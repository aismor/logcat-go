package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

var (
	androidGreen = color.NRGBA{R: 0x4A, G: 0xDE, B: 0x80, A: 0xFF}
	headerDarkBg = color.NRGBA{R: 0x0B, G: 0x0E, B: 0x14, A: 0xFF}
	surfaceDark  = color.NRGBA{R: 0x16, G: 0x1B, B: 0x22, A: 0xFF}
	subtitleGrey = color.NRGBA{R: 0x9C, G: 0xA3, B: 0xAF, A: 0xFF}
	titleWhite   = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	borderDark   = color.NRGBA{R: 0x2A, G: 0x2E, B: 0x38, A: 0xFF}
)

type logcatTheme struct {
	base   fyne.Theme
	isDark bool
}

func newLogcatTheme(dark bool) fyne.Theme {
	base := theme.LightTheme()
	if dark {
		base = theme.DarkTheme()
	}
	return &logcatTheme{base: base, isDark: dark}
}

func (t *logcatTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	if c, ok := logThemeColors(name); ok {
		return c
	}
	if !t.isDark {
		return t.base.Color(name, variant)
	}

	switch name {
	case theme.ColorNamePrimary:
		return androidGreen
	case theme.ColorNameForegroundOnPrimary:
		return color.NRGBA{R: 0x12, G: 0x12, B: 0x12, A: 0xFF}
	case theme.ColorNameHyperlink:
		return androidGreen
	case theme.ColorNameHeaderBackground:
		return headerDarkBg
	case theme.ColorNameBackground:
		return color.NRGBA{R: 0x12, G: 0x12, B: 0x12, A: 0xFF}
	case theme.ColorNameInputBackground:
		return surfaceDark
	case theme.ColorNameSeparator:
		return borderDark
	case theme.ColorNameButton:
		return surfaceDark
	case theme.ColorNameForeground:
		return titleWhite
	case theme.ColorNameDisabled:
		return color.NRGBA{R: 0x6B, G: 0x72, B: 0x80, A: 0xFF}
	case theme.ColorNamePlaceHolder:
		return subtitleGrey
	}
	return t.base.Color(name, variant)
}

func (t *logcatTheme) Font(style fyne.TextStyle) fyne.Resource {
	return t.base.Font(style)
}

func (t *logcatTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return t.base.Icon(name)
}

func (t *logcatTheme) Size(name fyne.ThemeSizeName) float32 {
	return t.base.Size(name)
}
