# Music Atlas Revival — Music Garden v0.5 and Public Contract v1

**Date:** 2026-08-08
**Status:** Implemented on feature branches; production cutover blocked by source-policy review
**Target release:** Music Garden v0.5.0
**Public contract:** music-garden.public v1.0.0
**Consumer:** `benstraw/benstrawbridge.com`

**Status history:** Planned → In Progress → Implemented on feature branches;
production cutover blocked by source-policy review.

## Summary

Revive Music Garden as the producer for a personal listening atlas. The garden
collects and canonicalizes private listening data, detects intentional ordered
album listening, enriches only observed entities, and writes a privacy-safe,
versioned export. Consumer sites render that contract without reading internal
stores or raw play events.

Success requires rich artist, genre, and listened-release pages; compatibility
with existing public routes; exclusion of synthetic legacy plays and exact
timestamps; atomic consumer sync; and removal of the legacy website aggregator
after two successful weekly v1 syncs.

## Recorded Product Decisions

- The site is a personal listening atlas, not a general music encyclopedia.
- Page publication is evidence-tiered; unplayed related entities are embedded
  but do not receive standalone pages.
- Artist pages require three primary/album-artist plays across two local dates,
  or one qualifying album session. Featured credits do not qualify.
- Genre pages require five real plays, two eligible artists, and approved
  editorial metadata.
- Public releases represent MusicBrainz release groups. A release is published
  only after an ordered session covers at least half of the exact edition and
  at least three distinct tracks.
- Ordered sessions allow skips, at most two unrelated tracks, and gaps no longer
  than 30 minutes. Sessions may cross midnight.
- Public exports contain derived dates and aggregates, never exact timestamps.
- Synthetic 2024 backfill is quarantined and cannot affect public output.
- Biographies and descriptions are source-led; optional personal Markdown lives
  in the consumer repository.
- Artist relationships cover aliases and person/group membership in v1.
- Images use provenance-first sourcing: Wikimedia and Cover Art Archive with
  explicit review/attribution, and Spotify only as an attributed fallback.
- Tracked review decisions plus CLI commands are the editorial interface.
- Publication is weekly. The first post-foundation milestone is a separately
  planned discovery journal.

## Implementation

### Preserve and simplify

- Preserve the existing local history on
  `codex/revival-archive-2026-08-08`, then implement from `origin/main` on
  `codex/revival-foundation`.
- Carry forward canonical IDs, entity stores, source clients, multi-artist
  attribution, taxonomy, review behavior, and primary-only artist stub creation.
- Persist only real play shards, canonical catalogs, taxonomy, review decisions,
  runtime-ignored source caches, and the committed public export.
- Perform normalization and aggregation in memory. Do not commit raw,
  normalized, or aggregated cache trees.
- Store catalogs under `data/catalog/`, with one-release read compatibility for
  old root paths and existing environment overrides.

### Canonical model and review

- Artist records carry source IDs, aliases, type, country, lifespan, genres,
  biography, media, relationships, external URLs, and provenance.
- Genre records carry garden-owned taxonomy, editorial data, listening stats,
  top entities, parent/child links, and up to six listening neighbors computed
  by cosine similarity with at least two shared eligible artists.
- Release-group records retain exact editions, ordered disc/track lists, artist
  credits, release types/dates, source IDs, artwork, and provenance.
- Resolve automatically only by exact source identity/link. Search-only matches
  require tracked approval.
- Store review decisions under `data/reviews/` and expose unified `review list`,
  `review show`, `review approve`, and `review reject` commands. Preserve the old
  genre review commands as one-release aliases.
- Allow only PD, CC0, CC BY, and CC BY-SA Wikimedia media in v1. Galleries are
  manually approved and limited to six images.

### Album sessions

- Use only real, non-legacy events.
- Match tracks against the exact edition and flatten `(disc, track)` order.
- Require `max(3, ceil(track_count * 0.5))` distinct forward-moving tracks.
- Allow skipped tracks, no more than two unrelated tracks, and event gaps up to
  30 minutes. Ignore duplicate tracks for coverage; a backward jump ends the
  current candidate.
- Roll qualifying sessions up to the release group. Publish session date and
  coverage only, while retaining separately labeled lifetime isolated-track
  evidence.

### Public contract

Add `music-garden export --contract public-v1 --out data/published/v1` producing:

```text
manifest.json
overview.json
artists.json
genres.json
releases.json
redirects.json
weeks/YYYY/YYYY-WNN.json
```

The manifest carries contract version, privacy level, published-through date,
timezone, source revision, counts, dataset hash, and per-file hashes without a
non-deterministic wall-clock value. JSON Schema Draft 2020-12 schemas live under
`schemas/public/v1/`. Consumers accept v1 minor additions and reject unsupported
major versions. Export generation validates into a temporary directory before
atomically replacing the previous export.

### Consumer and cutover

- Add a website adapter that prefers validated `data/music_garden/v1/` and
  temporarily falls back to `data/spotify/`.
- Validate version, schemas, and hashes and run the production Hugo build before
  replacing consumer data.
- Generate rich artist, genre, release, and existing weekly pages with Hugo
  content adapters. Preserve old routes with exported aliases and redirects.
- Merge personal Markdown from
  `assets/listening-notes/{artists,genres,releases}/{slug}.md`.
- Dual-run legacy and v1 outputs, compare representative routes and counts, and
  remove the old importer only after two successful weekly v1 syncs.
- Publish the producer Monday at 06:00 UTC and sync the consumer at 07:00 UTC.

## Test and Acceptance Plan

- Unit-test identity resolution, eligibility, review overrides, genre neighbors,
  album-session boundaries, deterministic export, redirects, hashes, schema
  validation, stale cleanup, and privacy exclusions.
- Run `go vet ./...`, `go test ./...`, and a production Go build.
- Test valid/invalid consumer contracts, legacy fallback, route preservation,
  absence of synthetic weeks and exact timestamps, representative entity pages,
  authored-note merging, media attribution, CSP, responsive rendering, and the
  production Hugo build.
- A failed producer or consumer validation must leave the previous valid data
  unchanged.
- Complete v0.5.0 only after producer output and the real consumer build pass
  together. Append commits, PRs, deviations, final test evidence, and release
  information below.

## Implementation Record

### 2026-08-08 implementation

- Archived the pre-revival producer state, including its existing 30 commits
  and eight working-tree changes, at commit `5f827fe9` on the pushed branch
  `codex/revival-archive-2026-08-08`.
- Implemented producer work on `codex/revival-foundation` and consumer work on
  `codex/music-atlas-consumer-v1`, leaving the website's unrelated branch
  untouched.
- Pushed producer implementation commit `57d243d0` and consumer implementation
  commit `9732992`. Draft PR creation remains pending because the local GitHub
  CLI credentials require reauthentication.
- Moved authoritative stores under `data/catalog/` with environment overrides
  and one-release root-path fallback. Real 2026 play shards were enriched with
  primary/featured/album artists, positions, and 886 complete Spotify editions.
- Added canonical artist relationships/aliases/lifespan, release-group edition
  metadata, unified versioned review decisions, album-session inference,
  deterministic public-v1 export, hashes, Draft 2020-12 schemas/fixtures,
  privacy checks, atomic replacement, and Monday 06:00 UTC publication.
- The generated fixture publishes through 2026-08-07 with 124 indexed artists,
  45 indexed genres, 26 weekly shards, and zero releases. The zero is expected:
  no real observed album reaches the agreed three-track/50-percent threshold,
  even before sequence rules are applied.
- Added the website v1 validator, staged production build, atomic installation,
  07:00 UTC schedule, temporary legacy fallback, rich artist/genre/release
  adapters and layouts, authored-note merge points, redirects, CSP widening,
  and automated route/privacy/CSP checks.

### Deviations and blockers

- Scheduled publication is credential-free. Spotify enrichment remains an
  explicit producer operation; duplicating cached OAuth credentials into a new
  workflow was rejected as an unnecessary security-boundary expansion.
- Cover Art Archive rendering is structurally supported by the edition-specific
  release model and CSP but is not activated because there are currently no
  qualifying releases and CAA does not provide a blanket reuse license.
- The legacy website fallback is intentionally retained. Removing it requires
  two successful scheduled v1 syncs, which cannot be truthfully completed in a
  single implementation session.
- Final source-policy review found that Spotify's Developer Policy effective
  2025-05-15 prohibits analysis producing derived listenership metrics, usage
  statistics, or user profiles. The planned public atlas may fall within this
  restriction. Do not deploy, remove the fallback, or create the `v0.5.0` tag
  until written Spotify clarification or legal review resolves this blocker.

### Test evidence

- `go vet ./...` — passed.
- `go test ./...` — passed, including MusicBrainz/MediaWiki local-server tests,
  album-session cases, deterministic export, checksum/stale cleanup, and privacy.
- `go build -o music-garden .` — passed with default version `v0.5.0`.
- Real public-v1 export generated and checksum-validated; a repository-wide
  scan of the export found no `played_at`, exact RFC3339 timestamps, or
  `synthetic-legacy` marker.
- The real export and consumer production build passed together. Release tag,
  PR links, merge revisions, and two scheduled sync results remain pending due
  to the GitHub authentication, source-policy, and temporal cutover gates above.
