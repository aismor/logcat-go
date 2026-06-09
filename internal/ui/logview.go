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

const uiRefreshDelay = 200 * time.Millisecond

type LogView struct {
	root             *logListRoot
	list             *widget.List
	store            *adb.LogStore
	parent           fyne.Window
	displayEntries   []model.LogEntry
	searchQuery      string
	autoFollow       bool
	selectedID       widget.ListItemID
	contentWidth     float32
	lastDisplayLen   int
	refreshPending   bool
	fullSyncPending  bool
}

func NewLogView(store *adb.LogStore, parent fyne.Window) *LogView {
	view := &LogView{
		store:        store,
		parent:       parent,
		autoFollow:   true,
		selectedID:   -1,
		contentWidth: 640,
	}

	view.list = widget.NewList(
		func() int {
			return len(view.displayEntries)
		},
		func() fyne.CanvasObject {
			return newLogRowItem()
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			if id < 0 || int(id) >= len(view.displayEntries) {
				return
			}
			height := updateLogRowItem(item, view.displayEntries[id], view.rowContentWidth())
			view.list.SetItemHeight(id, height)
		},
	)
	view.list.OnSelected = func(id widget.ListItemID) {
		view.selectedID = id
	}

	view.root = newLogListRoot(view, view.list)
	return view
}

func (v *LogView) Container() fyne.CanvasObject {
	return v.root
}

func (v *LogView) rowContentWidth() float32 {
	w := v.contentWidth - rowHorizontalPadding*2
	if w <= 16 {
		return 640
	}
	return w
}

func (v *LogView) setContentWidth(width float32) {
	if width <= 0 || width == v.contentWidth {
		return
	}
	v.contentWidth = width
	v.refreshHeightsFrom(0)
}

func (v *LogView) refreshHeightsFrom(start int) {
	if start < 0 {
		start = 0
	}
	w := v.rowContentWidth()
	for i := start; i < len(v.displayEntries); i++ {
		height := measureRowHeight(v.displayEntries[i], w)
		v.list.SetItemHeight(widget.ListItemID(i), height)
	}
}

type logListRoot struct {
	widget.BaseWidget
	view  *LogView
	list  *widget.List
	lastW float32
}

func newLogListRoot(view *LogView, list *widget.List) *logListRoot {
	r := &logListRoot{view: view, list: list}
	r.ExtendBaseWidget(r)
	return r
}

func (r *logListRoot) CreateRenderer() fyne.WidgetRenderer {
	return &logListRootRenderer{root: r}
}

type logListRootRenderer struct {
	root *logListRoot
}

func (r *logListRootRenderer) Layout(size fyne.Size) {
	r.root.list.Resize(size)
	r.root.list.Move(fyne.NewPos(0, 0))
	if size.Width > 0 && size.Width != r.root.lastW {
		r.root.lastW = size.Width
		r.root.view.setContentWidth(size.Width)
	}
}

func (r *logListRootRenderer) MinSize() fyne.Size {
	return fyne.NewSize(0, 200)
}

func (r *logListRootRenderer) Refresh() {
	r.root.list.Refresh()
	canvas.Refresh(r.root)
}

func (r *logListRootRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.root.list}
}

func (r *logListRootRenderer) Destroy() {}

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
	v.syncDisplayEntries()
	v.refreshHeightsFrom(0)
	v.list.Refresh()
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

func (v *LogView) syncDisplayEntries() {
	entries := v.store.All()
	if v.searchQuery != "" {
		entries = filterEntriesBySearch(entries, v.searchQuery)
	}
	if len(entries) > maxDisplayBlocks {
		entries = entries[len(entries)-maxDisplayBlocks:]
	}
	v.displayEntries = entries
}

func (v *LogView) scrollToEnd() {
	if len(v.displayEntries) == 0 {
		return
	}
	fyne.Do(func() {
		v.list.ScrollToBottom()
	})
}

func (v *LogView) Clear() {
	v.store.Clear()
	v.displayEntries = nil
	v.lastDisplayLen = 0
	v.selectedID = -1
	v.list.UnselectAll()
	v.list.Refresh()
}

func (v *LogView) ApplyBatch(entries []model.LogEntry, _ *model.LogEntry) {
	before := v.store.Len()

	for _, entry := range entries {
		if entry.IsUpdate {
			if !v.store.UpdateLast(entry) {
				v.store.Append(entry)
			}
			continue
		}
		v.store.Append(entry)
	}

	trimmed := len(entries) > 0 && v.store.Len() < before+len(entries)
	if trimmed || v.searchQuery != "" {
		v.scheduleRefresh(true)
		return
	}
	v.scheduleRefresh(false)
}

func (v *LogView) selectedEntry() (model.LogEntry, bool) {
	if v.selectedID < 0 || int(v.selectedID) >= len(v.displayEntries) {
		return model.LogEntry{}, false
	}
	return v.displayEntries[v.selectedID], true
}

func (v *LogView) SelectedText() string {
	entry, ok := v.selectedEntry()
	if !ok {
		return ""
	}
	return formatPlainLine(entry)
}

func (v *LogView) FormatSelectedJSON() error {
	if v.selectedID < 0 || int(v.selectedID) >= len(v.displayEntries) {
		return fmt.Errorf("clique em uma linha com JSON no log")
	}
	jsonText, ok := resolveJSONFromEntries(v.displayEntries, int(v.selectedID))
	if !ok {
		return fmt.Errorf("linha selecionada não contém JSON válido")
	}
	ShowJSONViewer(v.parent, jsonText)
	return nil
}

func (v *LogView) CopySelection(clipboard fyne.Clipboard, onComplete func()) (string, bool) {
	if entry, ok := v.selectedEntry(); ok {
		line := formatPlainLine(entry)
		clipboard.SetContent(line)
		return line, false
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
		}
		return
	}
	v.refreshPending = true
	if fullSync {
		v.fullSyncPending = true
	}

	time.AfterFunc(uiRefreshDelay, func() {
		fyne.Do(func() {
			prevLen := v.lastDisplayLen
			v.syncDisplayEntries()

			start := 0
			if !v.fullSyncPending && prevLen > 0 && len(v.displayEntries) >= prevLen {
				start = prevLen - 1
			}
			v.refreshHeightsFrom(start)
			v.lastDisplayLen = len(v.displayEntries)

			v.list.Refresh()
			if v.autoFollow {
				v.scrollToEnd()
			}
			v.refreshPending = false
			v.fullSyncPending = false
		})
	})
}
