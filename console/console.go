package console

import (
	"fmt"
	"strconv"

	"github.com/rubyattack3r/beaconcast/draw"
	"github.com/rubyattack3r/beaconcast/scene"
	"github.com/rubyattack3r/beaconcast/timeline"
)

// csPalette holds the colors of one color mode.
type csPalette struct {
	window, title, tableBG, tableHead, tableHeadText, row, rowText, split,
	tabsBG, tabActive, tabText, tabIdle, tabIdleText, console, text, info, good,
	bad, echo, statusBG, statusText, statusAccent, input, inputText, prompt,
	cursor, icon, page draw.Color
}

var csDark = csPalette{
	window: "#1c1c1c", title: "#8a8a8a", tableBG: "#141414", tableHead: "#262626",
	tableHeadText: "#b0b0b0", row: "#1f3f66", rowText: "#ffffff", split: "#2e2e2e",
	tabsBG: "#232323", tabActive: "#000000", tabText: "#e0e0e0", tabIdle: "#34393f", tabIdleText: "#b0b0b0",
	console: "#000000", text: "#d0d0d0", info: "#5fafff", good: "#5fd75f", bad: "#ff5f5f",
	statusBG: "#2b2b2b", statusText: "#d0d0d0", statusAccent: "#5fafff",
	echo: "#ffffff", input: "#000000", inputText: "#d0d0d0", prompt: "#ffffff",
	cursor: "#d0d0d0", icon: "#9fd3ff", page: "#0d1117",
}

var csLight = csPalette{
	window: "#e6e6e6", title: "#555555", tableBG: "#ffffff", tableHead: "#dcdcdc",
	tableHeadText: "#333333", row: "#b8cfe5", rowText: "#000000", split: "#cccccc",
	// The console stays dark in both modes, as in the client's light theme.
	tabsBG: "#d9d9d9", tabActive: "#ffffff", tabText: "#111111", tabIdle: "#c9d6e6", tabIdleText: "#333333",
	console: csDark.console, text: csDark.text, info: csDark.info, good: csDark.good, bad: csDark.bad,
	statusBG: "#e2e2e2", statusText: "#1c1c1c", statusAccent: "#1f5fa0",
	echo: csDark.echo, input: "#ffffff", inputText: "#000000", prompt: "#000000", cursor: "#1c1c1c", icon: "#3a6ea5", page: "#ffffff",
}

func csPaletteFor(sc *scene.Scene) csPalette {
	if sc.Mode == scene.ModeLight {
		return csLight
	}
	return csDark
}

const (
	csFont       = 14.0
	csSmall      = 13.0
	csLine       = 20.0
	csPad        = 16.0
	csTitleH     = 34.0
	csRowH       = 24.0
	csTableH     = csRowH*2 + 18
	csSplitH     = 6.0
	csTabsH      = 30.0
	csConsolePad = 10.0
	csStatusH    = 24.0
	csInputH     = 32.0
	csMinWidth   = 1000.0
)

// consoleLine is one rendered console line and when it appears.
type consoleLine struct {
	runs  []draw.Run
	start float64
}

func consoleLines(sc *scene.Scene, tl *timeline.Timeline) []consoleLine {
	c := csPaletteFor(sc)
	var lines []consoleLine
	add := func(at float64, runs ...draw.Run) { lines = append(lines, consoleLine{runs, at}) }
	for i, command := range sc.Commands {
		times := tl.Commands[i]
		add(times.Enter,
			draw.Run{Text: "beacon", Color: c.echo, Underline: true},
			run("> "+command.Input, c.echo))
		for j, event := range command.Events {
			at := times.Events[j]
			switch event.Kind {
			case scene.KindTask, scene.KindInfo:
				for _, line := range event.Lines() {
					add(at, run("[*] ", c.info), run(line, c.text))
				}
			case scene.KindCheckin:
				text := "host called home"
				if event.Bytes > 0 {
					text += fmt.Sprintf(", sent: %d bytes", event.Bytes)
				}
				add(at, run("[+] ", c.good), run(text, c.text))
			case scene.KindOutput:
				add(at, run("[+] ", c.good), run("received output:", c.text))
				for _, line := range event.Lines() {
					add(at, run(line, c.text))
				}
			case scene.KindError:
				for _, line := range event.Lines() {
					add(at, run("[-] ", c.bad), run(line, c.text))
				}
			}
		}
	}
	return lines
}

type csColumn struct {
	heading string
	width   float64
	value   string
}

// csMinWidths keeps the default table's familiar proportions.
var csMinWidths = map[string]float64{
	"external": 120, "internal": 110, "listener": 86, "user": 80, "computer": 92,
	"note": 112, "process": 130, "pid": 64, "arch": 56, "last": 80,
}

// sessionColumns returns the session table columns in scene order. Known
// columns show session values; any other column shows its [session.extra]
// value. The "last" column is filled in over time.
func sessionColumns(s scene.Session) []csColumn {
	values := map[string]string{
		"external": s.External, "internal": s.Internal, "listener": s.Listener,
		"user": s.User, "computer": s.Computer, "note": s.Note,
		"process": s.Process, "pid": strconv.Itoa(s.PID), "arch": s.Arch,
	}
	var columns []csColumn
	for _, key := range s.Columns {
		value, known := values[key]
		if !known {
			value = s.Extra[key]
		}
		heading := key
		if key == "internal" {
			heading += " \u25b2" // the client sorts sessions by internal address
		}
		width := max(csMinWidths[key], textWidth(heading, csSmall)+24, textWidth(value, csSmall)+24)
		if key == "last" {
			value = ""
		}
		columns = append(columns, csColumn{heading, width, value})
	}
	return columns
}

func layout(sc *scene.Scene, tl *timeline.Timeline) *draw.Document {
	c := csPaletteFor(sc)
	s := sc.Session
	columns := sessionColumns(s)
	lines := consoleLines(sc, tl)

	width := csMinWidth
	tableWidth := csPad + 8
	for _, column := range columns {
		tableWidth += column.width
	}
	width = max(width, tableWidth+csPad)
	for _, line := range lines {
		total := 0.0
		for _, r := range line.runs {
			total += textWidth(r.Text, csFont)
		}
		width = max(width, total+2*csPad)
	}
	for _, command := range sc.Commands {
		width = max(width, textWidth("beacon> "+command.Input, csFont)+2*csPad+draw.Advance*csFont)
	}

	tableTop := csTitleH
	tabsTop := tableTop + csTableH + csSplitH
	consoleTop := tabsTop + csTabsH
	chromeH := tabsTop + csTabsH + csStatusH + csInputH
	rows := len(lines) + 1
	if fit := int((maxHeight(sc) - chromeH - 2*csConsolePad) / csLine); rows > fit {
		rows = max(fit, 3)
	}
	consoleH := float64(rows)*csLine + 2*csConsolePad
	statusTop := consoleTop + consoleH
	inputTop := statusTop + csStatusH
	height := inputTop + csInputH

	static := []draw.Shape{
		draw.Rect{W: width, H: height, Fill: c.window, Radius: 10},
	}
	static = append(static, windowDots(17)...)
	if sc.Title != "" {
		static = append(static, draw.Text{X: width / 2, Y: 22, Size: csSmall, Anchor: draw.AnchorMiddle, Runs: []draw.Run{run(sc.Title, c.title)}})
	}
	static = append(static,
		draw.Rect{Y: tableTop, W: width, H: csTableH, Fill: c.tableBG},
		draw.Rect{Y: tableTop, W: width, H: csRowH, Fill: c.tableHead},
		draw.Rect{Y: tableTop + csRowH, W: width, H: csRowH, Fill: c.row},
		draw.Rect{X: csPad - 8, Y: tableTop + csRowH + 7, W: 12, H: 9, Radius: 1, Stroke: c.icon, StrokeWidth: 1.5},
	)
	x := csPad + 8
	lastX, hasLast := 0.0, false
	for _, column := range columns {
		static = append(static, label(x, tableTop+16, csSmall, c.tableHeadText, column.heading))
		if column.value != "" {
			static = append(static, label(x, tableTop+csRowH+16, csSmall, c.rowText, column.value))
		}
		if column.heading == "last" {
			lastX, hasLast = x, true
		}
		x += column.width
	}

	tab := fmt.Sprintf("Beacon %s@%d  X", s.Internal, s.PID)
	idleTab := "Event Log  X"
	static = append(static,
		draw.Rect{Y: tableTop + csTableH, W: width, H: csSplitH, Fill: c.split},
		draw.Rect{Y: tabsTop, W: width, H: csTabsH, Fill: c.tabsBG},
		draw.Rect{X: csPad - 8, Y: tabsTop + 4, W: textWidth(idleTab, csSmall) + 24, H: csTabsH - 4, Fill: c.tabIdle, Radius: 4, Corners: draw.TopLeft | draw.TopRight},
		label(csPad+4, tabsTop+20, csSmall, c.tabIdleText, idleTab),
		draw.Rect{X: csPad + 118, Y: tabsTop + 4, W: textWidth(tab, csSmall) + 24, H: csTabsH - 4, Fill: c.tabActive, Radius: 4, Corners: draw.TopLeft | draw.TopRight},
		label(csPad+130, tabsTop+20, csSmall, c.tabText, tab),
		draw.Rect{Y: consoleTop, W: width, H: consoleH, Fill: c.console},
		draw.Rect{Y: statusTop, W: width, H: csStatusH, Fill: c.statusBG},
		draw.Text{X: csPad, Y: statusTop + 17, Size: csSmall, Runs: statusRuns(c, s)},
		draw.Rect{Y: inputTop, W: width, H: csInputH, Fill: c.input, Radius: 10, Corners: draw.BottomLeft | draw.BottomRight},
	)
	baseline := inputTop + csInputH/2 + 5
	static = append(static, draw.Text{X: csPad, Y: baseline, Size: csFont, Runs: []draw.Run{
		{Text: "beacon", Color: c.prompt, Underline: true}, run("> ", c.prompt),
	}})

	in := input{
		X: csPad + textWidth("beacon> ", csFont), Baseline: baseline, Size: csFont, Color: c.inputText,
		Cursor: func(x float64) draw.Rect {
			return draw.Rect{X: x, Y: inputTop + 8, W: draw.Advance * csFont, H: csInputH - 16, Fill: c.cursor}
		},
	}
	var timed []draw.Element
	timed = append(timed, in.idle(tl)...)
	for i, command := range sc.Commands {
		timed = append(timed, in.typing(command.Input, tl.Commands[i])...)
	}
	var console []draw.Element
	for i, line := range lines {
		y := consoleTop + csConsolePad + float64(i)*csLine + 15
		console = append(console, draw.Element{
			Shape: draw.Text{X: csPad, Y: y, Size: csFont, Runs: line.runs},
			Start: line.start, End: tl.Duration,
		})
	}
	timed = append(timed, scroll(console, consoleTop+csConsolePad, float64(rows)*csLine, csLine, tl.Duration)...)
	timed = append(timed, lastCheckin(tl, width-csPad, statusTop+17, csSmall, draw.AnchorEnd, c.statusText, func(seconds int) string {
		return "last:" + lastLabel(seconds)
	})...)
	if hasLast {
		timed = append(timed, lastCheckin(tl, lastX, tableTop+csRowH+16, csSmall, draw.AnchorStart, c.rowText, lastLabel)...)
	}

	return &draw.Document{Width: width, Height: height, Duration: tl.Duration, Background: c.page, Static: static, Timed: timed}
}

// statusRuns formats the status bar like the client:
// "[COMPUTER] - arch |  user | pid - arch  | Note: note".
func statusRuns(c csPalette, s scene.Session) []draw.Run {
	text := fmt.Sprintf(" - %s |  %s | %d - %s", s.Arch, s.User, s.PID, s.Arch)
	if s.Note != "" {
		text += "  | Note: " + s.Note
	}
	return []draw.Run{run("["+s.Computer+"]", c.statusAccent), run(text, c.statusText)}
}

// lastLabel formats the time since the last check-in: a short round-trip
// time right after a check-in, then whole seconds.
func lastLabel(seconds int) string {
	if seconds == 0 {
		return "42ms"
	}
	return fmt.Sprintf("%ds", seconds)
}
