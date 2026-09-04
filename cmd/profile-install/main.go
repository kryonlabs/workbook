package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type profileManifest struct {
	Name        string `json:"name"`
	Command     string `json:"command"`
	Script      string `json:"script"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	DataEnv     string `json:"data_env"`
	Desktop     bool   `json:"desktop"`
	Categories  string `json:"categories"`
}

func main() {
	root := flag.String("root", "", "package root prefix")
	profilesDir := flag.String("profiles", "profiles", "profile manifest directory")
	binDir := flag.String("bin-dir", "/usr/bin", "directory for wrapper commands")
	appDir := flag.String("app-dir", "", "desktop entry directory")
	binary := flag.String("binary", "workbook", "base workbook binary name")
	flag.Parse()

	profiles, err := readProfiles(*profilesDir)
	if err != nil {
		die(err)
	}
	for _, profile := range profiles {
		if err := installWrapper(*root, *binDir, *binary, profile); err != nil {
			die(err)
		}
		if *appDir != "" && profile.Desktop {
			if err := installDesktop(*root, *appDir, profile); err != nil {
				die(err)
			}
		}
	}
}

func readProfiles(dir string) ([]profileManifest, error) {
	matches, err := filepath.Glob(filepath.Join(dir, "*.kry"))
	if err != nil {
		return nil, err
	}
	sort.Strings(matches)
	profiles := make([]profileManifest, 0, len(matches))
	for _, path := range matches {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var profile profileManifest
		if err := json.Unmarshal(data, &profile); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		if profile.Name == "" || profile.Command == "" {
			return nil, fmt.Errorf("%s: profile name and command are required", path)
		}
		if profile.DisplayName == "" {
			profile.DisplayName = profile.Command
		}
		if profile.Description == "" {
			profile.Description = profile.DisplayName + " workbook profile"
		}
		if profile.Categories == "" {
			profile.Categories = "Office;"
		}
		profiles = append(profiles, profile)
	}
	return profiles, nil
}

func installWrapper(root, binDir, binary string, profile profileManifest) error {
	dir := filepath.Join(root, strings.TrimPrefix(binDir, string(filepath.Separator)))
	if filepath.IsAbs(binDir) && root == "" {
		dir = binDir
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, profile.Command)
	if profile.Script != "" {
		data, err := os.ReadFile(profile.Script)
		if err != nil {
			return err
		}
		return os.WriteFile(path, data, 0o755)
	}
	script := fmt.Sprintf(`#!/usr/bin/env sh
set -eu

dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
exec -a %s "$dir/%s" -profile %s "$@"
`, shellWord(profile.Command), shellWord(binary), shellWord(profile.Name))
	return os.WriteFile(path, []byte(script), 0o755)
}

func installDesktop(root, appDir string, profile profileManifest) error {
	dir := filepath.Join(root, strings.TrimPrefix(appDir, string(filepath.Separator)))
	if filepath.IsAbs(appDir) && root == "" {
		dir = appDir
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, profile.Command+".desktop")
	text := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=%s
Comment=%s
Exec=%s
Terminal=false
Categories=%s
StartupNotify=true
StartupWMClass=%s
`, profile.DisplayName, profile.Description, profile.Command, profile.Categories, profile.Command)
	return os.WriteFile(path, []byte(text), 0o644)
}

func shellWord(s string) string {
	return strings.NewReplacer("'", "", "\n", "", "\r", "", "\t", "").Replace(s)
}

func die(err error) {
	fmt.Fprintln(os.Stderr, "profile-install:", err)
	os.Exit(1)
}
