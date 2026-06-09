package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	headerHeight = 88
	headerIcon   = 72
	pillRadius   = 8
	pillPadX     = 8
	pillPadY     = 2
	headerPadX   = 12
	headerPadY   = 8
)

func appTopBar(liveBadge fyne.CanvasObject, themeControl fyne.CanvasObject) fyne.CanvasObject {
	th := fyne.CurrentApp().Settings().Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()

	bg := canvas.NewRectangle(th.Color(theme.ColorNameHeaderBackground, v))
	bg.SetMinSize(fyne.NewSize(0, headerHeight))

	icon := canvas.NewImageFromResource(AppIcon())
	icon.FillMode = canvas.ImageFillContain
	icon.SetMinSize(fyne.NewSize(headerIcon, headerIcon))

	iconSlot := canvas.NewRectangle(color.Transparent)
	iconSlot.SetMinSize(fyne.NewSize(headerIcon, headerIcon))

	subtitle := canvas.NewText("Desenvolvido by @aismor", mutedTextColor(v))
	subtitle.TextSize = th.Size(theme.SizeNameCaptionText)

	titleBlock := container.NewVBox(brandedTitle(), subtitle)
	titleSlot := canvas.NewRectangle(color.Transparent)
	titleSlot.SetMinSize(fyne.NewSize(0, headerIcon))

	leftInner := container.NewHBox(
		container.NewStack(iconSlot, icon),
		spacer(10),
		container.NewStack(titleSlot, container.NewCenter(titleBlock)),
	)
	left := container.NewCenter(leftInner)

	themeLabel := canvas.NewText("Tema:", mutedTextColor(v))
	themeLabel.TextSize = th.Size(theme.SizeNameCaptionText)

	rightInner := container.NewHBox(
		liveBadge,
		spacer(12),
		themeLabel,
		spacer(6),
		themeControl,
	)
	right := container.NewCenter(rightInner)

	row := container.NewBorder(nil, nil, left, right, nil)
	padded := container.New(&insetLayout{padX: headerPadX, padY: headerPadY}, row)
	return container.NewStack(bg, padded)
}

func brandedTitle() fyne.CanvasObject {
	th := fyne.CurrentApp().Settings().Theme()
	size := th.Size(theme.SizeNameSubHeadingText)

	logcat := canvas.NewText("Logcat ", titleWhite)
	logcat.TextSize = size
	logcat.TextStyle = fyne.TextStyle{Bold: true}

	goText := canvas.NewText("Go", androidGreen)
	goText.TextSize = size
	goText.TextStyle = fyne.TextStyle{Bold: true}

	return container.NewHBox(logcat, goText)
}

func newLiveToggleButton(active bool, onChange func(bool)) *liveToggleButton {
	t := &liveToggleButton{active: active, onChange: onChange}
	t.ExtendBaseWidget(t)
	t.applyVisual()
	return t
}

type liveToggleButton struct {
	widget.BaseWidget
	active   bool
	onChange func(bool)
	dot      *canvas.Circle
	label    *canvas.Text
	bg       *canvas.Rectangle
	border   *canvas.Rectangle
}

func (t *liveToggleButton) CreateRenderer() fyne.WidgetRenderer {
	th := fyne.CurrentApp().Settings().Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()

	t.dot = canvas.NewCircle(androidGreen)
	t.dot.Resize(fyne.NewSize(7, 7))

	t.label = canvas.NewText("Ao vivo", titleWhite)
	t.label.TextSize = th.Size(theme.SizeNameCaptionText)

	t.bg = canvas.NewRectangle(th.Color(theme.ColorNameInputBackground, v))
	t.bg.CornerRadius = pillRadius

	t.border = canvas.NewRectangle(color.Transparent)
	t.border.CornerRadius = pillRadius
	t.border.StrokeWidth = 1

	t.applyVisual()
	return &liveToggleRenderer{btn: t}
}

func (t *liveToggleButton) Tapped(*fyne.PointEvent) {
	t.SetActive(!t.active)
	if t.onChange != nil {
		t.onChange(t.active)
	}
}

func (t *liveToggleButton) SetActive(active bool) {
	if t.active == active {
		return
	}
	t.active = active
	t.applyVisual()
	t.Refresh()
}

func (t *liveToggleButton) Active() bool {
	return t.active
}

func (t *liveToggleButton) applyVisual() {
	if t.dot == nil || t.label == nil || t.border == nil {
		return
	}
	th := fyne.CurrentApp().Settings().Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()
	if t.active {
		t.dot.FillColor = androidGreen
		t.label.Color = titleWhite
		t.border.StrokeColor = androidGreen
	} else {
		t.dot.FillColor = th.Color(theme.ColorNameDisabled, v)
		t.label.Color = mutedTextColor(v)
		t.border.StrokeColor = th.Color(theme.ColorNameSeparator, v)
	}
}

type liveToggleRenderer struct {
	btn *liveToggleButton
}

func (r *liveToggleRenderer) Layout(size fyne.Size) {
	min := r.MinSize()
	w := min.Width
	if size.Width > w {
		w = size.Width
	}
	h := min.Height
	yOff := float32(0)
	if size.Height > h {
		yOff = (size.Height - h) / 2
	}

	r.btn.bg.Resize(fyne.NewSize(w, h))
	r.btn.bg.Move(fyne.NewPos(0, yOff))
	r.btn.border.Resize(fyne.NewSize(w, h))
	r.btn.border.Move(fyne.NewPos(0, yOff))

	dotSize := float32(7)
	labelSize := r.btn.label.MinSize()
	contentH := dotSize
	if labelSize.Height > contentH {
		contentH = labelSize.Height
	}
	y := yOff + (h-contentH)/2

	r.btn.dot.Resize(fyne.NewSize(dotSize, dotSize))
	r.btn.dot.Move(fyne.NewPos(pillPadX, y+(contentH-dotSize)/2))

	labelX := pillPadX + dotSize + 5
	r.btn.label.Move(fyne.NewPos(labelX, y+(contentH-labelSize.Height)/2))
}

func (r *liveToggleRenderer) MinSize() fyne.Size {
	labelSize := r.btn.label.MinSize()
	w := pillPadX*2 + 7 + 5 + labelSize.Width
	h := pillPadY*2 + labelSize.Height
	if h < pillPadY*2+7 {
		h = pillPadY*2 + 7
	}
	return fyne.NewSize(w, h)
}

func (r *liveToggleRenderer) Refresh() {
	r.btn.applyVisual()
	r.btn.dot.Refresh()
	r.btn.label.Refresh()
	r.btn.bg.Refresh()
	r.btn.border.Refresh()
	canvas.Refresh(r.btn)
}

func (r *liveToggleRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.btn.bg, r.btn.border, r.btn.dot, r.btn.label}
}

func (r *liveToggleRenderer) Destroy() {}

func newThemeMenuButton(parent fyne.Window, onChange func(string)) fyne.CanvasObject {
	current := ThemeDisplayName(CurrentTheme())
	themeIcon := themeModeIcon(CurrentTheme())
	btn := widget.NewButtonWithIcon(current, themeIcon, nil)
	btn.Importance = widget.LowImportance

	btn.OnTapped = func() {
		menu := fyne.NewMenu("",
			fyne.NewMenuItem("Escuro", func() {
				ApplyTheme(ThemeDark)
				btn.SetText("Escuro")
				btn.SetIcon(themeModeIcon(ThemeDark))
				onChange(ThemeDark)
			}),
			fyne.NewMenuItem("Claro", func() {
				ApplyTheme(ThemeLight)
				btn.SetText("Claro")
				btn.SetIcon(themeModeIcon(ThemeLight))
				onChange(ThemeLight)
			}),
		)
		pop := widget.NewPopUpMenu(menu, parent.Canvas())
		pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(btn)
		pop.ShowAtPosition(pos.Add(fyne.NewPos(0, btn.Size().Height)))
	}

	return pillWrap(btn)
}

func themeModeIcon(mode string) fyne.Resource {
	if mode == ThemeLight {
		return theme.ColorChromaticIcon()
	}
	return theme.NewThemedResource(theme.MediaRecordIcon())
}

func pillWrap(content fyne.CanvasObject) fyne.CanvasObject {
	th := fyne.CurrentApp().Settings().Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()

	fill := canvas.NewRectangle(th.Color(theme.ColorNameInputBackground, v))
	fill.CornerRadius = pillRadius

	border := canvas.NewRectangle(color.Transparent)
	border.CornerRadius = pillRadius
	border.StrokeWidth = 1
	border.StrokeColor = th.Color(theme.ColorNameSeparator, v)

	padded := container.New(&insetLayout{padX: pillPadX, padY: pillPadY}, content)
	wrapped := container.NewStack(fill, border, padded)
	return container.NewCenter(wrapped)
}

func mutedTextColor(v fyne.ThemeVariant) color.Color {
	if v == theme.VariantLight {
		return subtitleGrey
	}
	return subtitleGrey
}

func spacer(w float32) fyne.CanvasObject {
	r := canvas.NewRectangle(color.Transparent)
	r.SetMinSize(fyne.NewSize(w, 1))
	return r
}

type insetLayout struct {
	padX, padY float32
}

func (l *insetLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) == 0 {
		return
	}
	objects[0].Resize(fyne.NewSize(size.Width-l.padX*2, size.Height-l.padY*2))
	objects[0].Move(fyne.NewPos(l.padX, l.padY))
}

func (l *insetLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) == 0 {
		return fyne.Size{}
	}
	min := objects[0].MinSize()
	return fyne.NewSize(min.Width+l.padX*2, min.Height+l.padY*2)
}

func cardHeading(title, subtitle string, actions fyne.CanvasObject) fyne.CanvasObject {
	head := widget.NewLabel(title)
	head.TextStyle = fyne.TextStyle{Bold: true}

	sub := widget.NewLabel(subtitle)
	sub.Importance = widget.LowImportance

	titleCol := container.NewVBox(head, sub)
	if actions == nil {
		return titleCol
	}
	return container.NewBorder(nil, nil, nil, actions, titleCol)
}

func statusFooter(left, right fyne.CanvasObject) fyne.CanvasObject {
	th := fyne.CurrentApp().Settings().Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()

	bg := canvas.NewRectangle(th.Color(theme.ColorNameHeaderBackground, v))
	bg.SetMinSize(fyne.NewSize(0, 36))

	row := container.NewBorder(nil, nil, left, right, nil)
	padded := container.NewPadded(row)
	return container.NewStack(bg, padded)
}

func statusSegment(icon fyne.Resource, text *widget.Label) fyne.CanvasObject {
	return container.NewHBox(widget.NewIcon(icon), text)
}

func newConnectionStatus(connected bool, device string) (*canvas.Circle, *widget.Label) {
	dot := canvas.NewCircle(androidGreen)
	dot.Resize(fyne.NewSize(8, 8))

	label := widget.NewLabel("Desconectado")
	label.Importance = widget.LowImportance

	if connected && device != "" {
		label.SetText("Conectado a " + device)
	} else {
		th := fyne.CurrentApp().Settings().Theme()
		v := fyne.CurrentApp().Settings().ThemeVariant()
		dot.FillColor = th.Color(theme.ColorNameDisabled, v)
	}
	return dot, label
}
