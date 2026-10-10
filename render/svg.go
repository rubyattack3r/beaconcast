// Package render draws documents as animated SVG, PNG frames, and GIFs.
package render

import (
	"encoding/base64"
	"fmt"
	"html"
	"strconv"
	"strings"

	"github.com/rubyattack3r/beaconcast/draw"
	"github.com/rubyattack3r/beaconcast/internal/fonts"
)

const fontStack = "BeaconcastMono, 'JetBrains Mono', monospace"

func num(v float64) string {
	return strconv.FormatFloat(float64(int64(v*100+copySign(0.5, v)))/100, 'f', -1, 64)
}

func copySign(magnitude, sign float64) float64 {
	if sign < 0 {
		return -magnitude
	}
	return magnitude
}

func rectSVG(r draw.Rect) string {
	fill := string(r.Fill)
	if fill == "" {
		fill = "none"
	}
	stroke := ""
	if r.Stroke != "" {
		stroke = fmt.Sprintf(` stroke="%s" stroke-width="%s"`, r.Stroke, num(r.StrokeWidth))
	}
	corners := r.RoundedCorners()
	switch corners {
	case 0:
		return fmt.Sprintf(`<rect x="%s" y="%s" width="%s" height="%s" fill="%s"%s/>`,
			num(r.X), num(r.Y), num(r.W), num(r.H), fill, stroke)
	case draw.AllCorners:
		return fmt.Sprintf(`<rect x="%s" y="%s" width="%s" height="%s" rx="%s" fill="%s"%s/>`,
			num(r.X), num(r.Y), num(r.W), num(r.H), num(r.Radius), fill, stroke)
	}
	radius := func(c draw.Corners) float64 {
		if corners&c != 0 {
			return r.Radius
		}
		return 0
	}
	tl, tr, br, bl := radius(draw.TopLeft), radius(draw.TopRight), radius(draw.BottomRight), radius(draw.BottomLeft)
	x, y, w, h := r.X, r.Y, r.W, r.H
	var d strings.Builder
	fmt.Fprintf(&d, "M%s %sH%s", num(x+tl), num(y), num(x+w-tr))
	if tr > 0 {
		fmt.Fprintf(&d, "A%s %s 0 0 1 %s %s", num(tr), num(tr), num(x+w), num(y+tr))
	}
	fmt.Fprintf(&d, "V%s", num(y+h-br))
	if br > 0 {
		fmt.Fprintf(&d, "A%s %s 0 0 1 %s %s", num(br), num(br), num(x+w-br), num(y+h))
	}
	fmt.Fprintf(&d, "H%s", num(x+bl))
	if bl > 0 {
		fmt.Fprintf(&d, "A%s %s 0 0 1 %s %s", num(bl), num(bl), num(x), num(y+h-bl))
	}
	fmt.Fprintf(&d, "V%s", num(y+tl))
	if tl > 0 {
		fmt.Fprintf(&d, "A%s %s 0 0 1 %s %s", num(tl), num(tl), num(x+tl), num(y))
	}
	d.WriteString("Z")
	return fmt.Sprintf(`<path d="%s" fill="%s"%s/>`, d.String(), fill, stroke)
}

func textSVG(t draw.Text) string {
	var b strings.Builder
	anchor := ""
	switch t.Anchor {
	case draw.AnchorMiddle:
		anchor = ` text-anchor="middle"`
	case draw.AnchorEnd:
		anchor = ` text-anchor="end"`
	}
	fmt.Fprintf(&b, `<text x="%s" y="%s" font-size="%s"%s>`, num(t.X), num(t.Y), num(t.Size), anchor)
	for _, r := range t.Runs {
		fmt.Fprintf(&b, `<tspan fill="%s"`, r.Color)
		if r.Bold {
			b.WriteString(` font-weight="700"`)
		}
		if r.Underline {
			b.WriteString(` text-decoration="underline"`)
		}
		// Browsers collapse repeated spaces in SVG text; no-break spaces
		// keep the monospaced columns aligned.
		fmt.Fprintf(&b, ">%s</tspan>", strings.ReplaceAll(html.EscapeString(r.Text), " ", "\u00a0"))
	}
	b.WriteString("</text>")
	return b.String()
}

func shapeSVG(shape draw.Shape) string {
	switch s := shape.(type) {
	case draw.Rect:
		return rectSVG(s)
	case draw.Text:
		return textSVG(s)
	}
	return ""
}

func header(doc *draw.Document) string {
	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%s" height="%s" viewBox="0 0 %s %s" font-family="%s">`,
		num(doc.Width), num(doc.Height), num(doc.Width), num(doc.Height), fontStack)
}

func fontFaces(doc *draw.Document) (string, error) {
	var b strings.Builder
	b.WriteString("/* Font: subset of JetBrains Mono, Copyright 2020 The JetBrains Mono Project Authors, SIL Open Font License 1.1. */")
	for _, bold := range []bool{false, true} {
		text := doc.Text(bold)
		if bold && text == "" {
			continue
		}
		woff, err := fonts.WOFF(text, bold)
		if err != nil {
			return "", err
		}
		weight := 400
		if bold {
			weight = 700
		}
		fmt.Fprintf(&b, "@font-face{font-family:BeaconcastMono;font-weight:%d;src:url(data:font/woff;base64,%s) format('woff')}",
			weight, base64.StdEncoding.EncodeToString(woff))
	}
	return b.String(), nil
}

// SVG renders the document as an animated SVG. Each timed element gets a CSS
// animation that shows it during its window once per loop.
func SVG(doc *draw.Document) ([]byte, error) {
	faces, err := fontFaces(doc)
	if err != nil {
		return nil, err
	}
	type window struct{ start, end string }
	names := map[window]string{}
	var rules, body strings.Builder
	percent := func(t float64) string {
		return strconv.FormatFloat(float64(int64(t/doc.Duration*100000+0.5))/1000, 'f', -1, 64)
	}
	for _, element := range doc.Timed {
		svg := shapeSVG(element.Shape)
		if element.Blink {
			svg = `<g class="blink">` + svg + "</g>"
		}
		if element.Start <= 0 && element.End >= doc.Duration {
			body.WriteString(svg + "\n")
			continue
		}
		w := window{percent(element.Start), percent(element.End)}
		name, ok := names[w]
		if !ok {
			name = "k" + strconv.Itoa(len(names))
			names[w] = name
			fmt.Fprintf(&rules, "@keyframes %s{", name)
			if element.Start > 0 {
				rules.WriteString("0%{opacity:0}")
			}
			fmt.Fprintf(&rules, "%s%%{opacity:1}", w.start)
			if element.End < doc.Duration {
				fmt.Fprintf(&rules, "%s%%{opacity:0}100%%{opacity:0}", w.end)
			} else {
				rules.WriteString("100%{opacity:1}")
			}
			fmt.Fprintf(&rules, "}.%s{opacity:0;animation:%s %ss step-end infinite}", name, name, num(doc.Duration))
		}
		fmt.Fprintf(&body, "<g class=\"%s\">%s</g>\n", name, svg)
	}

	var out strings.Builder
	out.WriteString(header(doc) + "\n<style>")
	out.WriteString(faces)
	out.WriteString("@keyframes blink{0%{opacity:1}50%{opacity:0}}.blink{animation:blink 1s step-end infinite}")
	out.WriteString(rules.String())
	out.WriteString("</style>\n")
	for _, shape := range doc.Static {
		out.WriteString(shapeSVG(shape) + "\n")
	}
	out.WriteString(body.String())
	out.WriteString("</svg>\n")
	return []byte(out.String()), nil
}

// StillSVG renders the document as it appears at time t, without animation.
func StillSVG(doc *draw.Document, t float64) ([]byte, error) {
	faces, err := fontFaces(doc)
	if err != nil {
		return nil, err
	}
	var out strings.Builder
	out.WriteString(header(doc) + "\n<style>" + faces + "</style>\n")
	for _, shape := range doc.At(t) {
		out.WriteString(shapeSVG(shape) + "\n")
	}
	out.WriteString("</svg>\n")
	return []byte(out.String()), nil
}
