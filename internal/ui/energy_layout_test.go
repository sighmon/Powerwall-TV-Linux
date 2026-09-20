package ui

import (
	"math"
	"testing"
)

func TestEnergyPathsKeepGeometryAcrossSceneSizes(t *testing.T) {
	base := calculateEnergyPaths(sceneRect{width: designWidth, height: designHeight})
	tests := []struct {
		name                       string
		width, height, scale, x, y float64
	}{
		{"cropped", 800, 600, 1, 0, 0},
		{"large", 1920, 1080, 1, 0, 0},
		{"4K", 3840, 2160, 1, 0, 0},
		{"ultrawide", 3440, 1440, 1, 0, 0},
		{"zoom and offsets", 1920, 1080, 1.2, 0.1, -0.1},
		{"zoom out", 1280, 720, 0.8, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scene := calculateSceneRect(tt.width, tt.height, tt.scale, tt.x, tt.y)
			got := calculateEnergyPaths(scene)
			scale := scene.width / designWidth
			for _, pair := range []struct {
				name      string
				got, want []energyPoint
			}{
				{"solar", got.solar, base.solar},
				{"powerwall", got.powerwall, base.powerwall},
				{"grid", got.grid, base.grid},
				{"home", got.home, base.home},
			} {
				for i, point := range pair.got {
					// Every vertex must retain its position relative to the background.
					x, y := (point.x-scene.x)/scale, (point.y-scene.y)/scale
					if math.Abs(x-pair.want[i].x) > 1e-9 || math.Abs(y-pair.want[i].y) > 1e-9 {
						t.Errorf("%s vertex %d moved in design coordinates: (%g, %g), want %+v", pair.name, i, x, y, pair.want[i])
					}
					if i > 0 {
						previous, basePrevious := pair.got[i-1], pair.want[i-1]
						angle := math.Atan2(point.y-previous.y, point.x-previous.x)
						baseAngle := math.Atan2(pair.want[i].y-basePrevious.y, pair.want[i].x-basePrevious.x)
						if math.Abs(angle-baseAngle) > 1e-9 {
							t.Errorf("%s segment %d changed angle: %g, want %g", pair.name, i, angle, baseAngle)
						}
					}
				}
			}
		})
	}
}
