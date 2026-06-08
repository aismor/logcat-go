package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

type LogEntry struct {
	widget.Entry
	onFormatJSON func(string)
}

func NewLogEntry(onFormatJSON func(string)) *LogEntry {
	e := &LogEntry{onFormatJSON: onFormatJSON}
	e.ExtendBaseWidget(e)
	e.MultiLine = true
	return e
}

func (e *LogEntry) TappedSecondary(pe *fyne.PointEvent) {
	if e.Disabled() && e.Password {
		return
	}

	app := fyne.CurrentApp()
	driver := app.Driver()
	canvas := driver.CanvasForObject(e)
	if focusable, ok := any(e).(fyne.Focusable); ok {
		canvas.Focus(focusable)
	}

	clipboard := app.Clipboard()
	cutItem := fyne.NewMenuItem("Recortar", func() {
		e.TypedShortcut(&fyne.ShortcutCut{Clipboard: clipboard})
	})
	copyItem := fyne.NewMenuItem("Copiar", func() {
		e.TypedShortcut(&fyne.ShortcutCopy{Clipboard: clipboard})
	})
	pasteItem := fyne.NewMenuItem("Colar", func() {
		e.TypedShortcut(&fyne.ShortcutPaste{Clipboard: clipboard})
	})
	selectAllItem := fyne.NewMenuItem("Selecionar tudo", func() {
		e.TypedShortcut(&fyne.ShortcutSelectAll{})
	})

	menuItems := []*fyne.MenuItem{cutItem, copyItem}

	if jsonText, ok := extractJSON(e.SelectedText()); ok {
		raw := jsonText
		formatItem := fyne.NewMenuItem("Formatar JSON", func() {
			if e.onFormatJSON != nil {
				e.onFormatJSON(raw)
			}
		})
		menuItems = append(menuItems, formatItem)
	}

	menuItems = append(menuItems, pasteItem, selectAllItem)

	entryPos := driver.AbsolutePositionForObject(e)
	popUp := widget.NewPopUpMenu(fyne.NewMenu("", menuItems...), canvas)
	popUp.ShowAtPosition(entryPos.Add(pe.Position))
}
