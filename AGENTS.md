# AGENTS.md — Beaconcast

Beaconcast is a standalone Go library and CLI that renders scripted Cobalt
Strike Beacon console demos as animated SVG and GIF. Scene files describe
what an operator types and what Beacon returns; the console package draws
them in the style of the client.

## Boundaries

- Rendering is offline and deterministic. Never read the clock, randomness,
  the network, or system fonts while rendering; the same scene and flags must
  produce byte-identical SVG and GIF output.
- Every renderer draws the same `draw.Document`. Do not add renderer-specific
  layout. When the SVG and raster output disagree, fix the shared shapes or
  the renderer, never one output alone.
- Imitate the client's look only. Do not embed Fortra logos, icons, or other
  copied assets, or name the product in rendered output.
- Keep the scene format stable. Unknown keys stay an error so typos are
  caught. When the client changes, prefer updating defaults and using
  `session.columns` and `session.extra` over adding one-off keys.
- Treat scene files as untrusted input. Keep the limits in `scene`,
  `console.MaxSize`, and `render` in force, add a bound for any new resource a
  scene can request, and document it in `docs/scene-format.md`.
- Keep reusable functionality in public packages. `internal/` is reserved for
  CLI wiring and font handling that callers should not import.
- Keep CLI presentation out of library packages. Library users must receive
  typed results and errors without terminal output or process exits.
- JetBrains Mono is the only font. Layout assumes its 0.6 em advance
  (`draw.Advance`); keep its OFL license beside the embedded files.
- Keep attribution complete. When dependencies change, run `make licenses`
  and update the module table in `THIRD_PARTY_NOTICES.md`; `make check` fails
  while a linked module's license is missing from `licenses/`.

## CLI conventions

- Use Cobra commands under `internal/cli`; keep `cmd/beaconcast` as a small
  entrypoint.
- Use kebab-case flags and action-oriented subcommands; group management
  commands under a noun, such as `scene validate`.
- Keep `--mode` consistent across `render`, `frame`, and `scene validate`.
- Write result data to stdout and diagnostics to stderr.
- Write outputs beside the scene by default and accept `-` to skip one.
- Exit non-zero only when a command fails.

## Appearance changes

A change to how the console looks lands with:

1. Both color modes updated; the console stays dark in both.
2. Reviewed golden frames in `render/testdata` (`go test ./render -update`).
3. Passing in-bounds and palette contrast tests.
4. Updated example renders (`make examples`), `docs/console.md`, and a
   `CHANGELOG.md` entry.

## Project layout

- `cmd/beaconcast/`: executable entrypoint
- `internal/cli/`: Cobra commands and output
- `internal/fonts/`: embedded JetBrains Mono, subsetting, and raster faces
- `beaconcast.go`: convenience API over the public packages
- `scene/`: scene file format, validation, and limits
- `timeline/`: absolute times for keystrokes, commands, and events
- `draw/`: shapes shared by the layout and renderers
- `console/`: Beacon console layout, palettes, and scrolling
- `render/`: animated SVG, still SVG, PNG frames, and GIF encoding
- `examples/`: example scene and its renders
- `docs/`: architecture, scene format, and console documentation
- `licenses/`: license texts of linked Go modules, written by `make licenses`
- `scripts/`: maintenance scripts
- `.github/`: CI, CodeQL, release workflows, and Dependabot

## Validation

Run the complete local gate before committing:

```bash
make check
nix flake check
```

`make check` must format-check, vet, lint Go and workflows, check dependency
licenses, test, and build the CLI. CI runs the same gate in the flake's dev
shell, plus `nix flake check`, a GoReleaser snapshot build, and `make
vulncheck`. Pin every GitHub Action to a commit SHA with a `# ratchet:`
comment. `nix flake check`
must reproduce the packaged build through the locked Nix toolchain. Regenerate
golden frames with `go test ./render -update` only after reviewing the new
images.

## Documentation and Changelog

- Keep `README.md`, `docs/`, CLI help, examples, and release archives
  consistent with behavior.
- Write each document for its audience: the README and `docs/console.md` and
  `docs/scene-format.md` for users, `docs/architecture.md` and this file for
  contributors, and `THIRD_PARTY_NOTICES.md` for attribution.
- State each fact in one place and link to it elsewhere instead of repeating
  it.
- Follow [Keep a Changelog](https://keepachangelog.com/en/1.1.0/): put
  user-facing changes under `Unreleased`, group them by applicable types such
  as `Added`, `Changed`, `Fixed`, and `Security`, and move them to a dated
  version section only when releasing. Do not use the changelog as a work log.

## Releasing

1. Move the `Unreleased` entries in `CHANGELOG.md` to a new
   `## [X.Y.Z] - YYYY-MM-DD` section.
2. Commit, then tag and push: `git tag vX.Y.Z && git push origin vX.Y.Z`.
3. The release workflow publishes the GoReleaser archives with that
   changelog section as release notes and attests their build provenance.
   It fails if the changelog has no section for the tag.
