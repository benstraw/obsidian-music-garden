# Remove Canonical Track Persistence

## Summary

Reverse the abandoned `tracks.json` split and simplify the product model so `plays/` is the only durable track-level source of truth. The canonical store remains responsible for genres, artists, releases, and editorial state. Track-oriented normalized and aggregated outputs are rebuilt directly from plays instead of persisted track entities.

## Key Changes

- Remove the split-store experiment:
  - no `data/tracks.json`
  - no `MUSIC_TRACKS_PATH`
  - no `migrate-store-layout`
  - no `replay-upstream-state`
- Remove canonical track persistence from `genres.Store`.
- Keep `TrackSlug` on play records, but derive it deterministically from canonical artist slug + track name instead of looking up a stored track entity.
- Rebuild normalized and aggregated track files from plays during data-layer sync.
- Drop legacy `topTracks.json` import as a source of canonical track metadata or legacy track counts.

## Implementation Notes

- `ResolvePlay` still canonicalizes artists and releases; it now only stamps a deterministic `TrackSlug`.
- Genre top-track sections are computed from canonicalized plays and play-owned metadata.
- Aggregated artist and release legacy counts no longer come from stored track entities.
- Legacy import continues to preserve artist and genre compatibility mappings, but ignores track snapshots.

## Tests

- `ResolvePlay` still sets canonical artist, release, and track slugs.
- Aggregated genre top tracks still populate from plays.
- Normalized and aggregated track JSON are generated from plays without store-backed track records.
- Legacy import ignores `topTracks.json` for canonical track persistence.
- Full suite passes: `go test ./...`, `go vet ./...`, `go build -o music-garden .`.

## Defaults

- `plays/` is the only durable track truth.
- Track metadata in existing play shards is sufficient for downstream track summaries.
- Website-side legacy page preservation remains outside this repo.
