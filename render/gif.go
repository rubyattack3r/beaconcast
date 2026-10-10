package render

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"io"
	"math"
	"sort"

	"github.com/rubyattack3r/beaconcast/draw"
)

// Limits on GIF encoding.
const (
	MaxFPS    = 50
	MaxFrames = 6000
)

type gifFrame struct {
	img   *image.RGBA // cropped to the region that changed
	same  []bool      // pixels in img that match the previous frame
	start int         // centiseconds
}

// GIF encodes the document as a looping GIF at the given frame rate.
// Identical consecutive frames are merged, and each frame stores only the
// region that changed since the previous one.
func GIF(w io.Writer, doc *draw.Document, fps int, background draw.Color) error {
	if fps < 1 || fps > MaxFPS {
		return fmt.Errorf("frame rate must be between 1 and %d", MaxFPS)
	}
	if frames := doc.Duration * float64(fps); frames > MaxFrames {
		return fmt.Errorf("GIF would need %.0f frames; the limit is %d (lower the frame rate)", frames, MaxFrames)
	}
	base, err := newCanvas(doc, background)
	if err != nil {
		return err
	}
	for _, shape := range doc.Static {
		if err := base.draw(shape); err != nil {
			return err
		}
	}

	var frames []gifFrame
	var previous *image.RGBA
	histogram := map[color.RGBA]int{}
	count := int(math.Ceil(doc.Duration * float64(fps)))
	for i := 0; i < count; i++ {
		t := float64(i) / float64(fps)
		canvas := &rasterizer{img: cloneRGBA(base.img)}
		for _, element := range doc.Timed {
			if element.Visible(t) {
				if err := canvas.draw(element.Shape); err != nil {
					return err
				}
			}
		}
		changed := canvas.img.Bounds()
		if previous != nil {
			changed = diffBounds(previous, canvas.img)
			if changed.Empty() {
				continue
			}
		}
		crop := cloneRGBA(canvas.img.SubImage(changed).(*image.RGBA))
		same := make([]bool, changed.Dx()*changed.Dy())
		for y := changed.Min.Y; y < changed.Max.Y; y++ {
			for x := changed.Min.X; x < changed.Max.X; x++ {
				c := crop.RGBAAt(x, y)
				if previous != nil && previous.RGBAAt(x, y) == c {
					same[(y-changed.Min.Y)*changed.Dx()+x-changed.Min.X] = true
					continue
				}
				histogram[c]++
			}
		}
		frames = append(frames, gifFrame{img: crop, same: same, start: int(math.Round(t * 100))})
		previous = canvas.img
	}

	// The last palette entry is transparent: unchanged pixels inside a
	// frame's changed region use it, which compresses far better.
	palette := buildPalette(histogram)
	opaque := palette
	transparent := uint8(len(palette))
	palette = append(palette, color.RGBA{})
	nearest := map[color.RGBA]uint8{}
	out := &gif.GIF{LoopCount: 0, Config: image.Config{
		ColorModel: palette, Width: base.img.Bounds().Dx(), Height: base.img.Bounds().Dy(),
	}}
	end := int(math.Round(doc.Duration * 100))
	for i, frame := range frames {
		next := end
		if i+1 < len(frames) {
			next = frames[i+1].start
		}
		bounds := frame.img.Bounds()
		paletted := image.NewPaletted(bounds, palette)
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				if frame.same[(y-bounds.Min.Y)*bounds.Dx()+x-bounds.Min.X] {
					paletted.SetColorIndex(x, y, transparent)
					continue
				}
				c := frame.img.RGBAAt(x, y)
				index, ok := nearest[c]
				if !ok {
					index = uint8(opaque.Index(c))
					nearest[c] = index
				}
				paletted.SetColorIndex(x, y, index)
			}
		}
		out.Image = append(out.Image, paletted)
		out.Delay = append(out.Delay, max(2, next-frame.start))
		out.Disposal = append(out.Disposal, gif.DisposalNone)
	}
	var buffer bytes.Buffer
	if err := gif.EncodeAll(&buffer, out); err != nil {
		return err
	}
	_, err = w.Write(buffer.Bytes())
	return err
}

func cloneRGBA(src *image.RGBA) *image.RGBA {
	dst := image.NewRGBA(src.Bounds())
	for y := src.Bounds().Min.Y; y < src.Bounds().Max.Y; y++ {
		copy(dst.Pix[dst.PixOffset(src.Bounds().Min.X, y):dst.PixOffset(src.Bounds().Max.X, y)],
			src.Pix[src.PixOffset(src.Bounds().Min.X, y):src.PixOffset(src.Bounds().Max.X, y)])
	}
	return dst
}

func diffBounds(a, b *image.RGBA) image.Rectangle {
	bounds := a.Bounds()
	minX, minY, maxX, maxY := bounds.Max.X, bounds.Max.Y, bounds.Min.X, bounds.Min.Y
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		row := a.PixOffset(bounds.Min.X, y)
		end := a.PixOffset(bounds.Max.X, y)
		if bytes.Equal(a.Pix[row:end], b.Pix[row:end]) {
			continue
		}
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			i := a.PixOffset(x, y)
			if !bytes.Equal(a.Pix[i:i+4], b.Pix[i:i+4]) {
				minX, maxX = min(minX, x), max(maxX, x+1)
				minY, maxY = min(minY, y), max(maxY, y+1)
			}
		}
	}
	if maxX <= minX {
		return image.Rectangle{}
	}
	return image.Rect(minX, minY, maxX, maxY)
}

// buildPalette keeps the 255 most frequent colors; the rest map to their
// nearest neighbor. Flat UI colors dominate, so text edges lose little.
func buildPalette(histogram map[color.RGBA]int) color.Palette {
	colors := make([]color.RGBA, 0, len(histogram))
	for c := range histogram {
		colors = append(colors, c)
	}
	sort.Slice(colors, func(i, j int) bool {
		if histogram[colors[i]] != histogram[colors[j]] {
			return histogram[colors[i]] > histogram[colors[j]]
		}
		a, b := colors[i], colors[j]
		return uint32(a.R)<<16|uint32(a.G)<<8|uint32(a.B) < uint32(b.R)<<16|uint32(b.G)<<8|uint32(b.B)
	})
	if len(colors) > 255 {
		colors = colors[:255]
	}
	palette := make(color.Palette, len(colors))
	for i, c := range colors {
		palette[i] = c
	}
	return palette
}
