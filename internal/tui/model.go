package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbletea"

	"github.com/cy83rt00n/workspace-manager/internal/config"
	"github.com/cy83rt00n/workspace-manager/internal/workspace"
)

const (
	// numRA — количество позиций правой панели (3 действия + 5 инструментов).
	numRA = 8
	// spinnerTimeout — таймаут асинхронного действия (паритет 60s Python
	// subprocess-timeout в wsm_cli).
	spinnerTimeout = 60 * time.Second
	// displayWidth/displayHeight — размер кадра до первого WindowSizeMsg.
	displayWidth  = 80
	displayHeight = 24
)

// configFields — четыре поля основной формы конфига (дословно wsm_tui.py).
var configFields = []Field{
	{Label: "Alias:", Key: "alias"},
	{Label: "Remote (alias:/path):", Key: "remote"},
	{Label: "Local mount:", Key: "mount"},
	{Label: "Editor command:", Key: "editor"},
}

// keysField — единственное поле диалога генерации ключей.
var keysField = []Field{
	{Label: "Key name:", Key: "keyname"},
}

// formMode — режим активной формы.
type formMode int

const (
	formNone formMode = iota
	formNew
	formEdit
	formKeys
)

// projectItem — проект списка: config.Project + вычисленный mounted-флаг
// (mounted не хранится в config.Project; он поставляется внешним IsMounted).
type projectItem struct {
	config.Project
	mounted bool
}

// Options — инжектируемые зависимости модели (для изоляции и тестов).
type Options struct {
	ConfigDir  string
	Engine     *workspace.Engine
	CheckNetFn func(ctx context.Context, alias string) (bool, string)
	KeygenFn   func(ctx context.Context, name string) (private, public string, err error)
	Version    string
}

// Model — состояние приложения (bubbletea-модель).
type Model struct {
	opts     Options
	messages chan string

	projects  []projectItem
	pidx      int // индекс выбранного проекта
	pscroll   int // верхняя строка видимой области списка
	aidx      int // индекс действия правой панели (0..7)
	focusLeft bool

	needsRefresh bool
	message      string
	quit         bool

	width, height int

	form            *FormState
	formMode        formMode
	formEditOldConf string
	confirm         *confirmState
	desktop         []string
	spinner         *spinnerState
	help            bool
	tooSmall        bool

	fake any // тестовый хук на fake-зависимости (в production всегда nil)
}

// confirmState — активный Yes/No-диалог (удаление конфига).
type confirmState struct {
	title  string
	lines  []string
	name   string
	choice int // 0=Yes, 1=No (дефолт No, как Python)
}

// spinnerState — полёт асинхронного действия (mount/unmount/run/connect).
type spinnerState struct {
	frame  int
	action string
}

// сработано: тип сообщений spinner'а (см. Update).
type spinnerTickMsg struct{}

type spinnerDoneMsg struct {
	err error
}

// New конструирует модель с needsRefresh=true (первый кадр грузит проекты).
func New(o Options) *Model {
	m := &Model{
		opts:      o,
		messages:  make(chan string, 16),
		formMode:  formNone,
		focusLeft: true, // паритет Python: focus = 'left'
	}
	m.needsRefresh = true
	m.updateSize(displayWidth, displayHeight)
	m.refresh()
	return m
}

// Messages возвращает канал вывода Engine (неблокирующие отправители ниже).
func (m *Model) Messages() chan string {
	return m.messages
}

// Out/ErrOut/Spin/SpinDone — методы-колбэки для workspace.Engine: форматируют и
// неблокирующе отправляют строку в канал messages (Python их печатал; TUI прячет
// их за спиннером — канал просто дренируется, финальная ошибка приходит через err).
func (m *Model) Out(format string, args ...any)    { m.sendEngineLine(fmt.Sprintf(format, args...)) }
func (m *Model) ErrOut(format string, args ...any) { m.sendEngineLine(fmt.Sprintf(format, args...)) }
func (m *Model) Spin(format string, args ...any)   { m.sendEngineLine(fmt.Sprintf(format, args...)) }
func (m *Model) SpinDone(format string, args ...any) {
	m.sendEngineLine(fmt.Sprintf(format, args...))
}

func (m *Model) sendEngineLine(s string) {
	select {
	case m.messages <- s:
	default:
	}
}

// drainMessages — чайная команда, дренирующая канал messages до конца программы.
func (m *Model) drainMessages() tea.Cmd {
	return func() tea.Msg {
		for range m.messages {
		}
		return nil
	}
}

// Init запускает подписку на канал вывода.
func (m *Model) Init() tea.Cmd {
	return m.drainMessages()
}

// Update применяет сообщение bubbletea.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch t := msg.(type) {
	case tea.WindowSizeMsg:
		m.updateSize(t.Width, t.Height)
		return m, nil
	case spinnerTickMsg:
		if m.spinner != nil {
			m.spinner.frame = (m.spinner.frame + 1) % len([]rune(SpinnerChars))
			return m, m.tickSpinner()
		}
		return m, nil
	case spinnerDoneMsg:
		m.spinner = nil
		m.needsRefresh = true
		if t.err != nil && t.err.Error() != "" {
			m.message = lastLine(t.err.Error())
		}
		return m, nil
	case tea.KeyMsg:
		return m, m.step(mapKey(t))
	}
	return m, nil
}

// tickSpinner — следующий тик спиннера (80ms), только пока spinner активен.
func (m *Model) tickSpinner() tea.Cmd {
	return tea.Tick(80*time.Millisecond, func(time.Time) tea.Msg {
		return spinnerTickMsg{}
	})
}

// View рендерит текущий кадр.
func (m *Model) View() string {
	if m.quit && m.spinner == nil {
		return ""
	}
	lines := m.viewReduced()
	return strings.Join(lines, "\n")
}

// refresh пересчитывает проекты и клампит индексы.
func (m *Model) refresh() {
	m.projects = loadProjects(m.opts)
	if len(m.projects) == 0 {
		m.pidx = 0
		m.pscroll = 0
		return
	}
	if m.pidx >= len(m.projects) {
		m.pidx = len(m.projects) - 1
	}
	m.needsRefresh = false
}

// loadProjects читает проекты и вычисляет mounted для каждого.
func loadProjects(o Options) []projectItem {
	projects, err := config.Projects(o.ConfigDir)
	if err != nil {
		return nil
	}
	out := make([]projectItem, 0, len(projects))
	for _, p := range projects {
		it := projectItem{Project: p}
		if o.Engine != nil && o.Engine.System != nil {
			it.mounted = o.Engine.System.IsMounted(p.LocalMount)
		}
		out = append(out, it)
	}
	return out
}
