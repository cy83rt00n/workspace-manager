package tui

import (
	"context"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/charmbracelet/bubbletea"

	"github.com/cy83rt00n/workspace-manager/internal/config"
)

// keyKind — тип абстрактной клавиши (для чистых тестов без bubbletea).
type keyKind int

const (
	keyRune keyKind = iota // печатная руна (латиница/кириллица)
	keyUp
	keyDown
	keyLeft
	keyRight
	keyEnter
	keyTab
	keyEsc
	keyBackspace
	keyDelete
	keyHome
	keyEnd
	keyF1
	keyF2
	keyF3
	keyF4
	keyF5
	keyF6
	keyF7
	keyF8
	keyF9
	keyF10
)

// keyEvent — абстрактная клавиша модели. Rune значим только при keyRune.
type keyEvent struct {
	kind keyKind
	rune rune
}

// keyRuneEvent возвращает символьную клавишу.
func keyRuneEvent(r rune) keyEvent { return keyEvent{kind: keyRune, rune: r} }

// mapKey преобразует tea.KeyMsg в keyEvent.
func mapKey(msg tea.KeyMsg) keyEvent {
	s := msg.String()
	switch s {
	case "up":
		return keyEvent{kind: keyUp}
	case "down":
		return keyEvent{kind: keyDown}
	case "left":
		return keyEvent{kind: keyLeft}
	case "right":
		return keyEvent{kind: keyRight}
	case "enter":
		return keyEvent{kind: keyEnter}
	case "tab", "shift+tab":
		return keyEvent{kind: keyTab}
	case "esc":
		return keyEvent{kind: keyEsc}
	case "backspace":
		return keyEvent{kind: keyBackspace}
	case "delete":
		return keyEvent{kind: keyDelete}
	case "home":
		return keyEvent{kind: keyHome}
	case "end":
		return keyEvent{kind: keyEnd}
	case "f1":
		return keyEvent{kind: keyF1}
	case "f2":
		return keyEvent{kind: keyF2}
	case "f3":
		return keyEvent{kind: keyF3}
	case "f4":
		return keyEvent{kind: keyF4}
	case "f5":
		return keyEvent{kind: keyF5}
	case "f6":
		return keyEvent{kind: keyF6}
	case "f7":
		return keyEvent{kind: keyF7}
	case "f8":
		return keyEvent{kind: keyF8}
	case "f9":
		return keyEvent{kind: keyF9}
	case "f10":
		return keyEvent{kind: keyF10}
	}
	r, _ := utf8.DecodeRuneInString(s)
	if r != utf8.RuneError && r != 0 && utf8.RuneCountInString(s) == 1 {
		return keyRuneEvent(r)
	}
	return keyEvent{}
}

// isRune проверяет, что клавиша — одна из перечисленных рун.
func isRune(k keyEvent, runes ...rune) bool {
	if k.kind != keyRune {
		return false
	}
	for _, r := range runes {
		if k.rune == r {
			return true
		}
	}
	return false
}

// isQuit — клавиша выхода (проверяется ПЕРВОЙ; Esc всегда выход — паритет-квирк).
func isQuit(k keyEvent) bool {
	return k.kind == keyEsc || k.kind == keyF10 || isRune(k, 'q', 'Q', 'й', 'Й')
}

// globalAction — таблица глобальных хоткеев (дословно _handle_key wsm_tui.py).
// Возвращает пустую строку, если клавиша не хоткей.
func globalAction(k keyEvent) string {
	switch k.kind {
	case keyF3:
		return "mount"
	case keyF4:
		return "run"
	case keyF5:
		return "connect"
	case keyF6:
		return "unmount"
	case keyF7:
		return "new"
	case keyF8:
		return "keys"
	case keyF9:
		return "desktop"
	case keyDelete:
		return "delete"
	}
	if k.kind != keyRune {
		return ""
	}
	switch k.rune {
	case 'm', 'M', 'ь', 'Ь':
		return "mount"
	case 'r', 'R', 'к', 'К':
		return "run"
	case 's', 'S', 'ы', 'Ы':
		return "connect"
	case 'u', 'U', 'г', 'Г':
		return "unmount"
	case 'c', 'C', 'с', 'С':
		return "new"
	case 'e', 'E', 'у', 'У':
		return "edit"
	case 'k', 'K', 'л', 'Л':
		return "keys"
	case 'd', 'D', 'в', 'В':
		return "desktop"
	case 'x', 'X', 'ч', 'Ч':
		return "delete"
	}
	return ""
}

// isNavDown/isNavUp — навигационные клавиши списков (кириллица включена).
func isNavDown(k keyEvent) bool { return k.kind == keyDown || isRune(k, 'j', 'о') }
func isNavUp(k keyEvent) bool   { return k.kind == keyUp || isRune(k, 'k', 'л') }

// step применяет одну клавишу (порт главного цикла wsm_tui.py). Порядок:
// модалы → выход → help → переключение панелей → глобальные хоткеи (до
// навигации — квирк k/л) → навигация. Возвращает команду для spinner.
func (m *Model) step(k keyEvent) tea.Cmd {
	if m.tooSmall {
		if isQuit(k) {
			m.quit = true
		} else {
			m.tooSmall = false
			m.needsRefresh = true
		}
		return nil
	}
	if m.help {
		m.help = false
		m.needsRefresh = true
		return nil
	}
	if m.desktop != nil {
		m.desktop = nil
		return nil
	}
	if m.confirm != nil {
		return m.stepConfirm(k)
	}
	if m.form != nil {
		return m.stepForm(k)
	}
	if m.spinner != nil {
		return nil
	}

	if isQuit(k) {
		m.quit = true
		return nil
	}
	if k.kind == keyF1 {
		m.help = true
		return nil
	}

	if m.focusLeft && (k.kind == keyRight || k.kind == keyTab) {
		m.focusLeft = false
		return nil
	}
	if !m.focusLeft && (k.kind == keyLeft || k.kind == keyTab) {
		m.focusLeft = true
		return nil
	}

	var cmd tea.Cmd
	if action := globalAction(k); action != "" {
		cmd = m.executeAction(action)
	}
	m.navigate(k)
	return cmd
}

// navigate — порт блока навигации главного цикла.
func (m *Model) navigate(k keyEvent) {
	if m.focusLeft {
		switch {
		case isNavDown(k):
			if len(m.projects) > 0 && m.pidx < len(m.projects)-1 {
				m.pidx++
			}
		case isNavUp(k):
			if m.pidx > 0 {
				m.pidx--
			}
		case k.kind == keyEnter:
			if len(m.projects) > 0 {
				m.focusLeft = false
				m.aidx = 0
			}
		}
		return
	}
	if len(m.projects) == 0 {
		return
	}
	switch {
	case isNavDown(k):
		if m.aidx < numRA-1 {
			m.aidx++
		}
	case isNavUp(k):
		if m.aidx > 0 {
			m.aidx--
		}
	case k.kind == keyEnter:
		m.executeAction(actionForIndex(m.projects[m.pidx].mounted, m.aidx))
	}
}

// actionForIndex — Enter-гетика правой панели (дословно wsm_tui.py). Квирк:
// label «Browse» при mounted диспатчит run.
func actionForIndex(mounted bool, aidx int) string {
	switch aidx {
	case 0:
		if mounted {
			return "unmount"
		}
		return "mount"
	case 1:
		return "run"
	case 2:
		return "connect"
	case 3:
		return "new"
	case 4:
		return "edit"
	case 5:
		return "keys"
	case 6:
		return "desktop"
	default:
		return "delete"
	}
}

// executeAction выполняет глобальное/Enter действие. Паритет-квирк: при пустом
// списке проектов НИЧЕГО не происходит (Python `_do_global_action` первым делом
// `if not projects: return False`), включая new/keys — создать первый проект из
// TUI нельзя, hint «F7 / c — create one» в Python не работает.
func (m *Model) executeAction(action string) tea.Cmd {
	if len(m.projects) == 0 {
		return nil
	}
	switch action {
	case "mount", "unmount", "connect":
		name := m.projects[m.pidx].Name
		editor := m.projects[m.pidx].EditorCmd
		return m.startAction(action, name, editor)
	case "run":
		return m.startAction(action, m.projects[m.pidx].Name, "")
	case "new":
		m.openConfigForm(formNew, nil)
	case "edit":
		m.openConfigForm(formEdit, &m.projects[m.pidx].Project)
	case "keys":
		m.openKeysForm()
	case "desktop":
		m.desktop = config.DesktopSnippetText(m.projects[m.pidx].Name)
	case "delete":
		m.openDeleteConfirm()
	}
	return nil
}

// startAction запускает асинхронное действие под спиннером (ctx 60s).
func (m *Model) startAction(action, name, editor string) tea.Cmd {
	m.spinner = &spinnerState{action: action}
	eng := m.opts.Engine
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), spinnerTimeout)
		defer cancel()
		var err error
		switch action {
		case "mount":
			err = eng.MountProject(ctx, name)
		case "unmount":
			err = eng.UnmountProject(ctx, name)
		case "run":
			err = eng.RunProject(ctx, name)
		case "connect":
			err = eng.ConnectProject(ctx, name, editor)
		}
		return spinnerDoneMsg{err: err}
	}
}

// openConfigForm открывает форму new/edit с prefill (editor по умолчанию "zed").
func (m *Model) openConfigForm(mode formMode, src *config.Project) {
	title := "Create Config"
	m.formMode = mode
	var prefill map[string]string
	switch mode {
	case formEdit:
		title = "Edit Config"
		if src == nil {
			m.form, m.formMode = nil, formNone
			return
		}
		editor := src.EditorCmd
		if editor == "" {
			editor = "zed"
		}
		prefill = map[string]string{
			"alias":  src.Name,
			"remote": src.RemotePath,
			"mount":  src.LocalMount,
			"editor": editor,
		}
		m.formEditOldConf = src.ConfPath
	default:
		prefill = map[string]string{"editor": "zed"}
		m.formEditOldConf = ""
	}
	m.form = NewFormState(title, configFields, prefill)
}

// openKeysForm открывает форму генерации ключей.
func (m *Model) openKeysForm() {
	m.formMode = formKeys
	m.form = NewFormState("Generate ED25519 Key", keysField, nil)
}

// openDeleteConfirm открывает confirm-диалог удаления (дефолт No).
func (m *Model) openDeleteConfirm() {
	if len(m.projects) == 0 {
		return
	}
	proj := m.projects[m.pidx]
	ok, reason := config.CanDeleteConfig(proj.ConfPath, proj.mounted)
	if !ok {
		m.message = reason
		return
	}
	m.confirm = &confirmState{
		title: "Confirm Delete",
		lines: []string{
			`Delete config "` + proj.Name + `"?`,
			"File: " + filepath.Join(m.opts.ConfigDir, proj.Name+config.ConfigSuffix),
		},
		name:   proj.Name,
		choice: 1,
	}
}

// stepConfirm обрабатывает клавишу в confirm-диалоге.
func (m *Model) stepConfirm(k keyEvent) tea.Cmd {
	choice, done, accepted := ConfirmInput(m.confirm.choice, toInput(k))
	if !done {
		m.confirm.choice = choice
		return nil
	}
	name := m.confirm.name
	m.confirm = nil
	m.needsRefresh = true
	if !accepted {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), spinnerTimeout)
	defer cancel()
	if err := m.opts.Engine.DeleteProject(ctx, name, true); err != nil && err.Error() != "" {
		m.message = lastLine(err.Error())
	}
	return nil
}

// stepForm обрабатывает клавишу в форме (new/edit/keys).
func (m *Model) stepForm(k keyEvent) tea.Cmd {
	result, done := m.form.HandleKey(toInput(k))
	if !done {
		return nil
	}
	m.needsRefresh = true
	if result == nil {
		m.form, m.formMode = nil, formNone
		return nil
	}

	if m.formMode == formKeys {
		m.form, m.formMode = nil, formNone
		name, has := result["keyname"]
		if !has || strings.TrimSpace(name) == "" {
			return nil
		}
		if m.opts.KeygenFn == nil {
			return nil
		}
		priv, pub, err := m.opts.KeygenFn(context.Background(), name)
		if err != nil {
			m.message = err.Error()
			return nil
		}
		m.message = "Key generated.\nPrivate: " + priv + "\nPublic:  " + pub
		return nil
	}

	aliasRaw := result["alias"]
	alias := sanitizeAlias(aliasRaw)
	remote := result["remote"]
	mountRaw := result["mount"]
	mountExpanded := config.ExpandMountPath(mountRaw)
	editor := result["editor"]
	if editor == "" {
		editor = "zed"
	}

	ok, errMsg := config.Validate(aliasRaw, remote, mountRaw)
	if !ok {
		m.form.Message = errMsg
		return nil
	}
	checkAlias := strings.SplitN(remote, ":", 2)[0]
	if m.opts.CheckNetFn != nil {
		if netOK, netErr := m.opts.CheckNetFn(context.Background(), checkAlias); !netOK {
			m.message = "Network check FAILED: " + netErr
			m.form, m.formMode = nil, formNone
			return nil
		}
	}
	oldConf := ""
	if m.formMode == formEdit {
		oldConf = m.formEditOldConf
	}
	if _, err := config.Save(m.opts.ConfigDir, alias, remote, mountExpanded, editor, oldConf); err != nil {
		m.message = lastLine(err.Error())
	}
	m.form, m.formMode = nil, formNone
	m.needsRefresh = true
	return nil
}

// toInput конвертирует keyEvent в Input формы.
func toInput(k keyEvent) Input {
	switch k.kind {
	case keyRune:
		return KeyChar(k.rune)
	case keyLeft:
		return KeyCode(Left)
	case keyRight:
		return KeyCode(Right)
	case keyBackspace:
		return KeyCode(Backspace)
	case keyDelete:
		return KeyCode(Delete)
	case keyHome:
		return KeyCode(Home)
	case keyEnd:
		return KeyCode(End)
	case keyTab:
		return KeyCode(Tab)
	case keyEnter:
		return KeyCode(Enter)
	case keyEsc:
		return KeyCode(Esc)
	default:
		return KeyChar(0) // игнорируемая клавиша (вне 32..126)
	}
}

// sanitizeAlias — фильтр алиаса как в config.Validate (unicode буквы/цифры + _-).
func sanitizeAlias(alias string) string {
	var b strings.Builder
	for _, r := range alias {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
