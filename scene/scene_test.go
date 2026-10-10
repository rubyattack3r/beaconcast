package scene

import (
	"strings"
	"testing"
)

const minimal = `
[[command]]
input = "whoami"
  [[command.event]]
  at = 0.5
  kind = "output"
  text = "contoso\\alice"
`

func TestExampleParses(t *testing.T) {
	sc, err := Load("../examples/situational-awareness.toml")
	if err != nil {
		t.Fatal(err)
	}
	if len(sc.Commands) != 3 || sc.Session.Computer != "WKS01" || sc.Mode != ModeDark {
		t.Fatalf("unexpected scene: %+v", sc)
	}
}

func TestDefaults(t *testing.T) {
	sc, err := Parse(minimal)
	if err != nil {
		t.Fatal(err)
	}
	if sc.Timing.Start != 1.0 || sc.Timing.Hold != 4.0 || sc.Session.PID != 4120 {
		t.Fatalf("defaults not applied: %+v %+v", sc.Timing, sc.Session)
	}
}

func TestExplicitZeroTiming(t *testing.T) {
	sc, err := Parse("[timing]\nafter_typing = 0\nstart = 0\n" + minimal)
	if err != nil {
		t.Fatal(err)
	}
	if sc.Timing.AfterTyping != 0 || sc.Timing.Start != 0 {
		t.Fatalf("explicit zero replaced by default: %+v", sc.Timing)
	}
}

func TestEventLines(t *testing.T) {
	e := Event{Text: "one\ntwo\n"}
	if got := e.Lines(); len(got) != 2 || got[1] != "two" {
		t.Fatalf("Lines() = %q", got)
	}
}

func TestInvalidScenes(t *testing.T) {
	cases := map[string]string{
		"no commands":  `mode = "dark"`,
		"unknown key":  "colour = \"red\"\n" + minimal,
		"unknown kind": strings.Replace(minimal, `"output"`, `"result"`, 1),
		"empty input":  strings.Replace(minimal, `"whoami"`, `" "`, 1),
		"missing text": strings.Replace(minimal, `text = "contoso\\alice"`, ``, 1),
		"negative at":  strings.Replace(minimal, "at = 0.5", "at = -1", 1),
		"out of order": minimal + "  [[command.event]]\n  at = 0.1\n  kind = \"info\"\n  text = \"x\"\n",
		"bad timing":   "[timing]\nhold = -1\n" + minimal,
	}
	for name, text := range cases {
		if _, err := Parse(text); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestLimits(t *testing.T) {
	long := strings.Repeat("x", MaxInputRunes+1)
	if _, err := Parse(strings.Replace(minimal, `"whoami"`, `"`+long+`"`, 1)); err == nil {
		t.Error("expected input length error")
	}
	lines := strings.Repeat(`line\n`, MaxLines+1)
	if _, err := Parse(strings.Replace(minimal, `text = "contoso\\alice"`, `text = "`+lines+`"`, 1)); err == nil {
		t.Error("expected line limit error")
	}
	if _, err := Parse("[timing]\nhold = 700\n" + minimal); err == nil {
		t.Error("expected duration limit error")
	}
}

func TestModeAndSpeed(t *testing.T) {
	sc, err := Parse("mode = \"light\"\n[timing]\nspeed = 2\n" + minimal)
	if err != nil {
		t.Fatal(err)
	}
	if sc.Mode != ModeLight || sc.Timing.Speed != 2 {
		t.Fatalf("mode %q speed %v", sc.Mode, sc.Timing.Speed)
	}
	if sc, _ := Parse(minimal); sc.Mode != ModeDark {
		t.Fatalf("default mode %q", sc.Mode)
	}
	for _, bad := range []string{"mode = \"dim\"\n", "[timing]\nspeed = 0\n", "[timing]\nspeed = 50\n"} {
		if _, err := Parse(bad + minimal); err == nil {
			t.Errorf("expected an error for %q", bad)
		}
	}
}

func TestHeight(t *testing.T) {
	if sc, err := Parse("height = 480\n" + minimal); err != nil || sc.Height != 480 {
		t.Fatalf("height = %v, %v", sc, err)
	}
	for _, bad := range []string{"height = 100\n", "height = 5000\n"} {
		if _, err := Parse(bad + minimal); err == nil {
			t.Errorf("expected an error for %q", bad)
		}
	}
}

func TestColumns(t *testing.T) {
	sc, err := Parse(minimal)
	if err != nil || len(sc.Session.Columns) != len(DefaultColumns) {
		t.Fatalf("default columns: %v, %v", sc.Session.Columns, err)
	}
	bad := map[string]string{
		"missing extra":  "[session]\ncolumns = [\"sleep\"]\n",
		"duplicate":      "[session]\ncolumns = [\"user\", \"user\"]\n",
		"built-in extra": "[session]\nextra = { user = \"x\" }\n",
		"too many":       "[session]\ncolumns = [" + strings.Repeat(`"user",`, MaxColumns+1) + "]\n",
		"long extra":     "[session]\nextra = { sleep = \"" + strings.Repeat("x", 65) + "\" }\n",
	}
	for name, text := range bad {
		if _, err := Parse(text + minimal); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}
