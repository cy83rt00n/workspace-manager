package tui

import "fmt"

// tableRowLE fkeys-данные для help-таблицы: (key, hasHotkey, hotkey, action)
// для левой и правой колонок, ДОСЛОВНО из wsm_render.py.
type helpKey struct {
	key string
	has bool
	ch  rune
	act string
}

// helpRows возвращает пять строк клавиш help-диалога (две колонки) — чистый
// источник данных (без глобальной переменной состояния).
func helpRows() [][2]helpKey {
	return [][2]helpKey{
		{
			{key: "F3", has: true, ch: 'm', act: "Mount"},
			{key: "F6", has: true, ch: 'u', act: "Unmount"},
		},
		{
			{key: "F4", has: true, ch: 'r', act: "Run"},
			{key: "F7", has: true, ch: 'c', act: "New config"},
		},
		{
			{key: "F5", has: true, ch: 's', act: "Connect"},
			{key: "F8", has: true, ch: 'e', act: "Edit config"},
		},
		{
			{key: "Del", has: true, ch: 'x', act: "Delete"},
			{key: "F9", has: true, ch: 'd', act: "Desktop"},
		},
		{
			{key: "F1", has: false, act: "Help"},
			{key: "F10", has: true, ch: 'q', act: "Quit"},
		},
	}
}

// HelpLines возвращает содержимое help-диалога: nav-строки, пустую строку,
// пять строк таблицы клавиш, затем:
//
//	""                                  (пустая строка)
//	"  Configs:  <confDir>"
//	"  Min size: <minW>×<minH>"          (× = U+00D7)
func HelpLines(confDir string, minW, minH int) []string {
	nav := []string{
		"  Tab / ←→  Switch panels",
		"  ↑↓ / j k  Navigate",
		"  Enter     Execute",
		"  Esc       Back to left panel",
	}
	tables := helpRows()
	lines := make([]string, 0, len(nav)+len(tables)+4)
	for _, n := range nav {
		lines = append(lines, n)
	}
	lines = append(lines, "")
	for _, pair := range tables {
		left := formatHelpKey(pair[0], true)
		right := formatHelpKey(pair[1], false)
		lines = append(lines, left+"  "+right)
	}
	lines = append(lines, "", fmt.Sprintf("  Configs:  %s", confDir), fmt.Sprintf("  Min size: %d×%d", minW, minH))
	return lines
}

// formatHelpKey форматирует одну колонку ключа (с двумя ведущими пробелами для
// левой); порт f'{k:<4}  / {c}  {a:<10}' и f'{k:<4}        {a:<10}'.
func formatHelpKey(k helpKey, left bool) string {
	prefix := ""
	if left {
		prefix = "  "
	}
	if k.has {
		return prefix + fmt.Sprintf("%-4s  / %s  %-10s", k.key, string(k.ch), k.act)
	}
	return prefix + fmt.Sprintf("%-4s        %-10s", k.key, k.act)
}

// HelpDialog рендерит help-бокс title "Help". dh=min(len(lines)+4, height-2);
// content_w=max(len строк)+6; dw=min(max(content_w,58), width-2). Строки lines
// выводятся с y=1+i и x=3 (по 3 пробела). Внизу "Press any key" на
// (dh-1, dw-15).
func HelpDialog(width, height int, confDir string, minW, minH int) []string {
	lines := HelpLines(confDir, minW, minH)
	dh := min(len(lines)+4, height-2)
	contentW := maxLineLen(lines) + 6
	dw := min(max(contentW, 58), width-2)
	dm := dims(dh, dw)
	grid := boxGrid(dm.w, dm.h, "Help")
	for i, line := range lines {
		safePut(grid, 1+i, 3, line)
	}
	safePut(grid, dh-1, dw-15, "Press any key")
	return gridToStrings(grid)
}

// ConfirmDialog рендерит диалог Yes/No (dh=6, dw=55). title — заголовок рамки;
// lines — строки тела (y=1+i, x=3). Кнопки "  Yes  " на x=10 и "  No   " на
// x=24 (строка y=3); активная кнопка choice (0=Yes, 1=No) — визуально
// выделяется (документируй: в String-представлении выделение передаётся
// вызывающему через стиль — здесь только текст кнопок).
func ConfirmDialog(width, height int, title string, lines []string, choice int) []string {
	dh, dw := 6, 55
	grid := boxGrid(dw, dh, title)
	for i, line := range lines {
		safePut(grid, 1+i, 3, line)
	}
	safePut(grid, 3, 10, "  Yes  ")
	safePut(grid, 3, 24, "  No   ")
	return gridToStrings(grid)
}

// ConfirmInput применяет клавишу key к текущему choice и сообщает, закрылся ли
// диалог и каков итог. ДОСЛОВНО port confirm_dialog из wsm_render.py:
//
//	LEFT            или h H р Р   → choice=0 (Yes), не закрыт
//	RIGHT           или l L д Д   → choice=1 (No),  не закрыт
//	Enter                        → закрыт, accepted = (choice==0)
//	Esc  или q Q й Й n N т Т      → choice=1, закрыт, accepted=false (No)
//	                    y Y н Н   → choice=0, закрыт, accepted=true (Yes)
//	прочее                       → без изменений, не закрыт
//
// Внимание (паритет эталона, НЕ «фикс»): латинская 'n' закрывает как No,
// русская 'н' закрывает как Yes.
func ConfirmInput(choice int, key Input) (newChoice int, done bool, accepted bool) {
	switch key.Kind {
	case Left:
		return 0, false, false
	case Right:
		return 1, false, false
	case Enter:
		return choice, true, choice == 0
	case Esc:
		return 1, true, false
	case Char:
		switch key.Rune {
		case 'h', 'H', 'р', 'Р':
			return 0, false, false
		case 'l', 'L', 'д', 'Д':
			return 1, false, false
		case 'q', 'Q', 'й', 'Й', 'n', 'N', 'т', 'Т':
			return 1, true, false
		case 'y', 'Y', 'н', 'Н':
			return 0, true, true
		default:
			return choice, false, false
		}
	default:
		return choice, false, false
	}
}

// TooSmallLines возвращает 5 строк дословно:
//
//	"Terminal Too Small"
//	""
//	fmt.Sprintf("Terminal too small (%dx%d). Need %dx%d+.", width, height, minW, minH)
//	""
//	"Press q to quit, any key to retry."
//
// где width/height — фактические размеры терминала (эталон подставляет max_w/max_h).
func TooSmallLines(width, height, minW, minH int) []string {
	return []string{
		"Terminal Too Small",
		"",
		fmt.Sprintf("Terminal too small (%dx%d). Need %dx%d+.", width, height, minW, minH),
		"",
		"Press q to quit, any key to retry.",
	}
}

// TooSmallDialog рендерит бокс title "Error"; dh=len(lines)+3; dw=max(60,width-2);
// строки с y=1+i, x=2.
func TooSmallDialog(width, height, minW, minH int) []string {
	lines := TooSmallLines(width, height, minW, minH)
	dh := len(lines) + 3
	dw := max(60, width-2)
	dm := dims(dh, dw)
	grid := boxGrid(dm.w, dm.h, "Error")
	for i, line := range lines {
		safePut(grid, 1+i, 2, line)
	}
	return gridToStrings(grid)
}

// MessageDialog: бокс "Message" dh=6, dw=min(lenRunes(msg)+6, width-2); msg на
// (y=2, x=3), усечённый Fit(..., dw-6); "Press any key" на (dh-1, dw-15).
func MessageDialog(width, height int, msg string) []string {
	dh := 6
	dw := min(len([]rune(msg))+6, width-2)
	dm := dims(dh, dw)
	grid := boxGrid(dm.w, dm.h, "Message")
	safePut(grid, 2, 3, Fit(msg, dw-6))
	safePut(grid, dh-1, dw-15, "Press any key")
	return gridToStrings(grid)
}

// DesktopDialog: бокс "Desktop Action"; dh=min(len(lines)+5, height-2);
// dw=min(max(lenRunes(lines...)+4, 50), width-2); lines на (y=2+i, x=3);
// "Press any key" на (dh-1, dw-15). lines передаёт вызывающий (6 строк
// config.DesktopSnippetText).
func DesktopDialog(width, height int, lines []string) []string {
	dh := min(len(lines)+5, height-2)
	dw := min(max(maxLineLen(lines)+4, 50), width-2)
	dm := dims(dh, dw)
	grid := boxGrid(dm.w, dm.h, "Desktop Action")
	for i, line := range lines {
		safePut(grid, 2+i, 3, line)
	}
	safePut(grid, dh-1, dw-15, "Press any key")
	return gridToStrings(grid)
}

// maxLineLen возвращает максимальную длину строки (в рунах) среди lines.
func maxLineLen(lines []string) int {
	m := 0
	for _, l := range lines {
		if n := len([]rune(l)); n > m {
			m = n
		}
	}
	return m
}

// dims содержит безопасные (>=1) размеры диалога.
type dimsPair struct {
	h int
	w int
}

// dims клампит высоту и ширину диалога к минимуму 1.
func dims(h, w int) dimsPair {
	if h < 1 {
		h = 1
	}
	if w < 1 {
		w = 1
	}
	return dimsPair{h: h, w: w}
}

// safePut пишет text в grid по (y,x) с усечением, семантика safe_addstr:
// не выйти за край, обойдя bottom-right cell на нижней строке.
func safePut(grid [][]rune, y, x int, text string) {
	if len(grid) == 0 || y < 0 || y >= len(grid) || x < 0 || x >= len(grid[0]) {
		return
	}
	w := len(grid[0])
	limit := w - x
	if y == len(grid)-1 {
		limit = w - x - 1
	}
	if limit < 0 {
		return
	}
	r := []rune(text)
	if limit > len(r) {
		limit = len(r)
	}
	for k := 0; k < limit; k++ {
		grid[y][x+k] = r[k]
	}
}
