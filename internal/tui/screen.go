package tui

import (
	"fmt"
	"strings"
)

// viewReduced возвращает текстовое представление текущего кадра (без
// терминала; используется и в View, и в тестах). View может быть вызвана
// повторно — она должна быть идемпотентной, кроме одноразового message.
func (m *Model) viewReduced() []string {
	if m.quit {
		return nil
	}
	if m.tooSmall {
		return TooSmallDialog(m.width, m.height, MinW, MinH)
	}
	if m.spinner != nil {
		return m.spinnerView()
	}
	if m.help {
		return HelpDialog(m.width, m.height, m.opts.ConfigDir, MinW, MinH)
	}
	if m.desktop != nil {
		return DesktopDialog(m.width, m.height, m.desktop)
	}
	if m.confirm != nil {
		return ConfirmDialog(m.width, m.height, m.confirm.title, m.confirm.lines, m.confirm.choice)
	}
	if m.form != nil {
		return m.form.View(m.width, m.height)
	}
	return m.mainView()
}

// spinnerView — модал спиннера (бокс 30×5 с надписью Please wait...).
func (m *Model) spinnerView() []string {
	const dh, dw = 5, 30
	grid := boxGrid(dw, dh, "Working")
	ch := '⠋'
	if m.spinner.frame >= 0 && m.spinner.frame < len([]rune(SpinnerChars)) {
		ch = []rune(SpinnerChars)[m.spinner.frame]
	}
	safePut(grid, 2, 3, fmt.Sprintf("  %c  Please wait...", ch))
	return gridToStrings(grid)
}

// mainView — основной двухпанельный кадр (дословная геометрия wsm_tui.py).
func (m *Model) mainView() []string {
	grid := newGrid(m.width, m.height)
	bodyH := m.height - 2
	if bodyH < 1 {
		bodyH = 1
	}
	divX := max(24, m.width/2)
	leftW := divX
	rightW := m.width - divX - 1
	if rightW < 1 {
		rightW = 1
	}

	// Левая панель.
	border := boxGrid(leftW, bodyH, "Projects")
	blit(grid, 0, 0, border)
	innerW := leftW - 2
	visibleH := bodyH - 2
	if len(m.projects) == 0 {
		if innerW > 0 {
			safePut(grid, 2, 2, "No projects.")
			safePut(grid, 3, 2, "Configs: "+m.opts.ConfigDir)
			safePut(grid, 4, 2, "F7 / c — create one")
		}
	} else {
		if m.pidx < m.pscroll {
			m.pscroll = m.pidx
		} else if m.pidx >= m.pscroll+visibleH {
			m.pscroll = m.pidx - visibleH + 1
		}
		for i := 0; i < visibleH; i++ {
			realIdx := m.pscroll + i
			y := 1 + i
			if y >= bodyH-1 || realIdx >= len(m.projects) {
				break
			}
			item := m.projects[realIdx]
			sel := realIdx == m.pidx && m.focusLeft
			prefix := " "
			if sel {
				prefix = "▶"
			}
			icon := "○"
			if item.mounted {
				icon = "●"
			}
			name := Fit(item.Name, innerW-6)
			padN := innerW - 5 - len([]rune(name))
			if padN < 1 {
				padN = 1
			}
			line := prefix + " " + name + strings.Repeat(" ", padN) + icon
			safePut(grid, y, 1, Fit(line, innerW))
		}
		y := 1 + bodyH - 2
		if y >= 0 && y < m.height {
			safePut(grid, y, 1, Fit(fmt.Sprintf(" %d project(s) ", len(m.projects)), leftW-2))
		}
	}

	// Разделитель.
	for i := 0; i < bodyH && i < m.height; i++ {
		grid[i][divX] = dividerRune(bodyH, i)
	}

	// Правая панель.
	if divX+1 < m.width {
		rborder := boxGrid(rightW, bodyH, "Actions")
		blit(grid, 0, divX+1, rborder)
	}
	rx := divX + 2
	rw := rightW - 2
	if rw < 1 {
		rw = 1
	}

	if len(m.projects) > 0 {
		sel := m.projects[m.pidx]
		actions := actionsFor(sel.mounted)
		for i, act := range actions {
			y := 1 + i*2
			if y >= bodyH-10 || y >= m.height {
				break
			}
			safePut(grid, y, rx, fmt.Sprintf("[ %-10s ]", act))
		}
		infoY := 1 + len(actions)*2 + 1 // = 8
		if infoY < bodyH-4 {
			safePut(grid, infoY, rx, strings.Repeat(string(HL), min(rw, rightW-4)))
			infoY++
			remote := sel.RemotePath
			if remote == "" {
				remote = "-"
			}
			if len([]rune(remote)) > rw-10 {
				r := []rune(remote)
				remote = string(r[:rw-13]) + "…"
			}
			safePut(grid, infoY, rx, Fit("Remote: "+remote, rw))
			infoY++
			local := sel.LocalMount
			if local == "" {
				local = "-"
			}
			if len([]rune(local)) > rw-9 {
				r := []rune(local)
				local = string(r[:rw-12]) + "…"
			}
			safePut(grid, infoY, rx, Fit("Local:  "+local, rw))
			infoY++
			editor := sel.EditorCmd
			if editor == "" {
				editor = "-"
			}
			safePut(grid, infoY, rx, Fit("Editor: "+editor, rw))
			infoY++
			status := "○ idle"
			if sel.mounted {
				status = "● mounted"
			}
			safePut(grid, infoY, rx, Fit("Status: "+status, rw))
			infoY++

			toolsY := 1 + bodyH - 13
			if toolsY > infoY {
				safePut(grid, toolsY, rx, strings.Repeat(string(HL), min(rw, rightW-4)))
				tools := []string{"New config", "Edit config", "SSH keys", "Desktop entry", "Delete config"}
				for i, tool := range tools {
					y := toolsY + 1 + i*2
					if y >= bodyH-2 || y >= m.height {
						break
					}
					safePut(grid, y, rx, fmt.Sprintf("[ %-14s ]", tool))
				}
			}
		}
	}

	// Статусбар: строка-бар на h-2, текст на h-1.
	if m.height-2 >= 0 {
		for j, r := range DrawBar(m.width) {
			if j < m.width {
				grid[m.height-2][j] = r
			}
		}
	}
	if m.height-1 >= 0 && m.width > 1 {
		text := fmt.Sprintf(" %d projects | %d mounted | Tab:switch | F1:help | F10:quit ",
			len(m.projects), countMounted(m.projects))
		if m.message != "" {
			text = " " + m.message + " "
			m.message = ""
		}
		safePut(grid, m.height-1, 1, Fit(text, m.width-2))
	}

	return gridToStrings(grid)
}

// countMounted считает смонтированные проекты.
func countMounted(items []projectItem) int {
	n := 0
	for _, it := range items {
		if it.mounted {
			n++
		}
	}
	return n
}

// actionsFor — подписи действий правой панели (дословно wsm_tui.py).
func actionsFor(mounted bool) []string {
	if mounted {
		return []string{"Unmount", "Browse", "Connect"}
	}
	return []string{"Mount", "Run", "Connect"}
}

// dividerRune — руна вертикального разделителя на строке i из height строк.
func dividerRune(height, i int) rune {
	switch {
	case i == 0:
		return TT
	case i == height-1:
		return BT
	default:
		return VL
	}
}

// newGrid — сетка width×height пробелов.
func newGrid(width, height int) [][]rune {
	grid := make([][]rune, height)
	for i := range grid {
		grid[i] = make([]rune, width)
		for j := range grid[i] {
			grid[i][j] = ' '
		}
	}
	return grid
}

// blit вписывает src поверх dst начиная с (y0,x0), по границе dst.
func blit(dst [][]rune, y0, x0 int, src [][]rune) {
	for i := range src {
		y := y0 + i
		if y < 0 || y >= len(dst) {
			continue
		}
		for j := range src[i] {
			x := x0 + j
			if x < 0 || x >= len(dst[y]) {
				continue
			}
			dst[y][x] = src[i][j]
		}
	}
}

// lastLine — последняя непустая строка s, обрезанная до 60 рун (порт Python
// stderr.strip().split('\n')[-1][:60]).
func lastLine(s string) string {
	lines := strings.Split(s, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if lines[i] != "" {
			return Fit(lines[i], 60)
		}
	}
	return ""
}

// updateSize — размер окна + флаг too-small guard.
func (m *Model) updateSize(w, h int) {
	m.width, m.height = w, h
	if w < MinW || h < MinH {
		m.tooSmall = true
	}
}
