// Package beaconcast renders scripted Cobalt Strike Beacon console demos as
// animated SVG and GIF. The CLI is an adapter over the public packages: scene
// loads and validates scene files, console lays a scene out as a
// draw.Document, and render draws documents as SVG, PNG, or GIF.
package beaconcast

import (
	"fmt"

	"github.com/rubyattack3r/beaconcast/console"
	"github.com/rubyattack3r/beaconcast/draw"
	"github.com/rubyattack3r/beaconcast/scene"
)

// Overrides replace scene settings, as the CLI's --mode does. Empty fields
// keep the scene's own values.
type Overrides struct {
	Mode string
}

// Apply validates and writes the non-empty overrides into a scene.
func (o Overrides) Apply(sc *scene.Scene) error {
	if o.Mode != "" {
		if err := scene.ValidateMode(o.Mode); err != nil {
			return err
		}
		sc.Mode = o.Mode
	}
	return nil
}

// Load reads a scene file, applies overrides, and lays it out.
func Load(path string, overrides Overrides) (*draw.Document, error) {
	sc, err := scene.Load(path)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if err := overrides.Apply(sc); err != nil {
		return nil, err
	}
	return console.Build(sc)
}
