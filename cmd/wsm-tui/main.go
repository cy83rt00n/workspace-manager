// Command wsm-tui is the optional NC-style TUI client for the WSM workspace
// manager. It shares the workspace Engine and config/system services with
// cmd/wsm (release gate: no business logic is duplicated); the presentation
// lives in internal/tui.
package main

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/cy83rt00n/workspace-manager/internal/config"
	"github.com/cy83rt00n/workspace-manager/internal/system"
	"github.com/cy83rt00n/workspace-manager/internal/tui"
	"github.com/cy83rt00n/workspace-manager/internal/version"
	"github.com/cy83rt00n/workspace-manager/internal/workspace"
)

// main — запуск TUI: wire-up реальных зависимостей и программы bubbletea.
func main() {
	dir := config.DefaultDir()
	eng := &workspace.Engine{
		ConfigDir: dir,
		System:    system.Default(),
		// Out/ErrOut/Spin/SpinDone подключаются ниже к каналу messages модели.
		// Confirm не заполняется: подтверждение удаления идёт через собственный
		// confirm-диалог TUI + DeleteProject(force=true).
	}

	cfg := tui.Options{
		ConfigDir: dir,
		Engine:    eng,
		CheckNetFn: func(ctx context.Context, alias string) (bool, string) {
			return system.Default().CheckNet(ctx, alias)
		},
		KeygenFn: func(ctx context.Context, name string) (string, string, error) {
			return system.Default().GenerateKeypair(ctx, name, dir)
		},
		Version: version.Version,
	}
	m := tui.New(cfg)
	eng.Out = m.Out
	eng.ErrOut = m.ErrOut
	eng.Spin = func(stage string) { m.Spin("%s", stage) }
	eng.SpinDone = func(msg string) { m.SpinDone("%s", msg) }

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "wsm-tui: %v\n", err)
		os.Exit(1)
	}
}
