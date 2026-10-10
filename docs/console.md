# Console

Beaconcast draws the parts of the client a Beacon demo needs:

- A title bar with window-control dots and, if the scene sets `title`, its
  text.
- The session table, with one selected session row whose `last` column resets
  at every check-in and then counts up.
- An Event Log tab and the selected `Beacon <internal>@<pid>` tab.
- The Beacon console, which scrolls whole lines once output fills it.
- The status bar, formatted like the client:
  `[computer] - arch |  user | pid - arch  | Note: note`, with a `last:Ns`
  counter on the right.
- The `beacon>` input line, where commands are typed before being echoed into
  the console with an underlined prompt.

## Events

| Event | Console output |
|---|---|
| `task` | `[*]` line with the text |
| `checkin` | `[+] host called home, sent: N bytes` (bytes optional) |
| `output` | `[+] received output:` followed by the text |
| `info` | `[*]` line with the text |
| `error` | `[-]` line with the text |

`checkin` and `output` also reset the session's `last` column and the status
bar's `last:` counter.

## Session table

The default columns are `external`, `internal`, `listener`, `user`,
`computer`, `note`, `process`, `pid`, `arch`, and `last`. Set
`session.columns` to reorder them, hide some, or add new ones; a column the
console does not know shows its value from `session.extra`:

```toml
[session]
columns = ["external", "internal", "user", "computer", "process", "pid", "arch", "sleep", "last"]
extra = { sleep = "60s 10%" }
```

Columns widen to fit their heading and value.

## Color modes

The [`mode`](scene-format.md#top-level) key or `--mode` selects the dark
(default) or light mode. The console stays black in both modes, as it does in the client's
light theme. The light mode follows the client's light theme elsewhere: a
light window frame and session table, a white selected tab beside a
blue-grey one, a light status bar, and a white input line. The dark mode
uses dark equivalents throughout. Colors are matched by eye rather than
taken from the client. Raster output fills the rounded window corners with
`#0d1117` in dark mode and `#ffffff` in light mode unless `--background`
names another color.
