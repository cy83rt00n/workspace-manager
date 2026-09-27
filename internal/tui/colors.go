package tui

import "github.com/charmbracelet/lipgloss"

const (
	// MinW / MinH — минимальные размеры терминала (Python MIN_W/MIN_H).
	MinW = 60
	MinH = 16
	// SpinnerChars — спиннер (Python SPINNER_CHARS), используется моделью BRIEF-008.
	SpinnerChars = "⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏"
)

// Palette возвращает десять базовых стилей, зеркально curses.init_pair(1..10)
// из wsm_tui.py. Индекс 0 не используется (нулевой стиль). Порядок ДОСЛОВНО:
//
//	1  green  on black
//	2  white  on black
//	3  black  on cyan
//	4  black  on white
//	5  cyan   on black
//	6  yellow on black
//	7  black  on yellow
//	8  blue   on black
//	9  black  on blue
//	10 red    on black
//
// Жирный (A_BOLD) в эталоне — это БОЛЬНОЙ атрибут, не часть пары: вызывающий
// применяет .Bold(true) поверх базового стиля.
func Palette() [11]lipgloss.Style {
	// lipgloss.Foreground/.Background с ANSI-кодами цветов.
	//
	// ОТКЛОНЕНИЕ от промпта: эталон (и первоначальный текст промпта) задавал
	// имена цветов ("green", "white", ...), но lipgloss/termenv НЕ умеет
	// маппить имена в ANSI-коды (только hex "#rrggbb" или числовые коды),
	// поэтому Render(...) не давал бы никакого ANSI-выхода и тест
	// «Render содержит \x1b[» был бы красным. Здесь используются числовые
	// ANSI-коды того же визуального смысла (0=black,1=red,2=green,3=yellow,
	// 4=blue,6=cyan,7=white), полностью сохраняя документированную палитру.
	var p [11]lipgloss.Style
	fg := [11]lipgloss.Color{
		"", "2", "7", "0", "0", "6",
		"3", "0", "4", "0", "1",
	}
	bg := [11]lipgloss.Color{
		"", "0", "0", "6", "7", "0",
		"0", "3", "0", "4", "0",
	}
	for i := 1; i <= 10; i++ {
		p[i] = lipgloss.NewStyle().Foreground(fg[i]).Background(bg[i])
	}
	return p
}

// Pair возвращает палитровый вход i (1..10); вне диапазона — нулевой стиль.
func Pair(i int) lipgloss.Style {
	p := Palette()
	if i < 1 || i > 10 {
		return lipgloss.NewStyle()
	}
	return p[i]
}
