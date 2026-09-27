package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// TestParseFileValue covers the line-by-line greedy parse of a single key. The
// baseline defect D2 (a '"' inside a value breaks the parse) and the
// first-match-wins rule are asserted explicitly.
func TestParseFileValue(t *testing.T) {
	cases := []struct {
		name       string
		content    string
		key        string
		want       string
		createFile bool
	}{
		{
			name:       "demo remote",
			content:    "remote_path = \"demo-alias:/opt/app\"\nlocal_mount = \"/home/u/.workspace/app\"\neditor_cmd = \"zed\"\n",
			key:        "remote_path",
			want:       "demo-alias:/opt/app",
			createFile: true,
		},
		{
			name:       "demo local",
			content:    "remote_path = \"demo-alias:/opt/app\"\nlocal_mount = \"/home/u/.workspace/app\"\neditor_cmd = \"zed\"\n",
			key:        "local_mount",
			want:       "/home/u/.workspace/app",
			createFile: true,
		},
		{
			name:       "demo editor",
			content:    "remote_path = \"demo-alias:/opt/app\"\nlocal_mount = \"/home/u/.workspace/app\"\neditor_cmd = \"zed\"\n",
			key:        "editor_cmd",
			want:       "zed",
			createFile: true,
		},
		{
			name:       "partial",
			content:    "remote_path = \"alias:/p\"\n",
			key:        "remote_path",
			want:       "alias:/p",
			createFile: true,
		},
		{
			name:       "partial missing key",
			content:    "remote_path = \"alias:/p\"\n",
			key:        "local_mount",
			want:       "",
			createFile: true,
		},
		{
			name:       "empty",
			content:    "",
			key:        "remote_path",
			want:       "",
			createFile: true,
		},
		{
			// quoted (D2): greediness of (.+) takes the capture up to the LAST
			// quote on the line. The stray '"' after "host:" is absorbed into the
			// value, so the parsed value is "host:\"/p" (the closing quote is the
			// one right after "/p").
			name:       "quoted",
			content:    "remote_path = \"host:\"/p\"\n",
			key:        "remote_path",
			want:       "host:\"/p",
			createFile: true,
		},
		{
			// duplicated: the FIRST matching line wins because the parse
			// returns on its first match (Python returns inside the loop).
			name:       "duplicated (first occurrence)",
			content:    "remote_path = \"v1\"\nremote_path = \"v2\"\n",
			key:        "remote_path",
			want:       "v1",
			createFile: true,
		},
		{
			name:       "malformed line skipped",
			content:    "no_equal_sign_here\nremote_path = \"ok\"\n",
			key:        "remote_path",
			want:       "ok",
			createFile: true,
		},
		{
			name:       "no file",
			content:    "",
			key:        "remote_path",
			want:       "",
			createFile: false,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			tmp := t.TempDir()
			path := filepath.Join(tmp, "x.config.toml")
			if tc.createFile {
				if err := os.WriteFile(path, []byte(tc.content), 0o600); err != nil {
					t.Fatalf("failed to write fixture %s: %v", path, err)
				}
			}
			got := parseFileValue(path, tc.key)
			if got != tc.want {
				t.Errorf("parseFileValue(%q) = %q, want %q", tc.key, got, tc.want)
			}
		})
	}
}

// TestFindProjectIn covers loading a single project: an existing file yields a
// fully populated Project, and a missing name yields nil.
func TestFindProjectIn(t *testing.T) {
	t.Run("existing project", func(t *testing.T) {
		tmp := t.TempDir()
		content := "remote_path = \"demo-alias:/opt/app\"\nlocal_mount = \"/home/u/.workspace/app\"\neditor_cmd = \"zed\"\n"
		path := filepath.Join(tmp, "demo"+ConfigSuffix)
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatalf("failed to write fixture %s: %v", path, err)
		}

		got := findProjectIn(tmp, "demo")
		if got == nil {
			t.Fatal("findProjectIn returned nil for an existing project")
		}
		want := Project{
			Name:       "demo",
			RemotePath: "demo-alias:/opt/app",
			LocalMount: "/home/u/.workspace/app",
			EditorCmd:  "zed",
			ConfPath:   filepath.Join(tmp, "demo"+ConfigSuffix),
		}
		if *got != want {
			t.Errorf("findProjectIn = %+v, want %+v", *got, want)
		}
	})

	t.Run("missing project", func(t *testing.T) {
		tmp := t.TempDir()
		if got := findProjectIn(tmp, "nosuchproj"); got != nil {
			t.Errorf("findProjectIn = %+v, want nil", *got)
		}
	})
}

// TestProjects covers discovery: empty existing dir, missing dir (empty and no
// error), filename ordering, and the suffix filter that excludes non-config
// files.
func TestProjects(t *testing.T) {
	t.Run("empty existing dir", func(t *testing.T) {
		tmp := t.TempDir()
		got, err := Projects(tmp)
		if err != nil {
			t.Fatalf("Projects returned error %v, want nil", err)
		}
		if len(got) != 0 {
			t.Errorf("len(Projects) = %d, want 0", len(got))
		}
	})

	t.Run("missing dir yields empty and nil error", func(t *testing.T) {
		tmp := t.TempDir()
		missing := filepath.Join(tmp, "nope")
		got, err := Projects(missing)
		if err != nil {
			t.Fatalf("Projects returned error %v, want nil", err)
		}
		if len(got) != 0 {
			t.Errorf("len(Projects) = %d, want 0", len(got))
		}
	})

	t.Run("sorted by filename", func(t *testing.T) {
		tmp := t.TempDir()
		// Write the three config files out of alphabetical order to prove the
		// result is sorted by filename regardless of creation order.
		fixtures := map[string]string{
			"zzz":   `remote_path = "zzz:/z"` + "\n" + `local_mount = "/m/z"` + "\n" + `editor_cmd = "zed"` + "\n",
			"alpha": `remote_path = "alpha:/a"` + "\n" + `local_mount = "/m/a"` + "\n" + `editor_cmd = "zed"` + "\n",
			"mid":   `remote_path = "mid:/m"` + "\n" + `local_mount = "/m/m"` + "\n" + `editor_cmd = "zed"` + "\n",
		}
		for name, content := range fixtures {
			path := filepath.Join(tmp, name+ConfigSuffix)
			if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
				t.Fatalf("failed to write fixture %s: %v", path, err)
			}
		}

		got, err := Projects(tmp)
		if err != nil {
			t.Fatalf("Projects returned error %v, want nil", err)
		}
		if len(got) != 3 {
			t.Fatalf("len(Projects) = %d, want 3", len(got))
		}
		wantOrder := []string{"alpha", "mid", "zzz"}
		for i, wantName := range wantOrder {
			if got[i].Name != wantName {
				t.Errorf("projects[%d].Name = %q, want %q", i, got[i].Name, wantName)
			}
			wantPath := filepath.Join(tmp, wantName+ConfigSuffix)
			if got[i].ConfPath != wantPath {
				t.Errorf("projects[%d].ConfPath = %q, want %q", i, got[i].ConfPath, wantPath)
			}
		}
	})

	t.Run("suffix filter excludes non-config files", func(t *testing.T) {
		tmp := t.TempDir()
		if err := os.WriteFile(filepath.Join(tmp, "notes.txt"), []byte("not a config"), 0o600); err != nil {
			t.Fatalf("failed to write fixture: %v", err)
		}
		alphaPath := filepath.Join(tmp, "alpha"+ConfigSuffix)
		if err := os.WriteFile(alphaPath, []byte("remote_path = \"host:/p\"\n"), 0o600); err != nil {
			t.Fatalf("failed to write fixture: %v", err)
		}

		got, err := Projects(tmp)
		if err != nil {
			t.Fatalf("Projects returned error %v, want nil", err)
		}
		if len(got) != 1 {
			t.Fatalf("len(Projects) = %d, want 1", len(got))
		}
		if got[0].Name != "alpha" {
			t.Errorf("projects[0].Name = %q, want %q", got[0].Name, "alpha")
		}
	})
}

// TestValidate covers the three validation rules against the exported message
// constants.
func TestValidate(t *testing.T) {
	cases := []struct {
		name       string
		alias      string
		remotePath string
		localMount string
		wantOK     bool
		wantMsg    string
	}{
		{name: "valid", alias: "demo", remotePath: "host:/p", localMount: "/mnt/x", wantOK: true, wantMsg: ""},
		{name: "space dropped", alias: "de mo", remotePath: "host:/p", localMount: "/mnt/x", wantOK: true, wantMsg: ""},
		{name: "dot dropped", alias: "de.mo", remotePath: "host:/p", localMount: "/mnt/x", wantOK: true, wantMsg: ""},
		{name: "dashes only", alias: "---", remotePath: "host:/p", localMount: "/mnt/x", wantOK: true, wantMsg: ""},
		{name: "exclamation dropped", alias: "!!!", remotePath: "host:/p", localMount: "/mnt/x", wantOK: false, wantMsg: AliasRequiredMessage},
		{name: "empty alias", alias: "", remotePath: "host:/p", localMount: "/mnt/x", wantOK: false, wantMsg: AliasRequiredMessage},
		{name: "no colon", alias: "demo", remotePath: "no-colon", localMount: "/mnt/x", wantOK: false, wantMsg: RemoteMustBeAliasPathMessage},
		{name: "two colons", alias: "demo", remotePath: "a:b:c", localMount: "/mnt/x", wantOK: false, wantMsg: RemoteMustBeAliasPathMessage},
		{name: "empty alias part", alias: "demo", remotePath: ":/p", localMount: "/mnt/x", wantOK: false, wantMsg: RemoteMustBeAliasPathMessage},
		{name: "relative mount", alias: "demo", remotePath: "host:/p", localMount: "rel/path", wantOK: false, wantMsg: LocalMountMustBeAbsoluteMessage},
		{name: "tilde work", alias: "demo", remotePath: "host:/p", localMount: "~/work", wantOK: true, wantMsg: ""},
		{name: "tilde alone", alias: "demo", remotePath: "host:/p", localMount: "~", wantOK: true, wantMsg: ""},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			gotOK, gotMsg := Validate(tc.alias, tc.remotePath, tc.localMount)
			if gotOK != tc.wantOK {
				t.Errorf("Validate ok = %v, want %v (msg=%q)", gotOK, tc.wantOK, gotMsg)
			}
			if gotMsg != tc.wantMsg {
				t.Errorf("Validate msg = %q, want %q", gotMsg, tc.wantMsg)
			}
		})
	}
}

// TestExpandMountPath covers tilde expansion against the real user home
// directory, plus the unchanged cases.
func TestExpandMountPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("os.UserHomeDir: %v", err)
	}

	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "tilde alone", in: "~", want: home},
		{name: "tilde work", in: "~/work", want: filepath.Join(home, "work")},
		{name: "absolute keeps", in: "/abs/path", want: "/abs/path"},
		{name: "relative keeps", in: "rel/path", want: "rel/path"},
		{name: "other user not expanded", in: "~otheruser/x", want: "~otheruser/x"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := ExpandMountPath(tc.in)
			if got != tc.want {
				t.Errorf("ExpandMountPath(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestSave covers the golden byte format (including the raw D2 value), parent
// directory creation, and the oldConf removal semantics.
func TestSave(t *testing.T) {
	t.Run("golden basic", func(t *testing.T) {
		tmp := t.TempDir()
		gotPath, err := Save(tmp, "demo", "host:/p", "/mnt/x", "zed", "")
		if err != nil {
			t.Fatalf("Save returned error: %v", err)
		}
		wantPath := filepath.Join(tmp, "demo"+ConfigSuffix)
		if gotPath != wantPath {
			t.Errorf("Save path = %q, want %q", gotPath, wantPath)
		}
		gotContent, err := os.ReadFile(wantPath)
		if err != nil {
			t.Fatalf("failed to read saved file: %v", err)
		}
		wantContent := "remote_path = \"host:/p\"\nlocal_mount = \"/mnt/x\"\neditor_cmd = \"zed\"\n"
		if string(gotContent) != wantContent {
			t.Errorf("content = %q, want %q", string(gotContent), wantContent)
		}
	})

	t.Run("golden raw D2 value", func(t *testing.T) {
		tmp := t.TempDir()
		gotPath, err := Save(tmp, "q", "host:\"/p", "/mnt/z", "", "")
		if err != nil {
			t.Fatalf("Save returned error: %v", err)
		}
		gotContent, err := os.ReadFile(gotPath)
		if err != nil {
			t.Fatalf("failed to read saved file: %v", err)
		}
		// The '"' inside the value is written as-is, NOT escaped (defect D2).
		wantContent := "remote_path = \"host:\"/p\"\nlocal_mount = \"/mnt/z\"\neditor_cmd = \"\"\n"
		if string(gotContent) != wantContent {
			t.Errorf("content = %q, want %q", string(gotContent), wantContent)
		}
	})

	t.Run("MkdirAll parents", func(t *testing.T) {
		tmp := t.TempDir()
		dir := filepath.Join(tmp, "a", "b", "c")
		gotPath, err := Save(dir, "demo", "host:/p", "/mnt/x", "zed", "")
		if err != nil {
			t.Fatalf("Save returned error: %v", err)
		}
		if _, statErr := os.Stat(dir); statErr != nil {
			t.Errorf("parent dir %s was not created: %v", dir, statErr)
		}
		if _, statErr := os.Stat(gotPath); statErr != nil {
			t.Errorf("config file %s was not written: %v", gotPath, statErr)
		}
	})

	t.Run("oldConf removed on path change", func(t *testing.T) {
		tmp := t.TempDir()
		old := filepath.Join(tmp, "old"+ConfigSuffix)
		if err := os.WriteFile(old, []byte("remote_path = \"old:/p\"\n"), 0o600); err != nil {
			t.Fatalf("failed to write oldConf: %v", err)
		}

		newName := "new"
		gotPath, err := Save(tmp, newName, "h:/p", "/m", "z", old)
		if err != nil {
			t.Fatalf("Save returned error: %v", err)
		}

		if _, statErr := os.Stat(old); statErr == nil {
			t.Errorf("oldConf %s should have been removed", old)
		}
		newPath := filepath.Join(tmp, newName+ConfigSuffix)
		if gotPath != newPath {
			t.Errorf("Save path = %q, want %q", gotPath, newPath)
		}
		if _, statErr := os.Stat(newPath); statErr != nil {
			t.Errorf("new config %s should exist: %v", newPath, statErr)
		}
	})

	t.Run("oldConf not removed on matching path", func(t *testing.T) {
		tmp := t.TempDir()
		name := "new"
		newPath := filepath.Join(tmp, name+ConfigSuffix)
		if _, err := Save(tmp, name, "h:/p", "/m", "z", ""); err != nil {
			t.Fatalf("first Save returned error: %v", err)
		}

		// Passing oldConf == newConf must NOT remove the file (it is the same
		// path), only overwrite it.
		if _, err := Save(tmp, name, "h2:/p", "/m2", "z", newPath); err != nil {
			t.Fatalf("second Save returned error: %v", err)
		}
		if _, statErr := os.Stat(newPath); statErr != nil {
			t.Errorf("config %s should still exist: %v", newPath, statErr)
		}
		gotContent, err := os.ReadFile(newPath)
		if err != nil {
			t.Fatalf("failed to read saved file: %v", err)
		}
		wantContent := "remote_path = \"h2:/p\"\nlocal_mount = \"/m2\"\neditor_cmd = \"z\"\n"
		if string(gotContent) != wantContent {
			t.Errorf("content = %q, want %q", string(gotContent), wantContent)
		}
	})
}

// TestCanDeleteConfig covers the two delete guards: a missing file and a
// mounted project both block the delete.
func TestCanDeleteConfig(t *testing.T) {
	tmp := t.TempDir()

	t.Run("missing file", func(t *testing.T) {
		path := filepath.Join(tmp, "nosuch"+ConfigSuffix)
		gotOK, gotReason := CanDeleteConfig(path, false)
		wantReason := fmt.Sprintf(ConfigNotFoundFormat, path)
		if gotOK {
			t.Errorf("CanDeleteConfig ok = true, want false")
		}
		if gotReason != wantReason {
			t.Errorf("CanDeleteConfig reason = %q, want %q", gotReason, wantReason)
		}
	})

	t.Run("existing and not mounted", func(t *testing.T) {
		path := filepath.Join(tmp, "demo"+ConfigSuffix)
		if err := os.WriteFile(path, []byte("remote_path = \"host:/p\"\n"), 0o600); err != nil {
			t.Fatalf("failed to write fixture: %v", err)
		}
		gotOK, gotReason := CanDeleteConfig(path, false)
		if !gotOK {
			t.Errorf("CanDeleteConfig ok = false, want true (reason=%q)", gotReason)
		}
		if gotReason != "" {
			t.Errorf("CanDeleteConfig reason = %q, want empty", gotReason)
		}
	})

	t.Run("existing but mounted", func(t *testing.T) {
		path := filepath.Join(tmp, "demo"+ConfigSuffix)
		gotOK, gotReason := CanDeleteConfig(path, true)
		if gotOK {
			t.Errorf("CanDeleteConfig ok = true, want false")
		}
		if gotReason != ProjectMountedMessage {
			t.Errorf("CanDeleteConfig reason = %q, want %q", gotReason, ProjectMountedMessage)
		}
	})
}

// TestDeleteConfigFile covers the guarded removal: a missing file or a mounted
// project returns the reason as an error and leaves the file untouched, while a
// safe project is removed.
func TestDeleteConfigFile(t *testing.T) {
	t.Run("missing file", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "nosuch"+ConfigSuffix)
		err := DeleteConfigFile(path, false)
		if err == nil {
			t.Fatal("DeleteConfigFile returned nil, want error")
		}
		want := fmt.Sprintf(ConfigNotFoundFormat, path)
		if err.Error() != want {
			t.Errorf("DeleteConfigFile error = %q, want %q", err.Error(), want)
		}
		if _, statErr := os.Stat(path); statErr == nil {
			t.Errorf("config %s should not exist", path)
		}
	})

	t.Run("mounted blocks delete", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "demo"+ConfigSuffix)
		if err := os.WriteFile(path, []byte("remote_path = \"host:/p\"\n"), 0o600); err != nil {
			t.Fatalf("failed to write fixture: %v", err)
		}
		err := DeleteConfigFile(path, true)
		if err == nil {
			t.Fatal("DeleteConfigFile returned nil, want error")
		}
		if err.Error() != ProjectMountedMessage {
			t.Errorf("DeleteConfigFile error = %q, want %q", err.Error(), ProjectMountedMessage)
		}
		if _, statErr := os.Stat(path); statErr != nil {
			t.Errorf("config %s should remain after blocked delete: %v", path, statErr)
		}
	})

	t.Run("safe project removed", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "demo"+ConfigSuffix)
		if err := os.WriteFile(path, []byte("remote_path = \"host:/p\"\n"), 0o600); err != nil {
			t.Fatalf("failed to write fixture: %v", err)
		}
		err := DeleteConfigFile(path, false)
		if err != nil {
			t.Fatalf("DeleteConfigFile returned error: %v", err)
		}
		if _, statErr := os.Stat(path); statErr == nil {
			t.Errorf("config %s should have been removed", path)
		}
	})
}

// TestDesktopSnippetText asserts the six-line XDG Desktop Action snippet with a
// raw project name interpolation.
func TestDesktopSnippetText(t *testing.T) {
	got := DesktopSnippetText("demo")
	want := []string{
		"Actions=demo;",
		"",
		"[Desktop Action demo]",
		"Name=Open Remote: demo",
		"Exec=bash -c 'source $HOME/.wsm && wsm run demo'",
		"Identifier=demo",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("DesktopSnippetText = %#v, want %#v", got, want)
	}
}
