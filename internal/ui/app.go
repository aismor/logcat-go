package ui

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
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
	modeSelect     *widget.Select
	packageEntry *packageSearchField
	searchEntry    *widget.Entry
	statusLeft     *widget.Label
	logView        *LogView
	store          *adb.LogStore

	startBtn    *widget.Button
	stopBtn     *widget.Button
	liveToggle  *liveToggleButton
	connDot       *canvas.Circle
	connLabel     *widget.Label
	fullScreen    bool

	devices          []model.Device
	packages         []model.PackageInfo
	selectedPackages []model.PackageInfo
	packagesLoaded   bool

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
	InitThemeFromPreferences()

	a.window = fyne.CurrentApp().NewWindow("Logcat Go")
	a.window.SetIcon(AppIcon())
	a.window.Resize(fyne.NewSize(1360, 860))
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

	a.modeSelect = widget.NewSelect([]string{
		model.ModeClean.String(),
		model.ModeFull.String(),
		model.ModeSearch.String(),
		model.ModeWarnErrorFatal.String(),
	}, nil)
	a.modeSelect.SetSelected(model.ModeClean.String())

	a.searchEntry = widget.NewEntry()
	a.searchEntry.SetPlaceHolder("Filtrar logs...")
	a.searchEntry.OnChanged = func(query string) {
		a.debounceSearch(query)
	}
	a.searchEntry.OnSubmitted = func(query string) {
		a.applySearch(query)
	}

	a.logView = NewLogView(a.store, a.window)

	a.statusLeft = widget.NewLabel("Pronto")
	a.statusLeft.Importance = widget.LowImportance

	themeBtn := newThemeMenuButton(a.window, func(mode string) {
		_ = mode
	})

	a.liveToggle = newLiveToggleButton(true, func(active bool) {
		a.logView.SetAutoFollow(active)
	})

	refreshDevicesBtn := a.iconToolButton(theme.ViewRefreshIcon(), "Atualizar devices", a.refreshDevices)
	formatJSONBtn := widget.NewButtonWithIcon("{ } JSON", theme.DocumentCreateIcon(), func() {
		if err := a.logView.FormatSelectedJSON(); err != nil {
			a.setStatusLeft(err.Error())
			return
		}
	})
	formatJSONBtn.Importance = widget.MediumImportance
	copyBtn := a.iconToolButton(theme.ContentCopyIcon(), "Copiar", func() {
		copied, async := a.logView.CopySelection(a.window.Clipboard(), func() {
			a.setStatusLeft("Log completo copiado.")
		})
		if async {
			a.setStatusLeft("Copiando log completo...")
			return
		}
		if copied == "" {
			a.setStatusLeft("Nenhum log para copiar.")
			return
		}
		if strings.TrimSpace(a.logView.SelectedText()) != "" {
			a.setStatusLeft("Seleção copiada.")
			return
		}
		a.setStatusLeft("Log completo copiado.")
	})
	clearToolsBtn := a.iconToolButton(theme.DeleteIcon(), "Limpar", func() {
		a.logView.Clear()
	})

	a.startBtn = widget.NewButtonWithIcon("Iniciar", theme.MediaPlayIcon(), func() {
		a.startLogcat()
	})
	a.startBtn.Importance = widget.HighImportance

	a.stopBtn = widget.NewButtonWithIcon("Parar", theme.MediaStopIcon(), func() {
		a.stopLogcat()
	})
	a.stopBtn.Importance = widget.MediumImportance
	a.stopBtn.Disable()

	filtersRow := container.NewGridWithColumns(4,
		a.deviceSelect,
		a.buildPackageSelector(),
		a.modeSelect,
		a.searchEntry,
	)
	toolsRow := container.NewHBox(refreshDevicesBtn, formatJSONBtn, copyBtn, clearToolsBtn)
	transportRow := container.NewHBox(a.startBtn, a.stopBtn)
	actionsRow := container.NewBorder(nil, nil, toolsRow, transportRow, nil)
	controlsBody := container.NewVBox(filtersRow, actionsRow)

	controlsHeading := cardHeading("Controles", "Device, filtros e captura", widget.NewIcon(theme.SettingsIcon()))
	controlsCard := widget.NewCard("", "", container.NewVBox(controlsHeading, controlsBody))

	clearLogsBtn := widget.NewButtonWithIcon("Limpar logs", theme.DeleteIcon(), func() {
		a.logView.Clear()
	})
	clearLogsBtn.Importance = widget.LowImportance

	fullscreenBtn := widget.NewButtonWithIcon("", theme.ViewFullScreenIcon(), func() {
		a.fullScreen = !a.fullScreen
		a.window.SetFullScreen(a.fullScreen)
	})
	fullscreenBtn.Importance = widget.LowImportance

	logHeading := cardHeading("Logs", "Selecione texto para copiar ou formatar JSON",
		container.NewHBox(clearLogsBtn, fullscreenBtn))
	logPanel := container.NewBorder(logHeading, nil, nil, nil, a.logView.Container())
	logCard := widget.NewCard("", "", logPanel)

	mainContent := container.NewBorder(controlsCard, nil, nil, nil, logCard)

	a.connDot, a.connLabel = newConnectionStatus(false, "")
	footer := statusFooter(
		statusSegment(theme.InfoIcon(), a.statusLeft),
		container.NewHBox(a.connDot, a.connLabel, widget.NewIcon(theme.ContentRedoIcon())),
	)

	topBar := appTopBar(a.liveToggle, themeBtn)
	mainArea := container.NewPadded(mainContent)
	body := container.NewBorder(topBar, footer, nil, nil, mainArea)
	return body
}

func (a *App) buildPackageSelector() fyne.CanvasObject {
	a.packageEntry = newPackageSearchField(a.openPackagePickerWithQuery)
	a.updatePackageField()
	return a.packageEntry
}

func (a *App) updatePackageField() {
	if a.packageEntry == nil {
		return
	}
	if len(a.selectedPackages) == 0 {
		a.packageEntry.SetDisplay("")
		return
	}
	a.packageEntry.SetDisplay(formatPackageSelection(a.selectedPackages))
}

func (a *App) iconToolButton(icon fyne.Resource, _ string, action func()) *widget.Button {
	btn := widget.NewButtonWithIcon("", icon, action)
	btn.Importance = widget.LowImportance
	return btn
}

func (a *App) updateConnectionBadge() {
	device := a.selectedDeviceSerial()
	if a.connDot == nil || a.connLabel == nil {
		return
	}
	th := fyne.CurrentApp().Settings().Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()
	if device != "" {
		a.connLabel.SetText("Conectado a " + device)
		a.connDot.FillColor = androidGreen
	} else {
		a.connLabel.SetText("Desconectado")
		a.connDot.FillColor = th.Color(theme.ColorNameDisabled, v)
	}
	a.connDot.Refresh()
	a.connLabel.Refresh()
}

func (a *App) setStatusLeft(msg string) {
	a.statusLeft.SetText(msg)
}

func (a *App) setStatus(msg string) {
	a.setStatusLeft(msg)
}

func (a *App) openPackagePickerWithQuery(query string) {
	if !a.packagesLoaded || len(a.packages) == 0 {
		device := a.selectedDeviceSerial()
		if device == "" {
			dialog.ShowInformation("Pacotes", "Selecione um device e aguarde o carregamento dos pacotes.", a.window)
			return
		}
		a.setStatusLeft("Carregando pacotes...")
		go func() {
			packages, err := adb.ListPackages(a.ctx, device)
			fyne.Do(func() {
				if err != nil {
					a.setStatusLeft("Erro ao listar pacotes: " + err.Error())
					return
				}
				a.packages = packages
				a.packagesLoaded = true
				a.setStatusLeft(fmt.Sprintf("%d pacotes carregados.", len(packages)))
				showPackagePicker(a.window, a.packages, a.selectedPackages, query, a.applyPackageSelection)
			})
		}()
		return
	}

	showPackagePicker(a.window, a.packages, a.selectedPackages, query, a.applyPackageSelection)
}

func (a *App) applyPackageSelection(packages []model.PackageInfo) {
	a.selectedPackages = packages
	a.updatePackageField()
	if len(packages) > 0 {
		a.setStatusLeft(fmt.Sprintf("%d pacote(s) selecionado(s).", len(packages)))
	}
}

func (a *App) refreshDevices() {
	go func() {
		devices, err := adb.ListDevices(a.ctx)
		fyne.Do(func() {
			if err != nil {
				a.setStatusLeft("Erro ao listar devices: " + err.Error())
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
				a.setStatusLeft("Nenhum device conectado.")
				a.updateConnectionBadge()
				return
			}

			a.setStatusLeft(fmt.Sprintf("%d device(s) encontrado(s).", len(options)))
			a.updateConnectionBadge()
		})
	}()
}

func (a *App) onDeviceChanged() {
	a.selectedPackages = nil
	a.packagesLoaded = false
	a.packages = nil
	a.updatePackageField()
	a.refreshPackages()
	a.stopLogcat()
	a.updateConnectionBadge()
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
				a.setStatusLeft("Erro ao listar pacotes: " + err.Error())
				return
			}

			a.packages = packages
			a.packagesLoaded = true
			a.setStatusLeft(fmt.Sprintf("%d pacotes carregados.", len(packages)))
			a.updateConnectionBadge()
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
		dialog.ShowInformation("Pacote", "Selecione ao menos um pacote em \"Buscar e selecionar pacotes\".", a.window)
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
	a.setStatusLeft(fmt.Sprintf("Logcat iniciado para %d pacote(s)", len(a.selectedPackages)))

	go func() {
		if err := session.Start(a.ctx); err != nil {
			fyne.Do(func() {
				a.setStatusLeft("Erro ao iniciar logcat: " + err.Error())
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
			a.setStatusLeft("Logcat encerrado.")
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
		a.setStatusLeft(fmt.Sprintf("Capturando logs · %d linha(s)", total))
		return
	}
	a.setStatusLeft(fmt.Sprintf("Busca %q · %d de %d linha(s)", query, visible, total))
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
