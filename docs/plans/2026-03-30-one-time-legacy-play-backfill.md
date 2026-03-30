# One-Time Historical Play Backfill from `topTracks.json`

## Summary

Add a one-time upstream import in `music-garden` that converts the old website's `../benstrawbridge.com/data/spotify/topTracks.json` into normal sharded play events under `data/plays/YYYY/YYYY-WNN.json`.

This keeps downstream simple: once historical plays exist in the canonical play stream, canonical aggregated genres and artists naturally reflect that history.

## Key Changes

- Add `music-garden import-legacy-plays` as a dedicated command, separate from `import-legacy`.
- Use `topTracks.json` as the playback source and `artists.json` only as support data.
- Assign approximate historical timestamps deterministically, spreading each artist's imported plays across their legacy `first_seen` / `last_seen` window when available, and a configurable fallback window otherwise.
- Mark imported plays with `source: legacy-backfill`.
- Write an import manifest under `data/import-manifests/legacy-plays.json` and hard-stop on reruns unless `--force` is passed.

## Public Interfaces

- New command:
  - `music-garden import-legacy-plays --source-dir PATH [--dry-run] [--force] [--artist VALUE] [--fallback-from YYYY-MM-DD] [--fallback-to YYYY-MM-DD]`
- New manifest path:
  - `data/import-manifests/legacy-plays.json`

Imported plays use the normal `models.Play` shape and are saved into the existing sharded `data/plays/` layout.

## Test Plan

- Parse `topTracks.json` fixtures into the expected number of synthetic plays.
- Imported plays carry `source: legacy-backfill`.
- Timestamps are deterministic and fall within the expected artist or fallback window.
- Real import is blocked when a manifest already exists unless `--force` is used.
- Imported plays resolve through the normal canonicalization path and increase canonical aggregated counts after rebuild.

## Assumptions

- This is a one-time migration for this repo and this site, not a reusable multi-project legacy framework.
- `topTracks.json` is the best available historical playback source.
- Approximate dates are acceptable as long as they are deterministic and plausibly historical.
