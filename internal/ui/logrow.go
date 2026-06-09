package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/aismor/logcat-go/internal/model"
)

const (
	rowHorizontalPadding = float32(8)
	rowVerticalPadding   = float32(6)
	defaultRowHeight     = float32(24)
)

type logRowParts struct {
	bg   *canvas.Rectangle
	text *widget.RichText
}

func newLogRowItem() fyne.CanvasObject {
	bg := canvas.NewRectangle(color.Transparent)
	rt := widget.NewRichText()
	rt.Wrapping = fyne.TextWrapBreak
	padded := container.NewPadded(rt)
	return container.NewStack(bg, padded)
}

func logRowFromItem(obj fyne.CanvasObject) *logRowParts {
	stack, ok := obj.(*fyne.Container)
	if !ok || len(stack.Objects) < 2 {
		return nil
	}
	bg, ok := stack.Objects[0].(*canvas.Rectangle)
	if !ok {
		return nil
	}
	padded, ok := stack.Objects[1].(*fyne.Container)
	if !ok || len(padded.Objects) == 0 {
		return nil
	}
	rt, ok := padded.Objects[0].(*widget.RichText)
	if !ok {
		return nil
	}
	return &logRowParts{bg: bg, text: rt}
}

func logSegments(entry model.LogEntry) []widget.RichTextSegment {
	mono := fyne.TextStyle{Monospace: true}
	level := normalizeLevel(entry.Level)
	levelColor := levelThemeColor(entry.Level)
	msgColor := messageThemeColor(entry.Level)

	metaStyle := widget.RichTextStyle{
		Inline:    true,
		ColorName: themeColorLogMeta,
		TextStyle: mono,
	}
	levelStyle := widget.RichTextStyle{
		Inline:    true,
		ColorName: levelColor,
		TextStyle: fyne.TextStyle{Monospace: true, Bold: true},
	}
	msgStyle := widget.RichTextStyle{
		Inline:    true,
		ColorName: msgColor,
		TextStyle: mono,
	}

	meta := lineMetaPrefix(entry)
	msg := messageText(entry)

	segments := make([]widget.RichTextSegment, 0, 3)
	if meta != "" {
		segments = append(segments, &widget.TextSegment{Text: meta + " ", Style: metaStyle})
	}
	segments = append(segments,
		&widget.TextSegment{Text: level + " ", Style: levelStyle},
		&widget.TextSegment{Text: msg, Style: msgStyle},
	)
	return segments
}

func measureRowHeight(entry model.LogEntry, contentWidth float32) float32 {
	if contentWidth <= 16 {
		return defaultRowHeight
	}

	rt := widget.NewRichText()
	rt.Wrapping = fyne.TextWrapBreak
	rt.Segments = logSegments(entry)
	rt.Resize(fyne.NewSize(contentWidth, 1))

	th := fyne.CurrentApp().Settings().Theme()
	padding := th.Size(theme.SizeNamePadding)
	return rt.MinSize().Height + rowVerticalPadding + padding
}

func updateLogRowItem(obj fyne.CanvasObject, entry model.LogEntry, contentWidth float32) float32 {
	parts := logRowFromItem(obj)
	if parts == nil {
		return defaultRowHeight
	}

	parts.bg.FillColor = levelBackground(entry.Level)
	parts.text.Segments = logSegments(entry)
	if contentWidth > 16 {
		parts.text.Resize(fyne.NewSize(contentWidth, 1))
	}

	height := parts.text.MinSize().Height
	if height < defaultRowHeight-rowVerticalPadding {
		height = defaultRowHeight - rowVerticalPadding
	}

	th := fyne.CurrentApp().Settings().Theme()
	total := height + rowVerticalPadding + th.Size(theme.SizeNamePadding)

	parts.bg.Refresh()
	parts.text.Refresh()
	return total
}
