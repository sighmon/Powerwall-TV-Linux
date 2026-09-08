package ui

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gotk3/gotk3/cairo"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
	"powerwall-tv-gtk/internal/api"
	"powerwall-tv-gtk/internal/auth"
	"powerwall-tv-gtk/internal/schedule"
	statepkg "powerwall-tv-gtk/internal/state"
	"powerwall-tv-gtk/internal/storage"
)

type dashboardWidgets struct {
	state          *statepkg.State
	scheduler      *schedule.Manager
	background     *cairo.Surface
	backgroundName string
	backgroundW    int
	backgroundH    int
	offGridSurface *cairo.Surface
	offGridWidth   int
	offGridHeight  int
	lines          *gtk.DrawingArea
	fixed          *gtk.Fixed
	width          int
	height         int
	bgName         string
	scene          sceneRect

	animPhase          float64
	introStarted       time.Time
	solarP             float64
	batteryP           float64
	homeP              float64
	gridP              float64
	batteryPercent     float64
	offGrid            bool
	siteCount          int
	currentSiteID      int
	sites              []api.Product
	gridBaseLabel      string
	carbonIntensity    *int
	renewablePercent   *float64
	batteryCount       float64
	backupReserve      float64
	stormWatch         bool
	siteInfoID         int
	refreshCount       int
	solarTodaySiteID   int
	solarTodayAt       time.Time
	solarTodayWh       float64
	runtime            runtimeEstimator
	vehicleMu          sync.Mutex
	vehicleCharge      map[string]api.VehicleChargeState
	vehicleFetchedAt   map[string]time.Time
	vehicleAttemptedAt map[string]time.Time
	vehicles           map[string]api.FleetVehicle
	vehicleListAt      time.Time
	vehicleCacheGen    uint64
	layoutGeneration   uint64
	metricBoxes        map[*gtk.Label]*gtk.Box

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

func buildDashboard(state *statepkg.State, charts *ChartsData, graphs *graphsWidgets, scheduler *schedule.Manager) (*gtk.Box, *dashboardWidgets, error) {
	root, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)

	w := &dashboardWidgets{state: state, scheduler: scheduler, charts: charts, graphs: graphs, vehicleCharge: make(map[string]api.VehicleChargeState), vehicleFetchedAt: make(map[string]time.Time), vehicleAttemptedAt: make(map[string]time.Time), vehicles: make(map[string]api.FleetVehicle), vehicleCacheGen: state.VehicleCacheGeneration, introStarted: time.Now()}
	for vin, cached := range storage.LoadVehicleChargeCache(time.Now()) {
		w.vehicleCharge[vin] = api.VehicleChargeState{BatteryLevel: cached.BatteryLevel, ChargingState: cached.ChargingState, MinutesToFullCharge: cached.MinutesToFullCharge}
		w.vehicleFetchedAt[vin] = cached.FetchedAt
	}

	overlay, _ := gtk.OverlayNew()
	w.lines, _ = gtk.DrawingAreaNew()
	w.lines.SetHExpand(true)
	w.lines.SetVExpand(true)
	w.lines.Connect("draw", func(_ *gtk.DrawingArea, cr *cairo.Context) {
		w.drawPowerLines(cr)
	})
	overlay.Add(w.lines)
	w.fixed, _ = gtk.FixedNew()
	w.fixed.SetHAlign(gtk.ALIGN_FILL)
	w.fixed.SetVAlign(gtk.ALIGN_FILL)
	w.fixed.SetHExpand(true)
	w.fixed.SetVExpand(true)
	w.fixed.SetOpacity(0)
	overlay.AddOverlay(w.fixed)
	overlay.SetHExpand(true)
	overlay.SetVExpand(true)
	root.PackStart(overlay, true, true, 0)

	w.siteName = overlayLabel("")
	w.energyVal = overlayLabel("")
	w.energyLbl = overlaySmallLabel("")
	for _, summary := range []*gtk.Label{w.siteName, w.energyVal, w.energyLbl} {
		summary.SetSizeRequest(260, -1)
		summary.SetXAlign(0)
	}
	w.solarVal = overlayLabel("")
	w.solarVal.SetMaxWidthChars(0)
	w.solarVal.SetWidthChars(0)
	w.solarVal.SetLineWrap(false)
	w.solarVal.SetSingleLineMode(true)
	w.solarVal.SetEllipsize(pango.ELLIPSIZE_NONE)
	w.solarVal.SetSizeRequest(240, -1)
	w.solarVal.SetXAlign(0.5)
	w.solarLbl = overlaySmallLabel("SOLAR")
	w.solarLbl.SetSizeRequest(240, -1)
	w.solarLbl.SetXAlign(0.5)
	w.homeVal = overlayLabel("")
	w.homeLbl = overlaySmallLabel("HOME")
	w.homeVal.SetSizeRequest(240, -1)
	w.homeVal.SetXAlign(0.5)
	w.homeLbl.SetSizeRequest(240, -1)
	w.homeLbl.SetXAlign(0.5)
	w.batteryVal = overlayLabel("")
	w.batteryVal.SetSizeRequest(240, -1)
	w.batteryVal.SetXAlign(0.5)
	w.batteryLbl = overlaySmallLabel("POWERWALL")
	w.batteryLbl.SetSizeRequest(240, -1)
	w.batteryLbl.SetXAlign(0.5)
	w.gridVal = overlayLabel("")
	w.gridVal.SetSizeRequest(240, -1)
	w.gridVal.SetXAlign(0.5)
	w.gridLbl = overlaySmallLabel("GRID")
	w.gridLbl.SetSizeRequest(240, -1)
	w.gridLbl.SetXAlign(0.5)
	w.vehicleVal = overlayLabel("")
	w.vehicleLbl = overlaySmallLabel("VEHICLE")
	w.vehicleVal.SetSizeRequest(240, -1)
	w.vehicleVal.SetXAlign(0.5)
	w.vehicleLbl.SetSizeRequest(240, -1)
	w.vehicleLbl.SetXAlign(0.5)
	w.error, _ = gtk.LabelNew("")
	w.error.SetHAlign(gtk.ALIGN_START)
	w.error.SetSizeRequest(260, -1)
	w.error.SetXAlign(0)
	w.error.SetMarkup("<span foreground='red'></span>")

	w.fixed.Put(w.siteName, 20, 20)
	w.metricBoxes = make(map[*gtk.Label]*gtk.Box)
	for _, pair := range [][2]*gtk.Label{
		{w.energyVal, w.energyLbl}, {w.solarVal, w.solarLbl},
		{w.homeVal, w.homeLbl}, {w.batteryVal, w.batteryLbl},
		{w.gridVal, w.gridLbl}, {w.vehicleVal, w.vehicleLbl},
	} {
		box, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, metricLineSpacing)
		for _, label := range pair {
			label.SetHAlign(gtk.ALIGN_FILL)
			box.PackStart(label, false, false, 0)
		}
		w.metricBoxes[pair[0]] = box
		w.fixed.Put(box, 0, 0)
		// Text arriving after the first layout can change the pair's size.
		// Recenter the pair without resetting its fonts or allocating its children.
		box.Connect("size-allocate", func() { w.positionMetrics() })
	}
	w.fixed.Put(w.error, 20, 120)

	overlay.Connect("size-allocate", func(_ *gtk.Overlay) {
		alloc := overlay.GetAllocation()
		w.resize(alloc.GetWidth(), alloc.GetHeight())
	})

	// initial refresh + periodic updates
	go refresh(state, w)
	glib.TimeoutAdd(10_000, func() bool {
		go refresh(state, w)
		return true
	})
	go refreshElectricityMaps(state, w)
	glib.TimeoutAdd(900_000, func() bool {
		go refreshElectricityMaps(state, w)
		return true
	})
	glib.TimeoutAdd(33, func() bool {
		if time.Since(w.introStarted) >= 1200*time.Millisecond {
			w.animPhase += 0.012
			if w.animPhase > 1 {
				w.animPhase -= 1
			}
		}
		if w.lines != nil {
			w.lines.QueueDraw()
		}
		w.fixed.SetOpacity(introEaseOut(w.introStarted, 1200*time.Millisecond, 700*time.Millisecond, time.Now()))
		return true
	})

	return root, w, nil
}

func refresh(state *statepkg.State, w *dashboardWidgets) {
	if state.Prefs.LoginMode == storage.LoginModeLocal && state.Prefs.GatewayIP == "demo" {
		refreshDemo(state, w)
		return
	}
	if state.Prefs.LoginMode == storage.LoginModeFleetAPI {
		refreshFleet(state, w)
		return
	}
	pw, _ := storage.GetGatewayPassword()
	client := api.NewLocalClient(state.Prefs.GatewayIP, state.Prefs.Username, pw)
	snap, err := client.FetchSnapshot()
	if err == nil && state.Prefs.WallConnectorIP != "" {
		var connectorErr error
		snap.WallConnector, connectorErr = api.FetchWallConnectorVitals(state.Prefs.WallConnectorIP)
		if connectorErr != nil {
			snap.Warning = "Failed to fetch Wall Connector vitals: " + connectorErr.Error()
		}
	}
	glib.IdleAdd(func() {
		if err != nil {
			setError(w.error, err.Error())
			return
		}
		setError(w.error, snap.Warning)
		siteName := snap.SiteName
		if siteName == "" {
			siteName = "Home"
		}
		setValue(w.siteName, siteName)
		if snap.Data.Solar.EnergyExported > 0 {
			setValue(w.energyVal, formatKwh(snap.Data.Solar.EnergyExported, state.Prefs.ShowLessPrecision))
			setSmall(w.energyLbl, "ENERGY GENERATED")
		} else {
			setValue(w.energyVal, "")
			setSmall(w.energyLbl, "")
		}
		w.solarP = snap.Data.Solar.InstantPower
		w.batteryP = snap.Data.Battery.InstantPower
		w.batteryPercent = snap.BatteryPercent.Percentage
		w.offGrid = snap.GridStatus.Status == "SystemIslandedActive" || snap.GridStatus.Status == "Inactive"
		w.setOffGridImage(w.offGrid)
		connectorPower := 0.0
		connectorStatus := ""
		if snap.WallConnector != nil {
			connectorPower = snap.WallConnector.Power()
			connectorStatus = snap.WallConnector.Status()
		}
		w.homeP = snap.Data.Load.InstantPower - connectorPower
		w.gridP = snap.Data.Site.InstantPower
		setValue(w.solarVal, formatKW(snap.Data.Solar.InstantPower, state.Prefs.ShowLessPrecision))
		setSmall(w.solarLbl, "SOLAR")
		setValue(w.homeVal, formatKW(w.homeP, state.Prefs.ShowLessPrecision))
		setSmall(w.homeLbl, "HOME")
		w.runtime.record(snap.Data.Battery.InstantPower, time.Now())
		w.stormWatch = false
		w.setBatteryLabels(snap.Data.Battery.InstantPower, snap.BatteryPercent.Percentage, snap.Data.Battery.Count, 0, state.Prefs)
		setValue(w.gridVal, formatKW(snap.Data.Site.InstantPower, state.Prefs.ShowLessPrecision))
		w.gridBaseLabel = gridLabelText(snap.GridStatus.Status == "SystemIslandedActive" || snap.GridStatus.Status == "Inactive")
		w.updateGridLabels(state.Prefs.ShowLessPrecision)
		if snap.WallConnector != nil {
			if connectorStatus == "Charging" {
				setValue(w.vehicleVal, formatKW(connectorPower, state.Prefs.ShowLessPrecision))
			} else {
				setValue(w.vehicleVal, connectorStatus)
			}
			setSmall(w.vehicleLbl, "VEHICLE")
		} else {
			setValue(w.vehicleVal, "")
			setSmall(w.vehicleLbl, "")
		}
		w.setBackground(selectHomeBackground(state, connectorPower, connectorStatus))

		if w.charts != nil {
			now := time.Now()
			w.charts.Solar.Add(now, snap.Data.Solar.InstantPower)
			w.charts.Battery.Add(now, snap.Data.Battery.InstantPower)
			w.charts.Home.Add(now, w.homeP)
			w.charts.Grid.Add(now, snap.Data.Site.InstantPower)
			w.charts.BatteryPercent.Add(now, snap.BatteryPercent.Percentage)
		}
		if w.lines != nil {
			w.lines.QueueDraw()
		}
	})
}

func refreshDemo(state *statepkg.State, w *dashboardWidgets) {
	phase := float64(time.Now().Unix()%120) / 120 * 2 * math.Pi
	home := 2304.0 + math.Sin(phase)*512
	solar := math.Max(0, home*0.7)
	battery := home * 0.2
	grid := home - solar - battery
	glib.IdleAdd(func() {
		setError(w.error, "")
		setValue(w.siteName, "Home sweet home")
		setValue(w.energyVal, formatKwh(4_096_000, state.Prefs.ShowLessPrecision))
		setSmall(w.energyLbl, "ENERGY GENERATED")
		w.solarP, w.batteryP, w.homeP, w.gridP = solar, battery, home, grid
		w.batteryPercent = 80
		w.offGrid = true
		w.setOffGridImage(true)
		setValue(w.solarVal, formatKW(solar, state.Prefs.ShowLessPrecision))
		setSmall(w.solarLbl, "SOLAR")
		setValue(w.homeVal, formatKW(home, state.Prefs.ShowLessPrecision))
		setSmall(w.homeLbl, "HOME")
		w.runtime.record(battery, time.Now())
		w.stormWatch = false
		w.setBatteryLabels(battery, 80, 1, 0, state.Prefs)
		w.gridBaseLabel = "OFF-GRID"
		w.updateGridLabels(state.Prefs.ShowLessPrecision)
		setValue(w.vehicleVal, formatKW(512, state.Prefs.ShowLessPrecision)+" · 80%")
		setSmall(w.vehicleLbl, "VEHICLE")
		w.setBackground("home-charger.png")
		now := time.Now()
		w.charts.Solar.Add(now, solar)
		w.charts.Battery.Add(now, battery)
		w.charts.Home.Add(now, home)
		w.charts.Grid.Add(now, grid)
		w.charts.BatteryPercent.Add(now, 80)
		w.lines.QueueDraw()
	})
}

func refreshFleet(state *statepkg.State, w *dashboardWidgets) {
	tok, err := auth.LoadStoredToken()
	if err != nil {
		glib.IdleAdd(func() {
			setError(w.error, "")
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
	var siteInfo *api.SiteInfo
	if err == nil && w.siteInfoID != snap.SiteID {
		siteInfo, _ = client.FetchSiteInfo(snap.SiteID)
	}
	if err == nil {
		w.fetchConnectedVehicleData(client, snap.Status.WallConnectors, snap.Vehicles)
	}
	var solarToday *float64
	if err == nil && (w.solarTodaySiteID != snap.SiteID || time.Since(w.solarTodayAt) >= time.Minute) {
		if total, fetchErr := client.FetchSolarEnergyToday(snap.SiteID, time.Now(), time.Now().Location().String()); fetchErr == nil {
			solarToday = &total
		}
	}
	glib.IdleAdd(func() {
		if err != nil {
			setError(w.error, err.Error())
			return
		}
		setError(w.error, "")
		state.Prefs.CurrentEnergySiteIdx = idx
		w.siteCount = snap.SiteCount
		w.currentSiteID = snap.SiteID
		w.sites = append([]api.Product(nil), snap.Sites...)
		if w.graphs != nil {
			w.graphs.setFleetSite(snap.SiteID)
		}
		_ = storage.SavePrefs(state.Prefs)
		if siteInfo != nil {
			w.siteInfoID = snap.SiteID
			w.batteryCount = siteInfo.Response.BatteryCount
			w.backupReserve = siteInfo.Response.BackupReservePercent
			w.state.FirmwareVersion = siteInfo.Response.Version
			w.state.InstallationDate = siteInfo.Response.InstallationDate
		}
		if solarToday != nil {
			w.solarTodayWh = *solarToday
			w.solarTodaySiteID = snap.SiteID
			w.solarTodayAt = time.Now()
		}
		name := snap.SiteName
		if siteInfo != nil {
			for _, candidate := range []string{siteInfo.Response.DisplayName, siteInfo.Response.SiteDisplayName, siteInfo.Response.SiteName} {
				if candidate != "" {
					name = candidate
					break
				}
			}
		}
		if name == "" {
			name = "Energy Site"
		}
		setValue(w.siteName, name)
		if w.solarTodaySiteID == snap.SiteID {
			setValue(w.energyVal, formatKwh(w.solarTodayWh, state.Prefs.ShowLessPrecision))
			setSmall(w.energyLbl, "ENERGY GENERATED TODAY")
		} else {
			setValue(w.energyVal, "")
			setSmall(w.energyLbl, "")
		}
		w.solarP = snap.Status.SolarPower
		w.batteryP = snap.Status.BatteryPower
		w.batteryPercent = snap.Status.BatteryPercent
		w.offGrid = fleetIsOffGrid(snap.Status.GridStatus, snap.Status.IslandStatus)
		w.setOffGridImage(w.offGrid)
		connectorPower, connectorStatus := fleetWallConnectorSummary(snap.Status.WallConnectors)
		w.homeP = snap.Status.LoadPower - connectorPower
		w.gridP = snap.Status.GridPower
		setValue(w.solarVal, formatKW(snap.Status.SolarPower, state.Prefs.ShowLessPrecision))
		setSmall(w.solarLbl, "SOLAR")
		setValue(w.homeVal, formatKW(w.homeP, state.Prefs.ShowLessPrecision))
		setSmall(w.homeLbl, "HOME")
		w.runtime.record(snap.Status.BatteryPower, time.Now())
		w.stormWatch = snap.Status.StormModeActive
		w.setBatteryLabels(snap.Status.BatteryPower, snap.Status.BatteryPercent, w.batteryCount, w.backupReserve, state.Prefs)
		setValue(w.gridVal, formatKW(snap.Status.GridPower, state.Prefs.ShowLessPrecision))
		w.gridBaseLabel = fleetGridLabelText(snap.Status.GridStatus, snap.Status.IslandStatus)
		w.updateGridLabels(state.Prefs.ShowLessPrecision)
		if len(snap.Status.WallConnectors) > 0 {
			batteryLevel := w.connectedVehicleBattery(snap.Status.WallConnectors)
			if connectorStatus == "Charging" {
				text := formatKW(connectorPower, state.Prefs.ShowLessPrecision)
				if batteryLevel != nil {
					text += fmt.Sprintf(" · %.0f%%", *batteryLevel)
				}
				setValue(w.vehicleVal, text)
			} else {
				text := connectorStatus
				if batteryLevel != nil {
					text += fmt.Sprintf(" · %.0f%%", *batteryLevel)
				}
				setValue(w.vehicleVal, text)
			}
			vehicleLabel := "VEHICLE"
			if len(snap.Status.WallConnectors) > 1 {
				vehicleLabel = fmt.Sprintf("VEHICLES (%d)", len(snap.Status.WallConnectors))
			}
			setSmall(w.vehicleLbl, vehicleLabel)
		} else {
			setValue(w.vehicleVal, "")
			setSmall(w.vehicleLbl, "")
		}
		w.setBackground(selectFleetHomeBackground(state, connectorPower, connectorStatus, snap.Status.WallConnectors))

		if w.charts != nil {
			now := time.Now()
			w.charts.Solar.Add(now, snap.Status.SolarPower)
			w.charts.Battery.Add(now, snap.Status.BatteryPower)
			w.charts.Home.Add(now, w.homeP)
			w.charts.Grid.Add(now, snap.Status.GridPower)
			w.charts.BatteryPercent.Add(now, snap.Status.BatteryPercent)
		}
		if w.lines != nil {
			w.lines.QueueDraw()
		}
	})
	if w.scheduler != nil {
		w.scheduler.ApplyDue(state, snap.SiteID, time.Now())
	}
}

func (w *dashboardWidgets) fetchConnectedVehicleData(client *api.FleetClient, connectors []api.WallConnector, productVehicles []api.FleetVehicle) {
	now := time.Now()
	w.vehicleMu.Lock()
	if w.vehicleCacheGen != w.state.VehicleCacheGeneration {
		w.vehicleCharge = make(map[string]api.VehicleChargeState)
		w.vehicleFetchedAt = make(map[string]time.Time)
		w.vehicleAttemptedAt = make(map[string]time.Time)
		w.vehicleCacheGen = w.state.VehicleCacheGeneration
	}
	for _, vehicle := range productVehicles {
		w.vehicles[vehicle.VIN] = vehicle
	}
	missingKnownVehicle := false
	for _, connector := range connectors {
		if connector.VIN != "" {
			if _, known := w.vehicles[connector.VIN]; !known {
				missingKnownVehicle = true
			}
		}
	}
	shouldRefreshList := missingKnownVehicle && (w.vehicleListAt.IsZero() || now.Sub(w.vehicleListAt) >= 15*time.Minute)
	w.vehicleMu.Unlock()
	if shouldRefreshList {
		if vehicles, err := client.FetchVehicles(); err == nil {
			w.vehicleMu.Lock()
			for _, vehicle := range vehicles {
				w.vehicles[vehicle.VIN] = vehicle
			}
			w.vehicleListAt = now
			w.vehicleMu.Unlock()
		}
	}
	for _, connector := range connectors {
		if connector.VIN == "" || connector.WallConnectorState == nil || (*connector.WallConnectorState != 1 && *connector.WallConnectorState != 4) {
			continue
		}
		interval := time.Hour
		if *connector.WallConnectorState == 1 {
			interval = 3 * time.Minute
			w.state.Prefs.LastChargingVIN = connector.VIN
			_ = storage.SavePrefs(w.state.Prefs)
		}
		w.vehicleMu.Lock()
		_, knownVehicle := w.vehicles[connector.VIN]
		last := w.vehicleFetchedAt[connector.VIN]
		lastAttempt := w.vehicleAttemptedAt[connector.VIN]
		w.vehicleMu.Unlock()
		if !knownVehicle || (!last.IsZero() && now.Sub(last) < interval) || (!lastAttempt.IsZero() && now.Sub(lastAttempt) < 5*time.Minute) {
			continue
		}
		w.vehicleMu.Lock()
		w.vehicleAttemptedAt[connector.VIN] = now
		w.vehicleMu.Unlock()
		charge, err := client.FetchVehicleData(connector.VIN)
		if err != nil {
			var statusErr *api.HTTPStatusError
			if errors.As(err, &statusErr) && vehicleUnavailable(statusErr) {
				w.vehicleMu.Lock()
				delete(w.vehicleCharge, connector.VIN)
				w.vehicleFetchedAt[connector.VIN] = now
				w.vehicleMu.Unlock()
				w.persistVehicleChargeCache()
			}
			continue
		}
		w.vehicleMu.Lock()
		w.vehicleCharge[connector.VIN] = *charge
		w.vehicleFetchedAt[connector.VIN] = now
		w.vehicleMu.Unlock()
		w.persistVehicleChargeCache()
	}
}

func vehicleUnavailable(err *api.HTTPStatusError) bool {
	if err == nil || err.StatusCode != 408 {
		return false
	}
	body := strings.ToLower(string(err.Body))
	return strings.Contains(body, "offline") || strings.Contains(body, "asleep") || strings.Contains(body, "unavailable")
}

func (w *dashboardWidgets) persistVehicleChargeCache() {
	w.vehicleMu.Lock()
	cache := make(map[string]storage.VehicleChargeCacheEntry, len(w.vehicleFetchedAt))
	for vin, fetchedAt := range w.vehicleFetchedAt {
		charge := w.vehicleCharge[vin]
		cache[vin] = storage.VehicleChargeCacheEntry{BatteryLevel: charge.BatteryLevel, ChargingState: charge.ChargingState, MinutesToFullCharge: charge.MinutesToFullCharge, FetchedAt: fetchedAt}
	}
	w.vehicleMu.Unlock()
	_ = storage.SaveVehicleChargeCache(cache)
}

func (w *dashboardWidgets) connectedVehicleBattery(connectors []api.WallConnector) *float64 {
	w.vehicleMu.Lock()
	defer w.vehicleMu.Unlock()
	for _, connector := range connectors {
		if charge, ok := w.vehicleCharge[connector.VIN]; ok && charge.BatteryLevel != nil {
			value := *charge.BatteryLevel
			return &value
		}
	}
	return nil
}

func (w *dashboardWidgets) setBatteryLabels(watts, percentage, count, reserve float64, prefs storage.Prefs) {
	w.refreshCount++
	arrow := "·"
	if watts > 40 {
		arrow = "▼"
	} else if watts < -40 {
		arrow = "▲"
	}
	setValue(w.batteryVal, fmt.Sprintf("%s %s %s", formatKW(watts, prefs.ShowLessPrecision), arrow, formatPercent(percentage, prefs.ShowLessPrecision)))
	store := w.scheduleStore()
	label := batteryStatusLabel(count, store, w.stormWatch)
	average := w.runtime.average(watts, 40, time.Now())
	estimate := runtimeEstimateString(average, count, percentage, reserve, 40)
	if estimate != "" && (prefs.AlwaysShowRuntimeEstimate || w.refreshCount%2 == 0) {
		setSmall(w.batteryLbl, estimate)
		return
	}
	setBatteryStatus(w.batteryLbl, label, scheduleActiveNow(store, time.Now()))
}

func (w *dashboardWidgets) scheduleStore() schedule.Store {
	if w.scheduler == nil {
		return schedule.Store{}
	}
	return w.scheduler.Snapshot()
}

func batteryStatusLabel(count float64, store schedule.Store, stormWatch bool) string {
	label := fmt.Sprintf("POWERWALL%s", batteryCountSuffix(count))
	activeSchedules := 0
	for _, item := range store.Schedules {
		if item.Enabled {
			activeSchedules++
		}
	}
	if store.Enabled && activeSchedules > 0 {
		label += " · ⏰"
		if activeSchedules > 1 {
			label += fmt.Sprintf(" %d", activeSchedules)
		}
	}
	if stormWatch {
		label += " · ⚡"
	}
	return label
}

func scheduleActiveNow(store schedule.Store, now time.Time) bool {
	if !store.Enabled {
		return false
	}
	hour, minute, _ := now.Clock()
	current := hour*60 + minute
	for _, item := range store.Schedules {
		if !item.Enabled {
			continue
		}
		if item.StartMinutes == item.EndMinutes ||
			(item.StartMinutes < item.EndMinutes && current >= item.StartMinutes && current < item.EndMinutes) ||
			(item.StartMinutes > item.EndMinutes && (current >= item.StartMinutes || current < item.EndMinutes)) {
			return true
		}
	}
	return false
}

func setBatteryStatus(lbl *gtk.Label, text string, active bool) {
	scale := labelTextScale(lbl)
	escaped := glib.MarkupEscapeText(text)
	if active {
		escaped = strings.Replace(escaped, "⏰", "<span foreground='#30d158'>⏰</span>", 1)
	}
	lbl.SetMarkup(fmt.Sprintf("<span foreground='white' weight='bold' size='%d' alpha='70%%'>%s</span>", int(12*1024*scale), escaped))
}

func refreshElectricityMaps(state *statepkg.State, w *dashboardWidgets) {
	key, _ := storage.GetElectricityMapsAPIKey()
	if key == "" || state.Prefs.ElectricityMapsZone == "" {
		return
	}
	data, err := api.FetchElectricityMaps(key, state.Prefs.ElectricityMapsZone)
	if err != nil {
		return // Optional data must not replace live Powerwall errors.
	}
	glib.IdleAdd(func() {
		w.carbonIntensity = &data.CarbonIntensity
		renewable := clamp(100-data.FossilFuelPercentage, 0, 100)
		w.renewablePercent = &renewable
		w.updateGridLabels(state.Prefs.ShowLessPrecision)
		w.relayout()
	})
}

func (w *dashboardWidgets) updateGridLabels(lessPrecision bool) {
	if w.gridLbl == nil || w.gridVal == nil {
		return
	}
	label := w.gridBaseLabel
	if label == "" {
		label = "GRID"
	}
	if w.carbonIntensity != nil {
		label += fmt.Sprintf(" · %d gCO2", *w.carbonIntensity)
	}
	if w.offGrid {
		setSmallColored(w.gridLbl, label, "#ff9500", "100%")
	} else {
		setSmall(w.gridLbl, label)
	}
	setGridValue(w.gridVal, formatKW(w.gridP, lessPrecision), w.renewablePercent)
}

func setGridValue(lbl *gtk.Label, power string, renewables *float64) {
	scale := labelTextScale(lbl)
	if renewables == nil {
		setValue(lbl, power)
		return
	}
	color := "#34c759"
	if *renewables < 25 {
		color = "#a2845e"
	} else if *renewables < 50 {
		color = "#ff9500"
	} else if *renewables < 75 {
		color = "#ffd60a"
	}
	lbl.SetMarkup(fmt.Sprintf("<span foreground='white' weight='bold' size='%d'>%s · <span foreground='%s'>%.1f%%</span></span>", int(18*1024*scale), glib.MarkupEscapeText(power), color, *renewables))
}

func fleetWallConnectorSummary(connectors []api.WallConnector) (power float64, status string) {
	status = "Idle"
	for _, connector := range connectors {
		if connector.WallConnectorPower != nil {
			power += *connector.WallConnectorPower
		}
		if connector.WallConnectorState == nil {
			continue
		}
		switch *connector.WallConnectorState {
		case 1:
			status = "Charging"
		case 4:
			if status != "Charging" {
				status = "Plugged in"
			}
		}
	}
	return power, status
}

func fleetGridLabelText(gridStatus, islandStatus string) string {
	label := "GRID"
	switch islandStatus {
	case "off_grid", "off_grid_unintentional":
		label = "GRID DOWN"
	case "off_grid_intentional":
		label = "OFF-GRID"
	default:
		if gridStatus == "SystemIslandedActive" || gridStatus == "Inactive" {
			label = "OFF-GRID"
		}
	}
	return label
}

func fleetIsOffGrid(gridStatus, islandStatus string) bool {
	switch islandStatus {
	case "off_grid", "off_grid_intentional", "off_grid_unintentional":
		return true
	case "on_grid":
		return false
	default:
		return gridStatus == "SystemIslandedActive" || gridStatus == "Inactive"
	}
}

func (w *dashboardWidgets) cycleSite(delta int) {
	if w == nil || w.state == nil || w.state.Prefs.LoginMode != storage.LoginModeFleetAPI || w.siteCount < 2 {
		return
	}
	next := w.state.Prefs.CurrentEnergySiteIdx + delta
	if next < 0 {
		next = 0
	}
	if next >= w.siteCount {
		next = w.siteCount - 1
	}
	if next == w.state.Prefs.CurrentEnergySiteIdx {
		return
	}
	w.state.Prefs.CurrentEnergySiteIdx = next
	_ = storage.SavePrefs(w.state.Prefs)
	go refreshFleet(w.state, w)
}

func formatKW(watts float64, lessPrecision bool) string {
	kw := watts / 1000
	if lessPrecision {
		return fmt.Sprintf("%.1f kW", math.Abs(kw))
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
	scale := labelTextScale(lbl)
	lbl.SetMarkup(fmt.Sprintf("<span foreground='white' weight='bold' size='%d'>%s</span>", int(18*1024*scale), glib.MarkupEscapeText(text)))
}

func setSmall(lbl *gtk.Label, text string) {
	scale := labelTextScale(lbl)
	lbl.SetMarkup(fmt.Sprintf("<span foreground='white' weight='bold' size='%d' alpha='70%%'>%s</span>", int(12*1024*scale), glib.MarkupEscapeText(text)))
}

func setSmallColored(lbl *gtk.Label, text, color, alpha string) {
	scale := labelTextScale(lbl)
	lbl.SetMarkup(fmt.Sprintf("<span foreground='%s' weight='bold' size='%d' alpha='%s'>%s</span>", color, int(12*1024*scale), alpha, glib.MarkupEscapeText(text)))
}

func setError(lbl *gtk.Label, text string) {
	lbl.SetMarkup(fmt.Sprintf("<span foreground='#ff453a'>%s</span>", glib.MarkupEscapeText(text)))
}

var overlayLabelScales sync.Map

func labelTextScale(lbl *gtk.Label) float64 {
	if value, ok := overlayLabelScales.Load(lbl); ok {
		return value.(float64)
	}
	return 1
}

func setLabelTextScale(lbl *gtk.Label, scale float64, small bool) {
	overlayLabelScales.Store(lbl, scale)
	text, _ := lbl.GetText()
	if small {
		setSmall(lbl, text)
	} else {
		setValue(lbl, text)
	}
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

func selectFleetHomeBackground(state *statepkg.State, wallConnectorPower float64, wallConnectorStatus string, connectors []api.WallConnector) string {
	if len(connectors) == 0 {
		return "home.png"
	}
	active := wallConnectorPower > 10 || wallConnectorStatus == "Charging" || wallConnectorStatus == "Plugged in"
	if !active {
		return "home-charger-empty.png"
	}
	vin := state.Prefs.LastChargingVIN
	for _, connector := range connectors {
		if connector.VIN != "" && connector.WallConnectorState != nil && (*connector.WallConnectorState == 1 || *connector.WallConnectorState == 4) {
			vin = connector.VIN
			break
		}
	}
	if len(vin) == 17 && (vin[3] == 'C' || vin[3] == 'c') {
		return "home-charger-cybertruck.png"
	}
	return "home-charger.png"
}

func resolveBackgroundPath(name string) string {
	candidates := make([]string, 0, 8)
	if v := os.Getenv("POWERWALL_TV_IMAGES_DIR"); v != "" {
		candidates = append(candidates, filepath.Join(v, name))
	}
	candidates = append(candidates,
		filepath.Join("/app/share/powerwall-tv/images", name),
	)
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(cwd, "assets/images", name),
			filepath.Join(cwd, "../powerwall-tv/Powerwall-TV/Images", name),
		)
	}
	if exe, err := os.Executable(); err == nil {
		base := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(base, "../share/powerwall-tv/images", name),
			filepath.Join(base, "assets/images", name),
			filepath.Join(base, "../powerwall-tv/Powerwall-TV/Images", name),
		)
	}
	for _, p := range candidates {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	return ""
}

func (w *dashboardWidgets) setBackground(name string) {
	w.bgName = name
	if w.scene.width == 0 || w.scene.height == 0 {
		return
	}
	targetWidth, targetHeight := int(w.scene.width), int(w.scene.height)
	if w.background != nil && w.backgroundName == name && w.backgroundW == targetWidth && w.backgroundH == targetHeight {
		return
	}
	path := resolveBackgroundPath(name)
	if path == "" {
		return
	}
	// fill/cover (no bars)
	pix, err := gdk.PixbufNewFromFileAtScale(path, targetWidth, targetHeight, false)
	if err != nil {
		return
	}
	surface, err := gdk.CairoSurfaceCreateFromPixbuf(pix, 1, nil)
	if err != nil {
		return
	}
	if w.background != nil {
		w.background.Close()
	}
	w.background = surface
	w.backgroundName = name
	w.backgroundW = targetWidth
	w.backgroundH = targetHeight
	w.lines.QueueDraw()
}

func (w *dashboardWidgets) drawPowerLines(cr *cairo.Context) {
	if w.width == 0 || w.height == 0 {
		return
	}
	ww, wh := w.scene.width, w.scene.height
	cx, cy := w.scene.x+ww/2, w.scene.y+wh/2
	cr.SetSourceRGB(22.0/255, 23.0/255, 24.0/255)
	cr.Paint()
	if w.background != nil {
		progress := introEaseOut(w.introStarted, 0, 1400*time.Millisecond, time.Now())
		scale := 1.04 - 0.04*progress
		cr.Save()
		cr.Rectangle(w.scene.x, w.scene.y, ww, wh)
		cr.Clip()
		cr.Translate(cx, cy)
		cr.Scale(scale, scale)
		cr.Translate(-cx, -cy)
		cr.SetSourceSurface(w.background, w.scene.x, w.scene.y)
		cr.PaintWithAlpha(progress)
		cr.Restore()
	}
	cr.PushGroup()
	indicatorHeight := 0.076 * wh * clamp(w.batteryPercent/100, 0, 1)
	indicatorWidth := math.Max(4, 0.0024*ww)
	indicatorBottomX := cx + 0.014*ww
	indicatorBottomY := cy + 0.244*wh
	cr.SetSourceRGB(0.36, 0.82, 0.38)
	cr.Rectangle(indicatorBottomX-indicatorWidth/2, indicatorBottomY-indicatorHeight, indicatorWidth, indicatorHeight)
	cr.Fill()
	hw, hh := ww/2, wh/2

	type np2 struct{ nx, ny float64 }
	toAbs := func(p np2) (float64, float64) {
		return cx + p.nx*hw, cy + p.ny*hh
	}

	// All points are normalized relative to screen centre:
	// nx = (x - centerX) / halfWidth, ny = (y - centerY) / halfHeight
	// Converted from the original 1280x720 tuning points so anchors scale cleanly.
	solarN := np2{nx: -0.1171875, ny: -0.5}                    // (565,180)
	pwrN := np2{nx: -0.046875, ny: 0.5833333333333334}         // (610,570)
	junctionLTN := np2{nx: 0.15625, ny: -0.08333333333333333}  // (740,330)
	junctionRBN := np2{nx: 0.234375, ny: 0.011111111111111112} // (790,364)

	solarX, solarY := toAbs(solarN)
	pwrX, pwrY := toAbs(pwrN)

	// Virtual junction anchors used for flow geometry.
	junctionL, junctionT := toAbs(junctionLTN)
	junctionR, junctionB := toAbs(junctionRBN)
	junctionCx, junctionCy := (junctionL+junctionR)/2, (junctionT+junctionB)/2

	type p2 struct{ x, y float64 }
	baseLineWidth := 4.0 * math.Max(1, ww/designWidth)
	pulseLineWidth := 5.0 * math.Max(1, ww/designWidth)
	const pulseTrail = 1.0
	const gatewayTransitFrac = 0.3 / 2.2 // 0.3s through gateway over ~2.2s animation cycle

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
		prog := localPhase * (1 + trail)   // so tail reaches destination before reset
		head := math.Min(1, prog)
		tail := math.Max(0, prog-trail)
		if !forward {
			head = 1 - math.Min(1, prog)
			tail = 1 - math.Max(0, prog-trail)
		}

		// Convert t in [0..1] to point on polyline.
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

		// Draw pulse by tracing the polyline section from tail->head so bends are respected.
		segmentRange := func(t0, t1 float64) []p2 {
			t0 = math.Max(0, math.Min(1, t0))
			t1 = math.Max(0, math.Min(1, t1))
			if t1 < t0 {
				t0, t1 = t1, t0
			}
			if len(path) < 2 {
				return nil
			}

			startDist := t0 * total
			endDist := t1 * total
			acc := 0.0
			pts := make([]p2, 0, len(path)+2)
			pts = append(pts, pointAt(t0))

			for i := 0; i < len(lens); i++ {
				next := acc + lens[i]
				if next > startDist && next < endDist {
					pts = append(pts, path[i+1])
				}
				acc = next
			}
			pts = append(pts, pointAt(t1))
			return pts
		}

		pts := segmentRange(tail, head)
		if len(pts) < 2 {
			return
		}
		cr.SetLineWidth(pulseLineWidth)
		cr.SetSourceRGBA(r, g, b, 0.95)
		cr.MoveTo(pts[0].x, pts[0].y)
		for i := 1; i < len(pts); i++ {
			cr.LineTo(pts[i].x, pts[i].y)
		}
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
	// Grid path: choose direction from computed balance so sign-convention noise
	// from APIs can't flip direction/colour.
	// Apply a true ±20W deadband for grid animations in both directions,
	// keyed off live measured grid power so tiny values (e.g. -0.008 kW) never animate.
	if !w.offGrid && math.Abs(w.gridP) >= 20 {
		// pathGridToHub is defined from gateway-ish anchor down to grid anchor,
		// so "forward" means gateway -> grid (export).
		gridForward := w.gridP < 0
		gridR, gridG, gridB := 0.72, 0.72, 0.72
		if gridForward {
			gridR, gridG, gridB = exportColor[0], exportColor[1], exportColor[2]
		}
		// For export (gateway->grid), always wait for the solar->gateway HEAD
		// to arrive at the gateway before starting the gateway->grid pulse.
		gridPulseStart := 0.0
		if gridForward {
			gridPulseStart = headArrival(dSolar)
		}
		pulsePath(pathGridToHub, true, gridForward, gridR, gridG, gridB, gridPulseStart, dGrid)
	}

	// Gateway -> Home colour follows the biggest contributor to HOME (not raw source power).
	// Grid-supplied home flow is always grey.
	if homeDemand > 30 {
		r, g, b := 0.72, 0.72, 0.72
		sourceDur := dGrid
		homeIsSolar := false
		if solarToHome >= batteryToHome && solarToHome >= gridImport {
			r, g, b = 0.98, 0.82, 0.24
			sourceDur = dSolar
			homeIsSolar = true
		} else if batteryToHome >= solarToHome && batteryToHome >= gridImport {
			r, g, b = 0.36, 0.82, 0.38
			sourceDur = dPwr
		}
		// Default start: after source HEAD reaches gateway, then traverse gateway.
		homeStart := headArrival(sourceDur) + gatewayTransitFrac
		// If home flow is solar, start exactly when gateway->powerwall solar charging starts.
		if homeIsSolar {
			homeStart = headArrival(dSolar)
		}
		// Keep Gateway->Home speed equal to Powerwall->Gateway speed (px per cycle).
		homeDur := dPwr
		if lPwr > 0 {
			homeDur = dPwr * (lHome / lPwr)
		}
		pulsePath(pathHubToHome, true, true, r, g, b, homeStart, homeDur)
	}
	if w.offGrid && w.offGridSurface != nil {
		centerX := w.scene.x + (0.5+0.151)*w.scene.width
		centerY := w.scene.y + (0.5+0.234)*w.scene.height
		cr.SetSourceSurface(w.offGridSurface, centerX-float64(w.offGridWidth)/2, centerY-float64(w.offGridHeight)/2)
		cr.Paint()
	}
	cr.PopGroupToSource()
	cr.PaintWithAlpha(introEaseOut(w.introStarted, 1200*time.Millisecond, 700*time.Millisecond, time.Now()))
}

func introEaseOut(start time.Time, delay, duration time.Duration, now time.Time) float64 {
	if start.IsZero() || duration <= 0 {
		return 1
	}
	t := float64(now.Sub(start)-delay) / float64(duration)
	t = clamp(t, 0, 1)
	return 1 - math.Pow(1-t, 3)
}

// resize defers layout until GTK has finished the current allocation pass. The
// generation guard coalesces rapid resize events and avoids allocation loops.
func (w *dashboardWidgets) resize(width, height int) {
	if w == nil || width <= 0 || height <= 0 {
		return
	}
	if w.width == width && w.height == height {
		return
	}
	w.width = width
	w.height = height
	w.layoutGeneration++
	generation := w.layoutGeneration
	glib.IdleAdd(func() {
		if w.layoutGeneration == generation && w.width == width && w.height == height {
			w.relayout()
		}
	})
}

func (w *dashboardWidgets) relayout() {
	if w.fixed == nil {
		return
	}
	ww, wh := float64(w.width), float64(w.height)
	if ww == 0 || wh == 0 {
		return
	}
	stateScale := 1.0
	horizontalOffset := 0.0
	verticalOffset := 0.0
	// Layout preferences are read on every allocation so saving Settings takes
	// effect immediately without rebuilding the dashboard.
	if w.state != nil {
		stateScale = w.state.Prefs.SceneScale
		horizontalOffset = w.state.Prefs.SceneHorizontalOffset
		verticalOffset = w.state.Prefs.SceneVerticalOffset
	}
	rightContentBound := 0.34
	if w.renewablePercent != nil || w.carbonIntensity != nil {
		rightContentBound = 0.40
	}
	w.scene = calculateSceneRectForContent(ww, wh, stateScale, horizontalOffset, verticalOffset, rightContentBound)
	scale := w.scene.width / designWidth
	textScale := math.Max(1, scale)
	for _, label := range []*gtk.Label{w.siteName, w.energyVal, w.solarVal, w.homeVal, w.batteryVal, w.vehicleVal} {
		setLabelTextScale(label, textScale, false)
	}
	for _, label := range []*gtk.Label{w.energyLbl, w.solarLbl, w.homeLbl, w.vehicleLbl} {
		setLabelTextScale(label, textScale, true)
	}
	overlayLabelScales.Store(w.batteryLbl, textScale)
	batteryText, _ := w.batteryLbl.GetText()
	setBatteryStatus(w.batteryLbl, batteryText, scheduleActiveNow(w.scheduleStore(), time.Now()))
	overlayLabelScales.Store(w.gridVal, textScale)
	overlayLabelScales.Store(w.gridLbl, textScale)
	gridText, _ := w.gridVal.GetText()
	if gridText != "" || w.gridBaseLabel != "" {
		w.updateGridLabels(w.state != nil && w.state.Prefs.ShowLessPrecision)
	}
	for _, summary := range []*gtk.Label{w.siteName, w.energyVal, w.energyLbl, w.error} {
		summary.SetSizeRequest(int(260*textScale), -1)
	}
	for _, metric := range []*gtk.Label{w.solarVal, w.solarLbl, w.homeVal, w.homeLbl, w.batteryVal, w.batteryLbl, w.gridVal, w.gridLbl, w.vehicleVal, w.vehicleLbl} {
		metric.SetSizeRequest(int(240*textScale), -1)
	}
	offX, offY := w.scene.x, w.scene.y
	move := func(widget gtk.IWidget, x, y float64) {
		w.fixed.Move(widget, int(offX+x*scale), int(offY+y*scale))
	}
	w.positionMetrics()
	if path := resolveBackgroundPath("off-grid.png"); path != "" {
		iconWidth := int(0.052 * w.scene.width)
		iconHeight := int(0.056 * w.scene.height)
		if iconWidth != w.offGridWidth || iconHeight != w.offGridHeight {
			if pix, err := gdk.PixbufNewFromFileAtScale(path, iconWidth, iconHeight, false); err == nil {
				if surface, err := gdk.CairoSurfaceCreateFromPixbuf(pix, 1, nil); err == nil {
					if w.offGridSurface != nil {
						w.offGridSurface.Close()
					}
					w.offGridSurface = surface
					w.offGridWidth = iconWidth
					w.offGridHeight = iconHeight
				}
			}
		}
	}

	// macOS uses an inline summary centred at x=-0.30, with a 260-point
	// leading-aligned message column.
	summaryX := offX + (0.5-0.30)*w.scene.width - 130*scale
	summaryY := offY + (0.5-0.38)*w.scene.height
	if ww < designWidth {
		// The source detaches the summary from the scene in a narrow desktop
		// window so it remains reachable while the 1280px scene is cropped.
		summaryX = math.Max(24, offX+0.04*w.scene.width)
		summaryY = 24
	}
	move(w.siteName, (summaryX-offX)/scale, (summaryY-offY)/scale)
	move(w.metricBoxes[w.energyVal], (summaryX-offX)/scale, (summaryY-offY)/scale+summaryEnergyValueY)
	move(w.error, (summaryX-offX)/scale, (summaryY-offY)/scale+87)
	if w.bgName != "" {
		w.setBackground(w.bgName)
	}
	if w.lines != nil {
		w.lines.QueueDraw()
	}
}

// Keep each pair centered on its scene anchor; GTK owns the spacing inside it.
func (w *dashboardWidgets) positionMetrics() {
	if w.scene.width <= 0 || w.fixed == nil {
		return
	}
	place := func(value *gtk.Label, nx, ny float64) {
		box := w.metricBoxes[value]
		_, width := box.GetPreferredWidth()
		_, height := box.GetPreferredHeight()
		x := int(w.scene.x + (0.5+nx)*w.scene.width - float64(width)/2)
		y := int(w.scene.y + (0.5+ny)*w.scene.height - float64(height)/2)
		w.fixed.Move(box, x, y)
	}
	place(w.solarVal, 0.087, -0.40)
	place(w.homeVal, 0.25, -0.30)
	place(w.batteryVal, 0.03, 0.40)
	gridX := 0.25
	if w.renewablePercent != nil || w.carbonIntensity != nil {
		gridX = 0.28
	}
	place(w.gridVal, gridX, 0.40)
	place(w.vehicleVal, -0.104, -0.40)
}

func (w *dashboardWidgets) setOffGridImage(show bool) {
	w.offGrid = show
	if w.lines != nil {
		w.lines.QueueDraw()
	}
}

func (w *dashboardWidgets) setSummaryVisible(visible bool) {
	for _, widget := range []*gtk.Label{w.siteName, w.energyVal, w.energyLbl, w.error} {
		if visible {
			widget.Show()
		} else {
			widget.Hide()
		}
	}
}

func (w *dashboardWidgets) summaryOverlapsScene() bool {
	if w == nil || w.width >= int(designWidth) {
		return false // The wide layout keeps the summary inside the scene.
	}
	scale := w.scene.width / designWidth
	x := math.Max(24, w.scene.x+0.04*w.scene.width)
	return w.scene.intersects(x, 24, 260*scale, 115*scale)
}
