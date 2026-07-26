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
    safe_addstr(win, y + h - 1, x + w - 1, LR, color)
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
