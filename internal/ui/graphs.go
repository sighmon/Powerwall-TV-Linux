package ui

import (
	"fmt"
	"time"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"powerwall-tv-gtk/internal/api"
	"powerwall-tv-gtk/internal/auth"
	statepkg "powerwall-tv-gtk/internal/state"
	"powerwall-tv-gtk/internal/storage"
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
	state     *statepkg.State
	siteID    int
	date      time.Time
	totals    api.HistoryTotals
	title     *gtk.Label
	subtitle  *gtk.Label
	loading   bool
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
	g.updateHeader()
	g.QueueDrawAll()
}

func buildGraphsView(charts *ChartsData, state *statepkg.State, onBack func()) (*gtk.Box, *graphsWidgets, error) {
	root, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 16)
	root.SetBorderWidth(20)
	root.SetCanFocus(true)
	root.SetHExpand(true)
	root.SetVExpand(true)
	root.SetSizeRequest(1000, 640)

	graphs := &graphsWidgets{state: state, date: time.Now()}

	header, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 12)
	previous, _ := gtk.ButtonNewWithLabel("‹")
	previous.SetTooltipText("Previous day")
	previous.Connect("clicked", func() { graphs.changeDay(-1) })
	next, _ := gtk.ButtonNewWithLabel("›")
	next.SetTooltipText("Next day")
	next.Connect("clicked", func() { graphs.changeDay(1) })
	titles, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 2)
	graphs.title, _ = gtk.LabelNew("")
	graphs.title.SetHAlign(gtk.ALIGN_CENTER)
	graphs.subtitle, _ = gtk.LabelNew("")
	graphs.subtitle.SetHAlign(gtk.ALIGN_CENTER)
	titles.PackStart(graphs.title, false, false, 0)
	titles.PackStart(graphs.subtitle, false, false, 0)
	header.PackStart(previous, false, false, 0)
	header.PackStart(titles, true, true, 0)
	header.PackEnd(next, false, false, 0)
	root.PackStart(header, false, false, 0)

	graphs.solar, _ = NewGraphView(charts.Solar, 0, 10000, [3]float64{0.95, 0.8, 0.2})
	graphs.solar.SetLabels("Time", "W")
	graphs.solar.SetDisplayScale(0.001)
	graphs.battery, _ = NewGraphView(charts.Battery, -4000, 4000, [3]float64{0.25, 0.5, 0.9})
	graphs.battery.SetLabels("Time", "W")
	graphs.battery.SetDisplayScale(0.001)
	graphs.battery.SetAutoRange(true)
	graphs.home, _ = NewGraphView(charts.Home, 0, 6000, [3]float64{0.95, 0.55, 0.2})
	graphs.home.SetLabels("Time", "W")
	graphs.home.SetDisplayScale(0.001)
	graphs.grid, _ = NewGraphView(charts.Grid, -6000, 6000, [3]float64{0.5, 0.5, 0.5})
	graphs.grid.SetLabels("Time", "W")
	graphs.grid.SetDisplayScale(0.001)
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

	graphs.topViews = []string{"battery", "solar", "home", "grid"}
	graphs.activeIdx = 0
	graphs.topBox = section("Powerwall Energy Flow", graphs.battery)
	graphs.topStack.AddNamed(graphs.topBox, "battery")
	graphs.topStack.AddNamed(section("Solar Power", graphs.solar), "solar")
	graphs.topStack.AddNamed(section("Home Power (W)", graphs.home), "home")
	graphs.topStack.AddNamed(section("Grid Power (W)", graphs.grid), "grid")
	graphs.topStack.SetVisibleChildName(graphs.topViews[graphs.activeIdx])

	graphs.bottomBox = section("Charge Level (%)", graphs.percent)

	root.PackStart(graphs.topStack, true, true, 0)
	root.PackStart(graphs.bottomBox, false, false, 0)
	graphs.bottomBox.SetHExpand(true)
	graphs.bottomBox.SetVExpand(true)
	graphs.bottomBox.SetSizeRequest(-1, 220)
	graphs.updateHeader()

	root.Connect("key-press-event", func(_ *gtk.Box, ev *gdk.Event) bool {
		key := gdk.EventKeyNewFromEvent(ev)
		switch key.KeyVal() {
		case gdk.KEY_Up:
			graphs.cycle(-1)
			return true
		case gdk.KEY_Down:
			graphs.cycle(1)
			return true
		case gdk.KEY_Left:
			graphs.changeDay(-1)
			return true
		case gdk.KEY_Right:
			graphs.changeDay(1)
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

func (g *graphsWidgets) setFleetSite(siteID int) { g.siteID = siteID }

func (g *graphsWidgets) changeDay(delta int) {
	next := g.date.AddDate(0, 0, delta)
	if next.After(time.Now()) {
		next = time.Now()
	}
	g.date = next
	g.loadHistory()
}

func (g *graphsWidgets) loadHistory() {
	if g == nil || g.loading || g.state == nil || g.state.Prefs.LoginMode != storage.LoginModeFleetAPI || g.siteID == 0 {
		return
	}
	token, err := auth.LoadStoredToken()
	if err != nil {
		g.subtitle.SetText("Log in to load Fleet history")
		return
	}
	g.loading = true
	g.subtitle.SetText("Loading…")
	client := api.NewFleetClient(g.state.Prefs.FleetBaseURL, token.AccessToken)
	date := g.date
	go func() {
		history, err := client.FetchCalendarHistory(g.siteID, date, time.Now().Location().String())
		glib.IdleAdd(func() {
			g.loading = false
			if err != nil {
				g.subtitle.SetText("Error: " + err.Error())
				return
			}
			g.totals = history.Totals
			g.solar.series.Replace(history.Solar)
			g.battery.series.Replace(history.Battery)
			g.home.series.Replace(history.Home)
			g.grid.series.Replace(history.Grid)
			g.percent.series.Replace(history.SOE)
			g.updateHeader()
			g.QueueDrawAll()
		})
	}()
}

func (g *graphsWidgets) updateHeader() {
	if g == nil || g.title == nil || g.subtitle == nil {
		return
	}
	titles := []string{"Powerwall", "Solar", "Home", "Grid"}
	subtitles := []string{"ENERGY FLOW", "SOLAR POWER", "HOME POWER", "GRID POWER"}
	totals := []float64{g.totals.BatteryWh, g.totals.SolarWh, g.totals.HomeWh, g.totals.GridWh}
	idx := g.activeIdx
	dateLabel := g.date.Format("2 Jan 2006")
	now := time.Now()
	if sameDay(g.date, now) {
		dateLabel = "Today"
	} else if sameDay(g.date, now.AddDate(0, 0, -1)) {
		dateLabel = "Yesterday"
	}
	g.title.SetMarkup(fmt.Sprintf("<big><b>%s</b></big>", titles[idx]))
	g.subtitle.SetMarkup(fmt.Sprintf("<small>%s · %s · %.1f kWh</small>", dateLabel, subtitles[idx], totals[idx]/1000))
}

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.In(a.Location()).Date()
	return ay == by && am == bm && ad == bd
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
