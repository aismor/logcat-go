package main

import (
	"fyne.io/fyne/v2/app"

	"github.com/aismor/logcat-go/internal/ui"
)

func main() {
	application := app.NewWithID("com.aismor.logcat-go")
	application.SetIcon(nil)

	logcatApp := ui.NewApp()
	logcatApp.Run()
}
