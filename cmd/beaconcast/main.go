package main

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/rubyattack3r/beaconcast/internal/cli"
)

// version is set by release builds; go install builds report the module
// version instead.
var version = "dev"

func main() {
	if info, ok := debug.ReadBuildInfo(); ok && version == "dev" &&
		info.Main.Version != "" && info.Main.Version != "(devel)" {
		version = info.Main.Version
	}
	command := cli.New(cli.Application{Version: version})
	if err := command.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "beaconcast: %v\n", err)
		os.Exit(1)
	}
}
