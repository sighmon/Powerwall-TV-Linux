package ui

import (
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
	fleetStatus     *gtk.Label
	loginMode       *gtk.ComboBoxText
}

func showSettingsDialog(parent gtk.IWindow, state *statepkg.State) error {
	dlg, _ := gtk.DialogNew()
	dlg.SetTitle("Settings")
	dlg.SetTransientFor(parent)
	dlg.SetModal(true)
	dlg.AddButton("Cancel", gtk.RESPONSE_CANCEL)
	dlg.AddButton("Save", gtk.RESPONSE_ACCEPT)

	content, _ := dlg.GetContentArea()

	notebook, _ := gtk.NotebookNew()
	content.Add(notebook)

	widgets := &settingsWidgets{}
	localTab, _ := buildLocalTab(widgets, state)
	fleetTab, _ := buildFleetTab(widgets, state, parent)
	prefsTab, _ := buildPrefsTab(widgets, state)

	notebook.AppendPage(localTab, label("Local"))
	notebook.AppendPage(fleetTab, label("Fleet API"))
	notebook.AppendPage(prefsTab, label("Display"))

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
	state.Prefs = p
	_ = storage.SavePrefs(p)

	if w.password != nil {
		if pw, _ := w.password.GetText(); pw != "" {
			_ = storage.SetGatewayPassword(pw)
		}
	}
}

func buildLocalTab(w *settingsWidgets, state *statepkg.State) (*gtk.Box, error) {
	box, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 8)
	box.SetBorderWidth(12)

	modeRow, combo := loginModeSelect(state.Prefs.LoginMode)
	w.loginMode = combo
	box.PackStart(modeRow, false, false, 0)

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

	return box, nil
}

func buildPrefsTab(w *settingsWidgets, state *statepkg.State) (*gtk.Box, error) {
	box, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 8)
	box.SetBorderWidth(12)

	w.lessPrecision, _ = gtk.CheckButtonNewWithLabel("Limit data to one decimal place")
	w.lessPrecision.SetActive(state.Prefs.ShowLessPrecision)
	box.PackStart(w.lessPrecision, false, false, 0)

	return box, nil
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
