# Data Model

`obsidian-music-garden` is a file-based data pipeline for personal music
knowledge gardening.

It separates:

- raw source responses
- normalized source-specific records
- aggregated canonical records
- generated Obsidian markdown output

The current implementation writes `raw/`, `normalized/`, and `aggregated/`
records today.

## Principles

### 1. File-based and durable

The repo does not require a database. Runtime state is stored in JSON files
under `data/`, plus markdown written into the user's Obsidian vault.

### 2. Canonical IDs are garden-owned

Spotify is the current collector, but Spotify IDs are not the garden's primary
identity model.

The garden owns:

- `artist_slug`
- `release_slug`
- canonical genre slugs

Canonical genre definitions are curated in `data/genre-taxonomy.json`.

Source IDs are preserved for provenance and matching.

### 3. Source data and publication are separate concerns

This repository is for personal music knowledge gardening and markdown
generation. It should not include monetization or affiliate logic.

If a separate public website repository consumes this data, that downstream
layer should make its own editorial and monetization decisions without changing
the role of this repo.

## Directory layout

Runtime data is organized under `data/`:

```text
data/
  genre-taxonomy.json
  raw/
    spotify/
      recently-played/
      artists/
      top-artists/
    musicbrainz/
    wikipedia/
  normalized/
    artists/
    releases/
    genres/
  aggregated/
    artists/
    releases/
    genres/
  plays/
    YYYY/
      YYYY-WNN.json
  genres.json
```

Markdown output is written to the Obsidian vault, not to a repo-root
`content/` directory.

Current output paths:

```text
{vault}/music/
  listening/
  artists/
```

Future output may also include:

```text
{vault}/music/
  albums/
  genres/
```

For repo-local generated markdown that is safe to version and reuse downstream,
genre pages can also be rendered under:

```text
content/
  genres/
```

## Layer meanings

### Raw

`data/raw/` stores unchanged API responses.

Purpose:

- reproducibility
- debugging
- re-running normalization later without hitting the source again
- keeping source payloads distinct from canonical records

Examples:

- Spotify `recently-played` snapshots
- Spotify artist batch responses
- Spotify top-artist responses
- Wikipedia search and summary snapshots

These files are source-shaped and should not be treated as stable internal
schema.

### Normalized

`data/normalized/` is the source-cleaned layer.

Purpose:

- clean a source payload into a stable per-source record
- map field names into garden conventions
- carry source attribution and timestamps
- prepare records for aggregation

Current examples:

- normalized artist record from Spotify
- normalized release record from Spotify
- normalized track record from Spotify
- normalized genre alias record for Spotify source genres
- normalized editorial genre record from Wikipedia

### Aggregated

`data/aggregated/` stores canonical merged records.

Purpose:

- one canonical file per artist slug
- one canonical file per release slug
- one canonical file per track slug
- one canonical file per genre slug
- preserve source links without letting any one source define identity

This is the layer intended to support future downstream consumers, including a
separate public website repository.

### Plays

`data/plays/` stores sharded listening history.

Purpose:

- efficient append/merge for collection
- local note generation
- deterministic weekly partitioning

Each play record includes source provenance plus canonical artist/release IDs.
and now canonical track IDs.

## Current entity support

### Artist

Canonical identity:

- `artist_slug`

Current external IDs/fields:

- Spotify artist ID
- MusicBrainz artist ID field in schema
- Spotify URL

Current aggregated record includes:

- canonical slug
- display name
- source genres
- canonical genres
- images
- update timestamp

### Release / album

Canonical identity:

- `release_slug`

Current external IDs/fields:

- Spotify album ID
- MusicBrainz release ID field in schema
- MusicBrainz release-group ID field in schema

Current aggregated record includes:

- canonical slug
- release name
- primary artist slug/name
- source IDs
- update timestamp

### Genre

Canonical identity:

- repo-owned canonical genre slug

Genres are not allowed to inherit identity directly from Spotify or any other
single source.

Current aggregated record includes:

- canonical slug
- display title
- aliases
- pending review state
- editorial summary and image metadata when available
- listening stats from local Spotify play history
- top artists, releases, and tracks for the genre
- source references that explain where major fields came from

### Track

Canonical identity:

- `track_slug`

Current external IDs/fields:

- Spotify track ID
- MusicBrainz track/recording ID field in schema
- Spotify URL

Current aggregated record includes:

- canonical slug
- track name
- primary artist slug/name
- release slug/name
- source IDs
- update timestamp

## Source attribution and timestamps

The schema is designed to preserve provenance.

Current mechanisms:

- raw snapshots retain the full source response body
- play records include `source`
- canonical records keep source IDs and source URLs
- canonical metadata records include `last_updated`
- aggregated genre records include explicit `source_refs` for taxonomy,
  listening-derived fields, Wikipedia editorial fields, and MusicBrainz-linked
  cross-source enrichment

This should be extended over time with more explicit source-link arrays if the
number of upstream systems grows.

## Genre policy

Genre identity is controlled by the garden.

Rules:

- canonical genre slugs belong to this repo
- source genre strings are normalized through a curated alias map
- unknown source genres are recorded for review instead of silently becoming
  canonical truth

This prevents Spotify, MusicBrainz, Wikipedia, or any future source from
independently defining genre identity.

## Downstream website support

The data model is intentionally shaped so a separate public website repo can:

- consume canonical artist/release/genre records
- reuse markdown or derived JSON safely
- preserve source boundaries
- add editorial logic without turning this repo into a publishing or affiliate
  system

That downstream use is a design goal, but it should remain downstream. This
repository should stay focused on collection, canonicalization, storage, and
Obsidian-oriented output.

## Genre Aggregation Flow

Genre aggregation is intentionally file-based and deterministic.

Inputs:

- `data/genre-taxonomy.json` for canonical slug, title, aliases, parent, and notes
- `data/genres.json` for merged canonical artist, release, track, and genre metadata
- `data/plays/YYYY/YYYY-WNN.json` for local listening-derived counts
- normalized Wikipedia and MusicBrainz records that have already been folded into
  the canonical store

Output:

- one JSON file per canonical genre slug under `data/aggregated/genres/`

The aggregate is designed to be the handoff record for later markdown
generation and later public-site consumption without making this repository a
publishing system.

## Genre Markdown Generation

Genre markdown pages are generated from `data/aggregated/genres/*.json`.

Properties:

- deterministic file names based on canonical genre slug
- stable front matter for Obsidian and future static-site use
- no affiliate or public-site-specific presentation logic
- safe to write to a sandbox output directory for review before writing to
  `content/genres/`
