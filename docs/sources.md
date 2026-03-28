# Sources

This document tracks the external data sources used, or planned for use, by
`obsidian-music-garden`.

It is an implementation reference, not legal advice. The project itself is
open source and non-commercial, but some data may later flow into a separate
downstream website project. That means source boundaries need to stay explicit.

## Source Matrix

| Source | What we collect | Where it is stored | How it is used in the garden | Access / rate-limit notes | Attribution / licensing review notes | Garden-only or downstream? |
|---|---|---|---|---|---|---|
| Spotify | User-authorized listening history, artist/album/track metadata, Spotify URLs, Spotify images | `data/raw/spotify/`, `data/plays/`, `data/normalized/`, `data/aggregated/`, `data/genres.json` | Primary listening collector and source of local play history | OAuth required. Must respect Spotify platform rules and API behavior | Review before public or commercial downstream use. Keep Spotify provenance attached. Avoid treating Spotify media as free-floating assets | Garden-first. Downstream website use should be reviewed carefully |
| MusicBrainz | Artist and release-group metadata, IDs, aliases, relationships, tags/genres | `data/raw/musicbrainz/`, `data/normalized/`, `data/aggregated/`, `data/genres.json` | Cross-source identity enrichment and metadata normalization | No API key currently required, but a proper User-Agent and polite request rate are required | Core data is generally friendlier than many commercial APIs, but not every adjacent data class should be treated as equally safe. Cover-art and supplementary-data handling still need review | Suitable for garden use and often suitable downstream, but review exact fields before public reuse |
| Wikipedia / Wikimedia text | Page title, summary/extract, canonical URL, source attribution metadata | `data/raw/wikipedia/`, `data/normalized/genres/`, `data/normalized/artists/`, `data/aggregated/`, `data/genres.json` | Editorial seed data for genre and artist pages | No API key required. Requests should still be polite and cacheable | Text reuse requires attribution and may have share-alike implications. Treat summaries as reviewable editorial seed data, not automatically free publishing copy | Garden use is fine. Downstream website use should carry attribution and be reviewed |
| Wikimedia Commons images | File-page URL, image URL, thumbnail metadata, author/license/attribution metadata when available | `data/raw/wikipedia/commons-images/`, `data/normalized/`, `data/aggregated/`, `data/genres.json` | Candidate image metadata only. Not blanket-approved media ingestion | No API key required. Use sparingly and cache results | Review each file individually. Commons is not one license. Per-image attribution and downstream suitability vary by file | Review required before downstream website use |
| Soundcharts (planned / optional) | Likely charting, audience, playlist, radio, or social-performance metadata, depending on licensed access | Not implemented yet. Would likely live under `data/raw/soundcharts/`, `data/normalized/`, and selected aggregated records | Optional future enrichment only | Expect account-level access, contractual limits, and stricter API usage terms than open sources | Do not assume open redistribution rights. Any integration should start from contract and permitted-use review, not from technical convenience | Treat as restricted until a specific agreement says otherwise |
| setlist.fm | Setlists, venues, dates, source links | Not a core part of the genre pipeline, but used in the CLI and rendered to notes/stdout | Concert note assistance rather than canonical music-graph enrichment | API key required. Free API use is non-commercial unless separately approved | Preserve attribution. Do not assume public/commercial republishing rights by default | Garden-only unless separately reviewed |

## Working Notes by Source

### Spotify

What we collect:

- recently played track history
- artist, release, and track metadata returned by the Spotify Web API
- Spotify IDs and URLs
- Spotify artist images

Where it lives:

- raw snapshots under `data/raw/spotify/`
- sharded listening history under `data/plays/`
- normalized and aggregated records under `data/normalized/` and `data/aggregated/`
- canonical merged metadata in `data/genres.json`

How we use it:

- it is the current listening collector
- it seeds artist, release, and track identities
- it feeds local listening stats and top-entity summaries

Review posture:

- fine for personal garden workflows
- keep Spotify provenance visible
- public or commercial downstream use should be reviewed before launch

### MusicBrainz

What we collect:

- artist metadata
- release-group and release identifiers
- aliases and tag/genre hints

Where it lives:

- raw responses under `data/raw/musicbrainz/`
- normalized artist/release/genre records under `data/normalized/`
- merged identifiers and records under `data/aggregated/` and `data/genres.json`

How we use it:

- cross-source enrichment
- canonical identity support
- source-neutral matching against Spotify-shaped runtime data

Access notes:

- proper User-Agent required
- polite rate limiting required
- local caching is used so the same lookups are not repeated unnecessarily

Review posture:

- practical for canonical IDs and metadata enrichment
- still review exact field classes before downstream public use
- do not blur MusicBrainz core metadata together with unrelated cover-art or image rights

### Wikipedia / Wikimedia text

What we collect:

- page candidates from search
- matched page title
- canonical URL
- short summary/extract
- ambiguity or not-found status when no clean match exists

Where it lives:

- raw responses under `data/raw/wikipedia/`
- normalized editorial records under `data/normalized/genres/` and `data/normalized/artists/`
- merged genre and artist records under `data/aggregated/` and `data/genres.json`

How we use it:

- editorial seed data for genre and artist pages
- source-aware summary content for personal knowledge gardening

Review posture:

- use is straightforward inside the garden
- downstream website reuse should preserve attribution and re-check obligations around text reuse and adaptation
- ambiguity should remain explicit rather than silently picking a weak match

### Wikimedia Commons images

What we collect:

- image candidate metadata
- file page URL
- image URL / thumbnail URL
- author / license / attribution fields where available

Where it lives:

- raw Commons lookups under `data/raw/wikipedia/commons-images/`
- normalized and aggregated image metadata alongside the associated genre or artist record

How we use it:

- candidate media metadata only
- later review aid, not blanket approval for website publication

Review posture:

- every file should be treated individually
- keep license and attribution metadata attached
- do not assume an image is safe downstream just because it came from Commons

### Soundcharts (planned / optional)

What we would likely collect:

- chart and audience context
- playlist or radio footprint
- trend/performance indicators

Where it would live:

- likely `data/raw/soundcharts/`
- then source-specific normalized records
- then carefully selected aggregated fields, if any

How we would use it:

- optional future enrichment only
- not a foundational identity source

Review posture:

- assume contract-driven restrictions
- do not implement downstream reuse until access terms and redistribution rules are understood

## Practical Boundary

Use this repo to:

- collect data for personal knowledge gardening
- normalize and aggregate records
- generate markdown and internal review artifacts

Treat a downstream website repo separately when it comes to:

- public redistribution
- commercial use
- editorial adaptation of source text
- image publication
- contract- or policy-sensitive data reuse

## Primary References

- Spotify Developer Policy: <https://developer.spotify.com/policy>
- Spotify Design & Branding Guidelines: <https://developer.spotify.com/documentation/design>
- MusicBrainz Data License: <https://musicbrainz.org/doc/About/Data_License>
- MusicBrainz Database licensing breakdown: <https://musicbrainz.org/doc/MusicBrainz_Database>
- Wikimedia Commons reuse guidance: <https://commons.wikimedia.org/wiki/Commons:Reusing_content_outside_Wikimedia>
- Wikimedia Commons license summaries: <https://commons.wikimedia.org/wiki/Commons:Reusing_content_outside_Wikimedia/licenses>
- setlist.fm API docs: <https://api.setlist.fm/>
- setlist.fm Terms of Use: <https://www.setlist.fm/help/terms>
