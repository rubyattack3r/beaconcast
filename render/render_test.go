package render

import (
	"bytes"
	"encoding/xml"
	"flag"
	"image"
	"image/draw"
	"image/gif"
	"image/png"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rubyattack3r/beaconcast/console"
	bdraw "github.com/rubyattack3r/beaconcast/draw"
	"github.com/rubyattack3r/beaconcast/scene"
)

var update = flag.Bool("update", false, "rewrite golden frames in testdata")

var modes = []string{scene.ModeDark, scene.ModeLight}

func example(t *testing.T, mode string) *bdraw.Document {
	t.Helper()
	sc, err := scene.Load("../examples/situational-awareness.toml")
	if err != nil {
		t.Fatal(err)
	}
	sc.Mode = mode
	doc, err := console.Build(sc)
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

func TestSVGIsDeterministicXML(t *testing.T) {
	for _, name := range modes {
		doc := example(t, name)
		first, err := SVG(doc)
		if err != nil {
			t.Fatal(err)
		}
		second, _ := SVG(doc)
		if !bytes.Equal(first, second) {
			t.Errorf("%s: SVG output differs between runs", name)
		}
		decoder := xml.NewDecoder(bytes.NewReader(first))
		for {
			if _, err := decoder.Token(); err == io.EOF {
				break
			} else if err != nil {
				t.Fatalf("%s: invalid XML: %v", name, err)
			}
		}
		if !strings.Contains(string(first), "@font-face") || !strings.Contains(string(first), "@keyframes") {
			t.Errorf("%s: SVG lacks embedded font or animation", name)
		}
	}
}

// TestGIFMatchesFrames replays the GIF and compares every frame with a
// directly rendered frame at the same time.
func TestGIFMatchesFrames(t *testing.T) {
	for _, name := range modes {
		doc := example(t, name)
		var buffer bytes.Buffer
		if err := GIF(&buffer, doc, 12, doc.Background); err != nil {
			t.Fatal(err)
		}
		decoded, err := gif.DecodeAll(&buffer)
		if err != nil {
			t.Fatal(err)
		}
		if decoded.LoopCount != 0 {
			t.Errorf("%s: GIF does not loop forever", name)
		}
		canvas := image.NewRGBA(image.Rect(0, 0, decoded.Config.Width, decoded.Config.Height))
		elapsed := 0
		total := 0
		for i, frame := range decoded.Image {
			draw.Draw(canvas, frame.Bounds(), frame, frame.Bounds().Min, draw.Over)
			want, err := Frame(doc, float64(elapsed)/100+0.001, doc.Background)
			if err != nil {
				t.Fatal(err)
			}
			if bad := mismatch(canvas, want); bad > 0.002 {
				t.Fatalf("%s: frame %d at %dcs differs in %.2f%% of pixels", name, i, elapsed, bad*100)
			}
			elapsed += decoded.Delay[i]
			total += decoded.Delay[i]
		}
		if want := int(math.Round(doc.Duration * 100)); total < want-2 || total > want+2 {
			t.Errorf("%s: GIF lasts %dcs, want %dcs", name, total, want)
		}
	}
}

// mismatch returns the fraction of pixels that differ noticeably. Palette
// quantization may shift rare antialiasing shades slightly.
func mismatch(a, b *image.RGBA) float64 {
	bad := 0
	for i := 0; i < len(a.Pix); i += 4 {
		for c := 0; c < 3; c++ {
			if d := int(a.Pix[i+c]) - int(b.Pix[i+c]); d > 24 || d < -24 {
				bad++
				break
			}
		}
	}
	return float64(bad) / float64(len(a.Pix)/4)
}

// TestGoldenFrames compares rendered frames with reviewed snapshots.
// Regenerate them with: go test ./render -update
func TestGoldenFrames(t *testing.T) {
	for _, name := range modes {
		doc := example(t, name)
		moments := []struct {
			label string
			at    float64
		}{{"typing", 2.2}, {"middle", doc.Duration / 2}, {"end", doc.Duration - 0.05}}
		for _, moment := range moments {
			got, err := Frame(doc, moment.at, doc.Background)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join("testdata", name+"-"+moment.label+".png")
			if *update {
				var buffer bytes.Buffer
				if err := png.Encode(&buffer, got); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, buffer.Bytes(), 0o644); err != nil {
					t.Fatal(err)
				}
				continue
			}
			file, err := os.Open(path)
			if err != nil {
				t.Fatalf("%v (run go test ./render -update)", err)
			}
			want, err := png.Decode(file)
			if closeErr := file.Close(); closeErr != nil {
				t.Fatal(closeErr)
			}
			if err != nil {
				t.Fatal(err)
			}
			wantRGBA := image.NewRGBA(want.Bounds())
			draw.Draw(wantRGBA, want.Bounds(), want, image.Point{}, draw.Src)
			if got.Bounds() != wantRGBA.Bounds() {
				t.Fatalf("%s: size %v, golden %v", path, got.Bounds(), wantRGBA.Bounds())
			}
			if bad := mismatch(got, wantRGBA); bad > 0.001 {
				t.Errorf("%s differs in %.2f%% of pixels", path, bad*100)
			}
		}
	}
}
