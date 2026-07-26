#!/usr/bin/env python3
"""WSM Workspace Manager — interactive curses TUI (MC / Norton style)."""

import curses
import os
import re
import subprocess
import sys
from pathlib import Path

from wsm_core import (CONF_DIR, VERSION, parse_toml, is_mounted,
                       load_projects, wsm_cli, check_net, match_key)
from wsm_render import (safe_addstr, draw_box, draw_vdivider, draw_bar,
                         HL, VL, UL, UR, LL, LR, LT, RT, TT, BT)

MIN_W, MIN_H = 60, 16


# ── Dialogs ────────────────────────────────────────────────────────

def create_config_dialog(stdscr, initial=None):
    """Create or edit project config.  Pass initial dict to pre-fill fields."""
    max_h, max_w = stdscr.getmaxyx()
    dh, dw = 14, min(62, max_w - 2)
    y0 = max(0, (max_h - dh) // 2)
    x0 = max(0, (max_w - dw) // 2)
    win = curses.newwin(dh, dw, y0, x0)
    win.keypad(True)
    curses.curs_set(1)

    fields = ['Alias:', 'Remote (alias:/path):', 'Local mount:', 'Editor command:']
    title = 'Edit Config' if initial else 'Create Config'
    if initial:
        remote = initial.get('remote_path', '')
        local = initial.get('local_mount', '')
        editor = initial.get('editor_cmd', 'zed')
        values = [initial.get('name', ''), remote, local, editor]
    else:
        values = ['', '', '', 'zed']
    cur = 0
    msg = ''

    while True:
        win.erase()
        draw_box(win, 0, 0, dh, dw, title, curses.color_pair(3))

        for i, label in enumerate(fields):
            y = 2 + i * 3
            safe_addstr(win, y, 3, label)
            display = values[i][:dw - 10]
            if i == cur:
                display += '█'
            attr = curses.A_REVERSE if i == cur else 0
            safe_addstr(win, y + 1, 3, f'  {display}', attr)

        safe_addstr(win, dh - 2, 2, 'Tab:next  Enter:save  Esc:cancel')
        if msg:
            safe_addstr(win, dh - 3, 2, msg, curses.color_pair(6))
        win.refresh()
        key = win.getch()

        if key == 27:  # Esc
            curses.curs_set(0); return None
        elif key == 9:  # Tab
            cur = (cur + 1) % len(fields); msg = ''
        elif key == 10:  # Enter — save
            alias = ''.join(c for c in values[0] if c.isalnum() or c in '_-')
            if not alias:
                msg = 'Alias required (a-z, 0-9, _, -)'
                continue
            if values[1].count(':') != 1 or not values[1].split(':')[0]:
                msg = 'Remote must be alias:/path'
                continue
            if not values[2].startswith('/'):
                msg = 'Local mount must be absolute path'
                continue
            curses.curs_set(0)
            return {
                'alias': alias,
                'remote_path': values[1],
                'local_mount': values[2],
                'editor_cmd': values[3] or 'zed',
            }
        elif key in (curses.KEY_BACKSPACE, 127, 8):
            values[cur] = values[cur][:-1]; msg = ''
        elif 32 <= key <= 126:
            values[cur] += chr(key); msg = ''


def generate_key_dialog(stdscr):
    max_h, max_w = stdscr.getmaxyx()
    dh, dw = 9, 55
    win = curses.newwin(dh, dw, (max_h - dh) // 2, (max_w - dw) // 2)
    win.keypad(True)
    curses.curs_set(1)
    keyname = ''

    while True:
        win.erase()
        draw_box(win, 0, 0, dh, dw, 'Generate ED25519 Key', curses.color_pair(3))
        safe_addstr(win, 2, 3, 'Key name:')
        safe_addstr(win, 3, 3, f'  {keyname}█')
        safe_addstr(win, 6, 2, 'Enter:generate  Esc:cancel')
        win.refresh()
        key = win.getch()

        if key == 27:
            curses.curs_set(0); return
        elif key == 10:
            if not keyname.strip(): continue
            clean = Path(keyname).stem
            clean = re.sub(r'\.(key|pub)$', '', clean)
            if not clean: continue
            private = CONF_DIR / f'{clean}.key'
            public = CONF_DIR / f'{clean}.pub'
            if private.exists(): continue
            subprocess.run(
                ['ssh-keygen', '-t', 'ed25519', '-f', str(private),
                 '-C', f'wsm-{clean}', '-N', ''],
                stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
            )
            subprocess.run(['mv', f'{private}.pub', str(public)],
                          stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
            win.erase()
            draw_box(win, 0, 0, dh, dw, 'Key Generated', curses.color_pair(3))
            safe_addstr(win, 2, 3, f'Private: {private}', curses.color_pair(1))
            safe_addstr(win, 3, 3, f'Public:  {public}', curses.color_pair(1))
            safe_addstr(win, 5, 2, '')
            safe_addstr(win, 6, dw - 15, 'Press any key')
            win.refresh()
            curses.curs_set(0); win.getch(); return
        elif key in (curses.KEY_BACKSPACE, 127, 8):
            keyname = keyname[:-1]
        elif 32 <= key <= 126:
            keyname += chr(key)


def desktop_snippet(stdscr, project):
    max_h, max_w = stdscr.getmaxyx()
    lines = [
        f'Actions={project};', '',
        f'[Desktop Action {project}]',
        f'Name=Open Remote: {project}',
        f'Exec=bash -c \'source $HOME/.wsm && wsm run {project}\'',
        f'Identifier={project}',
    ]
    dh = min(len(lines) + 5, max_h - 2)
    dw = min(max(len(max(lines, key=len)) + 4, 50), max_w - 2)
    win = curses.newwin(dh, dw, (max_h - dh) // 2, (max_w - dw) // 2)
    win.erase()
    draw_box(win, 0, 0, dh, dw, 'Desktop Action', curses.color_pair(3))
    for i, line in enumerate(lines):
        safe_addstr(win, 2 + i, 3, line)
    safe_addstr(win, dh - 1, dw - 15, 'Press any key')
    win.refresh(); win.getch()


def help_dialog(stdscr):
    max_h, max_w = stdscr.getmaxyx()
    rows = [
        ('F3', True,  'm', 'Mount',       'F6', True,  'u', 'Unmount'),
        ('F4', True,  'r', 'Run',         'F7', True,  'c', 'New config'),
        ('F5', True,  's', 'Connect',     'F8', True,  'e', 'Edit config'),
        ('Del',True,  'x', 'Delete',      'F9', True,  'd', 'Desktop'),
        ('F1', False, '',  'Help',        'F10',True,  'q', 'Quit'),
    ]
    lines = [
        '',
        '  Tab / ←→  Switch panels',
        '  ↑↓ / j k  Navigate',
        '  Enter     Execute',
        '  Esc       Back to left panel',
        '',
    ]
    for k1, s1, c1, a1, k2, s2, c2, a2 in rows:
        if s1:
            left = f'  {k1:<4}  / {c1}  {a1:<10}'
        else:
            left = f'  {k1:<4}        {a1:<10}'
        if s2:
            right = f'{k2:<4}  / {c2}  {a2:<10}'
        else:
            right = f'{k2:<4}        {a2:<10}'
        lines.append(f'{left}  {right}')
    lines += [
        '',
        f'Configs: {CONF_DIR}',
        f'Min size: {MIN_W}x{MIN_H}',
    ]
    dh = min(len(lines) + 4, max_h - 2)
    content_w = max((len(ln) for ln in lines), default=40) + 6
    dw = min(max(content_w, 58), max_w - 2)
    win = curses.newwin(dh, dw, (max_h - dh) // 2, (max_w - dw) // 2)
    win.erase()
    draw_box(win, 0, 0, dh, dw, 'Help', curses.color_pair(3))
    for i, line in enumerate(lines):
        attr = curses.A_BOLD if i == 0 else 0
        safe_addstr(win, 1 + i, 3, line, attr)
    safe_addstr(win, dh - 1, dw - 15, 'Press any key')
    win.refresh(); win.getch()


def too_small_dialog(stdscr):
    max_h, max_w = stdscr.getmaxyx()
    msg = f'Terminal too small ({max_w}x{max_h}). Need {MIN_W}x{MIN_H}+.'
    lines = ['Terminal Too Small', '', msg, '', 'Press q to quit, any key to retry.']
    dh, dw = len(lines) + 3, max(60, max_w - 2)
    win = curses.newwin(dh, dw, (max_h - dh) // 2, (max_w - dw) // 2)
    win.erase()
    draw_box(win, 0, 0, dh, dw, 'Error', curses.color_pair(4))
    for i, line in enumerate(lines):
        safe_addstr(win, 1 + i, 2, line, curses.A_BOLD if i == 0 else 0)
    win.refresh()
    return win.getch()


# ── Main TUI ────────────────────────────────────────────────────────

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
            key = too_small_dialog(stdscr)
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
                # Fixed layout: prefix + name (padded) + space + icon at right edge
                pad = inner_w - 5 - len(name)
                if pad < 1: pad = 1
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
        # Normalize: string chars → ord, special keys stay as int
        if isinstance(key, str):
            key = ord(key)

        if key in (27,) or match_key(key, 'q', 'Q', 'й', 'Й') or key == curses.KEY_F10:
            break
        if key == curses.KEY_F1:
            help_dialog(stdscr); needs_refresh = True; continue
        if focus == 'left' and key in (curses.KEY_RIGHT, ord('\t')):
            focus = 'right'; continue
        if focus == 'right' and key in (curses.KEY_LEFT, ord('\t'), 27):
            focus = 'left'; continue

        needs_refresh |= _handle_key(key, projects, pidx, aidx, focus, stdscr)

        # Navigation
        num_ra = 8  # right-panel action count (3 context + 5 tools)
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


def _handle_key(key, projects, pidx, _aidx, _focus, stdscr):
    """Handle F-key and single-key shortcuts."""
    fkeys = {
        curses.KEY_F3: 'mount', curses.KEY_F4: 'run', curses.KEY_F5: 'connect',
        curses.KEY_F6: 'unmount', curses.KEY_F7: 'new', curses.KEY_F8: 'keys',
        curses.KEY_F9: 'desktop', curses.KEY_F10: 'quit',
        curses.KEY_DC: 'delete',
    }
    singles = {}
    # Single-key shortcuts with Russian layout fallbacks
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
    """F-key or single-key action."""
    if not projects:
        return False
    project = projects[pidx]['name']

    if action == 'new':
        data = create_config_dialog(stdscr)
        if data:
            # Validate
            alias = data['remote_path'].split(':')[0]
            ok, err = check_net(alias)
            if not ok:
                _show_msg(stdscr, f'Network check FAILED: {err}')
                return True
            conf_file = CONF_DIR / f'{data["alias"]}.conf'
            CONF_DIR.mkdir(parents=True, exist_ok=True)
            with open(conf_file, 'w') as f:
                f.write(f'remote_path = "{data["remote_path"]}"\n')
                f.write(f'local_mount = "{data["local_mount"]}"\n')
                f.write(f'editor_cmd = "{data["editor_cmd"]}"\n')
        stdscr.erase(); stdscr.refresh(); return True
    elif action == 'keys':
        generate_key_dialog(stdscr)
        stdscr.erase(); stdscr.refresh(); return True
    elif action == 'desktop':
        desktop_snippet(stdscr, project)
        stdscr.erase(); stdscr.refresh(); return True
    elif action == 'delete':
        return _delete_config(projects, pidx, stdscr)
    elif action == 'edit':
        return _edit_config(projects, pidx, stdscr)
    else:
        # mount / run / connect / unmount
        r = wsm_cli(action, project)
        if r[2] != 0 and r[1]:
            _show_msg(stdscr, r[1].strip().split('\n')[-1][:60])
        return True


def _show_msg(stdscr, msg):
    h, w = stdscr.getmaxyx()
    dh, dw = 6, min(len(msg) + 6, w - 2)
    win = curses.newwin(dh, dw, (h - dh) // 2, (w - dw) // 2)
    win.erase()
    draw_box(win, 0, 0, dh, dw, 'Message', curses.color_pair(6))
    safe_addstr(win, 2, 3, msg[:dw - 6])
    safe_addstr(win, dh - 1, dw - 15, 'Press any key')
    win.refresh(); win.getch()


def _do_action(aidx, projects, pidx, stdscr):
    """Right-panel action via Enter."""
    project = projects[pidx]['name']
    mounted = projects[pidx]['mounted']

    if not mounted:
        context = {0: 'mount', 1: 'run', 2: 'connect'}
    else:
        context = {0: 'unmount', 1: 'run', 2: 'connect'}

    if aidx < 3:
        if aidx in context:
            r = wsm_cli(context[aidx], project)
            if r[2] != 0 and r[1]:
                _show_msg(stdscr, r[1].strip().split('\n')[-1][:60])
        return True

    if aidx == 3:
        data = create_config_dialog(stdscr)
        if data:
            alias = data['remote_path'].split(':')[0]
            ok, err = check_net(alias)
            if not ok:
                _show_msg(stdscr, f'Network check FAILED: {err}')
                return True
            conf_file = CONF_DIR / f'{data["alias"]}.conf'
            CONF_DIR.mkdir(parents=True, exist_ok=True)
            with open(conf_file, 'w') as f:
                f.write(f'remote_path = "{data["remote_path"]}"\n')
                f.write(f'local_mount = "{data["local_mount"]}"\n')
                f.write(f'editor_cmd = "{data["editor_cmd"]}"\n')
        stdscr.erase(); stdscr.refresh(); return True
    elif aidx == 4:
        return _edit_config(projects, pidx, stdscr)
    elif aidx == 5:
        generate_key_dialog(stdscr)
        stdscr.erase(); stdscr.refresh(); return True
    elif aidx == 6:
        desktop_snippet(stdscr, project)
        stdscr.erase(); stdscr.refresh(); return True
    elif aidx == 7:
        return _delete_config(projects, pidx, stdscr)
    return False


def _edit_config(projects, pidx, stdscr):
    """Edit selected project config — pre-fills fields from existing config."""
    if not projects:
        return False
    proj = projects[pidx]
    initial = {
        'name': proj['name'],
        'remote_path': proj['remote_path'] or '',
        'local_mount': proj['local_mount'] or '',
        'editor_cmd': proj['editor_cmd'] or 'zed',
    }
    data = create_config_dialog(stdscr, initial)
    if data:
        alias = data['remote_path'].split(':')[0]
        ok, err = check_net(alias)
        if not ok:
            _show_msg(stdscr, f'Network check FAILED: {err}')
            stdscr.erase(); stdscr.refresh()
            return True
        old_conf = Path(proj['conf'])
        new_conf = CONF_DIR / f'{data["alias"]}.conf'
        if str(old_conf) != str(new_conf) and old_conf.exists():
            old_conf.unlink()
        CONF_DIR.mkdir(parents=True, exist_ok=True)
        with open(new_conf, 'w') as f:
            f.write(f'remote_path = "{data["remote_path"]}"\n')
            f.write(f'local_mount = "{data["local_mount"]}"\n')
            f.write(f'editor_cmd = "{data["editor_cmd"]}"\n')
    stdscr.erase(); stdscr.refresh()
    return True


def _delete_config(projects, pidx, stdscr):
    """Delete selected project config with confirmation."""
    if not projects:
        return False
    name = projects[pidx]['name']
    if projects[pidx]['mounted']:
        _show_msg(stdscr, f'Unmount {name} first before deleting.')
        return True

    max_h, max_w = stdscr.getmaxyx()
    dh, dw = 6, 55
    win = curses.newwin(dh, dw, (max_h - dh) // 2, (max_w - dw) // 2)
    win.keypad(True)

    lines = [
        f'Delete config "{name}"?',
        f'File: {CONF_DIR / (name + ".conf")}',
    ]
    choice = 1  # 0=Yes, 1=No

    while True:
        win.erase()
        draw_box(win, 0, 0, dh, dw, 'Confirm Delete', curses.color_pair(10))
        for i, line in enumerate(lines):
            safe_addstr(win, 1 + i, 3, line)
        for i, lbl in enumerate(['  Yes  ', '  No   ']):
            attr = curses.color_pair(7) | curses.A_BOLD if i == choice else 0
            safe_addstr(win, 3, 10 + i * 14, lbl, attr)
        win.refresh()

        key = win.get_wch()
        if isinstance(key, str):
            key = ord(key)
        if key in (curses.KEY_LEFT,) or match_key(key, 'h', 'H', 'р', 'Р'):
            choice = 0
        elif key in (curses.KEY_RIGHT,) or match_key(key, 'l', 'L', 'д', 'Д'):
            choice = 1
        elif key == 10:
            break
        elif key in (27,) or match_key(key, 'q', 'Q', 'й', 'Й', 'n', 'N', 'т', 'Т'):
            choice = 1; break
        elif match_key(key, 'y', 'Y', 'н', 'Н'):
            choice = 0; break

    if choice == 0:
        conf_file = projects[pidx]['conf']
        try:
            Path(conf_file).unlink()
        except Exception as e:
            _show_msg(stdscr, f'Delete failed: {e}')
    return True


if __name__ == '__main__':
    curses.wrapper(main)
