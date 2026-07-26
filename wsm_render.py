#!/usr/bin/env python3
"""WSM Workspace Manager — TUI rendering helpers."""

import curses

# Box-drawing characters
HL = '\u2500'; VL = '\u2502'
UL = '\u250c'; UR = '\u2510'; LL = '\u2514'; LR = '\u2518'
LT = '\u251c'; RT = '\u2524'; TT = '\u252c'; BT = '\u2534'


def safe_addstr(win, y, x, text, color=0):
    """Add string to window, safe from bottom-right-corner crash."""
    h, w = win.getmaxyx()
    if y < 0 or y >= h or x >= w:
        return
    if y == h - 1:
        text = text[:w - x - 1] if w - x - 1 >= 0 else ''
    else:
        text = text[:w - x] if w - x >= 0 else ''
    if text:
        win.addstr(y, x, text, color)


def draw_box(win, y, x, h, w, title='', color=0):
    """Draw a bordered box.  Returns (inner_y, inner_x, inner_h, inner_w)."""
    safe_addstr(win, y, x, UL, color)
    safe_addstr(win, y, x + w - 1, UR, color)
    safe_addstr(win, y + h - 1, x, LL, color)
    # Bottom-right corner: must bypass safe_addstr (it blocks h-1,w-1)
    try:
        win.addstr(y + h - 1, x + w - 1, LR, color)
    except curses.error:
        pass
    safe_addstr(win, y, x + 1, HL * (w - 2), color)
    safe_addstr(win, y + h - 1, x + 1, HL * (w - 2), color)
    for i in range(1, h - 1):
        safe_addstr(win, y + i, x, VL, color)
        safe_addstr(win, y + i, x + w - 1, VL, color)
    if title:
        t = f' {title} '
        tx = x + (w - len(t)) // 2
        safe_addstr(win, y, tx, t, color | curses.A_BOLD)
    return (y + 1, x + 1, h - 2, w - 2)


def draw_vdivider(win, y, x, h, color=0):
    """Draw T-junctions and vertical divider between panels."""
    safe_addstr(win, y, x, TT, color)
    safe_addstr(win, y + h - 1, x, BT, color)
    for i in range(1, h - 1):
        safe_addstr(win, y + i, x, VL, color)


def draw_bar(win, y, x, w, color=0):
    """Draw horizontal bar with L/R T-junctions (for status bar)."""
    safe_addstr(win, y, x, LT, color)
    safe_addstr(win, y, x + w - 1, RT, color)
    safe_addstr(win, y, x + 1, HL * (w - 2), color)


def show_msg(stdscr, msg):
    """Show a one-line message dialog."""
    h, w = stdscr.getmaxyx()
    dh, dw = 6, min(len(msg) + 6, w - 2)
    win = curses.newwin(dh, dw, (h - dh) // 2, (w - dw) // 2)
    win.erase()
    draw_box(win, 0, 0, dh, dw, 'Message', curses.color_pair(6))
    safe_addstr(win, 2, 3, msg[:dw - 6])
    safe_addstr(win, dh - 1, dw - 15, 'Press any key')
    win.refresh(); win.getch()


def too_small_dialog(stdscr, min_w, min_h):
    """Show terminal-too-small error dialog."""
    max_h, max_w = stdscr.getmaxyx()
    msg = f'Terminal too small ({max_w}x{max_h}). Need {min_w}x{min_h}+.'
    lines = ['Terminal Too Small', '', msg, '', 'Press q to quit, any key to retry.']
    dh, dw = len(lines) + 3, max(60, max_w - 2)
    win = curses.newwin(dh, dw, (max_h - dh) // 2, (max_w - dw) // 2)
    win.erase()
    draw_box(win, 0, 0, dh, dw, 'Error', curses.color_pair(4))
    for i, line in enumerate(lines):
        safe_addstr(win, 1 + i, 2, line, curses.A_BOLD if i == 0 else 0)
    win.refresh()
    return win.getch()

MIN_W, MIN_H = 60, 16


def help_dialog(stdscr, conf_dir, min_w, min_h):
    """Show help/reference dialog."""
    max_h, max_w = stdscr.getmaxyx()
    nav = [
        'Tab / ←→  Switch panels',
        '↑↓ / j k  Navigate',
        'Enter     Execute',
        'Esc       Back to left panel',
    ]
    fkeys = [
        ('F3',  True, 'm', 'Mount',       'F6',  True, 'u', 'Unmount'),
        ('F4',  True, 'r', 'Run',         'F7',  True, 'c', 'New config'),
        ('F5',  True, 's', 'Connect',     'F8',  True, 'e', 'Edit config'),
        ('Del', True, 'x', 'Delete',      'F9',  True, 'd', 'Desktop'),
        ('F1',  False,'',  'Help',        'F10', True, 'q', 'Quit'),
    ]
    lines = []
    for line in nav:
        lines.append(f'  {line}')
    lines.append('')
    for k1, s1, c1, a1, k2, s2, c2, a2 in fkeys:
        if s1:
            left = f'  {k1:<4}  / {c1}  {a1:<10}'
        else:
            left = f'  {k1:<4}        {a1:<10}'
        if s2:
            right = f'{k2:<4}  / {c2}  {a2:<10}'
        else:
            right = f'{k2:<4}        {a2:<10}'
        lines.append(f'{left}  {right}')
    lines += ['', f'  Configs:  {conf_dir}', f'  Min size: {min_w}\u00d7{min_h}']
    dh = min(len(lines) + 4, max_h - 2)
    content_w = max((len(ln) for ln in lines), default=40) + 6
    dw = min(max(content_w, 58), max_w - 2)
    win = curses.newwin(dh, dw, (max_h - dh) // 2, (max_w - dw) // 2)
    win.erase()
    draw_box(win, 0, 0, dh, dw, 'Help', curses.color_pair(3))
    for i, line in enumerate(lines):
        safe_addstr(win, 1 + i, 3, line, curses.A_BOLD if i == 0 else 0)
    safe_addstr(win, dh - 1, dw - 15, 'Press any key')
    win.refresh(); win.getch()


def render_form(stdscr, title, fields, initial=None):
    """Generic form dialog.  Returns dict of {field_label: value} or None.
    fields: list of (label, key).  initial: dict of key->value."""
    max_h, max_w = stdscr.getmaxyx()
    dh = min(len(fields) * 3 + 6, max_h - 2)
    dw = min(62, max_w - 2)
    y0 = max(0, (max_h - dh) // 2)
    x0 = max(0, (max_w - dw) // 2)
    win = curses.newwin(dh, dw, y0, x0)
    win.keypad(True)
    curses.curs_set(1)

    values = [initial.get(key, '') if initial else '' for _, key in fields]
    cur = 0
    msg = ''

    while True:
        win.erase()
        draw_box(win, 0, 0, dh, dw, title, curses.color_pair(3))
        for i, (label, _) in enumerate(fields):
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

        if key == 27:
            curses.curs_set(0); return None
        elif key == 9:
            cur = (cur + 1) % len(fields); msg = ''
        elif key == 10:
            result = {}
            for (_, fkey), val in zip(fields, values):
                if not val and fkey == 'keyname':
                    continue
                result[fkey] = val
            curses.curs_set(0)
            return result
        elif key in (curses.KEY_BACKSPACE, 127, 8):
            values[cur] = values[cur][:-1]; msg = ''
        elif 32 <= key <= 126:
            values[cur] += chr(key); msg = ''


def confirm_dialog(stdscr, title, lines):
    """Yes/No confirmation dialog.  Returns True if Yes chosen."""
    max_h, max_w = stdscr.getmaxyx()
    dh, dw = 6, 55
    win = curses.newwin(dh, dw, (max_h - dh) // 2, (max_w - dw) // 2)
    win.keypad(True)
    choice = 1  # 0=Yes, 1=No
    while True:
        win.erase()
        draw_box(win, 0, 0, dh, dw, title, curses.color_pair(10) if 'Delete' in title else curses.color_pair(3))
        for i, line in enumerate(lines):
            safe_addstr(win, 1 + i, 3, line)
        for i, lbl in enumerate(['  Yes  ', '  No   ']):
            attr = curses.color_pair(7) | curses.A_BOLD if i == choice else 0
            safe_addstr(win, 3, 10 + i * 14, lbl, attr)
        win.refresh()
        key = win.get_wch()
        if isinstance(key, str):
            key = ord(key)
        if key in (curses.KEY_LEFT,) or key in {ord(c) for c in 'hHрР'}:
            choice = 0
        elif key in (curses.KEY_RIGHT,) or key in {ord(c) for c in 'lLдД'}:
            choice = 1
        elif key == 10:
            break
        elif key in (27,) or key in {ord(c) for c in 'qQйЙnNтТ'}:
            choice = 1; break
        elif key in {ord(c) for c in 'yYнН'}:
            choice = 0; break
    return choice == 0
