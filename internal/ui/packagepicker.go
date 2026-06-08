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

func filterPackages(packages []model.PackageInfo, query string) []model.PackageInfo {
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		if len(packages) > maxPackageResults {
			return packages[:maxPackageResults]
		}
		return packages
	}

	parts := strings.Fields(query)
	filtered := make([]model.PackageInfo, 0, 64)
	for _, pkg := range packages {
		name := strings.ToLower(pkg.Name)
		match := true
		for _, part := range parts {
			if !strings.Contains(name, part) {
				match = false
				break
			}
		}
		if match {
			filtered = append(filtered, pkg)
			if len(filtered) >= maxPackageResults {
				break
			}
		}
	}
	return filtered
}

func showPackagePicker(parent fyne.Window, packages []model.PackageInfo, current []model.PackageInfo, onApply func([]model.PackageInfo)) {
	selected := make(map[string]bool, len(current))
	for _, pkg := range current {
		selected[pkg.Name] = true
	}

	filtered := filterPackages(packages, "")
	countLabel := widget.NewLabel(selectionSummary(selected, len(packages)))
	countLabel.Importance = widget.LowImportance

	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder("Digite para filtrar (ex: pac, com.google)...")

	var list *widget.List
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
				countLabel.SetText(selectionSummary(selected, len(packages)))
			}
		},
	)

	applyFilter := func(query string) {
		filtered = filterPackages(packages, query)
		if len(filtered) == 0 {
			countLabel.SetText("Nenhum pacote encontrado para \"" + strings.TrimSpace(query) + "\"")
		} else {
			countLabel.SetText(filterPreviewSummary(len(filtered), selected, len(packages)))
		}
		list.Refresh()
	}

	searchEntry.OnChanged = applyFilter
	searchEntry.OnSubmitted = func(query string) {
		if len(filtered) == 1 {
			selected[filtered[0].Name] = true
			list.Refresh()
			countLabel.SetText(selectionSummary(selected, len(packages)))
			return
		}
		applyFilter(query)
	}

	selectAllBtn := widget.NewButtonWithIcon("Marcar visíveis", theme.ConfirmIcon(), func() {
		for _, pkg := range filtered {
			selected[pkg.Name] = true
		}
		list.Refresh()
		countLabel.SetText(selectionSummary(selected, len(packages)))
	})

	clearBtn := widget.NewButtonWithIcon("Limpar seleção", theme.CancelIcon(), func() {
		selected = make(map[string]bool)
		list.Refresh()
		countLabel.SetText(selectionSummary(selected, len(packages)))
	})

	header := widget.NewLabelWithStyle("Selecione um ou mais pacotes", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	searchBar := container.NewBorder(nil, nil, widget.NewIcon(theme.SearchIcon()), nil, searchEntry)
	toolbar := container.NewHBox(selectAllBtn, clearBtn)

	content := container.NewBorder(
		container.NewVBox(header, searchBar, toolbar, countLabel),
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

func filterPreviewSummary(visible int, selected map[string]bool, total int) string {
	base := strconv.Itoa(visible) + " exibidos"
	if visible >= maxPackageResults {
		base += " (limite — refine a busca)"
	}
	return base + " · " + selectionSummary(selected, total)
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
