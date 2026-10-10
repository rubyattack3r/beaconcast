// Package scene loads and validates beaconcast scene files.
//
// A scene lists the commands an operator types, the events each command
// produces, and the session shown in the client's session table.
package scene

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/BurntSushi/toml"
)

// Event kinds.
const (
	KindTask    = "task"    // the client acknowledged the command
	KindCheckin = "checkin" // Beacon checked in and picked up the task
	KindOutput  = "output"  // Beacon returned output
	KindInfo    = "info"    // an informational message, such as a download
	KindError   = "error"   // the command failed
)

// Limits bound the work a scene file can request.
const (
	MaxBytes      = 1 << 20 // scene file size
	MaxCommands   = 100
	MaxEvents     = 200 // per command
	MaxInputRunes = 400
	MaxLines      = 1000 // event text lines across the scene
	MaxDuration   = 600.0
	MinHeight     = 300 // smallest window height a scene may request
	MaxHeight     = 4096
)

// DefaultColumns is the client's default session table.
var DefaultColumns = []string{"external", "internal", "listener", "user", "computer", "note", "process", "pid", "arch", "last"}

// MaxColumns bounds the session table.
const MaxColumns = 20

// Color modes.
const (
	ModeDark  = "dark"
	ModeLight = "light"
)

var kinds = map[string]bool{
	KindTask: true, KindCheckin: true, KindOutput: true, KindInfo: true, KindError: true,
}

// Scene is a parsed scene file.
type Scene struct {
	Mode     string    `toml:"mode"`
	Title    string    `toml:"title"`
	Height   float64   `toml:"height"` // maximum window height in pixels; content scrolls beyond it
	Timing   Timing    `toml:"-"`
	RawTime  rawTiming `toml:"timing"`
	Session  Session   `toml:"session"`
	Commands []Command `toml:"command"`
}

// Timing controls the pacing of the animation, in seconds.
type Timing struct {
	Start        float64 `toml:"start"`         // before the first keystroke
	TypeInterval float64 `toml:"type_interval"` // between keystrokes
	AfterTyping  float64 `toml:"after_typing"`  // between the last keystroke and Enter
	Between      float64 `toml:"between"`       // after a command's last event
	Hold         float64 `toml:"hold"`          // after the final event, before looping
	Speed        float64 `toml:"speed"`         // playback multiplier applied to all of the above and to event times
}

// rawTiming distinguishes an explicit 0 from an unset value.
type rawTiming struct {
	Start        *float64 `toml:"start"`
	TypeInterval *float64 `toml:"type_interval"`
	AfterTyping  *float64 `toml:"after_typing"`
	Between      *float64 `toml:"between"`
	Hold         *float64 `toml:"hold"`
	Speed        *float64 `toml:"speed"`
}

// Session describes the Beacon session shown in the session table, the
// Beacon tab, and the status bar.
type Session struct {
	External string `toml:"external"`
	Internal string `toml:"internal"`
	Listener string `toml:"listener"`
	User     string `toml:"user"`
	Computer string `toml:"computer"`
	Process  string `toml:"process"`
	PID      int    `toml:"pid"`
	Arch     string `toml:"arch"`
	Note     string `toml:"note"`

	// Columns lists the session table columns in order. Known columns show
	// the fields above; others show their Extra value; "last" counts up
	// from each check-in.
	Columns []string          `toml:"columns"`
	Extra   map[string]string `toml:"extra"`
}

// Command is one typed command and the events it produces.
type Command struct {
	Input  string  `toml:"input"`
	Events []Event `toml:"event"`
}

// Event is something that happens At seconds after the command is entered.
type Event struct {
	At    float64 `toml:"at"`
	Kind  string  `toml:"kind"`
	Text  string  `toml:"text"`
	Bytes int     `toml:"bytes"`
}

// Lines splits the event text into display lines.
func (e Event) Lines() []string {
	if e.Text == "" {
		return nil
	}
	return strings.Split(strings.TrimRight(e.Text, "\n"), "\n")
}

// ValidateMode reports whether mode is empty, dark, or light.
func ValidateMode(mode string) error {
	switch mode {
	case "", ModeDark, ModeLight:
		return nil
	}
	return fmt.Errorf("mode must be %q or %q, got %q", ModeDark, ModeLight, mode)
}

// Load reads, validates, and fills defaults for a scene file.
func Load(path string) (*Scene, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("not a regular file")
	}
	if info.Size() > MaxBytes {
		return nil, fmt.Errorf("scene file exceeds %d bytes", MaxBytes)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(string(data))
}

// Parse decodes, validates, and fills defaults for scene text.
func Parse(text string) (*Scene, error) {
	if len(text) > MaxBytes {
		return nil, fmt.Errorf("scene exceeds %d bytes", MaxBytes)
	}
	var sc Scene
	meta, err := toml.Decode(text, &sc)
	if err != nil {
		return nil, err
	}
	if undecoded := meta.Undecoded(); len(undecoded) > 0 {
		keys := make([]string, len(undecoded))
		for i, key := range undecoded {
			keys[i] = key.String()
		}
		sort.Strings(keys)
		return nil, fmt.Errorf("unknown scene keys: %s", strings.Join(keys, ", "))
	}
	sc.fillDefaults()
	if err := sc.validate(); err != nil {
		return nil, err
	}
	return &sc, nil
}

func (sc *Scene) fillDefaults() {
	if sc.Mode == "" {
		sc.Mode = ModeDark
	}
	pick := func(value *float64, fallback float64) float64 {
		if value == nil {
			return fallback
		}
		return *value
	}
	raw := sc.RawTime
	sc.Timing = Timing{
		Start:        pick(raw.Start, 1.0),
		TypeInterval: pick(raw.TypeInterval, 0.05),
		AfterTyping:  pick(raw.AfterTyping, 0.45),
		Between:      pick(raw.Between, 1.4),
		Hold:         pick(raw.Hold, 4.0),
		Speed:        pick(raw.Speed, 1.0),
	}

	s := &sc.Session
	text := func(value *string, fallback string) {
		if *value == "" {
			*value = fallback
		}
	}
	text(&s.Internal, "10.10.10.20")
	text(&s.External, s.Internal)
	text(&s.Listener, "https")
	text(&s.User, "user")
	text(&s.Computer, "HOST")
	text(&s.Process, "rundll32.exe")
	text(&s.Arch, "x64")
	if s.PID == 0 {
		s.PID = 4120
	}
	if len(s.Columns) == 0 {
		s.Columns = append([]string(nil), DefaultColumns...)
	}
}

func (sc *Scene) validate() error {
	if len(sc.Commands) == 0 {
		return fmt.Errorf("scene has no [[command]] entries")
	}
	if len(sc.Commands) > MaxCommands {
		return fmt.Errorf("scene has %d commands; the limit is %d", len(sc.Commands), MaxCommands)
	}
	lines := 0
	total := sc.Timing.Start + sc.Timing.Hold
	timings := map[string]float64{
		"start": sc.Timing.Start, "type_interval": sc.Timing.TypeInterval,
		"after_typing": sc.Timing.AfterTyping, "between": sc.Timing.Between,
		"hold": sc.Timing.Hold,
	}
	for name, value := range timings {
		if value < 0 {
			return fmt.Errorf("timing.%s must not be negative", name)
		}
	}
	if sc.Timing.Speed < 0.1 || sc.Timing.Speed > 10 {
		return fmt.Errorf("timing.speed must be between 0.1 and 10")
	}
	if err := ValidateMode(sc.Mode); err != nil {
		return err
	}
	if err := sc.Session.validateColumns(); err != nil {
		return err
	}
	if sc.Height != 0 && (sc.Height < MinHeight || sc.Height > MaxHeight) {
		return fmt.Errorf("height must be between %d and %d pixels", MinHeight, MaxHeight)
	}
	for i, command := range sc.Commands {
		where := fmt.Sprintf("command %d", i+1)
		if strings.TrimSpace(command.Input) == "" {
			return fmt.Errorf("%s: input is required", where)
		}
		if strings.Contains(command.Input, "\n") {
			return fmt.Errorf("%s: input must be a single line", where)
		}
		if n := utf8.RuneCountInString(command.Input); n > MaxInputRunes {
			return fmt.Errorf("%s: input has %d characters; the limit is %d", where, n, MaxInputRunes)
		}
		if len(command.Events) > MaxEvents {
			return fmt.Errorf("%s: %d events; the limit is %d", where, len(command.Events), MaxEvents)
		}
		total += float64(utf8.RuneCountInString(command.Input))*sc.Timing.TypeInterval + sc.Timing.AfterTyping + sc.Timing.Between
		for j, event := range command.Events {
			at := fmt.Sprintf("%s, event %d", where, j+1)
			if !kinds[event.Kind] {
				return fmt.Errorf("%s: unknown kind %q (want task, checkin, output, info, or error)", at, event.Kind)
			}
			if event.At < 0 {
				return fmt.Errorf("%s: at must not be negative", at)
			}
			if j > 0 && event.At < command.Events[j-1].At {
				return fmt.Errorf("%s: events must be in time order", at)
			}
			if event.Kind != KindCheckin && event.Text == "" {
				return fmt.Errorf("%s: %s events need text", at, event.Kind)
			}
			lines += len(event.Lines())
		}
		if n := len(command.Events); n > 0 {
			total += command.Events[n-1].At
		}
	}
	if lines > MaxLines {
		return fmt.Errorf("scene has %d event lines; the limit is %d", lines, MaxLines)
	}
	if total /= sc.Timing.Speed; total > MaxDuration {
		return fmt.Errorf("scene lasts about %.0fs; the limit is %.0fs", total, MaxDuration)
	}
	return nil
}

var knownColumns = map[string]bool{
	"external": true, "internal": true, "listener": true, "user": true, "computer": true,
	"note": true, "process": true, "pid": true, "arch": true, "last": true,
}

func (s Session) validateColumns() error {
	if len(s.Columns) > MaxColumns {
		return fmt.Errorf("session.columns has %d entries; the limit is %d", len(s.Columns), MaxColumns)
	}
	seen := map[string]bool{}
	for _, column := range s.Columns {
		if column == "" || utf8.RuneCountInString(column) > 24 {
			return fmt.Errorf("session.columns entries must be 1 to 24 characters")
		}
		if seen[column] {
			return fmt.Errorf("session.columns lists %q twice", column)
		}
		seen[column] = true
		if _, ok := s.Extra[column]; !knownColumns[column] && !ok {
			return fmt.Errorf("session.columns lists %q, which needs a value in [session.extra]", column)
		}
	}
	for key, value := range s.Extra {
		if knownColumns[key] {
			return fmt.Errorf("session.extra.%s duplicates a built-in column; set session.%s instead", key, key)
		}
		if utf8.RuneCountInString(value) > 64 {
			return fmt.Errorf("session.extra.%s is longer than 64 characters", key)
		}
	}
	return nil
}
