package main

import (
	"path/filepath"
	"strings"
)

const (
	profileWorkbook = "workbook"
	profileCell     = "cell"
	profileGeld     = "geld"
)

func defaultProfile(argv0 string) string {
	if strings.EqualFold(filepath.Base(argv0), profileGeld) {
		return profileGeld
	}
	return profileWorkbook
}

func normalizedProfile(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", profileWorkbook, profileCell:
		return profileWorkbook
	case profileGeld:
		return profileGeld
	default:
		return strings.ToLower(strings.TrimSpace(s))
	}
}

func (a *app) isGeldProfile() bool {
	return a.profile == "" || a.profile == profileGeld
}

func (a *app) commandName() string {
	if a.isGeldProfile() {
		return profileGeld
	}
	if a.profile != "" {
		return a.profile
	}
	return profileWorkbook
}

func (a *app) windowTitle() string {
	if a.isGeldProfile() {
		return "Geld - Workbook"
	}
	return "Workbook"
}
