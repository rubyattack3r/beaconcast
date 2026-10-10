// Package console lays out a scene as an animated client window: a session
// table, a tab strip, the Beacon console, a status bar, and the beacon> input
// line.
package console

import (
	"fmt"
	"math"
	"sort"

	"github.com/rubyattack3r/beaconcast/draw"
	"github.com/rubyattack3r/beaconcast/scene"
	"github.com/rubyattack3r/beaconcast/timeline"
)

// Build lays out a scene as a draw.Document. It fails when the layout would
// exceed MaxSize.
func Build(sc *scene.Scene) (*draw.Document, error) {
	doc := layout(sc, timeline.Build(sc))
	if doc.Width > MaxSize || doc.Height > MaxSize {
		return nil, fmt.Errorf("layout is %.0fx%.0f pixels; the limit is %dx%d (shorten long lines)",
			doc.Width, doc.Height, MaxSize, MaxSize)
	}
	return doc, nil
}

// MaxSize bounds the width and height of a laid-out document in pixels.
const MaxSize = 4096

// DefaultHeight is the window height above which content scrolls, unless the
// scene sets its own height.
const DefaultHeight = 640.0

func maxHeight(sc *scene.Scene) float64 {
	if sc.Height > 0 {
		return sc.Height
	}
	return DefaultHeight
}

// scroll keeps the newest content inside the view [viewTop, viewTop+height).
// Elements are laid out from viewTop as if the view were unbounded; whenever
// visible content extends below the view, everything moves up by the
// overflow, rounded up to a multiple of step when step is positive. Each
// element is repeated for every scroll position it is visible in, and copies
// that fall completely outside the view are dropped. With step set to the
// line height, lines are always either fully in view or gone.
func scroll(elements []draw.Element, viewTop, height, step, duration float64) []draw.Element {
	times := []float64{0}
	for _, e := range elements {
		times = append(times, e.Start, e.End)
	}
	sort.Float64s(times)
	type window struct{ start, end, offset float64 }
	var windows []window
	for i, t := range times {
		end := duration
		if i+1 < len(times) {
			end = min(end, times[i+1])
		}
		if end <= t || t >= duration {
			continue
		}
		extent := viewTop
		for _, e := range elements {
			if e.Start <= t && t < e.End {
				_, bottom := draw.Bounds(e.Shape)
				extent = max(extent, bottom)
			}
		}
		offset := max(0, extent-(viewTop+height))
		if step > 0 && offset > 0 {
			offset = math.Ceil(offset/step-1e-9) * step
		}
		if n := len(windows); n > 0 && windows[n-1].offset == offset {
			windows[n-1].end = end
			continue
		}
		windows = append(windows, window{t, end, offset})
	}
	var out []draw.Element
	for _, e := range elements {
		for _, w := range windows {
			start, end := max(e.Start, w.start), min(e.End, w.end)
			if end <= start {
				continue
			}
			shape := draw.Shift(e.Shape, 0, -w.offset)
			if top, bottom := draw.Bounds(shape); bottom <= viewTop || top >= viewTop+height {
				continue
			}
			out = append(out, draw.Element{Shape: shape, Start: start, End: end, Blink: e.Blink})
		}
	}
	return out
}

func textWidth(text string, size float64) float64 { return draw.TextWidth(text, size) }

func run(text string, color draw.Color) draw.Run { return draw.Run{Text: text, Color: color} }

func label(x, y, size float64, color draw.Color, text string) draw.Text {
	return draw.Text{X: x, Y: y, Size: size, Runs: []draw.Run{run(text, color)}}
}

// input describes how the command line being typed is drawn.
type input struct {
	X, Baseline float64 // where the first typed rune goes
	Size        float64
	Color       draw.Color
	Cursor      func(x float64) draw.Rect // cursor drawn before the rune at x
}

// typing returns the keystroke and cursor elements for one command. Typed
// runes stay visible until the command is entered.
func (in input) typing(command string, c timeline.Command) []draw.Element {
	var elements []draw.Element
	advance := draw.Advance * in.Size
	runes := []rune(command)
	for i, r := range runes {
		x := in.X + float64(i)*advance
		if r != ' ' {
			elements = append(elements, draw.Element{
				Shape: draw.Text{X: x, Y: in.Baseline, Size: in.Size, Runs: []draw.Run{run(string(r), in.Color)}},
				Start: c.Keys[i], End: c.Enter,
			})
		}
		next := c.Enter
		if i+1 < len(runes) {
			next = c.Keys[i+1]
		}
		elements = append(elements, draw.Element{Shape: in.Cursor(x + advance), Start: c.Keys[i], End: next})
	}
	return elements
}

// idle returns blinking cursors for the time the input line is empty.
func (in input) idle(tl *timeline.Timeline) []draw.Element {
	var elements []draw.Element
	from := 0.0
	for _, c := range tl.Commands {
		elements = append(elements, draw.Element{Shape: in.Cursor(in.X), Start: from, End: c.TypeStart, Blink: true})
		from = c.Enter
	}
	return append(elements, draw.Element{Shape: in.Cursor(in.X), Start: from, End: tl.Duration, Blink: true})
}

// lastCheckin returns a counter of the time since the last check-in, drawn
// at (x, y): it resets at every check-in and counts up once per second.
func lastCheckin(tl *timeline.Timeline, x, y, size float64, anchor draw.Anchor, color draw.Color, format func(seconds int) string) []draw.Element {
	var elements []draw.Element
	checkins := append([]float64(nil), tl.Checkins...)
	sort.Float64s(checkins)
	checkins = append(checkins, tl.Duration)
	for i := 0; i+1 < len(checkins); i++ {
		from, until := checkins[i], checkins[i+1]
		for second := 0; from+float64(second) < until; second++ {
			end := from + float64(second+1)
			if end > until {
				end = until
			}
			elements = append(elements, draw.Element{
				Shape: draw.Text{X: x, Y: y, Size: size, Anchor: anchor, Runs: []draw.Run{run(format(second), color)}},
				Start: from + float64(second), End: end,
			})
		}
	}
	return elements
}

// windowDots returns the three window-control dots.
func windowDots(cy float64) []draw.Shape {
	var shapes []draw.Shape
	for i, color := range []draw.Color{"#ff5f57", "#febc2e", "#28c840"} {
		x := 14 + float64(i)*20
		shapes = append(shapes, draw.Rect{X: x, Y: cy - 6, W: 12, H: 12, Fill: color, Radius: 6})
	}
	return shapes
}
