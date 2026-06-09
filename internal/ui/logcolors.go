package ui

import (
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
)

const (
	themeColorLogMeta     fyne.ThemeColorName = "logMeta"
	themeColorLogVerbose  fyne.ThemeColorName = "logVerbose"
	themeColorLogDebug    fyne.ThemeColorName = "logDebug"
	themeColorLogInfo     fyne.ThemeColorName = "logInfo"
	themeColorLogWarn     fyne.ThemeColorName = "logWarn"
	themeColorLogError    fyne.ThemeColorName = "logError"
	themeColorLogMessage  fyne.ThemeColorName = "logMessage"
	themeColorLogWarnMsg  fyne.ThemeColorName = "logWarnMessage"
	themeColorLogErrorMsg fyne.ThemeColorName = "logErrorMessage"
)

var logMetaColor = color.NRGBA{R: 0x8B, G: 0x92, B: 0x9E, A: 0xFF}

func levelThemeColor(level string) fyne.ThemeColorName {
	switch strings.ToUpper(strings.TrimSpace(level)) {
	case "V":
		return themeColorLogVerbose
	case "D":
		return themeColorLogDebug
	case "I":
		return themeColorLogInfo
	case "W":
		return themeColorLogWarn
	case "E", "F", "A":
		return themeColorLogError
	default:
		return themeColorLogMessage
	}
}

func messageThemeColor(level string) fyne.ThemeColorName {
	switch strings.ToUpper(strings.TrimSpace(level)) {
	case "W":
		return themeColorLogWarnMsg
	case "E", "F", "A":
		return themeColorLogErrorMsg
	case "V", "D", "I":
		return levelThemeColor(level)
	default:
		return themeColorLogMessage
	}
}

func levelBackground(level string) color.Color {
	switch strings.ToUpper(strings.TrimSpace(level)) {
	case "W":
		return color.NRGBA{R: 0x3D, G: 0x32, B: 0x14, A: 0x55}
	case "E", "F", "A":
		return color.NRGBA{R: 0x3D, G: 0x18, B: 0x18, A: 0x66}
	default:
		return color.Transparent
	}
}

func logThemeColors(name fyne.ThemeColorName) (color.Color, bool) {
	switch name {
	case themeColorLogMeta:
		return logMetaColor, true
	case themeColorLogVerbose:
		return color.NRGBA{R: 0xAA, G: 0xAA, B: 0xAA, A: 0xFF}, true
	case themeColorLogDebug:
		return color.NRGBA{R: 0x6B, G: 0xAF, B: 0xFF, A: 0xFF}, true
	case themeColorLogInfo:
		return color.NRGBA{R: 0x98, G: 0xC3, B: 0x79, A: 0xFF}, true
	case themeColorLogWarn:
		return color.NRGBA{R: 0xE5, G: 0xC0, B: 0x7B, A: 0xFF}, true
	case themeColorLogError:
		return color.NRGBA{R: 0xE0, G: 0x6C, B: 0x75, A: 0xFF}, true
	case themeColorLogMessage:
		return color.NRGBA{R: 0xE5, G: 0xE7, B: 0xEB, A: 0xFF}, true
	case themeColorLogWarnMsg:
		return color.NRGBA{R: 0xF5, G: 0xD0, B: 0x90, A: 0xFF}, true
	case themeColorLogErrorMsg:
		return color.NRGBA{R: 0xFC, G: 0xA5, B: 0xA5, A: 0xFF}, true
	}
	return color.Transparent, false
}
