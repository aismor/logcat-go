package ui

import (
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/aismor/logcat-go/internal/model"
)

const maxPackageResults = 200

var systemPackagePrefixes = []string{
	"com.android.",
	"android.",
	"com.google.android.",
	"com.qualcomm.",
	"com.mediatek.",
	"com.sec.",
	"com.samsung.",
	"com.miui.",
	"com.xiaomi.",
	"com.huawei.",
	"com.coloros.",
	"com.oppo.",
	"com.vivo.",
	"com.oneplus.",
}

func isSystemPackage(name string) bool {
	if name == "android" {
		return true
	}
	for _, prefix := range systemPackagePrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func filterPackageList(packages []model.PackageInfo, query string, showSystem bool) []model.PackageInfo {
	query = strings.TrimSpace(strings.ToLower(query))
	parts := strings.Fields(query)

	filtered := make([]model.PackageInfo, 0, 64)
	for _, pkg := range packages {
		if !showSystem && isSystemPackage(pkg.Name) {
			continue
		}
		if query != "" {
			name := strings.ToLower(pkg.Name)
			match := true
			for _, part := range parts {
				if !strings.Contains(name, part) {
					match = false
					break
				}
			}
			if !match {
				continue
			}
		}
		filtered = append(filtered, pkg)
		if len(filtered) >= maxPackageResults {
			break
		}
	}
	return filtered
}

func showPackagePicker(parent fyne.Window, packages []model.PackageInfo, current []model.PackageInfo, initialQuery string, onApply func([]model.PackageInfo)) {
	selected := make(map[string]bool, len(current))
	for _, pkg := range current {
		selected[pkg.Name] = true
	}

	showSystem := false
	initialQuery = strings.TrimSpace(initialQuery)
	filtered := filterPackageList(packages, initialQuery, showSystem)
	countLabel := widget.NewLabel(packageListSummary(packages, filtered, selected, showSystem))
	countLabel.Importance = widget.LowImportance

	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder("Digite para buscar pacotes (ex: com.meuapp)...")
	if initialQuery != "" {
		searchEntry.SetText(initialQuery)
	}

	var list *widget.List
	var refreshList func()

	showSystemCheck := widget.NewCheck("Mostrar pacotes do sistema", func(checked bool) {
		showSystem = checked
		filtered = filterPackageList(packages, searchEntry.Text, showSystem)
		countLabel.SetText(packageListSummary(packages, filtered, selected, showSystem))
		refreshList()
	})
	showSystemCheck.SetChecked(false)

	list = widget.NewList(
		func() int {
			return len(filtered)
		},
		func() fyne.CanvasObject {
			check := widget.NewCheck("", nil)
			label := widget.NewLabel("")
			label.TextStyle = fyne.TextStyle{Monospace: true}
			label.Wrapping = fyne.TextWrapOff
			return container.NewHBox(check, label)
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			if id < 0 || id >= len(filtered) {
				return
			}
			pkg := filtered[id]
			row := item.(*fyne.Container)
			check := row.Objects[0].(*widget.Check)
			label := row.Objects[1].(*widget.Label)

			label.SetText(pkg.Name)
			check.SetChecked(selected[pkg.Name])
			check.OnChanged = func(checked bool) {
				if checked {
					selected[pkg.Name] = true
				} else {
					delete(selected, pkg.Name)
				}
				countLabel.SetText(packageListSummary(packages, filtered, selected, showSystem))
			}
		},
	)

	refreshList = func() {
		list.Refresh()
	}

	applyFilter := func(query string) {
		filtered = filterPackageList(packages, query, showSystem)
		if len(filtered) == 0 {
			countLabel.SetText("Nenhum pacote encontrado para \"" + strings.TrimSpace(query) + "\"")
		} else {
			countLabel.SetText(packageListSummary(packages, filtered, selected, showSystem))
		}
		list.Refresh()
	}

	searchEntry.OnChanged = applyFilter
	searchEntry.OnSubmitted = func(query string) {
		if len(filtered) == 1 {
			selected[filtered[0].Name] = true
			list.Refresh()
			countLabel.SetText(packageListSummary(packages, filtered, selected, showSystem))
			return
		}
		applyFilter(query)
	}

	selectAllBtn := widget.NewButtonWithIcon("Marcar visíveis", theme.ConfirmIcon(), func() {
		for _, pkg := range filtered {
			selected[pkg.Name] = true
		}
		list.Refresh()
		countLabel.SetText(packageListSummary(packages, filtered, selected, showSystem))
	})

	clearBtn := widget.NewButtonWithIcon("Limpar seleção", theme.CancelIcon(), func() {
		selected = make(map[string]bool)
		list.Refresh()
		countLabel.SetText(packageListSummary(packages, filtered, selected, showSystem))
	})

	header := widget.NewLabelWithStyle("Buscar e selecionar pacotes", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	searchBar := container.NewBorder(nil, nil, widget.NewIcon(theme.SearchIcon()), nil, searchEntry)
	toolbar := container.NewHBox(selectAllBtn, clearBtn)

	content := container.NewBorder(
		container.NewVBox(header, searchBar, showSystemCheck, toolbar, countLabel),
		nil, nil, nil,
		list,
	)
	content.Resize(fyne.NewSize(720, 520))

	d := dialog.NewCustomConfirm(
		"Pacotes instalados",
		"Confirmar",
		"Cancelar",
		content,
		func(ok bool) {
			if !ok {
				return
			}
			onApply(packagesFromSelection(packages, selected))
		},
		parent,
	)
	d.Resize(fyne.NewSize(760, 560))
	d.Show()
}

func packageListSummary(all, visible []model.PackageInfo, selected map[string]bool, showSystem bool) string {
	base := selectionSummary(selected, len(all))
	visibleCount := len(visible)
	if visibleCount >= maxPackageResults {
		base = strconv.Itoa(visibleCount) + " exibidos (limite — refine a busca) · " + base
	} else if visibleCount != len(all) {
		base = strconv.Itoa(visibleCount) + " exibidos · " + base
	}
	if !showSystem {
		hidden := countSystemPackages(all)
		if hidden > 0 {
			base += " · " + strconv.Itoa(hidden) + " do sistema ocultos"
		}
	}
	return base
}

func countSystemPackages(packages []model.PackageInfo) int {
	n := 0
	for _, pkg := range packages {
		if isSystemPackage(pkg.Name) {
			n++
		}
	}
	return n
}

func packagesFromSelection(all []model.PackageInfo, selected map[string]bool) []model.PackageInfo {
	known := make(map[string]model.PackageInfo, len(all))
	for _, pkg := range all {
		known[pkg.Name] = pkg
	}

	out := make([]model.PackageInfo, 0, len(selected))
	for name := range selected {
		if pkg, ok := known[name]; ok {
			out = append(out, pkg)
			continue
		}
		out = append(out, model.PackageInfo{Name: name})
	}
	return out
}

func selectionSummary(selected map[string]bool, total int) string {
	n := len(selected)
	switch {
	case n == 0:
		return "Nenhum pacote selecionado · " + strconv.Itoa(total) + " instalados"
	case n == 1:
		for name := range selected {
			return "1 selecionado: " + name
		}
	default:
		return strconv.Itoa(n) + " pacotes selecionados · " + strconv.Itoa(total) + " instalados"
	}
	return ""
}

func formatPackageSelection(packages []model.PackageInfo) string {
	switch len(packages) {
	case 0:
		return "Nenhum pacote selecionado"
	case 1:
		return packages[0].Name
	case 2:
		return packages[0].Name + ", " + packages[1].Name
	default:
		return packages[0].Name + " +" + strconv.Itoa(len(packages)-1) + " pacotes"
	}
}

type packageSearchField struct {
	widget.Entry
	onOpen      func(query string)
	updating    bool
	displayText string
}

func newPackageSearchField(onOpen func(string)) *packageSearchField {
	f := &packageSearchField{onOpen: onOpen}
	f.ExtendBaseWidget(f)
	f.SetPlaceHolder("Buscar e selecionar pacotes...")
	f.OnSubmitted = f.handleSubmit
	return f
}

func (f *packageSearchField) Tapped(*fyne.PointEvent) {
	if f.onOpen == nil {
		return
	}
	query := strings.TrimSpace(f.Text)
	if query == f.displayText {
		query = ""
	}
	f.onOpen(query)
}

func (f *packageSearchField) handleSubmit(query string) {
	if f.onOpen != nil {
		f.onOpen(strings.TrimSpace(query))
	}
	f.updating = true
	f.SetText(f.displayText)
	f.updating = false
}

func (f *packageSearchField) SetDisplay(text string) {
	f.displayText = text
	f.updating = true
	f.SetText(text)
	f.updating = false
}
