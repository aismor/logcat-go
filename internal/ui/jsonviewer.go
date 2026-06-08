package ui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type jsonSearchState struct {
	text    string
	query   string
	matches []int
	index   int
}

func ShowJSONViewer(parent fyne.Window, raw string) {
	formatted, err := formatJSON(raw)
	if err != nil {
		dialog.ShowError(fmt.Errorf("JSON inválido: %w", err), parent)
		return
	}

	win := fyne.CurrentApp().NewWindow("JSON formatado")
	win.Resize(fyne.NewSize(920, 720))
	win.CenterOnScreen()

	state := &jsonSearchState{text: formatted}

	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder("Buscar (Enter/F3 = próximo, Shift+Enter/Shift+F3 = anterior)...")

	statusLabel := widget.NewLabel("")
	statusLabel.TextStyle = fyne.TextStyle{Monospace: true}

	content := newJSONRichText(formatted, "", -1)
	scroll := container.NewVScroll(content)

	refreshContent := func() {
		content = newJSONRichText(state.text, state.query, state.index)
		scroll.Content = content
		content.Refresh()
		scroll.Refresh()

		if len(state.matches) == 0 || state.index < 0 || state.index >= len(state.matches) {
			statusLabel.SetText("0 ocorrências")
			return
		}

		offset := state.matches[state.index]
		row, _ := rowColFromOffset(state.text, offset)
		scrollToLine(scroll, content, row)
		statusLabel.SetText(fmt.Sprintf("%d / %d", state.index+1, len(state.matches)))
	}

	syncMatches := func() {
		state.query = strings.TrimSpace(searchEntry.Text)
		state.matches = findMatchOffsets(state.text, state.query)
		if len(state.matches) == 0 {
			state.index = 0
		} else if state.index >= len(state.matches) {
			state.index = len(state.matches) - 1
		}
	}

	nextMatch := func() {
		syncMatches()
		if len(state.matches) == 0 {
			refreshContent()
			return
		}
		state.index++
		if state.index >= len(state.matches) {
			state.index = 0
		}
		refreshContent()
	}

	prevMatch := func() {
		syncMatches()
		if len(state.matches) == 0 {
			refreshContent()
			return
		}
		state.index--
		if state.index < 0 {
			state.index = len(state.matches) - 1
		}
		refreshContent()
	}

	searchEntry.OnChanged = func(query string) {
		state.query = strings.TrimSpace(query)
		state.matches = findMatchOffsets(state.text, state.query)
		state.index = 0
		refreshContent()
	}

	searchEntry.OnSubmitted = func(string) {
		nextMatch()
	}

	prevBtn := widget.NewButtonWithIcon("Anterior", theme.NavigateBackIcon(), prevMatch)
	nextBtn := widget.NewButtonWithIcon("Próximo", theme.NavigateNextIcon(), nextMatch)
	copyBtn := widget.NewButtonWithIcon("Copiar JSON", theme.ContentCopyIcon(), func() {
		win.Clipboard().SetContent(state.text)
	})

	searchRow := container.NewBorder(
		nil, nil,
		widget.NewLabel("Buscar:"),
		container.NewHBox(prevBtn, nextBtn, copyBtn),
		searchEntry,
	)

	body := container.NewBorder(
		container.NewVBox(searchRow, statusLabel, widget.NewSeparator()),
		nil, nil, nil,
		scroll,
	)

	win.SetContent(body)
	win.Canvas().AddShortcut(&desktop.CustomShortcut{
		KeyName:  fyne.KeyF3,
		Modifier: fyne.KeyModifierShift,
	}, func(fyne.Shortcut) { prevMatch() })
	win.Canvas().AddShortcut(&desktop.CustomShortcut{
		KeyName: fyne.KeyF3,
	}, func(fyne.Shortcut) { nextMatch() })
	win.Canvas().AddShortcut(&desktop.CustomShortcut{
		KeyName:  fyne.KeyReturn,
		Modifier: fyne.KeyModifierShift,
	}, func(fyne.Shortcut) { prevMatch() })

	refreshContent()
	win.Show()
}

func newJSONRichText(text, query string, activeIndex int) *widget.RichText {
	rt := widget.NewRichText()
	rt.Wrapping = fyne.TextWrapOff
	rt.Scroll = fyne.ScrollNone

	normal := widget.RichTextStyle{
		Inline:    true,
		TextStyle: fyne.TextStyle{Monospace: true},
	}
	highlight := widget.RichTextStyle{
		Inline:    true,
		ColorName: theme.ColorNamePrimary,
		TextStyle: fyne.TextStyle{Monospace: true, Bold: true},
	}

	query = strings.TrimSpace(query)
	if query == "" {
		rt.Segments = []widget.RichTextSegment{
			&widget.TextSegment{Text: text, Style: normal},
		}
		return rt
	}

	matches := findMatchOffsets(text, query)
	if len(matches) == 0 || activeIndex < 0 || activeIndex >= len(matches) {
		rt.Segments = []widget.RichTextSegment{
			&widget.TextSegment{Text: text, Style: normal},
		}
		return rt
	}

	qLen := len(query)
	active := matches[activeIndex]
	end := active + qLen
	if end > len(text) {
		end = len(text)
	}

	segments := make([]widget.RichTextSegment, 0, 3)
	if active > 0 {
		segments = append(segments, &widget.TextSegment{Text: text[:active], Style: normal})
	}
	segments = append(segments, &widget.TextSegment{Text: text[active:end], Style: highlight})
	if end < len(text) {
		segments = append(segments, &widget.TextSegment{Text: text[end:], Style: normal})
	}
	rt.Segments = segments
	return rt
}

func scrollToLine(scroll *container.Scroll, content fyne.CanvasObject, row int) {
	content.Refresh()
	th := fyne.CurrentApp().Settings().Theme()
	lineHeight := fyne.MeasureText("A", th.Size(theme.SizeNameText), fyne.TextStyle{Monospace: true}).Height
	y := float32(row) * lineHeight
	if y < 0 {
		y = 0
	}
	scroll.ScrollToOffset(fyne.NewPos(0, y))
}
