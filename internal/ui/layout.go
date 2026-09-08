package ui

import "math"

const (
	designWidth         = 1280.0
	designHeight        = 720.0
	homeControlsPadding = 24
	summaryEnergyValueY = 30.0
	metricLineSpacing   = 4

	minSceneScale  = 0.80
	maxSceneScale  = 1.20
	minSceneOffset = -0.20
	maxSceneOffset = 0.20
)

func homeControlsPosition(availableHeight, controlsHeight int) (x, y int) {
	x = homeControlsPadding
	y = availableHeight - homeControlsPadding - controlsHeight
	if y < homeControlsPadding {
		y = homeControlsPadding
	}
	return x, y
}

type sceneRect struct {
	x, y, width, height float64
}

func (r sceneRect) intersects(x, y, width, height float64) bool {
	return r.width > 0 && r.height > 0 && width > 0 && height > 0 &&
		r.x < x+width && r.x+r.width > x && r.y < y+height && r.y+r.height > y
}

func clamp(value, low, high float64) float64 {
	return math.Min(high, math.Max(low, value))
}

// calculateSceneRect mirrors ContentView.sceneFrame in the source app. The
// scene never scales below its natural 1280x720 size; small windows crop it,
// while larger windows scale it uniformly. User offsets are relative to the
// available window, so they remain useful at every window size.
func calculateSceneRect(availableWidth, availableHeight, userScale, horizontalOffset, verticalOffset float64) sceneRect {
	return calculateSceneRectForContent(availableWidth, availableHeight, userScale, horizontalOffset, verticalOffset, 0.34)
}

func calculateSceneRectForContent(availableWidth, availableHeight, userScale, horizontalOffset, verticalOffset, rightContentBound float64) sceneRect {
	if availableWidth <= 0 || availableHeight <= 0 {
		return sceneRect{}
	}

	userScale = clamp(userScale, minSceneScale, maxSceneScale)
	horizontalOffset = clamp(horizontalOffset, minSceneOffset, maxSceneOffset)
	verticalOffset = clamp(verticalOffset, minSceneOffset, maxSceneOffset)

	fitScale := math.Max(1, math.Min(availableWidth/designWidth, availableHeight/designHeight))
	sceneWidth := designWidth * fitScale * userScale
	sceneHeight := designHeight * fitScale * userScale

	centeredX := (availableWidth - sceneWidth) / 2
	biasedX := centeredX - availableWidth*0.10
	rightContentBound = clamp(rightContentBound, 0, 0.5)
	rightContentX := sceneWidth * (0.5 + rightContentBound)
	keepRightVisibleX := availableWidth - rightContentX
	desiredX := math.Min(biasedX, keepRightVisibleX) + availableWidth*horizontalOffset
	lowerX := math.Min(0, availableWidth-sceneWidth)
	upperX := math.Max(0, availableWidth-sceneWidth)
	x := clamp(desiredX, lowerX, upperX)

	centeredY := (availableHeight - sceneHeight) / 2
	desiredY := centeredY + availableHeight*verticalOffset
	lowerY := math.Min(0, availableHeight-sceneHeight)
	upperY := math.Max(0, availableHeight-sceneHeight)
	y := clamp(desiredY, lowerY, upperY)

	return sceneRect{x: x, y: y, width: sceneWidth, height: sceneHeight}
}
