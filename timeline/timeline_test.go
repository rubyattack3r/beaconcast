package timeline

import (
	"math"
	"testing"

	"github.com/rubyattack3r/beaconcast/scene"
)

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestBuild(t *testing.T) {
	sc, err := scene.Parse(`
[timing]
start = 1
type_interval = 0.1
after_typing = 0.5
between = 2
hold = 3

[[command]]
input = "ab"
  [[command.event]]
  at = 1
  kind = "checkin"
  [[command.event]]
  at = 2
  kind = "output"
  text = "done"

[[command]]
input = "c"
`)
	if err != nil {
		t.Fatal(err)
	}
	tl := Build(sc)
	first := tl.Commands[0]
	if !near(first.Keys[1], 1.1) || !near(first.Enter, 1.7) || !near(first.Done, 3.7) {
		t.Fatalf("first command times: %+v", first)
	}
	second := tl.Commands[1]
	if !near(second.TypeStart, 5.7) || !near(second.Done, second.Enter) {
		t.Fatalf("second command times: %+v", second)
	}
	if !near(tl.Duration, second.Done+3) {
		t.Fatalf("duration %v", tl.Duration)
	}
	if len(tl.Checkins) != 3 || !near(tl.Checkins[1], 2.7) || !near(tl.Checkins[2], 3.7) {
		t.Fatalf("checkins %v", tl.Checkins)
	}
}

func TestSpeedScalesEverything(t *testing.T) {
	base := `
[[command]]
input = "ab"
  [[command.event]]
  at = 2
  kind = "output"
  text = "x"
`
	normal, _ := scene.Parse(base)
	fast, _ := scene.Parse("[timing]\nspeed = 2\n" + base)
	a, b := Build(normal), Build(fast)
	if !near(a.Duration, 2*b.Duration) || !near(a.Commands[0].Events[0], 2*b.Commands[0].Events[0]) {
		t.Fatalf("speed 2 durations: %v vs %v", a.Duration, b.Duration)
	}
}
