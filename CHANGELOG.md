# Changelog

All notable changes to this project are documented in this file. The format
is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

## [0.1.0] - 2026-10-10

### Added

- `beaconcast render` writes an animated SVG and a looping GIF of a scene.
- `beaconcast frame` writes a single still frame as PNG or SVG.
- `beaconcast scene validate` checks a scene without rendering it.
- TOML scene format for typed commands and their `task`, `checkin`,
  `output`, `info`, and `error` events, with adjustable timing and speed.
- Client layout with a session table, Beacon tab, console, status bar, and
  `beacon>` input line.
- Dark and light color modes.
- Console scrolling within a maximum window height.
- Configurable session table columns, including custom columns.
- Go packages for loading scenes, laying out the console, and rendering.
- Example scene using TrustedSec's Situational Awareness BOF output.
