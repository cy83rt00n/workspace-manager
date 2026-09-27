package tui

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cy83rt00n/workspace-manager/internal/config"
	"github.com/cy83rt00n/workspace-manager/internal/system"
	"github.com/cy83rt00n/workspace-manager/internal/workspace"
)

// fakeSystem — держит запись действий Engine для ассертов.
type fakeSystem struct {
	mounted      map[string]bool
	mountCalls   []string
	unmountCalls []string
	runCalls     int
	connectCalls []string
	checkOK      bool
	checkMsg     string
	mountErr     error
	unmountErr   error
	unmountAl    bool
	starts       []string
}

func (f *fakeSystem) CheckNet(ctx context.Context, alias string) (bool, string) {
	return f.checkOK, f.checkMsg
}

func (f *fakeSystem) IsMounted(path string) bool { return f.mounted[path] }

func (f *fakeSystem) Mount(ctx context.Context, remotePath, localMount string, poll func(int)) error {
	f.mountCalls = append(f.mountCalls, localMount)
	if poll != nil {
		poll(0)
	}
	return f.mountErr
}

func (f *fakeSystem) Unmount(ctx context.Context, localMount string, out func(string, ...any)) (bool, error) {
	f.unmountCalls = append(f.unmountCalls, localMount)
	if out != nil {
		out("Unmounting %s...", localMount)
	}
	return f.unmountAl, f.unmountErr
}

func (f *fakeSystem) Start(ctx context.Context, name string, args ...string) (*system.Proc, error) {
	f.starts = append(f.starts, name)
	return nil, nil
}

// newFakeModel строит модель с фейками и одним проектом demo (idle).
func newFakeModel(t *testing.T, projects []config.Project, mounted ...string) *Model {
	t.Helper()
	dir := t.TempDir()
	for _, p := range projects {
		if _, err := config.Save(dir, p.Name, p.RemotePath, p.LocalMount, p.EditorCmd, ""); err != nil {
			t.Fatalf("Save: %v", err)
		}
	}
	fs := &fakeSystem{mounted: map[string]bool{}}
	for _, m := range mounted {
		fs.mounted[m] = true
	}
	eng := &workspace.Engine{ConfigDir: dir, System: fs}
	m := New(Options{
		ConfigDir:  dir,
		Engine:     eng,
		CheckNetFn: func(context.Context, string) (bool, string) { return true, "" },
		KeygenFn: func(context.Context, string) (string, string, error) {
			return "/prv", "/pub", nil
		},
	})
	m.updateSize(80, 24)
	m.fake = fs
	return m
}

// demoProject — стандартная фикстура проекта.
func demoProject(name string) config.Project {
	return config.Project{
		Name:       name,
		RemotePath: "demo-alias:/opt/app",
		LocalMount: "/mnt/" + name,
		EditorCmd:  "zed",
	}
}

func TestGlobalHotkeyTable(t *testing.T) {
	project := demoProject("demo")
	actionCases := []struct {
		name string
		key  func() keyEvent
		act  string
	}{
		{"F3", func() keyEvent { return keyEvent{kind: keyF3} }, "mount"},
		{"F4", func() keyEvent { return keyEvent{kind: keyF4} }, "run"},
		{"F5", func() keyEvent { return keyEvent{kind: keyF5} }, "connect"},
		{"F6", func() keyEvent { return keyEvent{kind: keyF6} }, "unmount"},
		{"m", func() keyEvent { return keyRuneEvent('m') }, "mount"},
		{"M", func() keyEvent { return keyRuneEvent('M') }, "mount"},
		{"ь", func() keyEvent { return keyRuneEvent('ь') }, "mount"},
		{"Ь", func() keyEvent { return keyRuneEvent('Ь') }, "mount"},
		{"r", func() keyEvent { return keyRuneEvent('r') }, "run"},
		{"к", func() keyEvent { return keyRuneEvent('к') }, "run"},
		{"s", func() keyEvent { return keyRuneEvent('s') }, "connect"},
		{"ы", func() keyEvent { return keyRuneEvent('ы') }, "connect"},
		{"u", func() keyEvent { return keyRuneEvent('u') }, "unmount"},
		{"г", func() keyEvent { return keyRuneEvent('г') }, "unmount"},
	}
	for _, tc := range actionCases {
		t.Run(tc.name, func(t *testing.T) {
			m := newFakeModel(t, []config.Project{project})
			_ = m.step(tc.key())
			if m.spinner == nil || m.spinner.action != tc.act {
				t.Fatalf("%s: spinner.action = %v, want %s", tc.name,
					func() string {
						if m.spinner == nil {
							return "<nil>"
						}
						return m.spinner.action
					}(),
					tc.act)
			}
		})
	}
}

func TestHotkeyDialogsAndDelete(t *testing.T) {
	project := demoProject("demo")

	t.Run("new opens form", func(t *testing.T) {
		m := newFakeModel(t, []config.Project{project})
		m.step(keyRuneEvent('c'))
		if m.form == nil || m.formMode != formNew {
			t.Fatalf("форма new не открылась")
		}
		m.step(keyEvent{kind: keyEsc})
		if m.form != nil {
			t.Errorf("Esc не закрыл форму")
		}
	})
	t.Run("edit prefill", func(t *testing.T) {
		m := newFakeModel(t, []config.Project{project})
		m.step(keyRuneEvent('e'))
		if m.form == nil || m.formMode != formEdit {
			t.Fatalf("форма edit не открылась")
		}
		if m.form.Values[0] != "demo" || m.form.Values[2] != "/mnt/demo" {
			t.Errorf("prefill неверен: %v", m.form.Values)
		}
	})
	t.Run("keys opens keygen form", func(t *testing.T) {
		m := newFakeModel(t, []config.Project{project})
		m.step(keyEvent{kind: keyF8})
		if m.form == nil || m.formMode != formKeys || m.form.Title != "Generate ED25519 Key" {
			t.Fatalf("форма ключей не открылась: %+v", m.form)
		}
	})
	t.Run("desktop dialog", func(t *testing.T) {
		m := newFakeModel(t, []config.Project{project})
		m.step(keyRuneEvent('d'))
		if m.desktop == nil || len(m.desktop) != 6 || !strings.Contains(m.desktop[0], "Actions=demo;") {
			t.Fatalf("desktop-диалог неверен: %v", m.desktop)
		}
		m.step(keyEvent{kind: keyEnter})
		if m.desktop != nil {
			t.Errorf("desktop не закрылся")
		}
	})
	t.Run("F7 empty projects is a no-op quirk", func(t *testing.T) {
		m := newFakeModel(t, nil)
		m.step(keyEvent{kind: keyF7})
		if m.form != nil {
			t.Errorf("квирк: при пустых проектах new должен быть no-op")
		}
	})
	t.Run("delete confirm declines and confirms", func(t *testing.T) {
		m := newFakeModel(t, []config.Project{project})
		name := m.projects[0].Name
		path := filepath.Join(m.opts.ConfigDir, name+config.ConfigSuffix)
		m.step(keyEvent{kind: keyDelete})
		if m.confirm == nil {
			t.Fatalf("confirm не открылся")
		}
		m.step(keyRuneEvent('n'))
		if m.confirm != nil {
			t.Fatalf("confirm не закрылся после n")
		}
		if _, err := os.Stat(path); err != nil {
			t.Errorf("файл не должен был удалиться: %v", err)
		}
		m.step(keyEvent{kind: keyDelete})
		m.step(keyRuneEvent('y'))
		if _, err := os.Stat(path); err == nil {
			t.Errorf("файл должен быть удалён")
		}
	})
	t.Run("delete blocked when mounted", func(t *testing.T) {
		m := newFakeModel(t, []config.Project{project}, "/mnt/demo")
		m.step(keyEvent{kind: keyDelete})
		if m.confirm != nil {
			t.Errorf("confirm не должен открываться для mounted")
		}
		if m.message != config.ProjectMountedMessage {
			t.Errorf("message = %q", m.message)
		}
	})
}

func TestDispatchTableIdleMounted(t *testing.T) {
	cases := []struct {
		name     string
		mounted  bool
		aidx     int
		wantCall string // mount|unmount|run|connect|new|edit|keys|desktop|delete
	}{
		{"idle 0 mount", false, 0, "mount"},
		{"idle 1 run", false, 1, "run"},
		{"idle 2 connect", false, 2, "connect"},
		{"mounted 0 unmount", true, 0, "unmount"},
		{"mounted 1 browse runs run", true, 1, "run"},
		{"mounted 2 connect", true, 2, "connect"},
		{"tool 3 new", false, 3, "new"},
		{"tool 4 edit", false, 4, "edit"},
		{"tool 5 keys", false, 5, "keys"},
		{"tool 6 desktop", false, 6, "desktop"},
		{"tool 7 delete", false, 7, "delete"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mounted := []string{}
			if tc.mounted {
				mounted = []string{"/mnt/demo"}
			}
			m := newFakeModel(t, []config.Project{demoProject("demo")}, mounted...)
			m.focusLeft = false
			m.aidx = tc.aidx
			_ = m.step(keyEvent{kind: keyEnter})
			switch tc.wantCall {
			case "mount", "unmount", "run", "connect":
				if m.spinner == nil || m.spinner.action != tc.wantCall {
					t.Errorf("%s: spinner.action = %v", tc.wantCall,
						func() string {
							if m.spinner == nil {
								return "<nil>"
							}
							return m.spinner.action
						}())
				}
			case "new", "edit", "keys":
				if m.form == nil {
					t.Errorf("%s: форма не открылась", tc.wantCall)
				}
			case "desktop":
				if m.desktop == nil {
					t.Error("desktop не открылся")
				}
			case "delete":
				if m.confirm == nil {
					t.Error("confirm не открылся")
				}
			}
		})
	}
}

func TestKQuirkDoubleEffect(t *testing.T) {
	p1 := demoProject("one")
	p2 := demoProject("two")
	m := newFakeModel(t, []config.Project{p1, p2})
	m.pidx = 1
	m.step(keyRuneEvent('k'))
	if m.form == nil || m.formMode != formKeys {
		t.Errorf("квирк: k должен открыть keys-форму")
	}
	if m.pidx != 0 {
		t.Errorf("квирк: k должен также навигировать вверх; pidx=%d", m.pidx)
	}

	m2 := newFakeModel(t, []config.Project{p1, p2})
	m2.pidx = 1
	m2.step(keyRuneEvent('K'))
	if m2.form == nil || m2.formMode != formKeys {
		t.Errorf("K должен открыть keys-форму")
	}
	if m2.pidx != 1 {
		t.Errorf("K не должен навигировать; pidx=%d", m2.pidx)
	}
}

func TestQuitKeys(t *testing.T) {
	for _, k := range []keyEvent{
		keyRuneEvent('q'), keyRuneEvent('Q'), keyRuneEvent('й'), keyRuneEvent('Й'),
		{kind: keyEsc}, {kind: keyF10},
	} {
		m := newFakeModel(t, []config.Project{demoProject("demo")})
		m.step(k)
		if !m.quit {
			t.Errorf("клавиша %+v не вышла", k)
		}
	}
}

func TestStatusBarAndEmptyProjectList(t *testing.T) {
	t.Run("with project", func(t *testing.T) {
		m := newFakeModel(t, []config.Project{demoProject("demo")})
		lines := m.viewReduced()
		last := lines[len(lines)-1]
		if !strings.Contains(last, "1 projects | 0 mounted | Tab:switch | F1:help | F10:quit") {
			t.Errorf("статусбар: %q", last)
		}
	})
	t.Run("empty", func(t *testing.T) {
		m := newFakeModel(t, nil)
		joined := strings.Join(m.viewReduced(), "\n")
		for _, want := range []string{"No projects.", "F7 / c — create one", "0 projects | 0 mounted"} {
			if !strings.Contains(joined, want) {
				t.Errorf("нет %q", want)
			}
		}
		if !strings.Contains(joined, "Configs: ") {
			t.Errorf("нет строки Configs (усечение по ширине панели — норма): %q", joined[:min(120, len(joined))])
		}
	})
	t.Run("message overlay one-shot", func(t *testing.T) {
		m := newFakeModel(t, []config.Project{demoProject("demo")})
		m.message = "hello"
		lines := m.viewReduced()
		if !strings.Contains(strings.Join(lines, "\n"), " hello ") {
			t.Errorf("message-оверлей не показан")
		}
		lines2 := m.viewReduced()
		if strings.Contains(strings.Join(lines2, "\n"), " hello ") {
			t.Errorf("message должен быть одноразовым")
		}
	})
}

func TestProjectListIconsAndScroll(t *testing.T) {
	projects := []config.Project{demoProject("demo")}
	m := newFakeModel(t, projects, "/mnt/demo")
	joined := strings.Join(m.viewReduced(), "\n")
	if !strings.Contains(joined, "●") || !strings.Contains(joined, "▶") {
		t.Errorf("иконки: %q", joined)
	}
	// Скролл: 30 проектов, pidx за пределом видимого окна.
	many := make([]config.Project, 30)
	for i := range many {
		many[i] = demoProject("p" + strings.Repeat("x", i))
	}
	ms := newFakeModel(t, many)
	ms.pidx = 25
	ms.pscroll = 0
	ms.viewReduced()
	if ms.pscroll == 0 {
		t.Errorf("pscroll не сдвинулся при pidx=25")
	}
}

func TestFormValidationFlow(t *testing.T) {
	project := demoProject("demo")
	m := newFakeModel(t, []config.Project{project})
	m.step(keyRuneEvent('c'))
	if m.form == nil {
		t.Fatal("форма не открылась")
	}
	m.form.Values = []string{"demo", "no-colon", "/mnt/x", "zed"}
	m.step(keyEvent{kind: keyEnter})
	if m.form == nil {
		t.Fatal("форма закрылась при невалидном remote")
	}
	if m.form.Message != config.RemoteMustBeAliasPathMessage {
		t.Errorf("Message = %q", m.form.Message)
	}
	// Исправляем remote → сохранение.
	m.form.Values[1] = "host:/x"
	m.step(keyEvent{kind: keyEnter})
	if m.form != nil {
		t.Fatalf("форма не закрылась")
	}
	if _, err := os.Stat(filepath.Join(m.opts.ConfigDir, "demo.config.toml")); err != nil {
		t.Errorf("конфиг не сохранён: %v", err)
	}
}

func TestFormEditRenameRemovesOld(t *testing.T) {
	project := demoProject("old")
	m := newFakeModel(t, []config.Project{project})
	m.step(keyRuneEvent('e'))
	if m.form == nil {
		t.Fatal("форма edit не открылась")
	}
	m.form.Values = []string{"renamed", "host:/x", "/mnt/r", "zed"}
	m.step(keyEvent{kind: keyEnter})
	if _, err := os.Stat(filepath.Join(m.opts.ConfigDir, "renamed.config.toml")); err != nil {
		t.Errorf("новый конфиг не создан: %v", err)
	}
	if _, err := os.Stat(filepath.Join(m.opts.ConfigDir, "old.config.toml")); err == nil {
		t.Errorf("старый конфиг не удалён")
	}
}

func TestCheckNetFailClosesForm(t *testing.T) {
	dir := t.TempDir()
	fs := &fakeSystem{mounted: map[string]bool{}}
	eng := &workspace.Engine{ConfigDir: dir, System: fs}
	m := New(Options{
		ConfigDir: dir,
		Engine:    eng,
		CheckNetFn: func(context.Context, string) (bool, string) {
			return false, "down"
		},
	})
	m.updateSize(80, 24)
	m.fake = fs
	m.needsRefresh = true
	m.projects = []projectItem{{Project: demoProject("demo")}}
	m.step(keyRuneEvent('c'))
	m.form.Values = []string{"demo", "host:/x", "/mnt/x", "zed"}
	m.step(keyEvent{kind: keyEnter})
	if m.form != nil {
		t.Errorf("форма должна закрыться после CheckNet fail")
	}
	if !strings.Contains(m.message, "Network check FAILED: down") {
		t.Errorf("message = %q", m.message)
	}
}

func TestTooSmallGuard(t *testing.T) {
	m := newFakeModel(t, []config.Project{demoProject("demo")})
	m.updateSize(50, 10)
	if !m.tooSmall {
		t.Fatalf("tooSmall не поднят")
	}
	joined := strings.Join(m.viewReduced(), "\n")
	if !strings.Contains(joined, "Terminal too small (50x10)") {
		t.Errorf("нет too-small текста: %q", joined)
	}
	m.step(keyRuneEvent('a'))
	if m.tooSmall {
		t.Errorf("retry не сбросил tooSmall")
	}
	m.updateSize(50, 10)
	m.step(keyRuneEvent('q'))
	if !m.quit {
		t.Errorf("q в too-small не вышел")
	}
}

func TestTabPanelSwitching(t *testing.T) {
	m := newFakeModel(t, []config.Project{demoProject("demo")})
	m.step(keyEvent{kind: keyTab})
	if m.focusLeft {
		t.Errorf("Tab не переключил на правую панель")
	}
	m.step(keyEvent{kind: keyTab})
	if !m.focusLeft {
		t.Errorf("Tab не вернул на левую панель")
	}
}

func TestHelpDialogOpensAndCloses(t *testing.T) {
	m := newFakeModel(t, []config.Project{demoProject("demo")})
	m.step(keyEvent{kind: keyF1})
	if !m.help {
		t.Fatal("help не открылся")
	}
	joined := strings.Join(m.viewReduced(), "\n")
	if !strings.Contains(joined, "Tab / ←→  Switch panels") {
		t.Errorf("нет строки help")
	}
	m.step(keyRuneEvent('x'))
	if m.help {
		t.Errorf("help не закрылся")
	}
}

func TestEnterLeftPanelFocusMovesRight(t *testing.T) {
	m := newFakeModel(t, []config.Project{demoProject("demo")})
	m.step(keyEvent{kind: keyEnter})
	if m.focusLeft {
		t.Errorf("Enter не перевёл фокус вправо")
	}
	if m.aidx != 0 {
		t.Errorf("aidx не сброшен")
	}
}
