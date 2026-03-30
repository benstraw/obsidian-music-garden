# Multi-Artist Support, Legacy Snapshot Import, and Historical Play Backfill

## Summary

Implement multi-artist support as an additive extension to the play model, add a narrow `import-legacy` pipeline for the genuinely independent legacy Spotify snapshot files, and add a separate Spotify backfill command to enrich existing sharded plays with missing multi-artist metadata.

The updated scope is narrower than the original draft:

- exclude legacy `plays/` and `genres.json` from import because they are downstream copies of this repo
- treat `topTracks.json` as untimestamped historical count data, not play history
- keep current canonical slugs authoritative and persist compatibility artifacts for legacy website slugs

## Key Changes

### 1. Multi-artist support in the core play model

Additive changes only:

- add `models.PlayArtist` with `id`, `name`, and optional `spotify_url`
- add `Play.AdditionalArtists []PlayArtist` for track `artists[1:]`
- keep existing scalar artist fields as the primary artist contract for compatibility

Ingestion and canonicalization:

- extend Spotify fetch parsing so `spotifyAlbum` includes `artists[]`
- update `itemToPlay` to populate `AdditionalArtists`
- extend `genres.ResolvePlay` so primary and additional artists all flow through `ensureArtist`
- extend `genres.UncachedArtistIDs` to include all artist IDs from those artist slots
- extend `plays.mergePlay` to preserve richer slice data when duplicate plays merge
- extend weekly/daily artist rotation so featured artists from `AdditionalArtists` are included in the rotation set; keep the repeated-track grouping key unchanged

### 2. Historical play enrichment as a separate Spotify backfill

Add a new explicit command for existing sharded plays:

```sh
music-garden backfill-play-artists [--from-year YYYY] [--limit N] [--dry-run] [--verbose]
```

Behavior:

- scan existing `data/plays/YYYY/*.json`
- collect unique Spotify track IDs from plays that are missing `additional_artists` or `album_id`
- batch-fetch full track metadata from Spotify
- rewrite matching plays with:
  - `AdditionalArtists`
  - any missing `AlbumID` if present in Spotify track data
- resolve updated plays through `genres.ResolvePlay`
- persist rewritten sharded plays and then run the normal store save path once

Boundaries:

- this is not part of `collect`
- this is not part of `repair-plays`
- it is an explicit one-shot or occasional maintenance command for historical enrichment

### 3. Narrow legacy import pipeline

Add a new package and command:

- new package: `internal/importlegacy`
- new command: `music-garden import-legacy --source-dir PATH [--dry-run] [--verbose] [--audit-genres]`

Import scope:

- include:
  - `topArtists.json`
  - `topTracks.json`
  - `snapshot-2024-06.json`
  - optionally `artists.json` for secondary coverage and website-only aggregate hints
- exclude:
  - `plays/` because it is downstream-copied garden data
  - `genres.json` because it is downstream-copied garden data

Import behavior by file:

1. `topArtists.json`
   - primary artist import source
   - seed artists, Spotify genres, source genres, images, and other artist metadata already representable in the store
2. `snapshot-2024-06.json`
   - image-quality fallback and upgrade source for artists
3. `artists.json`
   - optional supplemental source for artist coverage gaps and genre labels
   - do not invent new canonical fields just to store website-only aggregates in this phase
4. `topTracks.json`
   - import artists, additional artists, album artists, releases, and tracks
   - preserve duplicate entries as untimestamped historical count data
   - do not create synthetic plays
   - do not touch `data/plays/`

Persistence for `topTracks.json`:

- add a track-level canonical metric such as `legacy_play_count`
- this represents untimestamped historical play-count evidence from the legacy dataset
- it remains distinct from true timestamped play history in `data/plays/`

### 4. Legacy compatibility artifacts

Keep current canonical slugs unchanged.

Add committed compatibility files in this repo:

- `data/legacy-artist-slugs.json`
  - legacy website artist slug -> canonical artist slug
  - include Spotify artist ID and display name where available
- `data/legacy-genre-slugs.json`
  - legacy website genre slug -> canonical genre slug where resolvable
  - unresolved entries remain explicit for downstream redirect and taxonomy work

Use these for:

- artist slug mismatches between website and current garden
- preserving SEO-sensitive legacy URL continuity downstream
- import audit output and verification

Genre reconciliation policy:

- `--audit-genres` should report legacy genre slugs and labels that do not resolve through the current taxonomy
- do not auto-create hundreds of canonical genres to satisfy old website URLs
- unresolved items go into the compatibility artifact and taxonomy backlog, not directly into canonical genre generation

## Public Interfaces and Types

Add or update:

- `models.PlayArtist`
- `models.Play.AdditionalArtists`
- `genres.TrackRecord.LegacyPlayCount` or equivalent canonical track-side field for untimestamped legacy counts
- CLI:
  - `music-garden import-legacy --source-dir PATH [--dry-run] [--verbose] [--audit-genres]`
  - `music-garden backfill-play-artists [--from-year YYYY] [--limit N] [--dry-run] [--verbose]`

Add persisted compatibility artifacts:

- `data/legacy-artist-slugs.json`
- `data/legacy-genre-slugs.json`

No existing `Play` field is removed or renamed.

## Test Plan

Add focused tests for each behavior.

Multi-artist core:

- `itemToPlay` populates additional artists correctly
- `ResolvePlay` ensures all referenced artists enter the canonical store
- `UncachedArtistIDs` includes uncached additional artists
- `mergePlay` prefers richer slice data on duplicate merges
- weekly/daily artist rotation includes featured artists while repeated-track grouping remains stable

Historical play backfill:

- dry-run reports candidate play updates without writing
- backfill rewrites matching plays with additional artists and missing album IDs
- repeated runs are idempotent
- backfill does not alter plays that already have richer artist arrays

Legacy import:

- `topArtists.json` imports artists and images correctly
- `snapshot-2024-06.json` upgrades artist images without downgrading richer existing data
- `topTracks.json` seeds track, release, and additional-artist relationships and sets `legacy_play_count`
- `topTracks.json` never creates play records or week shards
- `artists.json` only fills gaps and does not overwrite richer canonical metadata with weaker legacy values
- compatibility files contain expected mappings for known slug mismatches
- `--audit-genres` reports unresolved legacy genre slugs and labels without mutating taxonomy

Verification scenarios:

- legacy top-artist coverage materially increases canonical artist coverage
- known multi-artist tracks seed featured artists into the store
- known slug mismatch artists resolve via compatibility file, not canonical rename
- future stats can read both timestamped play counts and untimestamped `legacy_play_count`

## Assumptions and Defaults

- `data/plays/` remains the only timestamped chronological listening ledger
- legacy `topTracks.json` is treated as untimestamped historical count evidence, not play history
- current canonical slugs remain authoritative; legacy URL continuity is handled through compatibility artifacts
- `plays/` and `genres.json` from the website repo are excluded from import because they are synced copies of garden state
- `artists.json` is supplemental, not authoritative, unless it covers artists absent from `topArtists.json`
