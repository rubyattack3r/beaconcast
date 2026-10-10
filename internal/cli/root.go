package cli

import (
	"bytes"
	"fmt"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/rubyattack3r/beaconcast"
	"github.com/rubyattack3r/beaconcast/console"
	"github.com/rubyattack3r/beaconcast/draw"
	"github.com/rubyattack3r/beaconcast/render"
	"github.com/rubyattack3r/beaconcast/scene"
	"github.com/rubyattack3r/beaconcast/timeline"
	"github.com/spf13/cobra"
)

// Application configures the CLI.
type Application struct {
	Version string
}

// New returns the root beaconcast command.
func New(application Application) *cobra.Command {
	if application.Version == "" {
		application.Version = "dev"
	}
	root := &cobra.Command{
		Use:           "beaconcast",
		Short:         "Render scripted Beacon console demos as animated SVG and GIF",
		SilenceErrors: true,
		SilenceUsage:  true,
		Version:       application.Version,
		Args:          cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			return command.Help()
		},
	}
	root.SetVersionTemplate("beaconcast {{.Version}}\n")
	root.AddCommand(newRenderCommand())
	root.AddCommand(newFrameCommand())
	root.AddCommand(newSceneCommand())
	return root
}

// backgroundFor returns the --background color or the document's own.
func (flags *layoutFlags) backgroundFor(doc *draw.Document) draw.Color {
	if flags.background != "" {
		return draw.Color(flags.background)
	}
	return doc.Background
}

// layoutFlags are shared by commands that lay out a scene.
type layoutFlags struct {
	overrides  beaconcast.Overrides
	background string
}

func (flags *layoutFlags) register(command *cobra.Command) {
	registerOverrides(command, &flags.overrides)
	command.Flags().StringVar(&flags.background, "background", "", "color behind the rounded window corners in raster output (default: the mode's page color)")
}

func registerOverrides(command *cobra.Command, overrides *beaconcast.Overrides) {
	command.Flags().StringVar(&overrides.Mode, "mode", "", "color mode, dark or light, overriding the scene's mode")
}

func newRenderCommand() *cobra.Command {
	var flags layoutFlags
	var svgPath, gifPath string
	var fps int
	command := &cobra.Command{
		Use:   "render SCENE.toml",
		Short: "Write an animated SVG and a GIF for a scene",
		Args:  cobra.ExactArgs(1),
		RunE: func(command *cobra.Command, arguments []string) error {
			path := arguments[0]
			doc, err := beaconcast.Load(path, flags.overrides)
			if err != nil {
				return err
			}
			stem := strings.TrimSuffix(path, filepath.Ext(path))
			if svgPath == "" {
				svgPath = stem + ".svg"
			}
			if gifPath == "" {
				gifPath = stem + ".gif"
			}
			output := command.OutOrStdout()
			if svgPath != "-" {
				data, err := render.SVG(doc)
				if err != nil {
					return err
				}
				if err := writeFile(output, svgPath, data); err != nil {
					return err
				}
			}
			if gifPath != "-" {
				var buffer bytes.Buffer
				if err := render.GIF(&buffer, doc, fps, flags.backgroundFor(doc)); err != nil {
					return err
				}
				if err := writeFile(output, gifPath, buffer.Bytes()); err != nil {
					return err
				}
			}
			return nil
		},
	}
	flags.register(command)
	command.Flags().StringVar(&svgPath, "svg", "", `SVG output path (default: SCENE.svg; "-" to skip)`)
	command.Flags().StringVar(&gifPath, "gif", "", `GIF output path (default: SCENE.gif; "-" to skip)`)
	command.Flags().IntVar(&fps, "fps", 12, "GIF frames per second")
	return command
}

func newFrameCommand() *cobra.Command {
	var flags layoutFlags
	var at float64
	var outputPath string
	command := &cobra.Command{
		Use:   "frame SCENE.toml",
		Short: "Write one still frame of a scene as PNG or SVG",
		Args:  cobra.ExactArgs(1),
		RunE: func(command *cobra.Command, arguments []string) error {
			path := arguments[0]
			doc, err := beaconcast.Load(path, flags.overrides)
			if err != nil {
				return err
			}
			t := at
			if t < 0 {
				t = doc.Duration - 0.01
			}
			if t > doc.Duration {
				return fmt.Errorf("--at %.2f is past the end of the %.2fs loop", t, doc.Duration)
			}
			if outputPath == "" {
				outputPath = strings.TrimSuffix(path, filepath.Ext(path)) + ".png"
			}
			var data []byte
			if strings.EqualFold(filepath.Ext(outputPath), ".svg") {
				if data, err = render.StillSVG(doc, t); err != nil {
					return err
				}
			} else {
				img, err := render.Frame(doc, t, flags.backgroundFor(doc))
				if err != nil {
					return err
				}
				var buffer bytes.Buffer
				if err := png.Encode(&buffer, img); err != nil {
					return err
				}
				data = buffer.Bytes()
			}
			return writeFile(command.OutOrStdout(), outputPath, data)
		},
	}
	flags.register(command)
	command.Flags().Float64Var(&at, "at", -1, "time in seconds (default: the final frame)")
	command.Flags().StringVarP(&outputPath, "output", "o", "", "output path; .svg writes a still SVG (default: SCENE.png)")
	return command
}

func newSceneCommand() *cobra.Command {
	command := &cobra.Command{Use: "scene", Short: "Check scene files", Args: cobra.NoArgs}
	command.AddCommand(newSceneValidateCommand())
	return command
}

func newSceneValidateCommand() *cobra.Command {
	var overrides beaconcast.Overrides
	command := &cobra.Command{
		Use:   "validate SCENE.toml",
		Short: "Check a scene file and that its layout fits the limits",
		Args:  cobra.ExactArgs(1),
		RunE: func(command *cobra.Command, arguments []string) error {
			path := arguments[0]
			sc, err := scene.Load(path)
			if err != nil {
				return fmt.Errorf("%s: %w", path, err)
			}
			if err := overrides.Apply(sc); err != nil {
				return err
			}
			doc, err := console.Build(sc)
			if err != nil {
				return fmt.Errorf("%s: %w", path, err)
			}
			events := 0
			for _, c := range sc.Commands {
				events += len(c.Events)
			}
			_, err = fmt.Fprintf(command.OutOrStdout(), "%s: %d commands, %d events, %.1fs loop, %.0fx%.0f\n",
				path, len(sc.Commands), events, timeline.Build(sc).Duration, doc.Width, doc.Height)
			return err
		},
	}
	registerOverrides(command, &overrides)
	return command
}

func writeFile(output io.Writer, path string, data []byte) error {
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return err
	}
	_, err := fmt.Fprintf(output, "%s\t%d bytes\n", path, len(data))
	return err
}
