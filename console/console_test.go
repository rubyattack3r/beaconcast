package console

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/rubyattack3r/beaconcast/draw"
	"github.com/rubyattack3r/beaconcast/scene"
)

func load(t *testing.T) *scene.Scene {
	t.Helper()
	sc, err := scene.Load("../examples/situational-awareness.toml")
	if err != nil {
		t.Fatal(err)
	}
	return sc
}

func TestStaysInBounds(t *testing.T) {
	sc := load(t)
	for _, mode := range []string{scene.ModeDark, scene.ModeLight} {
		sc.Mode = mode
		doc, err := Build(sc)
		if err != nil {
			t.Fatalf("%s: %v", mode, err)
		}
		check := func(shape draw.Shape) {
			switch s := shape.(type) {
			case draw.Rect:
				if s.X < 0 || s.Y < 0 || s.X+s.W > doc.Width+0.01 || s.Y+s.H > doc.Height+0.01 {
					t.Errorf("%s: rect outside window: %+v", mode, s)
				}
			case draw.Text:
				width := 0.0
				for _, r := range s.Runs {
					width += float64(utf8.RuneCountInString(r.Text)) * draw.Advance * s.Size
				}
				left := s.X
				switch s.Anchor {
				case draw.AnchorMiddle:
					left -= width / 2
				case draw.AnchorEnd:
					left -= width
				}
				if left < 0 || left+width > doc.Width || s.Y > doc.Height {
					t.Errorf("%s: text outside window: %+v", mode, s)
				}
			}
		}
		for _, shape := range doc.Static {
			check(shape)
		}
		for _, element := range doc.Timed {
			check(element.Shape)
			if element.Start < 0 || element.End > doc.Duration+1e-9 || element.End < element.Start {
				t.Errorf("%s: bad window %v-%v", mode, element.Start, element.End)
			}
		}
		if doc.Background == "" {
			t.Errorf("%s: no background color", mode)
		}
	}
}

func TestScrollKeepsNewestContentInView(t *testing.T) {
	var elements []draw.Element
	for i := 0; i < 10; i++ {
		y := 100 + float64(i)*20 + 15
		elements = append(elements, draw.Element{Shape: draw.Text{Y: y, Size: 14, Runs: []draw.Run{{Text: "x"}}}, Start: float64(i), End: 20})
	}
	scrolled := scroll(elements, 100, 100, 20, 20)
	visibleAt := func(t float64) []float64 {
		var ys []float64
		for _, e := range scrolled {
			if e.Visible(t) {
				ys = append(ys, e.Shape.(draw.Text).Y)
			}
		}
		return ys
	}
	if got := visibleAt(3.5); len(got) != 4 || got[0] != 115 {
		t.Fatalf("before overflow: %v", got)
	}
	got := visibleAt(15)
	if len(got) != 5 {
		t.Fatalf("after overflow: %d rows visible, want 5: %v", len(got), got)
	}
	for _, y := range got {
		if top, bottom := draw.Bounds(draw.Text{Y: y, Size: 14}); top < 100 || bottom > 200 {
			t.Fatalf("row at %v outside the view", y)
		}
	}
}

func TestHeightCapsDocuments(t *testing.T) {
	sc := load(t)
	for _, height := range []float64{0, 480} {
		sc.Height = height
		doc, err := Build(sc)
		if err != nil {
			t.Fatal(err)
		}
		want := height
		if want == 0 {
			want = DefaultHeight
		}
		if doc.Height > want {
			t.Errorf("height %v exceeds %v", doc.Height, want)
		}
	}
}

func TestCustomColumns(t *testing.T) {
	sc, err := scene.Parse(`
[session]
columns = ["user", "computer", "sleep", "last"]
extra = { sleep = "60s 10%" }

[[command]]
input = "sleep 60 10"
`)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := Build(sc)
	if err != nil {
		t.Fatal(err)
	}
	var text strings.Builder
	for _, shape := range doc.Static {
		if s, ok := shape.(draw.Text); ok {
			for _, r := range s.Runs {
				text.WriteString(r.Text + "|")
			}
		}
	}
	got := text.String()
	for _, want := range []string{"sleep|", "60s 10%|", "computer|"} {
		if !strings.Contains(got, want) {
			t.Errorf("session table lacks %q: %s", want, got)
		}
	}
	if strings.Contains(got, "listener|") {
		t.Errorf("session table shows an unlisted column: %s", got)
	}
}

// TestPaletteContrast guards against text drawn in its own background color,
// such as console text inheriting a light-mode input color.
func TestPaletteContrast(t *testing.T) {
	luminance := func(c draw.Color) float64 {
		var r, g, b int
		if _, err := fmt.Sscanf(string(c), "#%02x%02x%02x", &r, &g, &b); err != nil {
			t.Fatalf("invalid color %q: %v", c, err)
		}
		return 0.2126*float64(r) + 0.7152*float64(g) + 0.0722*float64(b)
	}
	for name, c := range map[string]csPalette{"dark": csDark, "light": csLight} {
		pairs := []struct {
			what   string
			fg, bg draw.Color
		}{
			{"console text", c.text, c.console}, {"echo", c.echo, c.console},
			{"info", c.info, c.console}, {"success", c.good, c.console}, {"error", c.bad, c.console},
			{"prompt", c.prompt, c.input}, {"typed input", c.inputText, c.input}, {"cursor", c.cursor, c.input},
			{"status text", c.statusText, c.statusBG}, {"status computer", c.statusAccent, c.statusBG},
			{"selected tab", c.tabText, c.tabActive}, {"other tab", c.tabIdleText, c.tabIdle},
			{"table heading", c.tableHeadText, c.tableHead}, {"session row", c.rowText, c.row},
			{"title", c.title, c.window},
		}
		for _, p := range pairs {
			if d := luminance(p.fg) - luminance(p.bg); d > -60 && d < 60 {
				t.Errorf("%s: %s %s on %s has too little contrast", name, p.what, p.fg, p.bg)
			}
		}
	}
}
