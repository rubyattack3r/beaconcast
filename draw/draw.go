// Package draw defines the shapes that the console layout produces and
// renderers consume.
//
// Every renderer draws the same shapes, so the animated SVG and the raster
// frames always show the same picture at the same moment.
package draw

import "unicode/utf8"

// Advance is the width of every glyph as a fraction of the font size. All
// renderers use JetBrains Mono, whose glyphs advance 600/1000 em.
const Advance = 0.6

// TextWidth returns the rendered width of text at the given font size.
func TextWidth(text string, size float64) float64 {
	return float64(utf8.RuneCountInString(text)) * Advance * size
}

// Color is a CSS hex color such as "#1c1c1c".
type Color string

// Corners selects which corners of a Rect are rounded.
type Corners uint8

const (
	TopLeft Corners = 1 << iota
	TopRight
	BottomRight
	BottomLeft
	AllCorners = TopLeft | TopRight | BottomRight | BottomLeft
)

// Anchor is the horizontal alignment of a Text relative to its X.
type Anchor uint8

const (
	AnchorStart Anchor = iota
	AnchorMiddle
	AnchorEnd
)

// Shape is a Rect or a Text.
type Shape interface{ isShape() }

// Rect is a filled and optionally stroked rectangle.
type Rect struct {
	X, Y, W, H  float64
	Fill        Color // empty for no fill
	Stroke      Color // empty for no stroke
	StrokeWidth float64
	Radius      float64
	Corners     Corners // rounded corners; zero means all when Radius > 0
}

// Run is a span of text with one style.
type Run struct {
	Text      string
	Color     Color
	Bold      bool
	Underline bool
}

// Text is a single line of monospaced text; Y is the baseline.
type Text struct {
	X, Y   float64
	Size   float64
	Anchor Anchor
	Runs   []Run
}

func (Rect) isShape() {}
func (Text) isShape() {}

// RoundedCorners returns the corners that Radius applies to.
func (r Rect) RoundedCorners() Corners {
	if r.Radius <= 0 {
		return 0
	}
	if r.Corners == 0 {
		return AllCorners
	}
	return r.Corners
}

// Element is a shape visible from Start until End, in seconds. Blinking
// elements alternate between visible and hidden every half second.
type Element struct {
	Shape Shape
	Start float64
	End   float64
	Blink bool
}

// Document is a complete animation: static shapes drawn first, then timed
// elements in order.
type Document struct {
	Width, Height float64
	Duration      float64
	Background    Color // suggested color behind the window's rounded corners
	Static        []Shape
	Timed         []Element
}

// Visible reports whether the element is drawn at time t.
func (e Element) Visible(t float64) bool {
	if t < e.Start || t >= e.End {
		return false
	}
	if e.Blink {
		return int((t-e.Start)*2)%2 == 0
	}
	return true
}

// At returns the shapes drawn at time t, in drawing order.
func (d *Document) At(t float64) []Shape {
	shapes := append([]Shape(nil), d.Static...)
	for _, element := range d.Timed {
		if element.Visible(t) {
			shapes = append(shapes, element.Shape)
		}
	}
	return shapes
}

// Text returns every string the document can display in the given weight,
// for font subsetting.
func (d *Document) Text(bold bool) string {
	var all []byte
	add := func(shape Shape) {
		if text, ok := shape.(Text); ok {
			for _, run := range text.Runs {
				if run.Bold == bold {
					all = append(all, run.Text...)
				}
			}
		}
	}
	for _, shape := range d.Static {
		add(shape)
	}
	for _, element := range d.Timed {
		add(element.Shape)
	}
	return string(all)
}

// Bounds returns the vertical extent of a shape. Text extends from its
// baseline by the font's ascent and descent.
func Bounds(shape Shape) (top, bottom float64) {
	switch s := shape.(type) {
	case Rect:
		return s.Y, s.Y + s.H
	case Text:
		return s.Y - s.Size*0.8, s.Y + s.Size*0.25
	}
	return 0, 0
}

// Shift returns the shape moved by dx and dy.
func Shift(shape Shape, dx, dy float64) Shape {
	switch s := shape.(type) {
	case Rect:
		s.X += dx
		s.Y += dy
		return s
	case Text:
		s.X += dx
		s.Y += dy
		return s
	}
	return shape
}
