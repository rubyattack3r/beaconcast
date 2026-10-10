# Third-party notices

Beaconcast is released under the [MIT License](LICENSE). It includes or links
the following third-party software. The full license text of each Go module
is in [`licenses/`](licenses), under the module's path.

## Embedded font

[JetBrains Mono](https://github.com/JetBrains/JetBrainsMono) 2.304, Copyright
2020 The JetBrains Mono Project Authors, licensed under the SIL Open Font
License 1.1. The regular and bold fonts are embedded in the binary from
`internal/fonts`, with the license in [`internal/fonts/OFL.txt`](internal/fonts/OFL.txt).
Every SVG that Beaconcast renders embeds a subset of the font under the name
`BeaconcastMono` and carries this notice in a comment.

## Go modules linked into the binary

| Module | License |
|---|---|
| [github.com/andybalholm/brotli](https://github.com/andybalholm/brotli) | MIT |
| [github.com/BurntSushi/toml](https://github.com/BurntSushi/toml) | MIT |
| [github.com/golang/freetype](https://github.com/golang/freetype) | FreeType License (chosen from FreeType License or GPL-2.0) |
| [github.com/inconshreveable/mousetrap](https://github.com/inconshreveable/mousetrap) (Windows builds) | Apache-2.0 |
| [github.com/spf13/cobra](https://github.com/spf13/cobra) | Apache-2.0 |
| [github.com/spf13/pflag](https://github.com/spf13/pflag) | BSD-3-Clause |
| [github.com/tdewolff/font](https://github.com/tdewolff/font) | MIT |
| [github.com/tdewolff/parse](https://github.com/tdewolff/parse) | MIT |
| [golang.org/x/image](https://pkg.go.dev/golang.org/x/image) | BSD-3-Clause |
| [golang.org/x/sys](https://pkg.go.dev/golang.org/x/sys) | BSD-3-Clause |
| [golang.org/x/text](https://pkg.go.dev/golang.org/x/text) | BSD-3-Clause |

Portions of this software are based in part on the work of the FreeType Team
(<https://www.freetype.org>).

## Example scene

The example scene reproduces the task messages and output formats of
TrustedSec's [CS-Situational-Awareness-BOF](https://github.com/trustedsec/CS-Situational-Awareness-BOF)
for `uptime`, `ipconfig`, and `arp`, with made-up host data. No code from
that project is included.

## Trademarks

Cobalt Strike is a trademark of Fortra, LLC. Beaconcast imitates the
client's appearance without its logos or assets and is not affiliated with,
endorsed by, or sponsored by Fortra.
