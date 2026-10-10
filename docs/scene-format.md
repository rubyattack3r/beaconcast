# Scene format

A scene is a TOML file. Unknown keys are rejected, so a misspelled key fails
instead of being ignored. Scenes are treated as untrusted input and bounded
by the limits below.

## Top level

| Key | Default | Meaning |
|---|---|---|
| `mode` | `dark` | Color mode, `dark` or `light`; `--mode` overrides it |
| `title` | none | Text in the window's title bar |
| `height` | `640` | Maximum window height in pixels (300–4096); the console scrolls when output exceeds it |

## `[timing]`

All values are seconds. An explicit `0` is honored.

| Key | Default | Meaning |
|---|---|---|
| `start` | `1.0` | Before the first keystroke |
| `type_interval` | `0.05` | Between keystrokes |
| `after_typing` | `0.45` | Between the last keystroke and Enter |
| `between` | `1.4` | After a command's last event, before the next command |
| `hold` | `4.0` | After the final event, before the loop restarts |
| `speed` | `1.0` | Playback multiplier from 0.1 to 10 for all timings and event times |

## `[session]`

| Key | Default | Meaning |
|---|---|---|
| `external` | same as `internal` | External address column |
| `internal` | `10.10.10.20` | Internal address column and Beacon tab |
| `listener` | `https` | Listener column |
| `user` | `user` | User column and status bar |
| `computer` | `HOST` | Computer column and status bar |
| `process` | `rundll32.exe` | Process column |
| `pid` | `4120` | PID column, Beacon tab, and status bar |
| `arch` | `x64` | Arch column and status bar |
| `note` | empty | Note column |
| `columns` | all of the above and `last` | Session table columns in order |
| `extra` | none | Values for columns not listed above, such as `{ sleep = "60s" }` |

See [the console](console.md#session-table) for how columns are drawn.

## `[[command]]`

Each command has an `input`, the single line the operator types, and any
number of `[[command.event]]` entries in time order.

| Key | Meaning |
|---|---|
| `at` | Seconds after the command is entered |
| `kind` | `task`, `checkin`, `output`, `info`, or `error` |
| `text` | Message text; may contain newlines. Required except for `checkin` |
| `bytes` | Payload size reported by a `checkin` (optional) |

See [the console](console.md#events) for the lines each kind prints.

TOML literal strings (`'...'`) keep Windows backslashes as written; in basic
strings (`"..."`) write `\\` for a backslash.

## Limits

| Limit | Value |
|---|---|
| Scene file size | 1 MiB |
| Commands | 100 |
| Events per command | 200 |
| Input length | 400 characters |
| Event text lines in the scene | 1,000 |
| Loop length | 600 seconds |
| Session table columns | 20, with names up to 24 and extra values up to 64 characters |
| Laid-out image size | 4096 × 4096 pixels |
| GIF frames | 6,000, at 1–50 frames per second |
