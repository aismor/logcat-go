package ui

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/aismor/logcat-go/internal/adb"
	"github.com/aismor/logcat-go/internal/model"
)

type App struct {
	window fyne.Window
	ctx    context.Context
	cancel context.CancelFunc

	deviceSelect   *widget.Select
	packageSummary *widget.Label
	modeSelect     *widget.Select
	searchEntry    *widget.Entry
	statusLabel    *widget.Label
	logView        *LogView
	store          *adb.LogStore

	startBtn    *widget.Button
	stopBtn     *widget.Button
	packagesBtn *widget.Button
	followCheck *widget.Check

	devices           []model.Device
	packages          []model.PackageInfo
	selectedPackages  []model.PackageInfo
	packagesLoaded    bool

	sessionMu sync.Mutex
	session   *adb.LogcatSession

	searchTimer *time.Timer
}

func NewApp() *App {
	ctx, cancel := context.WithCancel(context.Background())
	return &App{
		ctx:    ctx,
		cancel: cancel,
		store:  adb.NewLogStore(0),
	}
}

func (a *App) Run() {
	a.window = fyne.CurrentApp().NewWindow("Logcat Go")
	a.window.Resize(fyne.NewSize(1200, 760))
	a.window.SetContent(a.buildLayout())
	a.window.SetCloseIntercept(func() {
		a.stopLogcat()
		a.cancel()
		a.window.Close()
	})
	a.refreshDevices()
	a.window.ShowAndRun()
}

func (a *App) buildLayout() fyne.CanvasObject {
	a.deviceSelect = widget.NewSelect([]string{}, func(string) {
		a.onDeviceChanged()
	})
	a.deviceSelect.PlaceHolder = "Device ADB"

	a.packageSummary = widget.NewLabel("Nenhum pacote")
	a.packageSummary.Wrapping = fyne.TextTruncate
	a.packageSummary.TextStyle = fyne.TextStyle{Monospace: true}

	a.packagesBtn = widget.NewButtonWithIcon("Pacotes", theme.SearchIcon(), func() {
		a.openPackagePicker()
	})
	a.packagesBtn.Importance = widget.MediumImportance

	a.modeSelect = widget.NewSelect([]string{
		model.ModeClean.String(),
		model.ModeFull.String(),
		model.ModeSearch.String(),
		model.ModeWarnErrorFatal.String(),
	}, nil)
	a.modeSelect.SetSelected(model.ModeClean.String())
	a.modeSelect.PlaceHolder = "Modo"

	a.searchEntry = widget.NewEntry()
	a.searchEntry.SetPlaceHolder("Filtrar logs...")
	a.searchEntry.OnChanged = func(query string) {
		a.debounceSearch(query)
	}
	a.searchEntry.OnSubmitted = func(query string) {
		a.applySearch(query)
	}

	a.logView = NewLogView(a.store, a.window)

	refreshDevicesBtn := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		a.refreshDevices()
	})
	refreshDevicesBtn.Importance = widget.LowImportance

	refreshPackagesBtn := widget.NewButtonWithIcon("", theme.ContentAddIcon(), func() {
		a.refreshPackages()
	})
	refreshPackagesBtn.Importance = widget.LowImportance

	clearBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
		a.logView.Clear()
	})
	clearBtn.Importance = widget.LowImportance

	copyBtn := widget.NewButtonWithIcon("", theme.ContentCopyIcon(), func() {
		copied, async := a.logView.CopySelection(a.window.Clipboard(), func() {
			a.setStatus("Log completo copiado.")
		})
		if async {
			a.setStatus("Copiando log completo...")
			return
		}
		if copied == "" {
			a.setStatus("Nenhum log para copiar.")
			return
		}
		if strings.TrimSpace(a.logView.SelectedText()) != "" {
			a.setStatus("Seleção copiada.")
			return
		}
		a.setStatus("Log completo copiado.")
	})
	copyBtn.Importance = widget.LowImportance

	formatJSONBtn := widget.NewButtonWithIcon("JSON", theme.DocumentCreateIcon(), func() {
		if err := a.logView.FormatSelectedJSON(); err != nil {
			a.setStatus(err.Error())
			return
		}
	})
	formatJSONBtn.Importance = widget.MediumImportance

	a.followCheck = widget.NewCheck("Ao vivo", func(checked bool) {
		a.logView.SetAutoFollow(checked)
	})
	a.followCheck.SetChecked(true)

	a.startBtn = widget.NewButtonWithIcon("Iniciar", theme.MediaPlayIcon(), func() {
		a.startLogcat()
	})
	a.startBtn.Importance = widget.HighImportance

	a.stopBtn = widget.NewButtonWithIcon("Parar", theme.MediaStopIcon(), func() {
		a.stopLogcat()
	})
	a.stopBtn.Importance = widget.DangerImportance
	a.stopBtn.Disable()

	a.statusLabel = widget.NewLabel("Pronto")
	a.statusLabel.Wrapping = fyne.TextWrapWord
	a.statusLabel.Importance = widget.LowImportance

	filtersRow := container.NewGridWithColumns(3, a.deviceSelect, a.modeSelect, a.searchEntry)
	actionsRow := container.NewHBox(
		refreshDevicesBtn,
		refreshPackagesBtn,
		formatJSONBtn,
		copyBtn,
		clearBtn,
		a.followCheck,
		a.startBtn,
		a.stopBtn,
	)
	toolbarRow := container.NewBorder(nil, nil, nil, actionsRow, filtersRow)

	packageRow := container.NewBorder(nil, nil, a.packagesBtn, nil, a.packageSummary)

	header := container.NewVBox(
		toolbarRow,
		packageRow,
		widget.NewSeparator(),
	)

	statusBar := container.NewBorder(
		nil, nil,
		widget.NewIcon(theme.InfoIcon()),
		nil,
		a.statusLabel,
	)

	return container.NewBorder(header, statusBar, nil, nil, a.logView.Container())
}

func (a *App) openPackagePicker() {
	if !a.packagesLoaded || len(a.packages) == 0 {
		device := a.selectedDeviceSerial()
		if device == "" {
			dialog.ShowInformation("Pacotes", "Selecione um device e aguarde o carregamento dos pacotes.", a.window)
			return
		}
		a.setStatus("Carregando pacotes...")
		go func() {
			packages, err := adb.ListPackages(a.ctx, device)
			fyne.Do(func() {
				if err != nil {
					a.setStatus("Erro ao listar pacotes: " + err.Error())
					return
				}
				a.packages = packages
				a.packagesLoaded = true
				a.setStatus(fmt.Sprintf("%d pacotes carregados.", len(packages)))
				showPackagePicker(a.window, a.packages, a.selectedPackages, a.applyPackageSelection)
			})
		}()
		return
	}

	showPackagePicker(a.window, a.packages, a.selectedPackages, a.applyPackageSelection)
}

func (a *App) applyPackageSelection(packages []model.PackageInfo) {
	a.selectedPackages = packages
	a.packageSummary.SetText(formatPackageSelection(packages))
	if len(packages) > 0 {
		a.setStatus(fmt.Sprintf("%d pacote(s) selecionado(s).", len(packages)))
	}
}

func (a *App) refreshDevices() {
	go func() {
		devices, err := adb.ListDevices(a.ctx)
		fyne.Do(func() {
			if err != nil {
				a.setStatus("Erro ao listar devices: " + err.Error())
				return
			}

			a.devices = devices
			options := make([]string, 0, len(devices))
			for _, d := range devices {
				if d.State != "device" {
					continue
				}
				options = append(options, d.Serial)
			}

			a.deviceSelect.Options = options
			a.deviceSelect.Refresh()

			if len(options) == 1 {
				a.deviceSelect.SetSelected(options[0])
				a.onDeviceChanged()
			}

			if len(options) == 0 {
				a.setStatus("Nenhum device conectado.")
				return
			}

			a.setStatus(fmt.Sprintf("%d device(s) encontrado(s).", len(options)))
		})
	}()
}

func (a *App) onDeviceChanged() {
	a.selectedPackages = nil
	a.packageSummary.SetText("Nenhum pacote selecionado")
	a.packagesLoaded = false
	a.packages = nil
	a.refreshPackages()
	a.stopLogcat()
}

func (a *App) refreshPackages() {
	device := a.selectedDeviceSerial()
	if device == "" {
		return
	}

	a.setStatus("Carregando pacotes...")
	go func() {
		packages, err := adb.ListPackages(a.ctx, device)
		fyne.Do(func() {
			if err != nil {
				a.setStatus("Erro ao listar pacotes: " + err.Error())
				return
			}

			a.packages = packages
			a.packagesLoaded = true
			a.setStatus(fmt.Sprintf("%d pacotes carregados. Clique em \"Escolher pacotes\" para filtrar.", len(packages)))
		})
	}()
}

func (a *App) selectedDeviceSerial() string {
	selected := a.deviceSelect.Selected
	if selected == "" {
		return ""
	}
	for _, d := range a.devices {
		if d.Serial == selected {
			return d.Serial
		}
	}
	return strings.Fields(selected)[0]
}

func (a *App) selectedMode() model.LogMode {
	switch a.modeSelect.Selected {
	case model.ModeFull.String():
		return model.ModeFull
	case model.ModeSearch.String():
		return model.ModeSearch
	case model.ModeWarnErrorFatal.String():
		return model.ModeWarnErrorFatal
	default:
		return model.ModeClean
	}
}

func (a *App) startLogcat() {
	device := a.selectedDeviceSerial()
	if device == "" {
		dialog.ShowInformation("Device", "Selecione um device.", a.window)
		return
	}

	if len(a.selectedPackages) == 0 {
		dialog.ShowInformation("Pacote", "Selecione ao menos um pacote em \"Escolher pacotes\".", a.window)
		return
	}

	a.stopLogcat()
	a.logView.Clear()
	a.logView.SetSearch(a.searchEntry.Text)

	filter := model.NewFilterConfig(a.selectedPackages, a.selectedMode(), "")
	session := adb.NewLogcatSession(device, filter)

	a.sessionMu.Lock()
	a.session = session
	a.sessionMu.Unlock()

	a.startBtn.Disable()
	a.stopBtn.Enable()
	a.setStatus(fmt.Sprintf("Logcat iniciado para %d pacote(s)", len(a.selectedPackages)))

	go func() {
		if err := session.Start(a.ctx); err != nil {
			fyne.Do(func() {
				a.setStatus("Erro ao iniciar logcat: " + err.Error())
				a.startBtn.Enable()
				a.stopBtn.Disable()
			})
			return
		}
		a.consumeLogcat(session)
	}()
}

func (a *App) consumeLogcat(session *adb.LogcatSession) {
	defer func() {
		fyne.Do(func() {
			a.setStatus("Logcat encerrado.")
			a.startBtn.Enable()
			a.stopBtn.Disable()
		})
	}()

	pending := make([]model.LogEntry, 0, 256)

	flush := func() {
		if len(pending) == 0 {
			return
		}

		batch := pending
		pending = nil

		fyne.Do(func() {
			a.logView.ApplyBatch(batch, nil)
			a.updateSearchStatus()
		})
	}

	ticker := time.NewTicker(400 * time.Millisecond)
	defer ticker.Stop()

	entries := session.Entries()
	for {
		select {
		case entry, ok := <-entries:
			if !ok {
				flush()
				<-session.Done()
				return
			}
			if entry.IsUpdate {
				pending = append(pending, entry)
				continue
			}
			pending = append(pending, entry)
		case <-ticker.C:
			flush()
		}
	}
}

func (a *App) debounceSearch(query string) {
	if a.searchTimer != nil {
		a.searchTimer.Stop()
	}
	a.searchTimer = time.AfterFunc(250*time.Millisecond, func() {
		fyne.Do(func() {
			a.applySearch(query)
		})
	})
}

func (a *App) applySearch(query string) {
	a.logView.SetSearch(query)
	a.updateSearchStatus()
}

func (a *App) updateSearchStatus() {
	a.sessionMu.Lock()
	running := a.session != nil
	a.sessionMu.Unlock()
	if !running {
		return
	}

	query := strings.TrimSpace(a.logView.SearchQuery())
	total := a.logView.StoreLen()
	visible := a.logView.FilteredLen()

	if query == "" {
		a.setStatus(fmt.Sprintf("Capturando logs · %d linha(s)", total))
		return
	}
	a.setStatus(fmt.Sprintf("Busca %q · %d de %d linha(s)", query, visible, total))
}

func (a *App) stopLogcat() {
	a.sessionMu.Lock()
	session := a.session
	a.session = nil
	a.sessionMu.Unlock()

	if session != nil {
		session.Stop()
	}

	a.startBtn.Enable()
	a.stopBtn.Disable()
}

func (a *App) setStatus(msg string) {
	a.statusLabel.SetText(msg)
}
