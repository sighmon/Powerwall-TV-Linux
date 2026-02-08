package ui

import (
	"fmt"
	"math"
	"time"

	"github.com/gotk3/gotk3/cairo"
	"github.com/gotk3/gotk3/gtk"
)

type point struct {
	t time.Time
	v float64
}

type TimeSeries struct {
	maxPoints int
	points    []point
}

func NewTimeSeries(max int) *TimeSeries {
	return &TimeSeries{maxPoints: max}
}

func (s *TimeSeries) Add(t time.Time, v float64) {
	s.points = append(s.points, point{t: t, v: v})
	if len(s.points) > s.maxPoints {
		s.points = s.points[len(s.points)-s.maxPoints:]
	}
}

func (s *TimeSeries) Values() []point {
	return s.points
}

type ChartsData struct {
	Solar          *TimeSeries
	Battery        *TimeSeries
	Home           *TimeSeries
	Grid           *TimeSeries
	BatteryPercent *TimeSeries
}

func NewChartsData() *ChartsData {
	return &ChartsData{
		Solar:          NewTimeSeries(240),
		Battery:        NewTimeSeries(240),
		Home:           NewTimeSeries(240),
		Grid:           NewTimeSeries(240),
		BatteryPercent: NewTimeSeries(240),
	}
}

type GraphView struct {
	area      *gtk.DrawingArea
	series    *TimeSeries
	color     [3]float64
	minY      float64
	maxY      float64
	autoRange bool
	xLabel    string
	yLabel    string
}

func NewGraphView(series *TimeSeries, minY, maxY float64, color [3]float64) (*GraphView, error) {
	area, _ := gtk.DrawingAreaNew()
	area.SetSizeRequest(600, 200)
	area.SetHExpand(true)
	area.SetVExpand(true)
	gv := &GraphView{area: area, series: series, color: color, minY: minY, maxY: maxY, xLabel: "Time", yLabel: "Value"}
	area.Connect("draw", func(_ *gtk.DrawingArea, cr *cairo.Context) {
		gv.draw(cr)
	})
	return gv, nil
}

func (g *GraphView) Widget() *gtk.DrawingArea { return g.area }

func (g *GraphView) QueueDraw() { g.area.QueueDraw() }

func (g *GraphView) SetLabels(xLabel, yLabel string) {
	g.xLabel = xLabel
	g.yLabel = yLabel
}

func (g *GraphView) SetAutoRange(v bool) {
	g.autoRange = v
}

func (g *GraphView) draw(cr *cairo.Context) {
	alloc := g.area.GetAllocation()
	w := float64(alloc.GetWidth())
	h := float64(alloc.GetHeight())

	// background (light, like SwiftUI charts)
	cr.SetSourceRGB(0.96, 0.96, 0.96)
	cr.Rectangle(0, 0, w, h)
	cr.Fill()

	// grid
	cr.SetSourceRGBA(0, 0, 0, 0.08)
	cr.SetLineWidth(1)
	for i := 0; i <= 4; i++ {
		y := 10 + (h-30)*float64(i)/4
		cr.MoveTo(40, y)
		cr.LineTo(w-10, y)
	}
	for i := 0; i <= 6; i++ {
		x := 40 + (w-60)*float64(i)/6
		cr.MoveTo(x, 10)
		cr.LineTo(x, h-20)
	}
	cr.Stroke()

	// axes
	cr.SetSourceRGBA(0, 0, 0, 0.2)
	cr.SetLineWidth(1)
	cr.MoveTo(40, 10)
	cr.LineTo(40, h-20)
	cr.LineTo(w-10, h-20)
	cr.Stroke()

	pts := g.series.Values()
	if len(pts) < 2 {
		return
	}

	minY := g.minY
	maxY := g.maxY
	if g.autoRange {
		dataMin := math.MaxFloat64
		dataMax := -math.MaxFloat64
		for _, p := range pts {
			if p.v < dataMin {
				dataMin = p.v
			}
			if p.v > dataMax {
				dataMax = p.v
			}
		}

		switch {
		case dataMin >= 0:
			// Positive-only: keep zero as lower extent.
			minY = 0
			span := dataMax - minY
			if span < 1 {
				span = 1
			}
			maxY = dataMax + span*0.1
		case dataMax <= 0:
			// Negative-only: keep zero as upper extent.
			maxY = 0
			span := maxY - dataMin
			if span < 1 {
				span = 1
			}
			minY = dataMin - span*0.1
		default:
			// Mixed: pad both ends.
			minY = dataMin
			maxY = dataMax
			span := maxY - minY
			if span < 1 {
				span = 1
			}
			pad := span * 0.1
			minY -= pad
			maxY += pad
		}
	}
	if maxY <= minY {
		minY = 0
		maxY = 100
	}

	// x range
	start := pts[0].t
	end := pts[len(pts)-1].t
	if end.Equal(start) {
		end = start.Add(time.Second)
	}

	plotW := w - 60
	plotH := h - 30

	// explicit zero line + zero tick label
	zeroY := 10 + plotH*(1-((0-minY)/(maxY-minY)))
	cr.SetSourceRGBA(0.55, 0.55, 0.55, 0.8)
	cr.SetLineWidth(1.2)
	cr.MoveTo(40, zeroY)
	cr.LineTo(w-10, zeroY)
	cr.Stroke()
	cr.SetSourceRGBA(0.35, 0.35, 0.35, 0.95)
	cr.SelectFontFace("Sans", cairo.FONT_SLANT_NORMAL, cairo.FONT_WEIGHT_BOLD)
	cr.SetFontSize(11)
	cr.MoveTo(16, zeroY+4)
	cr.ShowText("0")

	// axis tick labels
	cr.SetSourceRGBA(0, 0, 0, 0.5)
	cr.SelectFontFace("Sans", cairo.FONT_SLANT_NORMAL, cairo.FONT_WEIGHT_NORMAL)
	cr.SetFontSize(11)
	for i := 0; i <= 4; i++ {
		y := 10 + plotH*float64(i)/4
		val := maxY - (maxY-minY)*float64(i)/4
		if math.Abs(val) < 0.5 {
			continue // explicit zero label is drawn on zero line
		}
		cr.MoveTo(6, y+4)
		cr.ShowText(fmt.Sprintf("%.0f", val))
	}
	for i := 0; i <= 4; i++ {
		x := 40 + plotW*float64(i)/4
		t := start.Add(time.Duration(float64(end.Sub(start)) * float64(i) / 4))
		cr.MoveTo(x-12, h-6)
		cr.ShowText(t.Format("15:04"))
	}

	// Build render coords first.
	type xy struct{ x, y float64 }
	coords := make([]xy, 0, len(pts))
	for _, p := range pts {
		x := 40 + plotW*(p.t.Sub(start).Seconds()/end.Sub(start).Seconds())
		y := 10 + plotH*(1-((p.v-minY)/(maxY-minY)))
		if math.IsNaN(y) || math.IsInf(y, 0) {
			continue
		}
		coords = append(coords, xy{x: x, y: y})
	}
	if len(coords) < 2 {
		return
	}

	// area fill (anchor to 0 when in range; otherwise nearest edge)
	baselineVal := 0.0
	if baselineVal < minY {
		baselineVal = minY
	}
	if baselineVal > maxY {
		baselineVal = maxY
	}
	baselineY := 10 + plotH*(1-((baselineVal-minY)/(maxY-minY)))

	cr.SetSourceRGBA(g.color[0], g.color[1], g.color[2], 0.25)
	cr.MoveTo(coords[0].x, baselineY)
	for _, c := range coords {
		cr.LineTo(c.x, c.y)
	}
	cr.LineTo(coords[len(coords)-1].x, baselineY)
	cr.ClosePath()
	cr.Fill()

	// line (actual value, including negatives)
	cr.SetSourceRGB(g.color[0], g.color[1], g.color[2])
	cr.SetLineWidth(2)
	cr.MoveTo(coords[0].x, coords[0].y)
	for i := 1; i < len(coords); i++ {
		cr.LineTo(coords[i].x, coords[i].y)
	}
	cr.Stroke()
}
