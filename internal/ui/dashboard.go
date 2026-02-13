package ui

import (
	"context"
	"fmt"
	"math"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gotk3/gotk3/cairo"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
	"powerwall-tv-gtk/internal/api"
	"powerwall-tv-gtk/internal/auth"
	statepkg "powerwall-tv-gtk/internal/state"
	"powerwall-tv-gtk/internal/storage"
)

type dashboardWidgets struct {
	background *gtk.Image
	lines      *gtk.DrawingArea
	fixed      *gtk.Fixed
	width      int
	height     int
	bgName     string

	animPhase float64
	solarP    float64
	batteryP  float64
	homeP     float64
	gridP     float64

	siteName   *gtk.Label
	energyVal  *gtk.Label
	energyLbl  *gtk.Label
	solarVal   *gtk.Label
	solarLbl   *gtk.Label
	homeVal    *gtk.Label
	homeLbl    *gtk.Label
	batteryVal *gtk.Label
	batteryLbl *gtk.Label
	gridVal    *gtk.Label
	gridLbl    *gtk.Label
	vehicleVal *gtk.Label
	vehicleLbl *gtk.Label

	error  *gtk.Label
	charts *ChartsData
	graphs *graphsWidgets
}

func buildDashboard(state *statepkg.State, charts *ChartsData, graphs *graphsWidgets) (*gtk.Box, *dashboardWidgets, error) {
	root, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)

	w := &dashboardWidgets{charts: charts, graphs: graphs}

	overlay, _ := gtk.OverlayNew()
	w.background, _ = gtk.ImageNew()
	overlay.Add(w.background)
	w.lines, _ = gtk.DrawingAreaNew()
	w.lines.SetHExpand(true)
	w.lines.SetVExpand(true)
	w.lines.Connect("draw", func(_ *gtk.DrawingArea, cr *cairo.Context) {
		w.drawPowerLines(cr)
	})
	overlay.AddOverlay(w.lines)
	w.fixed, _ = gtk.FixedNew()
	overlay.AddOverlay(w.fixed)
	overlay.SetHExpand(true)
	overlay.SetVExpand(true)
	root.PackStart(overlay, true, true, 0)

	w.siteName = overlayLabel("")
	w.energyVal = overlayLabel("")
	w.energyLbl = overlaySmallLabel("")
	w.solarVal = overlayLabel("")
	w.solarVal.SetMaxWidthChars(0)
	w.solarVal.SetWidthChars(0)
	w.solarVal.SetLineWrap(false)
	w.solarVal.SetSingleLineMode(true)
	w.solarVal.SetEllipsize(pango.ELLIPSIZE_NONE)
	w.solarVal.SetSizeRequest(240, -1)
	w.solarVal.SetXAlign(1.0)
	w.solarLbl = overlaySmallLabel("SOLAR")
	w.solarLbl.SetSizeRequest(240, -1)
	w.solarLbl.SetXAlign(1.0)
	w.homeVal = overlayLabel("")
	w.homeLbl = overlaySmallLabel("HOME")
	w.batteryVal = overlayLabel("")
	w.batteryVal.SetSizeRequest(240, -1)
	w.batteryVal.SetXAlign(1.0)
	w.batteryLbl = overlaySmallLabel("POWERWALL")
	w.batteryLbl.SetSizeRequest(240, -1)
	w.batteryLbl.SetXAlign(1.0)
	w.gridVal = overlayLabel("")
	w.gridVal.SetSizeRequest(240, -1)
	w.gridVal.SetXAlign(0.0)
	w.gridLbl = overlaySmallLabel("GRID")
	w.gridLbl.SetSizeRequest(240, -1)
	w.gridLbl.SetXAlign(0.0)
	w.vehicleVal = overlayLabel("")
	w.vehicleLbl = overlaySmallLabel("VEHICLE")
	w.error, _ = gtk.LabelNew("")
	w.error.SetHAlign(gtk.ALIGN_START)
	w.error.SetMarkup("<span foreground='red'></span>")

	w.fixed.Put(w.siteName, 20, 20)
	w.fixed.Put(w.energyVal, 20, 60)
	w.fixed.Put(w.energyLbl, 20, 85)
	w.fixed.Put(w.solarVal, 404, 60)
	w.fixed.Put(w.solarLbl, 404, 85)
	w.fixed.Put(w.homeVal, 942, 60)
	w.fixed.Put(w.homeLbl, 942, 85)
	w.fixed.Put(w.batteryVal, 504, 615)
	w.fixed.Put(w.batteryLbl, 504, 640)
	w.fixed.Put(w.gridVal, 942, 615)
	w.fixed.Put(w.gridLbl, 942, 640)
	w.fixed.Put(w.vehicleVal, 60, 140)
	w.fixed.Put(w.vehicleLbl, 60, 165)
	w.fixed.Put(w.error, 20, 120)

	overlay.Connect("size-allocate", func(_ *gtk.Overlay) {
		alloc := overlay.GetAllocation()
		w.width = alloc.GetWidth()
		w.height = alloc.GetHeight()
		w.relayout()
	})

	// initial refresh + periodic updates
	go refresh(state, w)
	glib.TimeoutAdd(10_000, func() bool {
		go refresh(state, w)
		return true
	})
	glib.TimeoutAdd(33, func() bool {
		w.animPhase += 0.012
		if w.animPhase > 1 {
			w.animPhase -= 1
		}
		if w.lines != nil {
			w.lines.QueueDraw()
		}
		return true
	})

	return root, w, nil
}

func refresh(state *statepkg.State, w *dashboardWidgets) {
	if state.Prefs.LoginMode == storage.LoginModeFleetAPI {
		refreshFleet(state, w)
		return
	}
	pw, _ := storage.GetGatewayPassword()
	client := api.NewLocalClient(state.Prefs.GatewayIP, state.Prefs.Username, pw)
	snap, err := client.FetchSnapshot()
	glib.IdleAdd(func() {
		if err != nil {
			w.error.SetText(err.Error())
			return
		}
		w.error.SetText("")
		setValue(w.siteName, "Home")
		if snap.Data.Solar.EnergyExported > 0 {
			setValue(w.energyVal, formatKwh(snap.Data.Solar.EnergyExported, state.Prefs.ShowLessPrecision))
			setSmall(w.energyLbl, "ENERGY GENERATED")
		} else {
			setValue(w.energyVal, "")
			setSmall(w.energyLbl, "")
		}
		w.solarP = snap.Data.Solar.InstantPower
		w.batteryP = snap.Data.Battery.InstantPower
		w.homeP = snap.Data.Load.InstantPower
		w.gridP = snap.Data.Site.InstantPower
		setValue(w.solarVal, formatKW(snap.Data.Solar.InstantPower, state.Prefs.ShowLessPrecision))
		setSmall(w.solarLbl, "SOLAR")
		w.solarVal.SetMaxWidthChars(12)
		w.solarVal.SetLineWrap(true)
		setValue(w.homeVal, formatKW(snap.Data.Load.InstantPower, state.Prefs.ShowLessPrecision))
		setSmall(w.homeLbl, "HOME")
		setValue(w.batteryVal, fmt.Sprintf("%s · %s", formatKW(snap.Data.Battery.InstantPower, state.Prefs.ShowLessPrecision), formatPercent(snap.BatteryPercent.Percentage, state.Prefs.ShowLessPrecision)))
		setSmall(w.batteryLbl, fmt.Sprintf("POWERWALL%s", batteryCountSuffix(0)))
		setValue(w.gridVal, formatKW(snap.Data.Site.InstantPower, state.Prefs.ShowLessPrecision))
		setSmall(w.gridLbl, gridLabelText(false))
		setValue(w.vehicleVal, "")
		setSmall(w.vehicleLbl, "")
		w.setBackground(selectHomeBackground(state, 0, ""))

		if w.charts != nil {
			now := time.Now()
			w.charts.Solar.Add(now, snap.Data.Solar.InstantPower)
			w.charts.Battery.Add(now, snap.Data.Battery.InstantPower)
			w.charts.Home.Add(now, snap.Data.Load.InstantPower)
			w.charts.Grid.Add(now, snap.Data.Site.InstantPower)
			w.charts.BatteryPercent.Add(now, snap.BatteryPercent.Percentage)
		}
		if w.lines != nil {
			w.lines.QueueDraw()
		}
	})
}

func refreshFleet(state *statepkg.State, w *dashboardWidgets) {
	tok, err := auth.LoadStoredToken()
	if err != nil {
		glib.IdleAdd(func() {
			w.error.SetText("")
			w.siteName.SetMarkup("<b>Fleet</b>")
			w.energyVal.SetText("")
			w.energyLbl.SetText("")
			w.solarVal.SetText("")
			w.homeVal.SetText("")
			w.batteryVal.SetText("")
			w.gridVal.SetText("")
			w.setBackground(selectHomeBackground(state, 0, ""))
		})
		return
	}

	// refresh if expired (or within 2 minutes)
	if !tok.Expiry.IsZero() && time.Until(tok.Expiry) < 2*time.Minute {
		secrets := auth.LoadSecrets()
		cfg := auth.Config{ClientID: secrets.ClientID, ClientSecret: secrets.ClientSecret, FleetBaseURL: state.Prefs.FleetBaseURL}
		if tok.RefreshToken != "" && cfg.ClientID != "" && cfg.ClientSecret != "" {
			if newTok, err := auth.RefreshToken(context.Background(), cfg, tok.RefreshToken); err == nil {
				auth.SaveStoredToken(*newTok)
				tok.AccessToken = newTok.AccessToken
			}
		}
	}

	// resolve region base URL if needed
	if state.Prefs.FleetBaseURL == "https://fleet-api.prd.na.vn.cloud.tesla.com" {
		if base, err := api.ResolveFleetBaseURL(tok.AccessToken); err == nil && base != "" && base != state.Prefs.FleetBaseURL {
			state.Prefs.FleetBaseURL = base
			_ = storage.SavePrefs(state.Prefs)
		}
	}

	client := api.NewFleetClient(state.Prefs.FleetBaseURL, tok.AccessToken)
	snap, idx, err := client.FetchSnapshot(state.Prefs.CurrentEnergySiteIdx)
	glib.IdleAdd(func() {
		if err != nil {
			w.error.SetText(err.Error())
			return
		}
		w.error.SetText("")
		state.Prefs.CurrentEnergySiteIdx = idx
		_ = storage.SavePrefs(state.Prefs)
		name := snap.SiteName
		if name == "" {
			name = "Energy Site"
		}
		setValue(w.siteName, name)
		setValue(w.energyVal, "")
		setSmall(w.energyLbl, "")
		w.solarP = snap.Status.SolarPower
		w.batteryP = snap.Status.BatteryPower
		w.homeP = snap.Status.LoadPower
		w.gridP = snap.Status.GridPower
		setValue(w.solarVal, formatKW(snap.Status.SolarPower, state.Prefs.ShowLessPrecision))
		setSmall(w.solarLbl, "SOLAR")
		w.solarVal.SetMaxWidthChars(12)
		w.solarVal.SetLineWrap(true)
		setValue(w.homeVal, formatKW(snap.Status.LoadPower, state.Prefs.ShowLessPrecision))
		setSmall(w.homeLbl, "HOME")
		setValue(w.batteryVal, fmt.Sprintf("%s · %s", formatKW(snap.Status.BatteryPower, state.Prefs.ShowLessPrecision), formatPercent(snap.Status.BatteryPercent, state.Prefs.ShowLessPrecision)))
		setSmall(w.batteryLbl, fmt.Sprintf("POWERWALL%s", batteryCountSuffix(0)))
		setValue(w.gridVal, formatKW(snap.Status.GridPower, state.Prefs.ShowLessPrecision))
		setSmall(w.gridLbl, gridLabelText(snap.Status.GridStatus == "SystemIslandedActive" || snap.Status.GridStatus == "Inactive"))
		setValue(w.vehicleVal, "")
		setSmall(w.vehicleLbl, "")
		w.setBackground(selectHomeBackground(state, 0, ""))

		if w.charts != nil {
			now := time.Now()
			w.charts.Solar.Add(now, snap.Status.SolarPower)
			w.charts.Battery.Add(now, snap.Status.BatteryPower)
			w.charts.Home.Add(now, snap.Status.LoadPower)
			w.charts.Grid.Add(now, snap.Status.GridPower)
			w.charts.BatteryPercent.Add(now, snap.Status.BatteryPercent)
		}
		if w.lines != nil {
			w.lines.QueueDraw()
		}
	})
}

func formatKW(watts float64, lessPrecision bool) string {
	kw := watts / 1000
	if lessPrecision {
		return fmt.Sprintf("%.1f kW", kw)
	}
	return fmt.Sprintf("%.3f kW", kw)
}

func formatKwh(wh float64, lessPrecision bool) string {
	kwh := wh / 1000
	if kwh >= 1000 {
		return fmt.Sprintf("%s kWh", formatIntWithCommas(int64(kwh)))
	}
	if lessPrecision {
		return fmt.Sprintf("%.1f kWh", kwh)
	}
	return fmt.Sprintf("%.3f kWh", kwh)
}

func formatIntWithCommas(n int64) string {
	s := strconv.FormatInt(n, 10)
	if len(s) <= 3 {
		return s
	}
	neg := false
	if s[0] == '-' {
		neg = true
		s = s[1:]
	}
	out := make([]byte, 0, len(s)+len(s)/3)
	first := len(s) % 3
	if first == 0 {
		first = 3
	}
	out = append(out, s[:first]...)
	for i := first; i < len(s); i += 3 {
		out = append(out, ',')
		out = append(out, s[i:i+3]...)
	}
	if neg {
		return "-" + string(out)
	}
	return string(out)
}

func formatPercent(p float64, lessPrecision bool) string {
	if lessPrecision {
		return fmt.Sprintf("%.1f%%", p)
	}
	return fmt.Sprintf("%.1f%%", p)
}

func setValue(lbl *gtk.Label, text string) {
	lbl.SetMarkup(fmt.Sprintf("<span foreground='white' weight='bold'>%s</span>", text))
}

func setSmall(lbl *gtk.Label, text string) {
	lbl.SetMarkup(fmt.Sprintf("<span foreground='white' weight='bold' size='small' alpha='70%%'>%s</span>", text))
}

func overlayLabel(text string) *gtk.Label {
	lbl, _ := gtk.LabelNew("")
	lbl.SetHAlign(gtk.ALIGN_START)
	setValue(lbl, text)
	return lbl
}

func overlaySmallLabel(text string) *gtk.Label {
	lbl, _ := gtk.LabelNew("")
	lbl.SetHAlign(gtk.ALIGN_START)
	setSmall(lbl, text)
	return lbl
}

func batteryCountSuffix(count float64) string {
	if count <= 0 {
		return ""
	}
	return fmt.Sprintf(" · %.0fx", count)
}

func gridLabelText(offGrid bool) string {
	if offGrid {
		return "OFF-GRID"
	}
	return "GRID"
}

func selectHomeBackground(state *statepkg.State, wallConnectorPower float64, wallConnectorStatus string) string {
	hasConnector := state.Prefs.WallConnectorIP != ""
	if !hasConnector {
		return "home.png"
	}
	if wallConnectorPower > 10 || wallConnectorStatus == "Plugged in" {
		return "home-charger.png"
	}
	return "home-charger-empty.png"
}

func (w *dashboardWidgets) setBackground(name string) {
	w.bgName = name
	if w.background == nil || w.width == 0 || w.height == 0 {
		return
	}
	path := filepath.Join("/home/sighmon/Code/powerwall-tv/Powerwall-TV/Images", name)
	// fill/cover (no bars)
	pix, err := gdk.PixbufNewFromFileAtScale(path, w.width, w.height, false)
	if err != nil {
		return
	}
	w.background.SetFromPixbuf(pix)
}

func (w *dashboardWidgets) drawPowerLines(cr *cairo.Context) {
	if w.width == 0 || w.height == 0 {
		return
	}
	ww, wh := float64(w.width), float64(w.height)
	cx, cy := ww/2, wh/2
	hw, hh := ww/2, wh/2

	type np2 struct{ nx, ny float64 }
	toAbs := func(p np2) (float64, float64) {
		return cx + p.nx*hw, cy + p.ny*hh
	}

	// All points are normalized relative to screen centre:
	// nx = (x - centerX) / halfWidth, ny = (y - centerY) / halfHeight
	// Converted from the original 1280x720 tuning points so anchors scale cleanly.
	solarN := np2{nx: -0.1171875, ny: -0.5}               // (565,180)
	pwrN := np2{nx: -0.046875, ny: 0.5833333333333334}    // (610,570)
	junctionLTN := np2{nx: 0.15625, ny: -0.08333333333333333}  // (740,330)
	junctionRBN := np2{nx: 0.234375, ny: 0.011111111111111112} // (790,364)

	solarX, solarY := toAbs(solarN)
	pwrX, pwrY := toAbs(pwrN)

	// Virtual junction anchors used for flow geometry.
	junctionL, junctionT := toAbs(junctionLTN)
	junctionR, junctionB := toAbs(junctionRBN)
	junctionCx, junctionCy := (junctionL+junctionR)/2, (junctionT+junctionB)/2

	type p2 struct{ x, y float64 }
	const baseLineWidth = 4.0
	const pulseTrail = 1.0
	const gatewayTransitFrac = 0.3 / 2.2 // 0.3s through gateway over ~2.2s animation cycle
	drawBase := func(path []p2) {
		if len(path) < 2 {
			return
		}
		cr.SetLineWidth(baseLineWidth)
		cr.SetSourceRGBA(1, 1, 1, 0.16)
		cr.MoveTo(path[0].x, path[0].y)
		for i := 1; i < len(path); i++ {
			cr.LineTo(path[i].x, path[i].y)
		}
		cr.Stroke()
	}

	pulsePath := func(path []p2, on bool, forward bool, r, g, b float64, startAt, durationFrac float64) {
		if !on || len(path) < 2 {
			return
		}
		// Total path length
		lens := make([]float64, len(path)-1)
		total := 0.0
		for i := 0; i < len(path)-1; i++ {
			dx := path[i+1].x - path[i].x
			dy := path[i+1].y - path[i].y
			lens[i] = math.Hypot(dx, dy)
			total += lens[i]
		}
		if total <= 0 {
			return
		}

		trail := pulseTrail // pulse length as fraction of total path
		if durationFrac <= 0 {
			return
		}
		// Normalize start into [0..1) and allow wrap into next cycle so speed stays constant.
		startAt = math.Mod(startAt, 1)
		if startAt < 0 {
			startAt += 1
		}
		local := w.animPhase - startAt
		if local < 0 {
			local += 1
		}
		if local < 0 || local > durationFrac {
			return
		}
		localPhase := local / durationFrac // [0..1] over this path's duration
		prog := localPhase * (1 + trail)                     // so tail reaches destination before reset
		head := math.Min(1, prog)
		tail := math.Max(0, prog-trail)
		if !forward {
			head = 1 - math.Min(1, prog)
			tail = 1 - math.Max(0, prog-trail)
		}

		// Convert t in [0..1] to point on polyline
		pointAt := func(t float64) p2 {
			t = math.Max(0, math.Min(1, t))
			target := t * total
			acc := 0.0
			for i := 0; i < len(lens); i++ {
				if acc+lens[i] >= target {
					u := 0.0
					if lens[i] > 0 {
						u = (target - acc) / lens[i]
					}
					return p2{
						x: path[i].x + (path[i+1].x-path[i].x)*u,
						y: path[i].y + (path[i+1].y-path[i].y)*u,
					}
				}
				acc += lens[i]
			}
			return path[len(path)-1]
		}

		pHead := pointAt(head)
		pTail := pointAt(tail)
		cr.SetLineWidth(5)
		cr.SetSourceRGBA(r, g, b, 0.95)
		cr.MoveTo(pTail.x, pTail.y)
		cr.LineTo(pHead.x, pHead.y)
		cr.Stroke()
	}

	// Original guide geometry (used as reference for requested relative transforms)
	origSolarPath := []p2{{solarX, solarY}, {junctionL, junctionCy}}
	origPowerwallPath := []p2{{pwrX, pwrY}, {junctionCx, junctionB}}

	midpoint := func(a, b p2) p2 { return p2{(a.x + b.x) / 2, (a.y + b.y) / 2} }
	dist := func(a, b p2) float64 { return math.Hypot(b.x-a.x, b.y-a.y) }

	// Solar:
	// - Start where Powerwall currently ends.
	// - End directly down, 1/3 of Solar's current length.
	solarStart := origPowerwallPath[1]
	solarStart.x -= 2
	solarStart.y += 20
	solarCurrentLen := dist(origSolarPath[0], origSolarPath[1])
	solarLen := solarCurrentLen / 3.8
	solarEnd := p2{solarStart.x, solarStart.y + solarLen - 20}
	pathSolarToHub := []p2{solarStart, solarEnd}

	// Powerwall:
	// - Start at midpoint of current Powerwall path.
	// - End tiny bit lower/left of Solar end.
	powerwallStart := midpoint(origPowerwallPath[0], origPowerwallPath[1])
	powerwallStart.x -= 10
	powerwallStart.y += 3
	powerwallEnd := p2{solarEnd.x - 0.01*ww + 12, solarEnd.y + 0.005*wh + 17}
	pathPowerwallToHub := []p2{powerwallStart, powerwallEnd}

	// Grid:
	// - Start directly below Solar end, half Solar length below.
	// - End directly down.
	gridStart := p2{solarEnd.x, solarEnd.y + 0.5*solarLen + 10}
	gridEnd := p2{gridStart.x, gridStart.y + solarLen - 34}
	// Extend past the existing endpoint down/right to follow the background tail.
	gridTail := p2{gridEnd.x + 0.092*ww, gridEnd.y + 0.06*wh}
	pathGridToHub := []p2{gridStart, gridEnd, gridTail}

	// Home:
	// - Start 0.5% above and 2% right of Powerwall end.
	// - End with same vector (angle + length) as Powerwall.
	homeStart := p2{powerwallEnd.x + 0.02*ww - 8, powerwallEnd.y - 0.005*wh}
	pwrVec := p2{powerwallEnd.x - powerwallStart.x, powerwallEnd.y - powerwallStart.y}
	homeEnd := p2{homeStart.x + pwrVec.x - 10, homeStart.y + pwrVec.y + 2}
	pathHubToHome := []p2{homeStart, homeEnd}

	// Positional nudge requests (relative to current stroke width):
	// - Solar->Gateway and Gateway->Grid: right by 2x line width
	// - Powerwall->Gateway and Gateway->Home: up by 2x line width
	pathShift := 2 * baseLineWidth
	offsetPath := func(path []p2, dx, dy float64) []p2 {
		out := make([]p2, len(path))
		for i := range path {
			out[i] = p2{x: path[i].x + dx, y: path[i].y + dy}
		}
		return out
	}
	pathSolarToHub = offsetPath(pathSolarToHub, pathShift, 0)
	pathGridToHub = offsetPath(pathGridToHub, pathShift, 0)
	pathPowerwallToHub = offsetPath(pathPowerwallToHub, 0, -pathShift)
	pathHubToHome = offsetPath(pathHubToHome, 0, -pathShift)

	// Draw updated topology
	drawBase(pathSolarToHub)
	drawBase(pathPowerwallToHub)
	drawBase(pathGridToHub)
	drawBase(pathHubToHome)

	polyLen := func(path []p2) float64 {
		t := 0.0
		for i := 0; i < len(path)-1; i++ {
			t += math.Hypot(path[i+1].x-path[i].x, path[i+1].y-path[i].y)
		}
		return t
	}
	lSolar := polyLen(pathSolarToHub)
	lPwr := polyLen(pathPowerwallToHub)
	lGrid := polyLen(pathGridToHub)
	lHome := polyLen(pathHubToHome)
	maxLen := math.Max(math.Max(lSolar, lPwr), math.Max(lGrid, lHome))
	if maxLen <= 0 {
		return
	}
	// Duration share of the 0..1 cycle so apparent speed is constant across path lengths.
	dSolar := lSolar / maxLen
	dPwr := lPwr / maxLen
	dGrid := lGrid / maxLen

	// Estimate directional source flows from balance so direction/colour remain correct
	// even when API sign conventions vary.
	homeDemand := math.Max(0, w.homeP)
	solarSupply := math.Max(0, w.solarP)
	batteryDischarge := math.Max(0, w.batteryP) // +ve means battery discharging

	solarToHome := math.Min(solarSupply, homeDemand)
	remainingAfterSolar := math.Max(0, homeDemand-solarToHome)
	batteryToHome := math.Min(batteryDischarge, remainingAfterSolar)
	remainingAfterBattery := math.Max(0, remainingAfterSolar-batteryToHome)
	gridImport := remainingAfterBattery
	gridExport := math.Max(0, solarSupply+batteryDischarge-homeDemand)

	// Export source split for grid colour when exporting.
	solarExcess := math.Max(0, solarSupply-homeDemand)
	batteryToGrid := math.Max(0, gridExport-solarExcess)
	exportColor := [3]float64{0.98, 0.82, 0.24} // solar default
	if batteryToGrid >= solarExcess {
		exportColor = [3]float64{0.36, 0.82, 0.38}
	}

	headArrival := func(sourceDur float64) float64 {
		return sourceDur * (1.0 / (1.0 + pulseTrail))
	}

	// Incoming segments to gateway
	pulsePath(pathSolarToHub, solarSupply > 30, true, 0.98, 0.82, 0.24, 0, dSolar)

	// Powerwall path colour depends on direction/source:
	// - Discharging (powerwall -> gateway): green
	// - Charging (gateway -> powerwall): colour by dominant source into gateway (solar or grid)
	batteryOn := math.Abs(w.batteryP) > 30
	batteryForward := w.batteryP > 0 // true => powerwall -> gateway; false => gateway -> powerwall
	batteryR, batteryG, batteryB := 0.36, 0.82, 0.38
	batteryStart := 0.0
	if !batteryForward {
		batteryCharge := math.Max(0, -w.batteryP)
		solarExcessForCharge := math.Max(0, solarSupply-homeDemand)
		solarToBattery := math.Min(solarExcessForCharge, batteryCharge)
		gridToBattery := math.Max(0, batteryCharge-solarToBattery)
		if solarToBattery >= gridToBattery {
			batteryR, batteryG, batteryB = 0.98, 0.82, 0.24
			// Charging from solar: start only after solar pulse reaches gateway.
			batteryStart = headArrival(dSolar)
		} else {
			batteryR, batteryG, batteryB = 0.72, 0.72, 0.72
			// Charging from grid: start after grid pulse reaches gateway.
			batteryStart = headArrival(dGrid)
		}
	}
	pulsePath(pathPowerwallToHub, batteryOn, batteryForward, batteryR, batteryG, batteryB, batteryStart, dPwr)
	// Grid path: direction follows actual grid sign; colour reflects export source.
	gridOn := math.Abs(w.gridP) > 30 || gridImport > 30 || gridExport > 30
	// pathGridToHub is defined from gateway-ish anchor down to grid anchor,
	// so "forward" means gateway -> grid.
	gridForward := w.gridP < 0 // import: false (grid -> gateway), export: true (gateway -> grid)
	gridR, gridG, gridB := 0.72, 0.72, 0.72
	if w.gridP < -30 || gridExport > 30 {
		gridR, gridG, gridB = exportColor[0], exportColor[1], exportColor[2]
	}
	// For export (gateway->grid), start when the source HEAD reaches gateway,
	// then keep the extra +dPwr offset requested earlier.
	gridPulseStart := dPwr + gatewayTransitFrac
	if gridForward {
		if solarExcess >= batteryToGrid {
			gridPulseStart = headArrival(dSolar) + dPwr + gatewayTransitFrac
		} else {
			gridPulseStart = headArrival(dPwr) + dPwr + gatewayTransitFrac
		}
	}
	pulsePath(pathGridToHub, gridOn, gridForward, gridR, gridG, gridB, gridPulseStart, dGrid)

	// Gateway -> Home colour follows the biggest contributor to HOME (not raw source power).
	if homeDemand > 30 {
		r, g, b := 0.72, 0.72, 0.72
		sourceDur := dGrid
		if solarToHome >= batteryToHome && solarToHome >= gridImport {
			r, g, b = 0.98, 0.82, 0.24
			sourceDur = dSolar
		} else if batteryToHome >= solarToHome && batteryToHome >= gridImport {
			r, g, b = 0.36, 0.82, 0.38
			sourceDur = dPwr
		}
		// Start after winning source HEAD reaches gateway, then traverse gateway.
		// (No extra full-path offset; this keeps Home starting right after source arrival.)
		homeStart := headArrival(sourceDur) + gatewayTransitFrac
		// Keep Gateway->Home speed equal to Powerwall->Gateway speed (px per cycle).
		homeDur := dPwr
		if lPwr > 0 {
			homeDur = dPwr * (lHome / lPwr)
		}
		pulsePath(pathHubToHome, true, true, r, g, b, homeStart, homeDur)
	}
}

func (w *dashboardWidgets) relayout() {
	if w.fixed == nil {
		return
	}
	bw, bh := 1280.0, 720.0
	ww, wh := float64(w.width), float64(w.height)
	if ww == 0 || wh == 0 {
		return
	}
	scale := ww / bw
	if wh/bh < scale {
		scale = wh / bh
	}
	offX := (ww - bw*scale) / 2
	offY := (wh - bh*scale) / 2
	move := func(widget gtk.IWidget, x, y float64) {
		w.fixed.Move(widget, int(offX+x*scale), int(offY+y*scale))
	}
	moveRight := func(widget gtk.IWidget, rightX, y float64) {
		wwi, _ := widget.ToWidget().GetPreferredWidth()
		x := rightX - float64(wwi)
		w.fixed.Move(widget, int(offX+x*scale), int(offY+y*scale))
	}
	moveRightBox := func(widget gtk.IWidget, rightX, y float64) {
		w.fixed.Move(widget, int(offX+rightX*scale), int(offY+y*scale))
	}
	move(w.siteName, 20, 20)
	move(w.energyVal, 20, 60)
	move(w.energyLbl, 20, 85)
	moveRightBox(w.solarVal, 494, 60)
	moveRightBox(w.solarLbl, 494, 85)
	move(w.homeVal, 942, 60)
	move(w.homeLbl, 942, 85)
	moveRight(w.batteryVal, 644, 615)
	moveRight(w.batteryLbl, 644, 640)
	move(w.gridVal, 942, 615)
	move(w.gridLbl, 942, 640)
	move(w.vehicleVal, 60, 140)
	move(w.vehicleLbl, 60, 165)
	move(w.error, 20, 120)
	// background image is refreshed in data update path only.
	if w.lines != nil {
		w.lines.QueueDraw()
	}
}
