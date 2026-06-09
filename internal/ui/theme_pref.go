package ui

import (
	"fyne.io/fyne/v2"
)

const prefThemeKey = "ui.theme"

const (
	ThemeDark  = "dark"
	ThemeLight = "light"
)

func ThemeOptions() []string {
	return []string{"Escuro", "Claro"}
}

func ThemeDisplayName(mode string) string {
	if mode == ThemeLight {
		return "Claro"
	}
	return "Escuro"
}

func ThemeFromDisplay(name string) string {
	if name == "Claro" {
		return ThemeLight
	}
	return ThemeDark
}

func CurrentTheme() string {
	mode := fyne.CurrentApp().Preferences().StringWithFallback(prefThemeKey, ThemeDark)
	if mode == ThemeLight {
		return ThemeLight
	}
	return ThemeDark
}

func InitThemeFromPreferences() {
	ApplyTheme(CurrentTheme())
}

func ApplyTheme(mode string) {
	switch mode {
	case ThemeLight:
		fyne.CurrentApp().Settings().SetTheme(newLogcatTheme(false))
	default:
		fyne.CurrentApp().Settings().SetTheme(newLogcatTheme(true))
	}
	fyne.CurrentApp().Preferences().SetString(prefThemeKey, mode)
}
