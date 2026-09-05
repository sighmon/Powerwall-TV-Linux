package ui

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gotk3/gotk3/gtk"
	"powerwall-tv-gtk/internal/api"
	"powerwall-tv-gtk/internal/schedule"
	statepkg "powerwall-tv-gtk/internal/state"
)

type scheduleRow struct {
	box                      *gtk.Box
	name                     *gtk.Entry
	enabled                  *gtk.CheckButton
	site, startMode, endMode *gtk.ComboBoxText
	startHour, startMinute   *gtk.SpinButton
	endHour, endMinute       *gtk.SpinButton
	deleted                  bool
	id                       string
}

func showSchedulerDialog(parent gtk.IWindow, state *statepkg.State, manager *schedule.Manager, sites []api.Product, currentSiteID int) {
	dialog, _ := gtk.DialogNew()
	dialog.SetTitle("Powerwall Scheduler")
	dialog.SetTransientFor(parent)
	dialog.SetModal(true)
	dialog.SetDefaultSize(760, 680)
	dialog.AddButton("Run Due Schedules", gtk.ResponseType(1001))
	dialog.AddButton("Cancel", gtk.RESPONSE_CANCEL)
	dialog.AddButton("Save", gtk.RESPONSE_ACCEPT)
	content, _ := dialog.GetContentArea()
	root, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 12)
	root.SetBorderWidth(12)
	content.Add(root)

	snapshot := manager.Snapshot()
	active, _ := gtk.CheckButtonNewWithLabel("Scheduler Active")
	active.SetActive(snapshot.Enabled)
	root.PackStart(active, false, false, 0)
	status, _ := gtk.LabelNew(snapshot.LastRunStatus)
	status.SetHAlign(gtk.ALIGN_START)
	root.PackStart(status, false, false, 0)

	scroll, _ := gtk.ScrolledWindowNew(nil, nil)
	scroll.SetPolicy(gtk.POLICY_AUTOMATIC, gtk.POLICY_AUTOMATIC)
	rowsBox, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 12)
	scroll.Add(rowsBox)
	root.PackStart(scroll, true, true, 0)
	rows := make([]*scheduleRow, 0, len(snapshot.Schedules)+1)
	addRow := func(item schedule.Schedule) {
		row := buildScheduleRow(item, sites)
		rows = append(rows, row)
		rowsBox.PackStart(row.box, false, false, 0)
		row.box.ShowAll()
	}
	for _, item := range snapshot.Schedules {
		addRow(item)
	}
	add, _ := gtk.ButtonNewWithLabel("Add Schedule")
	add.Connect("clicked", func() { addRow(schedule.New()) })
	root.PackStart(add, false, false, 0)

	collect := func() schedule.Store {
		store := schedule.Store{Enabled: active.GetActive(), LastRunStatus: manager.Snapshot().LastRunStatus}
		previous := make(map[string]schedule.Schedule, len(snapshot.Schedules))
		for _, item := range snapshot.Schedules {
			previous[item.ID] = item
		}
		for _, row := range rows {
			if row.deleted {
				continue
			}
			name, _ := row.name.GetText()
			siteID, _ := strconv.Atoi(row.site.GetActiveID())
			siteName := ""
			for _, site := range sites {
				if site.EnergySiteID == siteID {
					siteName = site.DisplayLabel()
				}
			}
			prior := previous[row.id]
			store.Schedules = append(store.Schedules, schedule.Schedule{
				ID: row.id, Name: name, Enabled: row.enabled.GetActive(), EnergySiteID: siteID, EnergySiteName: siteName,
				StartMinutes: row.startHour.GetValueAsInt()*60 + row.startMinute.GetValueAsInt(),
				EndMinutes:   row.endHour.GetValueAsInt()*60 + row.endMinute.GetValueAsInt(),
				StartMode:    schedule.Mode(row.startMode.GetActiveID()), EndMode: schedule.Mode(row.endMode.GetActiveID()),
				LastAppliedStart: prior.LastAppliedStart, LastAppliedEnd: prior.LastAppliedEnd,
			})
		}
		return store
	}

	dialog.ShowAll()
	for {
		response := dialog.Run()
		if response == gtk.ResponseType(1001) {
			_ = manager.Replace(collect(), time.Now())
			status.SetText(manager.ApplyDue(state, currentSiteID, time.Now()))
			continue
		}
		if response == gtk.RESPONSE_ACCEPT {
			_ = manager.Replace(collect(), time.Now())
		}
		break
	}
	dialog.Destroy()
}

func buildScheduleRow(item schedule.Schedule, sites []api.Product) *scheduleRow {
	row := &scheduleRow{id: item.ID}
	row.box, _ = gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 6)
	row.box.SetMarginBottom(8)
	top, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 8)
	row.name, _ = gtk.EntryNew()
	row.name.SetText(item.Name)
	row.enabled, _ = gtk.CheckButtonNewWithLabel("Enabled")
	row.enabled.SetActive(item.Enabled)
	remove, _ := gtk.ButtonNewWithLabel("Delete")
	remove.Connect("clicked", func() { row.deleted = true; row.box.Hide() })
	top.PackStart(row.name, true, true, 0)
	top.PackStart(row.enabled, false, false, 0)
	top.PackStart(remove, false, false, 0)
	row.box.PackStart(top, false, false, 0)

	row.site, _ = gtk.ComboBoxTextNew()
	row.site.Append("0", "Select Home")
	found := false
	for _, site := range sites {
		id := strconv.Itoa(site.EnergySiteID)
		name := site.DisplayLabel()
		if name == "" {
			name = "Energy Site " + id
		}
		row.site.Append(id, fmt.Sprintf("%s (%s)", name, id))
		found = found || site.EnergySiteID == item.EnergySiteID
	}
	if item.EnergySiteID != 0 && !found {
		row.site.Append(strconv.Itoa(item.EnergySiteID), fmt.Sprintf("%s (%d)", item.EnergySiteName, item.EnergySiteID))
	}
	row.site.SetActiveID(strconv.Itoa(item.EnergySiteID))
	row.box.PackStart(labeledWidget("Home", row.site), false, false, 0)

	row.startMode = modeCombo(item.StartMode)
	row.endMode = modeCombo(item.EndMode)
	row.startHour, row.startMinute = timeControls(item.StartMinutes)
	row.endHour, row.endMinute = timeControls(item.EndMinutes)
	row.box.PackStart(labeledWidget("Start Mode", row.startMode), false, false, 0)
	row.box.PackStart(labeledTime("Start", row.startHour, row.startMinute), false, false, 0)
	row.box.PackStart(labeledWidget("End Mode", row.endMode), false, false, 0)
	row.box.PackStart(labeledTime("End", row.endHour, row.endMinute), false, false, 0)
	separator, _ := gtk.SeparatorNew(gtk.ORIENTATION_HORIZONTAL)
	row.box.PackStart(separator, false, false, 4)
	return row
}

func modeCombo(value schedule.Mode) *gtk.ComboBoxText {
	combo, _ := gtk.ComboBoxTextNew()
	for _, mode := range schedule.Modes {
		combo.Append(string(mode), mode.Title())
	}
	combo.SetActiveID(string(value))
	return combo
}

func timeControls(minutes int) (*gtk.SpinButton, *gtk.SpinButton) {
	hour, _ := gtk.SpinButtonNewWithRange(0, 23, 1)
	minute, _ := gtk.SpinButtonNewWithRange(0, 59, 1)
	hour.SetValue(float64(minutes / 60))
	minute.SetValue(float64(minutes % 60))
	return hour, minute
}

func labeledWidget(text string, widget gtk.IWidget) *gtk.Box {
	box, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 8)
	label, _ := gtk.LabelNew(text)
	label.SetSizeRequest(90, -1)
	label.SetHAlign(gtk.ALIGN_START)
	box.PackStart(label, false, false, 0)
	box.PackStart(widget, true, true, 0)
	return box
}

func labeledTime(text string, hour, minute *gtk.SpinButton) *gtk.Box {
	box := labeledWidget(text, hour)
	colon, _ := gtk.LabelNew(":")
	box.PackStart(colon, false, false, 0)
	box.PackStart(minute, false, false, 0)
	return box
}
