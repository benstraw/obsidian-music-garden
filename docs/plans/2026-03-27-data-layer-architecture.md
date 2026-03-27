# Plan: Layered Data Architecture

**Date:** 2026-03-27  
**Status:** Implemented in `v0.6.0`

## Summary

Introduce an explicit layered runtime data architecture so source snapshots,
source-cleaned records, canonical merged records, and Obsidian output are no
longer conceptually mixed together.

## Implemented in This Slice

- Added a dedicated `internal/datalayer` package with record types for:
  - raw source snapshots
  - normalized source-cleaned records
  - aggregated canonical artist/release/genre records
- Added directory layout creation for:
  - `data/raw/spotify/`
  - `data/raw/musicbrainz/`
  - `data/raw/wikipedia/`
  - `data/normalized/{artists,releases,genres}/`
  - `data/aggregated/{artists,releases,genres}/`
- Wired the current Spotify flows to persist:
  - raw `recently-played` snapshots
  - raw `artists` batch snapshots
  - raw `top-artists` snapshots
  - aggregated canonical artist/release/genre records
- Kept vault output unchanged. Obsidian markdown still writes to the user's
  vault under `music/`, not to a repo-root `content/` directory.

## Deferred

- Writing `data/normalized/` records
- MusicBrainz and Wikipedia adapters
- Album and genre markdown generation in the vault
