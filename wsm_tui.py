#!/usr/bin/env python3
"""WSM Workspace Manager — thin TUI client (MC / Norton style)."""

import curses
import os
import re
import subprocess
import sys
import threading
from pathlib import Path

from wsm_core import (CONF_DIR, VERSION, parse_toml, is_mounted,
                       load_projects, wsm_cli, check_net, match_key,
                       validate_config, save_config, can_delete_config,
                       delete_config_file, generate_keypair,
                       desktop_snippet_text)

from wsm_render import (safe_addstr, draw_box, draw_vdivider, draw_bar,
                         HL, VL, UL, UR, LL, LR, LT, RT, TT, BT,
                         show_msg, too_small_dialog, MIN_W, MIN_H,
                         help_dialog, render_form, confirm_dialog)

SPINNER_CHARS = '⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏'


# ── TUI-specific helpers ────────────────────────────────────────────

def spinner_modal(stdscr, action, project):
    """Run wsm_cli in thread while showing spinner modal."""
    result = [None, None, -1]
    def _run():
        stdout, stderr, rc = wsm_cli(action, project)
        result[0], result[1], result[2] = stdout, stderr, rc

    t = threading.Thread(target=_run, daemon=True)
    t.start()

    max_h, max_w = stdscr.getmaxyx()
    dh, dw = 5, 30
    win = curses.newwin(dh, dw, (max_h - dh) // 2, (max_w - dw) // 2)
    i = 0
    while t.is_alive():
        win.erase()
        draw_box(win, 0, 0, dh, dw, 'Working', curses.color_pair(6))
        safe_addstr(win, 2, 3, f'  {SPINNER_CHARS[i % len(SPINNER_CHARS)]}  Please wait...')
        win.refresh()
        i += 1
        curses.napms(80)
    t.join()
    return result[0], result[1], result[2]


def config_form(stdscr, initial=None):
    """Create/edit config form — thin wrapper over render_form with validation."""
    title = 'Edit Config' if initial else 'Create Config'
    fields = [
        ('Alias:', 'alias'),
        ('Remote (alias:/path):', 'remote'),
        ('Local mount:', 'mount'),
        ('Editor command:', 'editor'),
    ]
    if initial:
        prefill = {
            'alias': initial.get('name', ''),
            'remote': initial.get('remote_path', ''),
            'mount': initial.get('local_mount', ''),
            'editor': initial.get('editor_cmd', 'zed'),
        }
    else:
        prefill = {'editor': 'zed'}

    data = render_form(stdscr, title, fields, initial=prefill)
    if not data:
        return None

    alias_raw = data.get('alias', '')
    alias = ''.join(c for c in alias_raw if c.isalnum() or c in '_-')
    result = {
        'alias': alias,
        'remote_path': data.get('remote', ''),
        'local_mount': data.get('mount', ''),
        'editor_cmd': data.get('editor', 'zed'),
        '_raw_alias': alias_raw,
        '_raw_remote': data.get('remote', ''),
        '_raw_mount': data.get('mount', ''),
    }
    ok, err = validate_config(alias_raw, result['remote_path'], result['local_mount'])
    if not ok:
        show_msg(stdscr, err)
        return None
    return result


def generate_key_dialog(stdscr):
    """Key generation dialog — thin wrapper over render_form + core."""
    fields = [('Key name:', 'keyname')]
    data = render_form(stdscr, 'Generate ED25519 Key', fields)
    if not data or not data.get('keyname', '').strip():
        return
    try:
        private, public = generate_keypair(data['keyname'])
        show_msg(stdscr, f'Key generated.\nPrivate: {private}\nPublic:  {public}')
    except (ValueError, FileExistsError) as e:
        show_msg(stdscr, str(e))


def desktop_dialog(stdscr, project):
    """Show Desktop Action snippet — thin wrapper over core + render."""
    lines = desktop_snippet_text(project)
    max_h, max_w = stdscr.getmaxyx()
    dh = min(len(lines) + 5, max_h - 2)
    dw = min(max(len(max(lines, key=len)) + 4, 50), max_w - 2)
    win = curses.newwin(dh, dw, (max_h - dh) // 2, (max_w - dw) // 2)
    win.erase()
    draw_box(win, 0, 0, dh, dw, 'Desktop Action', curses.color_pair(3))
    for i, line in enumerate(lines):
        safe_addstr(win, 2 + i, 3, line)
    safe_addstr(win, dh - 1, dw - 15, 'Press any key')
    win.refresh(); win.getch()


# ── Main TUI loop ───────────────────────────────────────────────────

def main(stdscr):
    curses.curs_set(0)
    stdscr.keypad(True)

    curses.init_pair(1, curses.COLOR_GREEN, curses.COLOR_BLACK)
    curses.init_pair(2, curses.COLOR_WHITE, curses.COLOR_BLACK)
    curses.init_pair(3, curses.COLOR_BLACK, curses.COLOR_CYAN)
    curses.init_pair(4, curses.COLOR_BLACK, curses.COLOR_WHITE)
    curses.init_pair(5, curses.COLOR_CYAN, curses.COLOR_BLACK)
    curses.init_pair(6, curses.COLOR_YELLOW, curses.COLOR_BLACK)
    curses.init_pair(7, curses.COLOR_BLACK, curses.COLOR_YELLOW)
    curses.init_pair(8, curses.COLOR_BLUE, curses.COLOR_BLACK)
    curses.init_pair(9, curses.COLOR_BLACK, curses.COLOR_BLUE)
    curses.init_pair(10, curses.COLOR_RED, curses.COLOR_BLACK)

    projects = []
    pidx = 0; pscroll = 0; aidx = 0
    focus = 'left'
    needs_refresh = True
    message = ''

    while True:
        if needs_refresh:
            projects = load_projects()
            pidx = min(pidx, max(0, len(projects) - 1))
            needs_refresh = False

        h, w = stdscr.getmaxyx()

        if w < MIN_W or h < MIN_H:
            key = too_small_dialog(stdscr, MIN_W, MIN_H)
            if key in (27,) or match_key(key, 'q', 'Q', 'й', 'Й'):
                break
            stdscr.erase(); stdscr.refresh()
            needs_refresh = True
            continue

        stdscr.erase()

        # ── Geometry ──
        body_y, body_h = 0, h - 2
        divider_x = max(24, w // 2)
        left_w = divider_x
        right_w = w - divider_x - 1

        lborder = curses.color_pair(3) | (curses.A_BOLD if focus == 'left' else 0)
        rborder = curses.color_pair(3) | (curses.A_BOLD if focus == 'right' else 0)
        if focus != 'left':
            lborder = curses.color_pair(5)
        if focus != 'right':
            rborder = curses.color_pair(5)

        # ── Left panel ──
        draw_box(stdscr, body_y, 0, body_h, left_w, 'Projects', lborder)
        inner_y, inner_x, visible_h, inner_w = body_y + 1, 1, body_h - 2, left_w - 2

        if not projects:
            safe_addstr(stdscr, inner_y + 1, inner_x + 1, 'No projects.', curses.color_pair(2))
            safe_addstr(stdscr, inner_y + 2, inner_x + 1, f'Configs: {CONF_DIR}')
            safe_addstr(stdscr, inner_y + 3, inner_x + 1, 'F7 / c — create one')
        else:
            if pidx < pscroll:
                pscroll = pidx
            elif pidx >= pscroll + visible_h:
                pscroll = pidx - visible_h + 1

            for i, proj in enumerate(projects[pscroll:pscroll + visible_h]):
                y = inner_y + i
                if y >= inner_y + visible_h:
                    break
                real_idx = pscroll + i
                sel = real_idx == pidx and focus == 'left'

                icon = '●' if proj['mounted'] else '○'
                icolor = curses.color_pair(1) | curses.A_BOLD if proj['mounted'] else curses.color_pair(2)

                if sel:
                    stdscr.addstr(y, inner_x, ' ' * (left_w - 2), curses.color_pair(4))
                    line_attr = curses.color_pair(4)
                else:
                    line_attr = 0

                prefix = '▶' if sel else ' '
                name = proj['name'][:inner_w - 6]
                pad = max(1, inner_w - 5 - len(name))
                safe_addstr(stdscr, y, inner_x + 1,
                            f'{prefix} {name}{" " * pad}{icon}', line_attr)

            summ = f' {len(projects)} project(s) '
            safe_addstr(stdscr, body_y + body_h - 2, 1, summ[:left_w - 2], curses.color_pair(6))

        # ── Divider ──
        draw_vdivider(stdscr, body_y, divider_x, body_h, curses.color_pair(3) | curses.A_BOLD)

        # ── Right panel ──
        draw_box(stdscr, body_y, divider_x + 1, body_h, right_w, 'Actions', rborder)
        rx, rw = divider_x + 2, right_w - 2

        if projects:
            sel = projects[pidx]
            ry = body_y + 1

            if sel['mounted']:
                actions = ['Unmount', 'Browse', 'Connect']
            else:
                actions = ['Mount', 'Run', 'Connect']

            for i, act in enumerate(actions):
                y = ry + i * 2
                if y >= body_y + body_h - 10: break
                highlight = focus == 'right' and i == aidx
                label = f'[ {act:<10} ]'
                if highlight:
                    safe_addstr(stdscr, y, rx, label, curses.color_pair(7) | curses.A_BOLD)
                else:
                    safe_addstr(stdscr, y, rx, label)

            info_y = ry + len(actions) * 2 + 1
            if info_y < body_y + body_h - 4:
                safe_addstr(stdscr, info_y, rx, '─' * min(rw, right_w - 4), curses.color_pair(5))
                info_y += 1

                remote = sel['remote_path'] or '-'
                if len(remote) > rw - 10: remote = remote[:rw - 13] + '…'
                safe_addstr(stdscr, info_y, rx, f'Remote: {remote}'[:rw])
                info_y += 1
                local = str(sel['local_mount'] or '-')
                if len(local) > rw - 9: local = local[:rw - 12] + '…'
                safe_addstr(stdscr, info_y, rx, f'Local:  {local}'[:rw])
                info_y += 1
                safe_addstr(stdscr, info_y, rx, f'Editor: {sel["editor_cmd"] or "-"}'[:rw])
                info_y += 1
                status = '● mounted' if sel['mounted'] else '○ idle'
                safe_addstr(stdscr, info_y, rx, f'Status: {status}'[:rw],
                            curses.color_pair(1) if sel['mounted'] else curses.color_pair(2))

            tools_y = ry + body_h - 13
            if tools_y > info_y + 2:
                safe_addstr(stdscr, tools_y, rx, '─' * min(rw, right_w - 4), curses.color_pair(5))
                tools = ['New config', 'Edit config', 'SSH keys', 'Desktop entry', 'Delete config']
                for i, tool in enumerate(tools):
                    y = tools_y + 1 + i * 2
                    if y >= body_y + body_h - 2: break
                    highlight = focus == 'right' and i + 3 == aidx
                    label = f'[ {tool:<14} ]'
                    if highlight:
                        safe_addstr(stdscr, y, rx, label, curses.color_pair(7) | curses.A_BOLD)
                    else:
                        safe_addstr(stdscr, y, rx, label)

        # ── Status bar ──
        fbar_y = h - 2
        draw_bar(stdscr, fbar_y, 0, w)
        stdscr.addstr(fbar_y + 1, 0, ' ' * (w - 1), curses.color_pair(8))
        mnt = sum(1 for p in projects if p['mounted'])
        status = f' {len(projects)} projects | {mnt} mounted | Tab:switch | F1:help | F10:quit '
        if message:
            status = f' {message} '
            message = ''
        safe_addstr(stdscr, fbar_y + 1, 1, status[:w - 2], curses.color_pair(9))

        stdscr.refresh()

        # ── Input ──
        key = stdscr.get_wch()
        if isinstance(key, str):
            key = ord(key)

        if key in (27,) or match_key(key, 'q', 'Q', 'й', 'Й') or key == curses.KEY_F10:
            break
        if key == curses.KEY_F1:
            help_dialog(stdscr, CONF_DIR, MIN_W, MIN_H); needs_refresh = True; continue

        # Panel switching
        if focus == 'left' and key in (curses.KEY_RIGHT, ord('\t')):
            focus = 'right'; continue
        if focus == 'right' and key in (curses.KEY_LEFT, ord('\t'), 27):
            focus = 'left'; continue

        # Global hotkeys
        needs_refresh |= _handle_key(key, projects, pidx, aidx, focus, stdscr)

        # Navigation
        num_ra = 8
        if focus == 'left':
            if key in (curses.KEY_DOWN,) or match_key(key, 'j', 'о'):
                pidx = min(pidx + 1, len(projects) - 1) if projects else 0
            elif key in (curses.KEY_UP,) or match_key(key, 'k', 'л'):
                pidx = max(pidx - 1, 0)
            elif key == 10 and projects:
                focus = 'right'; aidx = 0
        elif focus == 'right' and projects:
            if key in (curses.KEY_DOWN,) or match_key(key, 'j', 'о'):
                aidx = min(aidx + 1, num_ra - 1)
            elif key in (curses.KEY_UP,) or match_key(key, 'k', 'л'):
                aidx = max(aidx - 1, 0)
            elif key == 10:
                needs_refresh |= _do_action(aidx, projects, pidx, stdscr)

    curses.curs_set(1)


# ── Action dispatchers ──────────────────────────────────────────────

def _handle_key(key, projects, pidx, _aidx, _focus, stdscr):
    """Handle F-key and single-key shortcuts."""
    fkeys = {
        curses.KEY_F3: 'mount', curses.KEY_F4: 'run', curses.KEY_F5: 'connect',
        curses.KEY_F6: 'unmount', curses.KEY_F7: 'new', curses.KEY_F8: 'keys',
        curses.KEY_F9: 'desktop', curses.KEY_F10: 'quit',
        curses.KEY_DC: 'delete',
    }
    singles = {}
    for ch, act in [
        ('mMьЬ', 'mount'), ('rRкК', 'run'), ('sSыЫ', 'connect'),
        ('uUгГ', 'unmount'), ('cCсС', 'new'), ('eEуУ', 'edit'),
        ('kKлЛ', 'keys'), ('dDвВ', 'desktop'), ('xXчЧ', 'delete'),
    ]:
        for c in ch:
            singles[ord(c)] = act
    action = fkeys.get(key) or singles.get(key)
    if action:
        return _do_global_action(action, projects, pidx, stdscr)
    return False


def _do_global_action(action, projects, pidx, stdscr):
    if not projects:
        return False
    project = projects[pidx]['name']

    if action == 'new':
        data = config_form(stdscr)
        if data:
            alias = data['remote_path'].split(':')[0]
            ok, err = check_net(alias)
            if not ok:
                show_msg(stdscr, f'Network check FAILED: {err}')
                return True
            save_config(data['alias'], data['remote_path'],
                       data['local_mount'], data['editor_cmd'])
        stdscr.erase(); stdscr.refresh(); return True
    elif action == 'keys':
        generate_key_dialog(stdscr)
        stdscr.erase(); stdscr.refresh(); return True
    elif action == 'desktop':
        desktop_dialog(stdscr, project)
        stdscr.erase(); stdscr.refresh(); return True
    elif action == 'delete':
        return _delete_config(projects, pidx, stdscr)
    elif action == 'edit':
        return _edit_config(projects, pidx, stdscr)
    else:
        r = spinner_modal(stdscr, action, project)
        if r[2] != 0 and r[1]:
            show_msg(stdscr, r[1].strip().split('\n')[-1][:60])
        return True


def _do_action(aidx, projects, pidx, stdscr):
    project = projects[pidx]['name']
    mounted = projects[pidx]['mounted']

    if not mounted:
        context = {0: 'mount', 1: 'run', 2: 'connect'}
    else:
        context = {0: 'unmount', 1: 'run', 2: 'connect'}

    if aidx < 3:
        if aidx in context:
            r = spinner_modal(stdscr, context[aidx], project)
            if r[2] != 0 and r[1]:
                show_msg(stdscr, r[1].strip().split('\n')[-1][:60])
        return True

    if aidx == 3:
        data = config_form(stdscr)
        if data:
            alias = data['remote_path'].split(':')[0]
            ok, err = check_net(alias)
            if not ok:
                show_msg(stdscr, f'Network check FAILED: {err}')
                return True
            save_config(data['alias'], data['remote_path'],
                       data['local_mount'], data['editor_cmd'])
        stdscr.erase(); stdscr.refresh(); return True
    elif aidx == 4:
        return _edit_config(projects, pidx, stdscr)
    elif aidx == 5:
        generate_key_dialog(stdscr)
        stdscr.erase(); stdscr.refresh(); return True
    elif aidx == 6:
        desktop_dialog(stdscr, project)
        stdscr.erase(); stdscr.refresh(); return True
    elif aidx == 7:
        return _delete_config(projects, pidx, stdscr)
    return False


def _edit_config(projects, pidx, stdscr):
    if not projects:
        return False
    proj = projects[pidx]
    initial = {
        'name': proj['name'],
        'remote_path': proj['remote_path'] or '',
        'local_mount': proj['local_mount'] or '',
        'editor_cmd': proj['editor_cmd'] or 'zed',
    }
    data = config_form(stdscr, initial)
    if data:
        alias = data['remote_path'].split(':')[0]
        ok, err = check_net(alias)
        if not ok:
            show_msg(stdscr, f'Network check FAILED: {err}')
            stdscr.erase(); stdscr.refresh()
            return True
        save_config(data['alias'], data['remote_path'],
                   data['local_mount'], data['editor_cmd'],
                   old_conf=proj['conf'])
    stdscr.erase(); stdscr.refresh()
    return True


def _delete_config(projects, pidx, stdscr):
    if not projects:
        return False
    name = projects[pidx]['name']
    local_mount = projects[pidx]['local_mount']

    ok, reason = can_delete_config(projects[pidx]['conf'], local_mount)
    if not ok:
        show_msg(stdscr, reason)
        return True

    lines = [
        f'Delete config "{name}"?',
        f'File: {CONF_DIR / (name + ".config.toml")}',
    ]
    if confirm_dialog(stdscr, 'Confirm Delete', lines):
        try:
            delete_config_file(projects[pidx]['conf'], local_mount)
        except (ValueError, OSError) as e:
            show_msg(stdscr, f'Delete failed: {e}')
    return True


if __name__ == '__main__':
    curses.wrapper(main)
