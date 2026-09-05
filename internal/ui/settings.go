package ui

import (
	"fmt"
	"runtime/debug"

	"github.com/gotk3/gotk3/gtk"
	"powerwall-tv-gtk/internal/auth"
	statepkg "powerwall-tv-gtk/internal/state"
	"powerwall-tv-gtk/internal/storage"
)

type settingsWidgets struct {
	gatewayIP       *gtk.Entry
	username        *gtk.Entry
	password        *gtk.Entry
	wallConnectorIP *gtk.Entry
	lessPrecision   *gtk.CheckButton
	alwaysRuntime   *gtk.CheckButton
	preventSaver    *gtk.CheckButton
	keepInFront     *gtk.CheckButton
	showInMenuBar   *gtk.CheckButton
	showScheduler   *gtk.CheckButton
	autoHideSummary *gtk.CheckButton
	autoHideButtons *gtk.CheckButton
	fleetStatus     *gtk.Label
	fleetToken      *gtk.Entry
	loginMode       *gtk.ComboBoxText
	sceneScale      *gtk.SpinButton
	sceneHorizontal *gtk.SpinButton
	sceneVertical   *gtk.SpinButton
	electricityKey  *gtk.Entry
	electricityZone *gtk.Entry
}

func showSettingsDialog(parent gtk.IWindow, state *statepkg.State) error {
	dlg, _ := gtk.DialogNew()
	dlg.SetTitle("Settings")
	dlg.SetTransientFor(parent)
	dlg.SetModal(true)
	dlg.AddButton("Cancel", gtk.RESPONSE_CANCEL)
	dlg.AddButton("Save", gtk.RESPONSE_ACCEPT)

	content, _ := dlg.GetContentArea()
	modeRow, combo := loginModeSelect(state.Prefs.LoginMode)
	widgets := &settingsWidgets{loginMode: combo}
	content.PackStart(modeRow, false, false, 8)

	notebook, _ := gtk.NotebookNew()
	content.Add(notebook)

	localTab, _ := buildLocalTab(widgets, state)
	fleetTab, _ := buildFleetTab(widgets, state, parent)
	prefsTab, _ := buildPrefsTab(widgets, state)
	infoTab, _ := buildInfoTab(state)

	notebook.AppendPage(localTab, label("Local"))
	notebook.AppendPage(fleetTab, label("Fleet API"))
	notebook.AppendPage(prefsTab, label("Display"))
	notebook.AppendPage(infoTab, label("Information"))

	dlg.ShowAll()
	resp := dlg.Run()
	if resp == gtk.RESPONSE_ACCEPT {
		persistSettings(widgets, state)
	}
	dlg.Destroy()
	return nil
}

func persistSettings(w *settingsWidgets, state *statepkg.State) {
	p := state.Prefs
	if w.loginMode != nil {
		if w.loginMode.GetActiveID() == "fleetAPI" {
			p.LoginMode = storage.LoginModeFleetAPI
		} else {
			p.LoginMode = storage.LoginModeLocal
		}
	}
	if w.gatewayIP != nil {
		p.GatewayIP, _ = w.gatewayIP.GetText()
	}
	if w.wallConnectorIP != nil {
		p.WallConnectorIP, _ = w.wallConnectorIP.GetText()
	}
	if w.username != nil {
		p.Username, _ = w.username.GetText()
	}
	if w.lessPrecision != nil {
		p.ShowLessPrecision = w.lessPrecision.GetActive()
	}
	if w.alwaysRuntime != nil {
		p.AlwaysShowRuntimeEstimate = w.alwaysRuntime.GetActive()
	}
	if w.preventSaver != nil {
		p.PreventScreenSaver = w.preventSaver.GetActive()
	}
	if w.keepInFront != nil {
		p.KeepWindowInFront = w.keepInFront.GetActive()
	}
	if w.showInMenuBar != nil {
		p.ShowInMenuBar = w.showInMenuBar.GetActive()
	}
	if w.showScheduler != nil {
		p.ShowSchedulerButton = w.showScheduler.GetActive()
	}
	if w.autoHideSummary != nil {
		p.AutoHideSummaryOnOverlap = w.autoHideSummary.GetActive()
	}
	if w.autoHideButtons != nil {
		p.AutoHideButtonsOnOverlap = w.autoHideButtons.GetActive()
	}
	if w.sceneScale != nil {
		p.SceneScale = w.sceneScale.GetValue()
	}
	if w.sceneHorizontal != nil {
		p.SceneHorizontalOffset = w.sceneHorizontal.GetValue()
	}
	if w.sceneVertical != nil {
		p.SceneVerticalOffset = w.sceneVertical.GetValue()
	}
	if w.electricityZone != nil {
		p.ElectricityMapsZone, _ = w.electricityZone.GetText()
	}
	state.Prefs = p
	_ = storage.SavePrefs(p)

	if w.password != nil {
		if pw, _ := w.password.GetText(); pw != "" {
			_ = storage.SetGatewayPassword(pw)
		}
	}
	if w.electricityKey != nil {
		key, _ := w.electricityKey.GetText()
		_ = storage.SetElectricityMapsAPIKey(key)
	}
	if w.fleetToken != nil {
		token, _ := w.fleetToken.GetText()
		_ = storage.SetFleetAccessToken(token)
	}
}

func buildLocalTab(w *settingsWidgets, state *statepkg.State) (*gtk.Box, error) {
	box, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 8)
	box.SetBorderWidth(12)

	row, entry := labeledEntryWithValue("Gateway IP", state.Prefs.GatewayIP)
	w.gatewayIP = entry
	box.PackStart(row, false, false, 0)

	row, entry = labeledEntryWithValue("Username", state.Prefs.Username)
	w.username = entry
	box.PackStart(row, false, false, 0)

	row, entry = labeledPassword("Password")
	w.password = entry
	box.PackStart(row, false, false, 0)

	row, entry = labeledEntryWithValue("Wall Connector IP", state.Prefs.WallConnectorIP)
	w.wallConnectorIP = entry
	box.PackStart(row, false, false, 0)

	return box, nil
}

func buildFleetTab(w *settingsWidgets, state *statepkg.State, parent gtk.IWindow) (*gtk.Box, error) {
	box, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 8)
	box.SetBorderWidth(12)

	statusText := "Not logged in"
	if tok, err := auth.LoadStoredToken(); err == nil && tok.AccessToken != "" {
		statusText = "Logged in"
		if !tok.Expiry.IsZero() {
			statusText = "Logged in (expires " + tok.Expiry.Format("15:04") + ")"
		}
	}
	w.fleetStatus, _ = gtk.LabelNew(statusText)
	w.fleetStatus.SetHAlign(gtk.ALIGN_START)
	box.PackStart(w.fleetStatus, false, false, 0)
	row, tokenEntry := labeledPassword("Access token")
	w.fleetToken = tokenEntry
	if token, err := storage.GetFleetAccessToken(); err == nil {
		w.fleetToken.SetText(token)
	}
	box.PackStart(row, false, false, 0)

	btn, _ := gtk.ButtonNewWithLabel("Login with Tesla")
	btn.Connect("clicked", func() {
		go startFleetLogin(parent, state, w.fleetStatus)
	})
	box.PackStart(btn, false, false, 0)

	logout, _ := gtk.ButtonNewWithLabel("Log out")
	logout.Connect("clicked", func() {
		storage.ClearFleetTokens()
		w.fleetStatus.SetText("Not logged in")
	})
	box.PackStart(logout, false, false, 0)

	deleteAll, _ := gtk.ButtonNewWithLabel("Delete all Fleet settings")
	deleteAll.Connect("clicked", func() {
		confirm := gtk.MessageDialogNew(parent, gtk.DIALOG_MODAL, gtk.MESSAGE_WARNING, gtk.BUTTONS_OK_CANCEL, "Delete all Fleet and display settings?")
		if confirm.Run() == gtk.RESPONSE_OK {
			storage.ClearFleetTokens()
			_ = storage.ClearVehicleChargeCache()
			state.VehicleCacheGeneration++
			_ = storage.SetElectricityMapsAPIKey("")
			defaults := storage.DefaultPrefs()
			state.Prefs.CurrentEnergySiteIdx = 0
			state.Prefs.FleetBaseURL = defaults.FleetBaseURL
			state.Prefs.ElectricityMapsZone = ""
			state.Prefs.KeepWindowInFront = false
			state.Prefs.AlwaysShowRuntimeEstimate = false
			state.Prefs.ShowSchedulerButton = false
			state.Prefs.AutoHideSummaryOnOverlap = true
			state.Prefs.AutoHideButtonsOnOverlap = true
			state.Prefs.SceneScale = 1
			state.Prefs.SceneHorizontalOffset = 0
			state.Prefs.SceneVerticalOffset = 0
			state.Prefs.LastChargingVIN = ""
			_ = storage.SavePrefs(state.Prefs)
			w.fleetToken.SetText("")
			w.fleetStatus.SetText("Not logged in")
			if w.electricityKey != nil {
				w.electricityKey.SetText("")
			}
			if w.electricityZone != nil {
				w.electricityZone.SetText("")
			}
			if w.keepInFront != nil {
				w.keepInFront.SetActive(false)
			}
			if w.alwaysRuntime != nil {
				w.alwaysRuntime.SetActive(false)
			}
			if w.showScheduler != nil {
				w.showScheduler.SetActive(false)
			}
			if w.autoHideSummary != nil {
				w.autoHideSummary.SetActive(true)
			}
			if w.autoHideButtons != nil {
				w.autoHideButtons.SetActive(true)
			}
			if w.sceneScale != nil {
				w.sceneScale.SetValue(1)
			}
			if w.sceneHorizontal != nil {
				w.sceneHorizontal.SetValue(0)
			}
			if w.sceneVertical != nil {
				w.sceneVertical.SetValue(0)
			}
		}
		confirm.Destroy()
	})
	box.PackEnd(deleteAll, false, false, 8)

	return box, nil
}

func buildInfoTab(state *statepkg.State) (*gtk.Box, error) {
	box, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 8)
	box.SetBorderWidth(12)
	for _, line := range []string{
		"Version: " + appVersion(),
		"Firmware: " + valueOrDash(state.FirmwareVersion),
		"Installed: " + valueOrDash(state.InstallationDate),
		"Base: " + state.Prefs.FleetBaseURL,
		"Last charging VIN: " + valueOrDash(state.Prefs.LastChargingVIN),
	} {
		label, _ := gtk.LabelNew(line)
		label.SetHAlign(gtk.ALIGN_START)
		label.SetSelectable(true)
		box.PackStart(label, false, false, 0)
	}
	disclaimer, _ := gtk.LabelNew("This is an unofficial app – not affiliated with Tesla, Inc. Tesla, Powerwall, and related marks are trademarks of Tesla, Inc.")
	disclaimer.SetLineWrap(true)
	disclaimer.SetHAlign(gtk.ALIGN_START)
	box.PackStart(disclaimer, false, false, 8)
	return box, nil
}

func appVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		version := info.Main.Version
		if version != "" && version != "(devel)" {
			return version
		}
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" && len(setting.Value) >= 7 {
				return fmt.Sprintf("development (%s)", setting.Value[:7])
			}
		}
	}
	return "development"
}

func valueOrDash(value string) string {
	if value == "" {
		return "-"
	}
	return value
}

func buildPrefsTab(w *settingsWidgets, state *statepkg.State) (*gtk.Box, error) {
	box, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 8)
	box.SetBorderWidth(12)

	w.lessPrecision, _ = gtk.CheckButtonNewWithLabel("Limit data to one decimal place")
	w.lessPrecision.SetActive(state.Prefs.ShowLessPrecision)
	box.PackStart(w.lessPrecision, false, false, 0)
	w.alwaysRuntime, _ = gtk.CheckButtonNewWithLabel("Always show Powerwall estimate")
	w.alwaysRuntime.SetActive(state.Prefs.AlwaysShowRuntimeEstimate)
	box.PackStart(w.alwaysRuntime, false, false, 0)
	w.preventSaver, _ = gtk.CheckButtonNewWithLabel("Prevent screen saver from showing")
	w.preventSaver.SetActive(state.Prefs.PreventScreenSaver)
	w.preventSaver.SetTooltipText("Keeping the screen on may increase power usage and risk burn-in.")
	box.PackStart(w.preventSaver, false, false, 0)
	w.keepInFront, _ = gtk.CheckButtonNewWithLabel("Keep window in front")
	w.keepInFront.SetActive(state.Prefs.KeepWindowInFront)
	box.PackStart(w.keepInFront, false, false, 0)
	w.showInMenuBar, _ = gtk.CheckButtonNewWithLabel("Show in menu bar")
	w.showInMenuBar.SetActive(state.Prefs.ShowInMenuBar)
	box.PackStart(w.showInMenuBar, false, false, 0)
	w.showScheduler, _ = gtk.CheckButtonNewWithLabel("Show schedule (beta)")
	w.showScheduler.SetActive(state.Prefs.ShowSchedulerButton)
	box.PackStart(w.showScheduler, false, false, 0)
	w.autoHideSummary, _ = gtk.CheckButtonNewWithLabel("Auto-hide home summary")
	w.autoHideSummary.SetActive(state.Prefs.AutoHideSummaryOnOverlap)
	box.PackStart(w.autoHideSummary, false, false, 0)
	w.autoHideButtons, _ = gtk.CheckButtonNewWithLabel("Auto-hide buttons")
	w.autoHideButtons.SetActive(state.Prefs.AutoHideButtonsOnOverlap)
	box.PackStart(w.autoHideButtons, false, false, 0)

	row, entry := labeledPassword("Electricity Maps API key")
	w.electricityKey = entry
	if key, err := storage.GetElectricityMapsAPIKey(); err == nil {
		w.electricityKey.SetText(key)
	}
	box.PackStart(row, false, false, 0)
	row, entry = labeledEntryWithValue("Electricity Maps zone (e.g. AU-SA)", state.Prefs.ElectricityMapsZone)
	w.electricityZone = entry
	box.PackStart(row, false, false, 0)

	w.sceneScale = labeledSpin(box, "Scene scale", state.Prefs.SceneScale, minSceneScale, maxSceneScale, 0.05)
	w.sceneHorizontal = labeledSpin(box, "Horizontal offset", state.Prefs.SceneHorizontalOffset, minSceneOffset, maxSceneOffset, 0.02)
	w.sceneVertical = labeledSpin(box, "Vertical offset", state.Prefs.SceneVerticalOffset, minSceneOffset, maxSceneOffset, 0.02)

	reset, _ := gtk.ButtonNewWithLabel("Reset scene layout")
	reset.Connect("clicked", func() {
		w.sceneScale.SetValue(1)
		w.sceneHorizontal.SetValue(0)
		w.sceneVertical.SetValue(0)
	})
	box.PackStart(reset, false, false, 4)

	return box, nil
}

func labeledSpin(box *gtk.Box, text string, value, minimum, maximum, step float64) *gtk.SpinButton {
	row, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 8)
	lbl, _ := gtk.LabelNew(text)
	lbl.SetHAlign(gtk.ALIGN_START)
	spin, _ := gtk.SpinButtonNewWithRange(minimum, maximum, step)
	spin.SetValue(value)
	spin.SetDigits(2)
	row.PackStart(lbl, true, true, 0)
	row.PackEnd(spin, false, false, 0)
	box.PackStart(row, false, false, 0)
	return spin
}

func labeledEntryWithValue(labelText, value string) (*gtk.Box, *gtk.Entry) {
	row, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 4)
	lbl, _ := gtk.LabelNew(labelText)
	lbl.SetHAlign(gtk.ALIGN_START)
	entry, _ := gtk.EntryNew()
	entry.SetText(value)
	row.PackStart(lbl, false, false, 0)
	row.PackStart(entry, false, false, 0)
	return row, entry
}

func labeledPassword(labelText string) (*gtk.Box, *gtk.Entry) {
	row, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 4)
	lbl, _ := gtk.LabelNew(labelText)
	lbl.SetHAlign(gtk.ALIGN_START)
	entry, _ := gtk.EntryNew()
	entry.SetVisibility(false)
	row.PackStart(lbl, false, false, 0)
	row.PackStart(entry, false, false, 0)
	return row, entry
}

func loginModeSelect(current storage.LoginMode) (*gtk.Box, *gtk.ComboBoxText) {
	row, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 4)
	lbl, _ := gtk.LabelNew("Login Mode")
	lbl.SetHAlign(gtk.ALIGN_START)
	combo, _ := gtk.ComboBoxTextNew()
	combo.Append("local", "Local")
	combo.Append("fleetAPI", "Fleet API")
	if current == storage.LoginModeFleetAPI {
		combo.SetActiveID("fleetAPI")
	} else {
		combo.SetActiveID("local")
	}
	row.PackStart(lbl, false, false, 0)
	row.PackStart(combo, false, false, 0)
	return row, combo
}

func label(text string) *gtk.Label {
	lbl, _ := gtk.LabelNew(text)
	return lbl
}
