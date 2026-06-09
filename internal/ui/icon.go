package ui

import (
	_ "embed"

	"fyne.io/fyne/v2"
)

//go:embed resources/logcat-go.png
var appIconPNG []byte

func AppIcon() fyne.Resource {
	return fyne.NewStaticResource("logcat-go.png", appIconPNG)
}
