package render

import (
	"fmt"
	"image"
	"image/color"
	imagedraw "image/draw"
	"math"
	"strconv"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"

	"github.com/rubyattack3r/beaconcast/draw"
	"github.com/rubyattack3r/beaconcast/internal/fonts"
)

// ParseColor parses a #rgb or #rrggbb color.
func ParseColor(c draw.Color) (color.RGBA, error) {
	s := string(c)
	if len(s) == 4 && s[0] == '#' {
		s = "#" + string([]byte{s[1], s[1], s[2], s[2], s[3], s[3]})
	}
	if len(s) != 7 || s[0] != '#' {
		return color.RGBA{}, fmt.Errorf("invalid color %q", c)
	}
	v, err := strconv.ParseUint(s[1:], 16, 32)
	if err != nil {
		return color.RGBA{}, fmt.Errorf("invalid color %q", c)
	}
	return color.RGBA{uint8(v >> 16), uint8(v >> 8), uint8(v), 0xff}, nil
}

// rasterizer draws shapes onto an RGBA image.
type rasterizer struct {
	img *image.RGBA
}

func newCanvas(doc *draw.Document, background draw.Color) (*rasterizer, error) {
	bg, err := ParseColor(background)
	if err != nil {
		return nil, err
	}
	img := image.NewRGBA(image.Rect(0, 0, int(math.Ceil(doc.Width)), int(math.Ceil(doc.Height))))
	imagedraw.Draw(img, img.Bounds(), image.NewUniform(bg), image.Point{}, imagedraw.Src)
	return &rasterizer{img: img}, nil
}

func (r *rasterizer) draw(shape draw.Shape) error {
	switch s := shape.(type) {
	case draw.Rect:
		return r.rect(s)
	case draw.Text:
		return r.text(s)
	}
	return nil
}

func (r *rasterizer) blend(x, y int, c color.RGBA, coverage float64) {
	if coverage <= 0 {
		return
	}
	if coverage >= 1 {
		r.img.SetRGBA(x, y, c)
		return
	}
	dst := r.img.RGBAAt(x, y)
	mix := func(a, b uint8) uint8 { return uint8(float64(a)*(1-coverage) + float64(b)*coverage + 0.5) }
	r.img.SetRGBA(x, y, color.RGBA{mix(dst.R, c.R), mix(dst.G, c.G), mix(dst.B, c.B), 0xff})
}

// distance returns the signed distance from (px, py) to the rectangle's
// edge, negative inside, honoring rounded corners.
func distance(s draw.Rect, px, py float64) float64 {
	corners := s.RoundedCorners()
	radius := math.Min(s.Radius, math.Min(s.W, s.H)/2)
	if corners != 0 && radius > 0 {
		cx, cy := 0.0, 0.0
		var corner draw.Corners
		switch {
		case px < s.X+radius && py < s.Y+radius:
			corner, cx, cy = draw.TopLeft, s.X+radius, s.Y+radius
		case px > s.X+s.W-radius && py < s.Y+radius:
			corner, cx, cy = draw.TopRight, s.X+s.W-radius, s.Y+radius
		case px > s.X+s.W-radius && py > s.Y+s.H-radius:
			corner, cx, cy = draw.BottomRight, s.X+s.W-radius, s.Y+s.H-radius
		case px < s.X+radius && py > s.Y+s.H-radius:
			corner, cx, cy = draw.BottomLeft, s.X+radius, s.Y+s.H-radius
		}
		if corner != 0 && corners&corner != 0 {
			return math.Hypot(px-cx, py-cy) - radius
		}
	}
	dx := math.Max(s.X-px, px-(s.X+s.W))
	dy := math.Max(s.Y-py, py-(s.Y+s.H))
	return math.Max(dx, dy)
}

func clamp01(v float64) float64 { return math.Max(0, math.Min(1, v)) }

func (r *rasterizer) rect(s draw.Rect) error {
	var fill, stroke color.RGBA
	var err error
	if s.Fill != "" {
		if fill, err = ParseColor(s.Fill); err != nil {
			return err
		}
	}
	if s.Stroke != "" {
		if stroke, err = ParseColor(s.Stroke); err != nil {
			return err
		}
	}
	half := s.StrokeWidth / 2
	bounds := image.Rect(int(math.Floor(s.X-half-1)), int(math.Floor(s.Y-half-1)),
		int(math.Ceil(s.X+s.W+half+1)), int(math.Ceil(s.Y+s.H+half+1))).Intersect(r.img.Bounds())
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			d := distance(s, float64(x)+0.5, float64(y)+0.5)
			if s.Fill != "" {
				r.blend(x, y, fill, clamp01(0.5-d))
			}
			if s.Stroke != "" {
				outer := clamp01(0.5 - (d - half))
				inner := clamp01(0.5 - (d + half))
				r.blend(x, y, stroke, outer-inner)
			}
		}
	}
	return nil
}

func (r *rasterizer) text(t draw.Text) error {
	total := 0.0
	for _, run := range t.Runs {
		total += float64(len([]rune(run.Text))) * draw.Advance * t.Size
	}
	x := t.X
	switch t.Anchor {
	case draw.AnchorMiddle:
		x -= total / 2
	case draw.AnchorEnd:
		x -= total
	}
	for _, run := range t.Runs {
		face, err := fonts.Face(t.Size, run.Bold)
		if err != nil {
			return err
		}
		c, err := ParseColor(run.Color)
		if err != nil {
			return err
		}
		drawer := font.Drawer{Dst: r.img, Src: image.NewUniform(c), Face: face,
			Dot: fixed.Point26_6{X: fixed.Int26_6(x * 64), Y: fixed.Int26_6(t.Y * 64)}}
		drawer.DrawString(run.Text)
		advance := float64(len([]rune(run.Text))) * draw.Advance * t.Size
		if run.Underline {
			if err := r.rect(draw.Rect{X: x, Y: t.Y + 2, W: advance, H: 1, Fill: run.Color}); err != nil {
				return err
			}
		}
		x += advance
	}
	return nil
}

// Frame renders the document as it appears at time t.
func Frame(doc *draw.Document, t float64, background draw.Color) (*image.RGBA, error) {
	canvas, err := newCanvas(doc, background)
	if err != nil {
		return nil, err
	}
	for _, shape := range doc.At(t) {
		if err := canvas.draw(shape); err != nil {
			return nil, err
		}
	}
	return canvas.img, nil
}
