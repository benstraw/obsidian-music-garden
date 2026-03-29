# Commands

All commands use this runtime path precedence:
1. CLI flags (where applicable)
2. Environment variables
3. `MUSIC_STATE_DIR` files (`.env`, `tokens.json`) unless `MUSIC_PLAYS_DIR` and/or `MUSIC_GENRES_PATH` override the data paths
4. Current working directory fallback (with warning)

OAuth tokens auto-refresh if they are expiring within 5 minutes.

---

## auth

```bash
./music-garden auth
```

Runs the full OAuth 2.0 authorization code flow:

1. Builds a Spotify authorization URL with the required scopes
2. Opens it in your default browser (macOS `open`)
3. For **localhost redirect URIs**: starts a local HTTP server on `:8888` to
   capture the callback automatically
4. For **external redirect URIs**: prompts you to paste the full redirect URL
   from your browser's address bar
5. Exchanges the authorization code for access and refresh tokens
6. Saves tokens to the effective `tokens.json` path (mode `0600`)

If the browser does not open automatically, the full auth URL is printed to
stdout — copy and paste it manually.

Tokens auto-refresh on subsequent commands. You should only need to run `auth`
once, unless `tokens.json` is deleted or the refresh token expires.

This command is source-specific: it authenticates the current Spotify collector,
not the garden as a whole.

**Requires:** `SPOTIFY_CLIENT_ID`, `SPOTIFY_CLIENT_SECRET`, `SPOTIFY_REDIRECT_URI` in `.env` or environment.

---

## collect

```bash
./music-garden collect
```

Fetches the last 50 recently-played tracks from the Spotify API and merges
them into the weekly shard file for the current ISO week under the effective
plays directory.

**Behaviour:**
1. Calls `GET /me/player/recently-played?limit=50`
2. Filters out podcast episodes (items with no `track` key)
3. On first run after upgrade: migrates `data/plays.json` → sharded layout and renames the legacy file to `data/plays.json.bak`
4. Resolves canonical `artist_slug` and `release_slug` values while preserving Spotify IDs for provenance
5. Routes each new play to its ISO week file (`data/plays/YYYY/YYYY-WNN.json`), merging with the existing file
6. Deduplicates by `played_at` — existing plays are never duplicated
7. If `MUSIC_AUTO_DAILY_ON_COLLECT_SPOTIFY=1`, regenerates today's daily note
   (`spotify-YYYY-MM-DD.md`) so it stays up to date as new plays arrive
8. Writes unchanged Spotify API snapshots to `data/raw/spotify/` and refreshes
   canonical aggregated records under `data/aggregated/`

**Output:** `{playsDir}/YYYY/YYYY-WNN.json` (e.g. `data/plays/2026/2026-W11.json`)

**Additional persisted data:**
- `data/raw/spotify/recently-played/` — unchanged collect snapshots
- `data/raw/spotify/artists/` — unchanged artist batch snapshots fetched during metadata hydration
- `data/aggregated/artists|releases|genres/` — canonical merged records

Since Spotify only returns the last 50 plays, running `collect` 5× daily
ensures no plays are lost to the 50-track API cap.

The files under `data/plays/` are the durable listening ledger for the garden.
They are not raw Spotify payloads; they are the project-shaped play records
that downstream enrichment, aggregation, and markdown generation build on.

---

## repair-plays

```bash
./music-garden repair-plays
```

Rewrites existing weekly shards under `data/plays/` so older thin records pick
up the richer canonical fields now supported by the project, including source,
canonical artist/release/track slugs, and any resolvable album identifiers.

Use this after schema upgrades or if your historical play files predate the
current canonical play model.

---

## backfill-play-artists

```bash
./music-garden backfill-play-artists [--from-year YYYY] [--limit N] [--dry-run] [--verbose]
```

Enriches existing sharded play history by fetching full Spotify track metadata
for plays that are still missing `additional_artists` or `album_id`.

This command is intentionally separate from `collect`. It is a one-shot or
occasional maintenance pass for historical play shards.

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--from-year` | all years | Only rewrite plays whose `played_at` starts with this year |
| `--limit` | unlimited | Max unique track IDs to fetch from Spotify |
| `--dry-run` | false | Report candidate updates without writing |
| `--verbose` | false | Print each fetched track while backfilling |

---

## weekly

```bash
./music-garden weekly [--date YYYY-MM-DD] [--out-dir DIR]
```

Generates a weekly markdown note for the ISO week (Mon–Sun) containing the
given date (default: the current week).

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--date` | today | Any date within the target week |
| `--out-dir` | vault root | Override the output root for safe sandbox generation |

**What it does:**
1. Determines the ISO week (Monday 00:00 → Sunday 23:59 local time)
2. Filters the effective sharded play history for plays that fall within the week
3. Creates artist stubs for new artists (see below)
4. Writes the weekly note (always overwrites if it already exists)

**Output:** `{outputRoot}/music/listening/spotify-YYYY-Www.md`

**Weekly note sections:**
- YAML frontmatter (`type: note`, `tags: [music, weekly-music]`, `created`, `week`)
- Stats block: play count, unique tracks/artists/albums, total listening time
- Repeated Tracks (≥2 plays in the week)
- Albums This Week (sorted by play count)
- Artists in Rotation (wikilinks, sorted alphabetically)
- New Artists (first appearance — no stub existed before this run)
- Notes (empty section)

**Artist stubs** — created at `{outputRoot}/music/artists/{Name}.md` for every
artist in the week's plays. Never overwrites an existing stub. Each stub
includes frontmatter (`type: resource`, `tags: [music/artist]`, `artist_slug`,
`spotify_artist_id`, `musicbrainz_artist_id`, `spotify_url`, `genres`) and a
dataview query that lists all weekly notes linking to the artist.

---

## daily

```bash
./music-garden daily [--date YYYY-MM-DD] [--out-dir DIR]
```

Generates a daily markdown note for the given calendar date (default: today).

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--date` | today | Date in `YYYY-MM-DD` (interpreted in local timezone) |
| `--out-dir` | vault root | Override the output root for safe sandbox generation |

**Behaviour:**
1. Loads the effective sharded play history
2. Filters plays for the local calendar day
3. Creates missing artist stubs for artists heard that day
4. If no plays exist for that day, exits without writing a file
5. If a note already exists, skips (never overwrites)
6. Otherwise writes the daily note

**Output:** `{outputRoot}/music/listening/spotify-YYYY-MM-DD.md`

**Daily note sections:**
- YAML frontmatter (`type: note`, `tags: [music, daily-music]`, `created`, `date`)
- Stats block: play count, unique tracks/artists/albums, total listening time
- Play Log with local times, track, artist wikilink, album
- Songs Played (all song+artist+album combinations with play counts)
- Artists Played (all artists with play counts)
- Albums Played (all album+artist combinations with play counts)
- Notes (empty section)

**Artist stubs:** `daily` also creates missing artist stubs at
`{outputRoot}/music/artists/{Name}.md` for artists heard on that day.

---

## catch-up

```bash
./music-garden catch-up [--weeks N] [--out-dir DIR]
```

Scans the vault's listening directory for missing weekly and daily notes and
generates only what is missing. Existing notes are never overwritten.

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--weeks` | 8 | Number of weeks back to scan |
| `--out-dir` | vault root | Override the output root for safe sandbox generation |

**Behaviour:**
1. Checks for `spotify-YYYY-Www.md` in `{outputRoot}/music/listening/` for each
   of the last N weeks
2. Generates missing weekly notes in chronological order (oldest first)
3. Loads the effective plays file once, then checks the last `N*7` days for missing
   `spotify-YYYY-MM-DD.md` files
4. Generates missing daily notes (skips days with no plays)

This is the preferred command for the scheduled Sunday run — it fills any
gaps from missed `collect` windows without overwriting notes you have already
edited.

---

## import-legacy

```bash
./music-garden import-legacy --source-dir /path/to/benstrawbridge.com/data/spotify [--dry-run] [--verbose] [--audit-genres]
```

Imports legacy Spotify snapshot data from the older website repository into the
garden's canonical store.

**Import scope:**
- `topArtists.json`
- `snapshot-2024-06.json`
- `artists.json` as a supplemental source
- `topTracks.json` as untimestamped historical count data plus track/release/artist enrichment

**Explicitly excluded:**
- `plays/` because those shards are synced copies of the garden ledger
- `genres.json` because it is a synced copy of the garden store

`topTracks.json` does **not** create synthetic play records. Repeated track
entries are preserved as `legacy_play_count` on canonical track records rather
than being written into `data/plays/`.

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--source-dir` | required | Legacy `data/spotify` directory |
| `--dry-run` | false | Report import counts without writing store or compatibility files |
| `--verbose` | false | Print imported artists and tracks |
| `--audit-genres` | false | Report unresolved genre labels and legacy website genre slugs |

This command also writes compatibility artifacts for downstream URL continuity:
- `data/legacy-artist-slugs.json`
- `data/legacy-genre-slugs.json`

---

## persona

```bash
./music-garden persona [--out-dir DIR]
```

Regenerates the Music Taste context pack at
`{outputRoot}/01-ai-brain/context-packs/Music Taste.md` (always overwrites).

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--out-dir` | vault root | Override the output root for safe sandbox generation |

**What it fetches:**
- Top 50 artists for `short_term` (~4 weeks), `medium_term` (~6 months), `long_term` (all time)
- This week's plays from the effective plays file for the Recent Rotation section
- Unchanged Spotify top-artist responses are stored under `data/raw/spotify/top-artists/`

**Context pack sections:**
- Current Top Artists (last ~4 weeks)
- Top Artists (last ~6 months)
- All-Time Top Artists
- Top Genres (derived from short_term artist genres, deduplicated, up to 15)
- Recent Rotation (unique artists heard this week, sorted)
- Notes (empty)

Intended to be read by AI assistants when creating playlists, recommending
music, or discussing musical taste.

---

## genre-backfill

```bash
./music-garden genre-backfill
```

Backfills canonical artist metadata in `data/genres.json` from existing play history.

**Behaviour:**
1. Loads all effective play shards
2. Loads the effective canonical metadata store
3. Finds Spotify artist IDs present in play history but missing canonical genre metadata
4. Fetches artist details from Spotify in batches of 50
5. Writes canonical artist genres, source genres, release records, and artist images into `data/genres.json`
6. Updates existing artist stubs in the vault with any newly cached genres
7. Stores unchanged Spotify artist batch responses in `data/raw/spotify/artists/`
   and rewrites aggregated canonical records in `data/aggregated/`

Use this after importing historical plays or if `collect` ran before the canonical metadata store existed.

---

## image-backfill

```bash
./music-garden image-backfill
```

Fetches Spotify artist profile images for canonical artist records that currently
have no `images` entry in `data/genres.json`.

**Behaviour:**
1. Loads the effective canonical metadata store
2. Finds canonical artist records whose `images` array is missing or empty
3. Fetches artist details from Spotify in batches of 50
4. Updates only the `images` field for those cache entries
5. Stores unchanged Spotify artist batch responses in `data/raw/spotify/artists/`
   and rewrites aggregated canonical records in `data/aggregated/`

This command is metadata-only: it updates `data/genres.json` but does not modify
weekly notes, daily notes, or artist stubs.

---

## genre-report

```bash
./music-garden genre-report
```

Loads the curated canonical genre taxonomy from `data/genre-taxonomy.json` and
compares it with the unknown source labels currently tracked in `data/genres.json`.

The report lists:

- known mappings
- unknown source labels
- collisions where the same normalized alias points at multiple canonical slugs

Use this command while curating genre definitions and aliases.

---

## genre-review

```bash
./music-garden genre-review --slug indie-rock
./music-garden genre-review --queue [--limit 20]
```

Surfaces the editorial evidence bundle for one canonical genre, or prints a
ranked queue of draft mapped genres that are strongest promotion candidates.

`--slug` prints:

- taxonomy state (`mapped` or `pending`)
- workflow state (`draft` or `publishable`)
- aliases
- Wikipedia title / URL / summary presence
- image presence
- listening stats
- top artist/release/track counts
- whether an aggregated record exists
- whether a generated markdown page already exists in the target output dir

`--queue` prints the top draft review candidates ranked by:

1. local play count
2. enrichment completeness (summary, image, parent)
3. slug

---

## genre-promote

```bash
./music-garden genre-promote --rule ready-basic [--dry-run] [--limit 25] [--min-plays N]
./music-garden genre-promote --slug indie-rock --state publishable
./music-garden genre-promote --slug indie-rock --state draft
```

Sets the editorial workflow state used by batch genre-page generation.

Supported states:

- `draft`
- `publishable`

Supported rules:

- `ready-basic`
  - taxonomy mapped
  - not pending
  - has Wikipedia summary
  - at least 5 plays by default
- `ready-loose`
  - taxonomy mapped
  - not pending
  - has Wikipedia summary **or** at least 10 plays by default
- `any-play`
  - taxonomy mapped
  - not pending
  - at least 1 play by default
- `any-play-rich`
  - taxonomy mapped
  - not pending
  - at least 1 play by default
  - has Wikipedia summary
  - has image
- `no-play-rich`
  - taxonomy mapped
  - not pending
  - has Wikipedia summary
  - has image

Rule-based promotion creates minimal canonical genre records in
`data/genres.json` when a mapped aggregated genre does not already have one.

Manual promotion of a taxonomy-pending genre is allowed, but batch page
generation still skips it until the taxonomy issue is resolved.

---

## aggregate-genre

```bash
./music-garden aggregate-genre --slug indie-rock
```

Rebuilds one canonical genre aggregate under `data/aggregated/genres/` by
combining:

- canonical taxonomy fields from `data/genre-taxonomy.json`
- local canonicalized Spotify listening history from `data/plays/`
- MusicBrainz identifiers already merged into canonical artist/release records
- Wikipedia editorial summary and image metadata already merged into
  `data/genres.json`

The output record is deterministic and intended for later Obsidian markdown
generation or downstream website consumption.

---

## aggregate-genres

```bash
./music-garden aggregate-genres
```

Batch rebuilds every known canonical genre aggregate in
`data/aggregated/genres/`.

Use this after:

- substantial listening imports
- MusicBrainz enrichment runs
- Wikipedia genre enrichment runs
- taxonomy curation changes

---

## sync-data-layer

```bash
./music-garden sync-data-layer
```

Rebuilds the full file-based data layer from the canonical store and sharded
play history.

This refreshes:

- `data/normalized/artists/`
- `data/normalized/releases/`
- `data/normalized/tracks/`
- `data/normalized/genres/`
- `data/aggregated/artists/`
- `data/aggregated/releases/`
- `data/aggregated/tracks/`
- `data/aggregated/genres/`

Use this after structural metadata changes when you want artist, release, and
track aggregates refreshed alongside genres.

---

## generate-genre-pages

```bash
./music-garden generate-genre-pages [--out-dir ./content/genres] [--slug indie-rock] [--limit 2]
```

Renders Obsidian-friendly markdown genre pages from
`data/aggregated/genres/*.json`.

Default output is repo-local `content/genres/`. For safe review runs, pass a
different `--out-dir`, for example:

```bash
./music-garden generate-genre-pages --out-dir ./sandbox/music/genres --limit 2
```

Batch generation only renders genres that are both:

- not taxonomy-pending
- marked `publishable`

To force generation for a draft or pending genre during inspection, pass its
slug explicitly with `--slug`.

Each page includes front matter, a concise summary, listening stats, top local
artists/albums/tracks, source notes, and related genre links when available.

---

## setlist

```bash
./music-garden setlist <artist> [--date YYYY-MM-DD]
```

Looks up a setlist on setlist.fm and prints it to stdout. No vault files are written.

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--date` | today | Date of the concert in YYYY-MM-DD format |

**Requires:** `SETLISTFM_API_KEY` in `.env`.
Get a key at [setlist.fm/settings/apps](https://www.setlist.fm/settings/apps).

**Output format:**

```
Artist Name — Venue Name — City, ST
2026-02-21

Set 1:
1. Song Title
2. Song Title
...

Encore:
1. Song Title

Setlist.fm: https://www.setlist.fm/setlist/...
```

**Concert note workflow:**
1. During or after a show, open the Templater template `Concert Note` in Obsidian — it prompts for artist and venue, then renames the file to `YYYY-MM-DD - Artist - Venue.md` and places it in `music/concerts/`
2. Run `music-garden setlist "<Artist>" --date YYYY-MM-DD`, copy the output, and paste it into the Set List section of the note
3. The artist stub's Concerts Dataview block will automatically pick up the new note via the `music/live-artist/<Artist Name>` tag

---

## musicbrainz-enrich-artist

```bash
./music-garden musicbrainz-enrich-artist --spotify-id <spotify-artist-id> --name "Artist Name"
```

Looks up a MusicBrainz artist from a Spotify seed, saves the raw search and
lookup payloads under `data/raw/musicbrainz/`, writes a normalized MusicBrainz
artist record under `data/normalized/artists/`, and merges the result into the
canonical metadata store in `data/genres.json`.

MusicBrainz genres/tags are also passed through the curated canonical genre taxonomy,
so this command can enrich both artist identity and genre metadata.

---

## musicbrainz-backfill-artists

```bash
./music-garden musicbrainz-backfill-artists [--limit N] [--refresh]
```

Walks canonical artists in `data/genres.json` and runs MusicBrainz enrichment in
batch. By default it skips artists that already have a MusicBrainz artist ID.

Use `--refresh` to force re-enrichment of already matched artists.

---

## musicbrainz-enrich-album

```bash
./music-garden musicbrainz-enrich-album --artist "Artist Name" --name "Album Name" --spotify-album-id <spotify-album-id>
```

Searches MusicBrainz release-groups for an album seed, stores the raw search
and lookup payloads under `data/raw/musicbrainz/`, writes a normalized
MusicBrainz release record under `data/normalized/releases/`, and merges the
release-group identifiers into the canonical metadata store in `data/genres.json`.

Release-group genres/tags are also normalized into canonical genre records
under `data/normalized/genres/`.

---

## musicbrainz-backfill-albums

```bash
./music-garden musicbrainz-backfill-albums [--limit N] [--refresh]
```

Walks canonical releases in `data/genres.json` and runs MusicBrainz release-group
enrichment in batch. By default it skips releases that already have a
MusicBrainz release-group ID.

Use `--refresh` to force re-enrichment of already matched releases.

---

## wikipedia-backfill-genres

```bash
./music-garden wikipedia-backfill-genres [--limit N] [--refresh] [--slug canonical-slug]
```

Walks known canonical genre slugs and runs Wikipedia/Wikimedia enrichment in
batch. By default it skips genre records that already have a matched Wikipedia
page.

Use `--refresh` to force re-enrichment of already matched genre pages.
Use `--slug` to refresh one canonical genre directly.

---

## wikipedia-backfill-artists

```bash
./music-garden wikipedia-backfill-artists [--limit N] [--refresh]
```

Walks canonical artists and runs Wikipedia/Wikimedia enrichment in batch. By
default it skips artist records that already have a matched Wikipedia page.

Use `--refresh` to force re-enrichment of already matched artist pages.

---

## doctor

```bash
./music-garden doctor
```

Prints effective runtime configuration and diagnostics in one place:

1. Working directory and executable path
2. Effective `.env`, `tokens.json`, `data/plays/` (plays dir), `data/plays.json` (legacy, if present), templates, vault/listening paths
3. Effective `data/genres.json` path and any data-path overrides
4. State-dir fallback warnings
5. Launchd labels and expected log paths
6. Best-effort loaded/not-loaded launchd job status

Exit code is `0` when no issues are found and nonzero when warnings/errors are detected.
