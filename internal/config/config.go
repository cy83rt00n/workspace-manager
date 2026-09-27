// Package config implements the configuration domain of the WSM workspace
// manager: parsing, validation, saving, discovery, delete guards and the
// desktop-action snippet. It is a pure domain function: it runs no subprocess,
// opens no network connection and performs no mount check; mount state is
// supplied externally by the caller.
//
// Baseline defects are preserved as-is and are deliberately NOT fixed:
//
//	D2  parse/save without escaping: the regex ^<key>\s*=\s*"(.+)" is greedy,
//	    and a '"' inside a value breaks the parse (and Save writes it raw).
//	D7  the project name is interpolated raw inside a single-quoted shell in the
//	    desktop snippet (a name containing a single quote breaks the shell).
//	D15 Save is not atomic.
//
// Intentional structural (non-contractual) divergences from the Python
// baseline:
//
//	the mounted field is moved out of this domain (it is delivered in stage
//	P1.3 by an external mount check, not parsed here); the "~otheruser" tilde
//	form is NOT expanded and stays as-is.
package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

// ConfigSuffix is the filename suffix of a single project config file. The
// project name is the filename without this suffix.
const ConfigSuffix = ".config.toml"

// AliasRequiredMessage is returned when the cleaned alias is empty.
const AliasRequiredMessage = "Alias required (a-z, 0-9, _, -)"

// RemoteMustBeAliasPathMessage is returned when remote_path is not exactly one
// colon with a non-empty alias before it.
const RemoteMustBeAliasPathMessage = "Remote must be alias:/path"

// LocalMountMustBeAbsoluteMessage is returned when the local mount path does not
// start with a slash (after ~ expansion).
const LocalMountMustBeAbsoluteMessage = "Local mount must be absolute path"

// ConfigNotFoundFormat is the "not found" reason for a delete operation. The
// single %s is the absolute config file path.
const ConfigNotFoundFormat = "Config not found: %s"

// ProjectMountedMessage is returned when a delete is blocked because the project
// is currently mounted.
const ProjectMountedMessage = "Project is mounted. Unmount first."

// Project is one discovered project configuration. Name is the config filename
// without the .config.toml suffix. RemotePath, LocalMount and EditorCmd are
// parsed values (each may be empty for a missing key). ConfPath is the absolute
// path of the parsed config file. Mounted is intentionally absent: it is
// delivered later (stage P1.3) from an external mount check, not from this
// domain.
type Project struct {
	Name       string
	RemotePath string
	LocalMount string
	EditorCmd  string
	ConfPath   string
}

// DefaultDir returns the user's per-user config directory: <home>/.config/workspace .
// It is the single place that resolves the home directory; all other functions
// receive their directory as an explicit dir argument (no package globals).
//
// It returns "" when os.UserHomeDir reports an error (it never panics).
func DefaultDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "workspace")
}

// parseFileValue reads a single config file and returns the value of key, or ""
// if the key or file is absent. It is a faithful port of parse_toml: it scans
// line by line with the regex ^<key>\s*=\s*"(.+)" (no $ anchor), where the
// capture group is greedy to the last quote on the line. The FIRST matching
// line wins and parsing stops there (Python returns inside the loop on the
// first match). A malformed line is skipped. A missing file or missing key
// yields "".
func parseFileValue(path, key string) string {
	re := regexp.MustCompile(`^` + regexp.QuoteMeta(key) + `\s*=\s*"(.+)"`)
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if matches := re.FindStringSubmatch(line); len(matches) >= 2 {
			return matches[1]
		}
	}
	return ""
}

// findProjectIn loads the single project named name from dir. It reads
// <dir>/<name>.config.toml and returns a Project populated from the three
// recognised keys. It returns nil when the config file does not exist.
func findProjectIn(dir, name string) *Project {
	path := filepath.Join(dir, name+ConfigSuffix)
	if _, err := os.Stat(path); err != nil {
		return nil
	}
	return &Project{
		Name:       name,
		RemotePath: parseFileValue(path, "remote_path"),
		LocalMount: parseFileValue(path, "local_mount"),
		EditorCmd:  parseFileValue(path, "editor_cmd"),
		ConfPath:   path,
	}
}

// Projects discovers all project configs in dir and returns them sorted by
// config filename (lexicographically on the file name, then suffix-stripped to
// the project name). It returns an empty slice, with no error, when dir does not
// exist (mirroring the Python CONF_DIR.exists() guard). Mounted status is
// intentionally not part of the result: it is delivered by an external check in
// stage P1.3.
func Projects(dir string) ([]Project, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return []Project{}, nil
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ConfigSuffix) {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)

	projects := make([]Project, 0, len(names))
	for _, filename := range names {
		name := strings.TrimSuffix(filename, ConfigSuffix)
		if p := findProjectIn(dir, name); p != nil {
			projects = append(projects, *p)
		}
	}
	return projects, nil
}

// Validate checks a config triple. It returns ok=false with an exact diagnostic
// message at the first failing rule, or ok=true with an empty message. Rules:
//
//  1. alias is reduced to its alphanumeric (Unicode letters/digits) plus '_' and
//     '-' characters; empty -> AliasRequiredMessage.
//  2. remotePath must contain exactly one ':' with a non-empty part before it,
//     else -> RemoteMustBeAliasPathMessage.
//  3. localMount: leading "~" is expanded through ExpandMountPath; the result
//     must start with '/', else -> LocalMountMustBeAbsoluteMessage.
func Validate(alias, remotePath, localMount string) (bool, string) {
	var cleaned strings.Builder
	for _, r := range alias {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' {
			cleaned.WriteRune(r)
		}
	}
	if cleaned.Len() == 0 {
		return false, AliasRequiredMessage
	}

	if strings.Count(remotePath, ":") != 1 {
		return false, RemoteMustBeAliasPathMessage
	}
	if parts := strings.SplitN(remotePath, ":", 2); parts[0] == "" {
		return false, RemoteMustBeAliasPathMessage
	}

	expanded := ExpandMountPath(localMount)
	if !strings.HasPrefix(expanded, "/") {
		return false, LocalMountMustBeAbsoluteMessage
	}
	return true, ""
}

// ExpandMountPath expands a leading "~" against the user home directory:
//
//	"~"      -> home
//	"~/..."  -> home + "/..." (everything after the tilde)
//	anything else (including "~otheruser", a leading "~x" without slash) is
//	            returned unchanged.
//
// The "~otheruser" case is intentionally NOT expanded in Go (documented
// structural divergence; it is an unreachable user edge). A relative path is
// returned unchanged — Validate rejects it via the absolute-path rule.
func ExpandMountPath(localMount string) string {
	if localMount == "~" || strings.HasPrefix(localMount, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return localMount
		}
		if localMount == "~" {
			return home
		}
		return home + localMount[1:]
	}
	return localMount
}

// Save writes a project config file at <dir>/<alias>.config.toml and returns its
// path. It creates dir (with parents) via MkdirAll. If oldConf is non-empty, is
// not equal to the new config path, and names an existing file, that file is
// removed first. The content is EXACTLY three lines, values substituted RAW with
// no escaping (baseline defect D2 preserved): a '"' inside a value is written
// as-is and will later break re-parsing. Права создаваемых файлов/каталогов —
// 0644/0755 (паритет типичного umask 022 у Python; Go umask не учитывает, при
// строгом umask расхождение с Python допустимо).
func Save(dir, alias, remotePath, localMount, editorCmd, oldConf string) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	newConf := filepath.Join(dir, alias+ConfigSuffix)
	if oldConf != "" && oldConf != newConf {
		if _, err := os.Stat(oldConf); err == nil {
			if err := os.Remove(oldConf); err != nil {
				return "", err
			}
		}
	}

	var content strings.Builder
	content.WriteString(`remote_path = "`)
	content.WriteString(remotePath)
	content.WriteString("\"\n")
	content.WriteString(`local_mount = "`)
	content.WriteString(localMount)
	content.WriteString("\"\n")
	content.WriteString(`editor_cmd = "`)
	content.WriteString(editorCmd)
	content.WriteString("\"\n")

	if err := os.WriteFile(newConf, []byte(content.String()), 0o644); err != nil {
		return "", err
	}
	return newConf, nil
}

// CanDeleteConfig reports whether a config file may be deleted. It returns
// ok=false with an exact reason when the file does not exist
// (ConfigNotFoundFormat with the confPath) or when mounted is true. The mounted
// flag is supplied externally (stage P1.3 performs the actual mount check); this
// domain only enforces the guard.
func CanDeleteConfig(confPath string, mounted bool) (bool, string) {
	if _, err := os.Stat(confPath); err != nil {
		return false, fmt.Sprintf(ConfigNotFoundFormat, confPath)
	}
	if mounted {
		return false, ProjectMountedMessage
	}
	return true, ""
}

// DeleteConfigFile deletes confPath if CanDeleteConfig allows it, otherwise it
// returns an error whose message is the reason. A nil return means the file was
// removed.
func DeleteConfigFile(confPath string, mounted bool) error {
	ok, reason := CanDeleteConfig(confPath, mounted)
	if !ok {
		return errors.New(reason)
	}
	return os.Remove(confPath)
}

// DesktopSnippetText builds the six-line XDG Desktop Action snippet for a single
// project. The project name is interpolated raw, including inside the
// single-quoted Exec line (baseline defect D7 preserved: a name containing a
// single quote breaks the shell).
func DesktopSnippetText(project string) []string {
	return []string{
		"Actions=" + project + ";",
		"",
		"[Desktop Action " + project + "]",
		"Name=Open Remote: " + project,
		"Exec=bash -c 'source $HOME/.wsm && wsm run " + project + "'",
		"Identifier=" + project,
	}
}
