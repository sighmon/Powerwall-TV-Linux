package ui

import (
	"strings"
	"time"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"powerwall-tv-gtk/internal/auth"
	"powerwall-tv-gtk/internal/schedule"
	statepkg "powerwall-tv-gtk/internal/state"
	"powerwall-tv-gtk/internal/storage"
)

func buildMainView(state *statepkg.State, parent gtk.IWindow, app *gtk.Application) (*gtk.Overlay, error) {
	root, _ := gtk.OverlayNew()
	root.SetHExpand(true)
	root.SetVExpand(true)

	charts := NewChartsData()
	scheduler := schedule.Load()
	stack, _ := gtk.StackNew()
	stack.SetTransitionType(gtk.STACK_TRANSITION_TYPE_NONE)
	stack.SetTransitionDuration(0)
	stack.SetHHomogeneous(false)
	stack.SetVHomogeneous(false)
	stack.SetHExpand(true)
	stack.SetVExpand(true)

	graphsView, graphsWidgets, _ := buildGraphsView(charts, state, func() {
		stack.SetVisibleChildName("home")
	})

	dashboard, dashboardWidgets, _ := buildDashboard(state, charts, graphsWidgets, scheduler)
	statusIcon := buildStatusIcon(state, dashboardWidgets, parent, app)
	stack.AddNamed(dashboard, "home")
	stack.AddNamed(graphsView, "graphs")
	stack.SetVisibleChildName("home")

	root.Add(stack)

	// Footer buttons overlay
	footerFixed, _ := gtk.FixedNew()
	footer, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 12)
	footer.SetOpacity(0)
	footerIntroStarted := time.Now()
	glib.TimeoutAdd(33, func() bool {
		footer.SetOpacity(introEaseOut(footerIntroStarted, 1200*time.Millisecond, 700*time.Millisecond, time.Now()))
		return time.Since(footerIntroStarted) < 2*time.Second
	})
	backBtn, _ := gtk.ButtonNewWithLabel("Back")
	backBtn.SetNoShowAll(true)
	backBtn.Connect("clicked", func() {
		stack.SetVisibleChildName("home")
		backBtn.Hide()
	})
	footer.PackStart(backBtn, false, false, 0)

	graphsBtn, _ := gtk.ButtonNewWithLabel("Graphs")
	graphsBtn.Connect("clicked", func() {
		stack.SetVisibleChildName("graphs")
		backBtn.Show()
		backBtn.GrabFocus()
		if graphsView != nil {
			graphsView.QueueDraw()
		}
		graphsWidgets.loadHistory()
	})
	footer.PackStart(graphsBtn, false, false, 0)

	schedulerBtn, _ := gtk.ButtonNewWithLabel("Scheduler")
	schedulerBtn.SetNoShowAll(true)
	schedulerBtn.Connect("clicked", func() {
		showSchedulerDialog(parent, state, scheduler, dashboardWidgets.sites, dashboardWidgets.currentSiteID)
	})
	footer.PackStart(schedulerBtn, false, false, 0)

	settingsBtn, _ := gtk.ButtonNewWithLabel("Settings")
	var revealOverlays func()
	var inhibitCookie uint
	applyDesktopPrefs := func() {
		parent.ToWindow().SetKeepAbove(state.Prefs.KeepWindowInFront)
		if statusIcon != nil {
			statusIcon.setVisible(state.Prefs.ShowInMenuBar)
		}
		if state.Prefs.PreventScreenSaver && inhibitCookie == 0 {
			inhibitCookie = app.Inhibited(parent, gtk.APPLICATION_INHIBIT_IDLE, "Powerwall TV display is active")
		} else if !state.Prefs.PreventScreenSaver && inhibitCookie != 0 {
			app.Uninhibit(inhibitCookie)
			inhibitCookie = 0
		}
		if state.Prefs.LoginMode == "fleetAPI" && state.Prefs.ShowSchedulerButton {
			schedulerBtn.Show()
		} else {
			schedulerBtn.Hide()
		}
	}
	applyDesktopPrefs()
	settingsBtn.Connect("clicked", func() {
		_ = showSettingsDialog(parent, state)
		applyDesktopPrefs()
		if dashboardWidgets != nil {
			dashboardWidgets.relayout()
		}
		if revealOverlays != nil {
			revealOverlays()
		}
	})
	footer.PackStart(settingsBtn, false, false, 0)

	stack.Connect("notify::visible-child-name", func() {
		name := stack.GetVisibleChildName()
		if name == "graphs" {
			backBtn.Show()
		} else {
			backBtn.Hide()
			alloc := root.GetAllocation()
			dashboardWidgets.resize(alloc.GetWidth(), alloc.GetHeight())
		}
	})

	backBtn.Hide()
	footerFixed.Put(footer, homeControlsPadding, homeControlsPadding)
	root.AddOverlay(footerFixed)
	autoHideGeneration := 0
	revealOverlays = func() {
		autoHideGeneration++
		generation := autoHideGeneration
		footer.Show()
		dashboardWidgets.setSummaryVisible(true)
		glib.TimeoutAdd(3000, func() bool {
			if generation != autoHideGeneration || stack.GetVisibleChildName() != "home" {
				return false
			}
			_, footerWidth := footer.GetPreferredWidth()
			_, footerHeight := footer.GetPreferredHeight()
			footerX, footerY := homeControlsPosition(dashboardWidgets.height, footerHeight)
			footerOverlaps := dashboardWidgets.scene.intersects(float64(footerX), float64(footerY), float64(footerWidth), float64(footerHeight))
			if state.Prefs.AutoHideButtonsOnOverlap && footerOverlaps {
				footer.Hide()
			}
			if state.Prefs.AutoHideSummaryOnOverlap && dashboardWidgets.summaryOverlapsScene() {
				dashboardWidgets.setSummaryVisible(false)
			}
			return false
		})
	}

	root.Connect("size-allocate", func(_ *gtk.Overlay) {
		alloc := root.GetAllocation()
		_, footerHeight := footer.GetPreferredHeight()
		footerX, footerY := homeControlsPosition(alloc.GetHeight(), footerHeight)
		footerFixed.Move(footer, footerX, footerY)
		if dashboardWidgets != nil && stack.GetVisibleChildName() == "home" {
			dashboardWidgets.resize(alloc.GetWidth(), alloc.GetHeight())
			revealOverlays()
		}
	})
	root.AddEvents(int(gdk.BUTTON_PRESS_MASK))
	root.Connect("button-press-event", func() bool {
		revealOverlays()
		return false
	})
	root.SetCanFocus(true)
	root.Connect("key-press-event", func(_ *gtk.Overlay, ev *gdk.Event) bool {
		if stack.GetVisibleChildName() != "home" || dashboardWidgets == nil {
			return false
		}
		key := gdk.EventKeyNewFromEvent(ev)
		switch key.KeyVal() {
		case gdk.KEY_Up:
			dashboardWidgets.cycleSite(-1)
			return true
		case gdk.KEY_Down:
			dashboardWidgets.cycleSite(1)
			return true
		case gdk.KEY_Escape:
			parent.ToWindow().Unfullscreen()
			return true
		}
		return false
	})
	root.Connect("map", func() { root.GrabFocus() })
	if state.Prefs.LoginMode == storage.LoginModeLocal && strings.TrimSpace(state.Prefs.GatewayIP) == "" {
		glib.IdleAdd(func() {
			_ = showSettingsDialog(parent, state)
			applyDesktopPrefs()
			dashboardWidgets.relayout()
		})
	} else if state.Prefs.LoginMode == storage.LoginModeFleetAPI {
		if _, err := auth.LoadStoredToken(); err != nil {
			go startFleetLogin(parent, state, dashboardWidgets.error)
		}
	}

	return root, nil
}
