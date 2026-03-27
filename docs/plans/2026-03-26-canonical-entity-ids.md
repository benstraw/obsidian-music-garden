# Plan: Canonical Artist, Release, and Genre IDs

**Date:** 2026-03-26  
**Status:** Implemented in `v0.6.0`

## Summary

Introduce garden-owned canonical IDs for artists, releases, and genres so the
project no longer treats Spotify IDs and raw Spotify genre strings as the
primary internal model.

## Implemented Changes

- Added canonical `artist_slug` and `release_slug` fields to persisted play
  records while preserving Spotify IDs and URLs as source metadata.
- Replaced the old artist-ID keyed `genres.json` cache with a canonical
  metadata store containing:
  - canonical artist records
  - canonical release records
  - source ID → canonical slug indexes
  - a curated genre alias table
  - pending/unmapped source genres for later review
- Added source-to-canonical resolution logic so collection canonicalizes plays
  before persistence.
- Added in-place migration for existing sharded play files so older
  Spotify-shaped records are rewritten with canonical fields on first run.
- Updated artist stub frontmatter to include canonical identity fields:
  `artist_slug`, `spotify_artist_id`, and `musicbrainz_artist_id`.
- Updated CLI/docs framing to describe music-garden as a multi-source garden
  with Spotify as the current collector rather than the entire product model.

## Defaults Chosen

- Canonical IDs are stable name-based slugs.
- Release identity is an internal release slug first, not a source ID.
- MusicBrainz fields exist in the schema now but are not populated by a
  resolver in this phase.
- Genre identity is controlled by a repo-owned alias map; unmapped source
  genres are recorded for review instead of silently becoming canonical.

## Tests Added or Updated

- Fetch mapping now verifies source and album ID fields.
- Genres/store tests cover:
  - legacy cache migration
  - canonical genre normalization
  - pending alias capture
  - canonical play resolution
- Plays tests cover in-place canonical migration of sharded files.
- Render tests now verify canonical genre slugs and artist stub metadata
  updates.
