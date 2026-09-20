package ui

import "math"

type energyPoint struct{ x, y float64 }

type energyPaths struct {
	solar, powerwall, grid, home []energyPoint
}

// Build every anchor and adjustment in design coordinates, then apply the
// background's uniform scale and translation. Pixel tuning must scale too,
// otherwise the segments change angle as the scene grows.
func calculateEnergyPaths(scene sceneRect) energyPaths {
	type p2 = energyPoint
	ww, wh := designWidth, designHeight
	cx, cy := ww/2, wh/2
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

	baseLineWidth := 4.0

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
	pathPowerwallToHub[len(pathPowerwallToHub)-1].y += 2
	pathHubToHome = offsetPath(pathHubToHome, 0, -pathShift)
	// Align the home endpoint and the full grid path with the background.
	const homeGridDrop = 3.0
	pathHubToHome[len(pathHubToHome)-1].y += homeGridDrop
	pathGridToHub = offsetPath(pathGridToHub, 0, homeGridDrop)

	paths := energyPaths{pathSolarToHub, pathPowerwallToHub, pathGridToHub, pathHubToHome}
	scale := scene.width / designWidth
	for _, path := range [][]energyPoint{paths.solar, paths.powerwall, paths.grid, paths.home} {
		for i := range path {
			path[i].x = scene.x + path[i].x*scale
			path[i].y = scene.y + path[i].y*scale
		}
	}
	return paths
}
