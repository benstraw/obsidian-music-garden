# Genre-Only `genres.json` with Separate Artist and Release Stores

## Summary
Refactor the canonical store so `data/genres.json` owns only genre taxonomy/editorial state. Move artist and release identity into dedicated canonical files, preserving the current model but removing non-genre bulk from `genres.json`.

Chosen defaults:
- `genres.json` becomes genre-only
- artists and releases remain persisted canonically, but in separate stores rather than being derived from `plays/`
- `plays/` remains the only durable track-level source of truth
- the refactor includes backward-compatible loading from the current mixed `data/genres.json`

## Key Changes
### Store layout
Introduce three canonical store files:
- `data/genres.json`: `version`, `genre_aliases`, `pending_genre_aliases`, `genre_records`
- `data/artists.json`: `version`, `artists`, `artist_slug_aliases`, `artist_source_index`
- `data/releases.json`: `version`, `releases`, `release_source_index`

Remove artist and release ownership from `internal/genres.Store`. Replace it with:
- a genre-focused store/type in `internal/genres`
- dedicated artist and release store types
- a small composite runtime loader/saver used by commands so command UX does not change

### Compatibility and migration
Support the current mixed file as an input format during transition:
- if `artists.json` or `releases.json` is missing, load artist/release data from the current mixed `data/genres.json`
- first save after upgrade writes the three-file layout
- `genres.json` is rewritten without artist/release sections
- no data should be lost during the first migration save

Add one explicit maintenance command:
- `music-garden migrate-canonical-store`

Behavior:
- loads old mixed or already-split layout
- rewrites canonical state into the three-file layout deterministically
- prints counts for moved artist/release records and indexes
- is idempotent

### Command/runtime behavior
Keep current command names and workflows unchanged. Internally:
- commands that currently call `genres.Load` / `genres.Save` switch to a shared composite loader/saver
- play resolution continues to canonicalize artist and release identity, but writes to the artist/release stores instead of `genres.json`
- data-layer aggregation and workflow commands read genre state from the genre store and entity identity from the artist/release stores
- legacy import and backfill continue to populate artist/release canonical state, but no longer through a mixed `genres.json`

### File paths and environment
Keep `MUSIC_GENRES_PATH` for the genre store.
Add optional overrides:
- `MUSIC_ARTISTS_PATH`
- `MUSIC_RELEASES_PATH`

Default paths:
- `data/genres.json`
- `data/artists.json`
- `data/releases.json`

### Versioning and cleanup
On completion:
- bump the app minor version
- update docs/usage text so the canonical store layout is documented
- keep backward-compatible loading for at least one release cycle so older local repos can migrate cleanly

## Public Interfaces / Types
- `genres.Store` becomes genre-only
- add artist and release store types with `Load`, `Save`, and normalization helpers
- add composite runtime load/save helpers used by `main.go`
- add `music-garden migrate-canonical-store`
- add `MUSIC_ARTISTS_PATH` and `MUSIC_RELEASES_PATH`

## Test Plan
- mixed-layout `data/genres.json` loads successfully into the new composite runtime state
- first save from mixed layout writes:
  - genre-only `data/genres.json`
  - populated `data/artists.json`
  - populated `data/releases.json`
- repeated migration/save is idempotent
- play collection and resolution still canonicalize artists and releases correctly after the split
- data-layer sync still builds aggregated artists, releases, and genre views correctly
- legacy import/backfill still populates artist/release identity correctly
- CLI commands that read canonical data continue to work unchanged
- build/test/vet all pass after the refactor

## Assumptions and Defaults
- `plays/` remains the only durable track-history source
- artist and release identity are still worth persisting canonically in this repo
- SEO or legacy website preservation remains out of scope here
- current local canonical curation in `genres.json` is authoritative and must survive the migration unchanged
