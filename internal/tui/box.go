package tui

// Псевдографика (Python wsm_render.py). Каждая — одиночная руна.
const (
	HL = '─' // U+2500 horizontal line
	VL = '│' // U+2502 vertical line
	UL = '┌' // U+250C upper-left corner
	UR = '┐' // U+2510 upper-right corner
	LL = '└' // U+2514 lower-left corner
	LR = '┘' // U+2518 lower-right corner
	LT = '├' // U+251C left T-junction
	RT = '┤' // U+2524 right T-junction
	TT = '┬' // U+252C top T-junction
	BT = '┴' // U+2534 bottom T-junction
)

// DrawBox возвращает height строк по width рун — рамку. Пустой title — рамка
// без заголовка; иначе title пишется как " <title> " по центру верхней грани
// (эталон draw_box). Выходная длина каждой строки РОВНО width рун.
//
// godoc обязателен по этому пункту: в Python нижний-правый угол рисовался в
// обход safe_addstr (курсоры падали бы на cell h-1,w-1); в string-представлении
// курсора нет, поэтому угол LR рисуется всегда, если заявлен. Это структурное
// (не контрактное) расхождение, задокументировать словами выше.
func DrawBox(width, height int, title string) []string {
	return gridToStrings(boxGrid(width, height, title))
}

// DrawVDivider возвращает вертикальный разделитель из height строк: TT сверху,
// BT снизу, VL между ними. Каждый элемент — ОДНА руна (одна колонка); столбец
// задаёт вызывающий при композиции.
func DrawVDivider(height int) []string {
	out := make([]string, height)
	if height < 1 {
		return out
	}
	for i := 0; i < height; i++ {
		switch {
		case i == 0:
			out[i] = string(TT)
		case i == height-1:
			out[i] = string(BT)
		default:
			out[i] = string(VL)
		}
	}
	return out
}

// DrawBar возвращает одну строку: LT + HL*(width-2) + RT (эталон draw_bar).
func DrawBar(width int) string {
	if width < 1 {
		return ""
	}
	if width == 1 {
		return string(LT)
	}
	var b []rune
	b = append(b, LT)
	for i := 0; i < width-2; i++ {
		b = append(b, HL)
	}
	b = append(b, RT)
	return string(b)
}

// Fit усекает s до maxWidth рун (семантика safe_addstr: не выйти за край).
// Отрицательный maxWidth → "" (эталон давал ” при w-x-1 < 0).
func Fit(s string, maxWidth int) string {
	if maxWidth < 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= maxWidth {
		return s
	}
	return string(r[:maxWidth])
}

// boxGrid строит сетку рамки (не пишет курсором, без terminal-ограничений).
// Углы расставлены с приоритетом UL > UR > LL > LR, поэтому в вырожденном
// случае 1×1 (все углы занимают одну клетку) остаётся UL.
func boxGrid(width, height int, title string) [][]rune {
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	grid := make([][]rune, height)
	for i := range grid {
		grid[i] = make([]rune, width)
		for j := range grid[i] {
			grid[i][j] = ' '
		}
	}
	// Горизонтальные грани (верх и низ).
	for j := 0; j < width; j++ {
		grid[0][j] = HL
		grid[height-1][j] = HL
	}
	// Вертикальные грани (лево и право).
	for i := 0; i < height; i++ {
		grid[i][0] = VL
		grid[i][width-1] = VL
	}
	// Углы: первый выигрывает, UL — наивысший приоритет (для 1×1 остаётся UL).
	setCorner := func(i, j int, r rune) {
		if grid[i][j] != UL && grid[i][j] != UR && grid[i][j] != LL && grid[i][j] != LR {
			grid[i][j] = r
		}
	}
	setCorner(0, 0, UL)
	setCorner(0, width-1, UR)
	setCorner(height-1, 0, LL)
	setCorner(height-1, width-1, LR)
	// Заголовок по центру верхней грани (эталон draw_box).
	if title != "" {
		t := " " + title + " "
		tr := []rune(t)
		tx := (width - len(tr)) / 2
		if tx >= 0 {
			for k, ch := range tr {
				if tx+k >= width {
					break
				}
				grid[0][tx+k] = ch
			}
		}
	}
	return grid
}

// gridToStrings конвертирует сетку рун в строки.
func gridToStrings(grid [][]rune) []string {
	out := make([]string, len(grid))
	for i := range grid {
		out[i] = string(grid[i])
	}
	return out
}
