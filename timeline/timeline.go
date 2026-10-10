// Package timeline turns scene pacing into absolute times.
package timeline

import "github.com/rubyattack3r/beaconcast/scene"

// Command holds the absolute times for one scene command.
type Command struct {
	TypeStart float64   // first keystroke
	Keys      []float64 // when each rune of the input appears
	Enter     float64   // when the command is submitted
	Events    []float64 // when each event happens
	Done      float64   // the command's last event, or Enter
}

// Timeline holds absolute times for a whole scene.
type Timeline struct {
	Commands []Command
	Checkins []float64 // when the session last checked in, including 0
	Duration float64
}

// Build computes the timeline for a scene.
func Build(sc *scene.Scene) *Timeline {
	tl := &Timeline{Checkins: []float64{0}}
	speed := sc.Timing.Speed
	if speed <= 0 {
		speed = 1
	}
	timing := sc.Timing
	start, typeInterval, afterTyping := timing.Start/speed, timing.TypeInterval/speed, timing.AfterTyping/speed
	between, hold := timing.Between/speed, timing.Hold/speed
	t := start
	for _, command := range sc.Commands {
		c := Command{TypeStart: t}
		runes := []rune(command.Input)
		for i := range runes {
			c.Keys = append(c.Keys, t+float64(i)*typeInterval)
		}
		c.Enter = t + float64(len(runes))*typeInterval + afterTyping
		c.Done = c.Enter
		for _, event := range command.Events {
			at := c.Enter + event.At/speed
			c.Events = append(c.Events, at)
			if at > c.Done {
				c.Done = at
			}
			if event.Kind == scene.KindCheckin || event.Kind == scene.KindOutput {
				tl.Checkins = append(tl.Checkins, at)
			}
		}
		tl.Commands = append(tl.Commands, c)
		t = c.Done + between
	}
	tl.Duration = t - between + hold
	return tl
}
