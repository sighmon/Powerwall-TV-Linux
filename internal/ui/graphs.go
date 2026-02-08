package ui

import (
	"fmt"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
)

type graphsWidgets struct {
	solar     *GraphView
	battery   *GraphView
	home      *GraphView
	grid      *GraphView
	percent   *GraphView
	topStack  *gtk.Stack
	topViews  []string
	activeIdx int
	topBox    *gtk.Box
	bottomBox *gtk.Box
}

func (g *graphsWidgets) QueueDrawAll() {
	if g == nil {
		return
	}
	g.solar.QueueDraw()
	g.battery.QueueDraw()
	g.home.QueueDraw()
	g.grid.QueueDraw()
	g.percent.QueueDraw()
}

func (g *graphsWidgets) cycle(delta int) {
	if g == nil || g.topStack == nil || len(g.topViews) == 0 {
		return
	}
	g.activeIdx = (g.activeIdx + delta + len(g.topViews)) % len(g.topViews)
	name := g.topViews[g.activeIdx]
	g.topStack.SetVisibleChildName(name)
	g.QueueDrawAll()
}

func buildGraphsView(charts *ChartsData, onBack func()) (*gtk.Box, *graphsWidgets, error) {
	root, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 16)
	root.SetBorderWidth(20)
	root.SetCanFocus(true)
	root.SetHExpand(true)
	root.SetVExpand(true)
	root.SetSizeRequest(1000, 640)

	graphs := &graphsWidgets{}

	graphs.solar, _ = NewGraphView(charts.Solar, 0, 10000, [3]float64{0.95, 0.8, 0.2})
	graphs.solar.SetLabels("Time", "W")
	graphs.battery, _ = NewGraphView(charts.Battery, -4000, 4000, [3]float64{0.25, 0.5, 0.9})
	graphs.battery.SetLabels("Time", "W")
	graphs.battery.SetAutoRange(true)
	graphs.home, _ = NewGraphView(charts.Home, 0, 6000, [3]float64{0.95, 0.55, 0.2})
	graphs.home.SetLabels("Time", "W")
	graphs.grid, _ = NewGraphView(charts.Grid, -6000, 6000, [3]float64{0.5, 0.5, 0.5})
	graphs.grid.SetLabels("Time", "W")
	graphs.grid.SetAutoRange(true)
	graphs.percent, _ = NewGraphView(charts.BatteryPercent, 0, 100, [3]float64{0.35, 0.75, 0.35})
	graphs.percent.SetLabels("Time", "%")

	graphs.topStack, _ = gtk.StackNew()
	graphs.topStack.SetTransitionType(gtk.STACK_TRANSITION_TYPE_SLIDE_UP_DOWN)
	graphs.topStack.SetTransitionDuration(150)
	graphs.topStack.SetHHomogeneous(false)
	graphs.topStack.SetVHomogeneous(false)
	graphs.topStack.SetHExpand(true)
	graphs.topStack.SetVExpand(true)
	graphs.topStack.SetSizeRequest(-1, 420)
	graphs.topStack.SetHAlign(gtk.ALIGN_FILL)
	graphs.topStack.SetVAlign(gtk.ALIGN_FILL)

	graphs.topViews = []string{"solar", "battery", "home", "grid"}
	graphs.activeIdx = 0
	graphs.topBox = section("Solar Power (W)", graphs.solar)
	graphs.topStack.AddNamed(graphs.topBox, "solar")
	graphs.topStack.AddNamed(section("Battery Power (W)", graphs.battery), "battery")
	graphs.topStack.AddNamed(section("Home Power (W)", graphs.home), "home")
	graphs.topStack.AddNamed(section("Grid Power (W)", graphs.grid), "grid")
	graphs.topStack.SetVisibleChildName(graphs.topViews[graphs.activeIdx])

	graphs.bottomBox = section("Charge Level (%)", graphs.percent)

	root.PackStart(graphs.topStack, true, true, 0)
	root.PackStart(graphs.bottomBox, false, false, 0)
	graphs.bottomBox.SetHExpand(true)
	graphs.bottomBox.SetVExpand(true)
	graphs.bottomBox.SetSizeRequest(-1, 220)

	root.Connect("key-press-event", func(_ *gtk.Box, ev *gdk.Event) bool {
		key := gdk.EventKeyNewFromEvent(ev)
		switch key.KeyVal() {
		case gdk.KEY_Up:
			graphs.cycle(-1)
			return true
		case gdk.KEY_Down:
			graphs.cycle(1)
			return true
		case gdk.KEY_Escape:
			if onBack != nil {
				onBack()
				return true
			}
		}
		return false
	})
	root.Connect("map", func() {
		root.GrabFocus()
	})
	// Let GTK drive sizing via expand/fill; avoid manual size requests during allocation.

	return root, graphs, nil
}

func section(title string, graph *GraphView) *gtk.Box {
	box, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 6)
	box.SetHExpand(true)
	box.SetVExpand(true)
	label, _ := gtk.LabelNew("")
	label.SetMarkup(fmt.Sprintf("<b>%s</b>", title))
	label.SetHAlign(gtk.ALIGN_START)
	label.SetMarginBottom(2)
	box.PackStart(label, false, false, 0)
	graph.Widget().SetHExpand(true)
	graph.Widget().SetVExpand(true)
	box.PackStart(graph.Widget(), true, true, 0)
	return box
}
