package ui

import (
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"

	"github.com/aismor/logcat-go/internal/adb"
	"github.com/aismor/logcat-go/internal/model"
)

const uiRefreshDelay = 250 * time.Millisecond

type LogView struct {
	root            *logViewRoot
	entry           *LogEntry
	store           *adb.LogStore
	parent          fyne.Window
	formattedBlocks []string
	pendingBlocks   []string
	displayText     string
	searchQuery     string
	autoFollow      bool
	updating        bool
	refreshPending  bool
	fullSyncPending bool
}

func NewLogView(store *adb.LogStore, parent fyne.Window) *LogView {
	view := &LogView{
		store:      store,
		parent:     parent,
		autoFollow: true,
	}

	view.entry = NewLogEntry(func(raw string) {
		ShowJSONViewer(parent, raw)
	})
	view.entry.Wrapping = fyne.TextWrapBreak
	view.entry.Scroll = fyne.ScrollNone
	view.entry.TextStyle = fyne.TextStyle{Monospace: true}
	view.entry.SetPlaceHolder("Logcat — clique em Iniciar para capturar logs ao vivo")
	view.entry.OnChanged = func(text string) {
		if view.updating || text == view.displayText {
			return
		}
		view.updating = true
		view.entry.SetText(view.displayText)
		view.updating = false
	}

	view.root = newLogViewRoot(view, view.entry)
	return view
}

func (v *LogView) Container() fyne.CanvasObject {
	return v.root
}

type logViewRoot struct {
	widget.BaseWidget
	view  *LogView
	entry *LogEntry
	lastW float32
}

func newLogViewRoot(view *LogView, entry *LogEntry) *logViewRoot {
	r := &logViewRoot{view: view, entry: entry}
	r.ExtendBaseWidget(r)
	return r
}

func (r *logViewRoot) CreateRenderer() fyne.WidgetRenderer {
	return &logViewRootRenderer{root: r}
}

type logViewRootRenderer struct {
	root *logViewRoot
}

func (r *logViewRootRenderer) Layout(size fyne.Size) {
	r.root.entry.Resize(size)
	r.root.entry.Move(fyne.NewPos(0, 0))
	if size.Width != r.root.lastW {
		r.root.lastW = size.Width
		r.root.entry.Refresh()
		if r.root.view.autoFollow {
			r.root.view.scrollToEnd()
		}
	}
}

func (r *logViewRootRenderer) MinSize() fyne.Size {
	return fyne.NewSize(0, 200)
}

func (r *logViewRootRenderer) Refresh() {
	r.root.entry.Refresh()
	canvas.Refresh(r.root)
}

func (r *logViewRootRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.root.entry}
}

func (r *logViewRootRenderer) Destroy() {}

func (v *LogView) SetAutoFollow(enabled bool) {
	v.autoFollow = enabled
	if enabled {
		v.scrollToEnd()
	}
}

func (v *LogView) AutoFollow() bool {
	return v.autoFollow
}

func (v *LogView) SetSearch(query string) {
	v.searchQuery = strings.TrimSpace(query)
	v.rebuildDisplayBlocks()
	v.applyDisplayText(false)
}

func (v *LogView) SearchQuery() string {
	return v.searchQuery
}

func (v *LogView) StoreLen() int {
	return v.store.Len()
}

func (v *LogView) FilteredLen() int {
	if v.searchQuery == "" {
		return v.store.Len()
	}
	return len(filterEntriesBySearch(v.store.All(), v.searchQuery))
}

func (v *LogView) rebuildDisplayBlocks() {
	entries := v.store.All()
	if v.searchQuery != "" {
		entries = filterEntriesBySearch(entries, v.searchQuery)
	}

	blocks := make([]string, 0, len(entries))
	for _, entry := range entries {
		blocks = append(blocks, formatDisplayBlock(entry))
	}
	v.formattedBlocks = blocks
	v.displayText = buildDisplayTextFromBlocks(blocks)
}

func (v *LogView) applyDisplayText(scroll bool) {
	if v.entry.Text == v.displayText {
		if scroll && v.autoFollow {
			v.scrollToEnd()
		}
		return
	}

	v.updating = true
	v.entry.SetText(v.displayText)
	v.updating = false
	v.entry.Refresh()

	if scroll && v.autoFollow {
		v.scrollToEnd()
	}
}

func (v *LogView) appendDisplayChunk(chunk string) {
	if chunk == "" {
		return
	}

	v.displayText += chunk
	v.updating = true
	if v.entry.Text == "" {
		v.entry.SetText(chunk)
	} else {
		v.entry.Append(chunk)
	}
	v.updating = false
	v.entry.Refresh()

	if v.autoFollow {
		v.scrollToEnd()
	}
}

func (v *LogView) scrollToEnd() {
	v.scrollToEndDeferred()
}

func (v *LogView) scrollToEndDeferred() {
	scroll := func() {
		if v.entry == nil {
			return
		}
		v.entry.Refresh()
		v.entry.ScrollToEnd()
	}

	fyne.Do(scroll)
	time.AfterFunc(20*time.Millisecond, func() { fyne.Do(scroll) })
	time.AfterFunc(80*time.Millisecond, func() { fyne.Do(scroll) })
	time.AfterFunc(200*time.Millisecond, func() { fyne.Do(scroll) })
	time.AfterFunc(400*time.Millisecond, func() { fyne.Do(scroll) })
}

func (v *LogView) Clear() {
	v.store.Clear()
	v.formattedBlocks = nil
	v.pendingBlocks = nil
	v.displayText = ""
	v.updating = true
	v.entry.SetText("")
	v.updating = false
}

func (v *LogView) ApplyBatch(entries []model.LogEntry, _ *model.LogEntry) {
	before := v.store.Len()

	for _, entry := range entries {
		v.store.Append(entry)
	}

	trimmed := len(entries) > 0 && v.store.Len() < before+len(entries)

	if trimmed || v.searchQuery != "" {
		v.pendingBlocks = nil
		v.scheduleRefresh(true)
		return
	}

	for _, entry := range entries {
		v.pendingBlocks = append(v.pendingBlocks, formatDisplayBlock(entry))
	}
	v.scheduleRefresh(false)
}

func (v *LogView) SelectedText() string {
	return v.entry.SelectedText()
}

func (v *LogView) FormatSelectedJSON() error {
	selected := strings.TrimSpace(v.entry.SelectedText())
	if selected == "" {
		return fmt.Errorf("selecione um trecho com JSON no log")
	}
	jsonText, ok := extractJSON(selected)
	if !ok {
		return fmt.Errorf("seleção não contém JSON válido")
	}
	ShowJSONViewer(v.parent, jsonText)
	return nil
}

func (v *LogView) CopySelection(clipboard fyne.Clipboard, onComplete func()) (string, bool) {
	if selected := v.entry.SelectedText(); selected != "" {
		clipboard.SetContent(selected)
		return selected, false
	}

	entries := v.store.All()
	query := v.searchQuery
	go func() {
		filtered := entries
		if query != "" {
			filtered = filterEntriesBySearch(entries, query)
		}
		text := strings.TrimSpace(buildPlainText(filtered))
		if text == "" {
			return
		}
		fyne.Do(func() {
			clipboard.SetContent(text)
			if onComplete != nil {
				onComplete()
			}
		})
	}()

	return "", true
}

func (v *LogView) scheduleRefresh(fullSync bool) {
	if v.refreshPending {
		if fullSync {
			v.fullSyncPending = true
			v.pendingBlocks = nil
		}
		return
	}
	v.refreshPending = true
	if fullSync {
		v.fullSyncPending = true
	}

	time.AfterFunc(uiRefreshDelay, func() {
		fyne.Do(func() {
			if v.fullSyncPending {
				v.rebuildDisplayBlocks()
				v.pendingBlocks = nil
				v.fullSyncPending = false
				v.applyDisplayText(v.autoFollow)
			} else if len(v.pendingBlocks) > 0 {
				blocks := v.pendingBlocks
				v.pendingBlocks = nil
				v.formattedBlocks = append(v.formattedBlocks, blocks...)

				newText := buildDisplayTextFromBlocks(v.formattedBlocks)
				if strings.HasPrefix(newText, "…\n") || newText != v.displayText+pendingChunk(blocks) {
					v.displayText = newText
					v.applyDisplayText(v.autoFollow)
				} else {
					v.appendDisplayChunk(pendingChunk(blocks))
				}
			}
			v.refreshPending = false
		})
	})
}

func pendingChunk(blocks []string) string {
	var b strings.Builder
	for i, block := range blocks {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(block)
	}
	if len(blocks) > 0 {
		b.WriteByte('\n')
	}
	return b.String()
}
