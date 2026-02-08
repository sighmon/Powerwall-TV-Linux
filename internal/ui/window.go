package ui

import (
	"github.com/gotk3/gotk3/gtk"
	statepkg "powerwall-tv-gtk/internal/state"
)

func buildMainView(state *statepkg.State, parent gtk.IWindow) (*gtk.Overlay, error) {
	root, _ := gtk.OverlayNew()
	root.SetHExpand(true)
	root.SetVExpand(true)

	charts := NewChartsData()
	stack, _ := gtk.StackNew()
	stack.SetTransitionType(gtk.STACK_TRANSITION_TYPE_NONE)
	stack.SetTransitionDuration(0)
	stack.SetHHomogeneous(false)
	stack.SetVHomogeneous(false)
	stack.SetHExpand(true)
	stack.SetVExpand(true)

	graphsView, graphsWidgets, _ := buildGraphsView(charts, func() {
		stack.SetVisibleChildName("home")
	})

	dashboard, dashboardWidgets, _ := buildDashboard(state, charts, graphsWidgets)
	stack.AddNamed(dashboard, "home")
	stack.AddNamed(graphsView, "graphs")
	stack.SetVisibleChildName("home")

	root.Add(stack)

	// Footer buttons overlay
	footerFixed, _ := gtk.FixedNew()
	footer, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 12)
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
	})
	footer.PackStart(graphsBtn, false, false, 0)

	settingsBtn, _ := gtk.ButtonNewWithLabel("Settings")
	settingsBtn.Connect("clicked", func() {
		_ = showSettingsDialog(parent, state)
	})
	footer.PackStart(settingsBtn, false, false, 0)

	stack.Connect("notify::visible-child-name", func() {
		name := stack.GetVisibleChildName()
		if name == "graphs" {
			backBtn.Show()
		} else {
			backBtn.Hide()
		}
	})

	backBtn.Hide()
	footerFixed.Put(footer, 20, 20)
	root.AddOverlay(footerFixed)

	root.Connect("size-allocate", func(_ *gtk.Overlay) {
		alloc := root.GetAllocation()
		footerFixed.Move(footer, 20, alloc.GetHeight()-80)
		if dashboardWidgets != nil && stack.GetVisibleChildName() == "home" {
			dashboardWidgets.width = alloc.GetWidth()
			dashboardWidgets.height = alloc.GetHeight()
			dashboardWidgets.relayout()
		}
	})

	return root, nil
}
