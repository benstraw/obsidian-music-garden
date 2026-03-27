# Licensing Notes

This document turns source-policy research into repo-level working rules.

It is not legal advice. It is the default engineering posture for
`obsidian-music-garden` as of **March 27, 2026**.

## Default posture

The repo should separate:

- source-authorized private/personal garden use
- public redistribution
- commercial downstream publishing

Those are not the same rights.

## Working rules

### 1. Treat Spotify as source-restricted

Spotify is the current collector, but Spotify should not define the legal or
architectural center of the project.

Default rule:
- storing Spotify-derived data for personal garden use is acceptable
- public downstream reuse should be conservative
- commercial downstream use should be treated as **review required**

Engineering implications:
- keep Spotify IDs and URLs as provenance, not as the garden's only identity
- keep canonical slugs source-neutral
- if a public site consumes garden output, preserve clear source boundaries
- if Spotify metadata/artwork is displayed publicly, include Spotify-required
  attribution/linking treatment

### 2. Prefer MusicBrainz for canonical identity, not for blanket reuse

MusicBrainz is strong for canonical entity resolution, but licensing is not one
single blanket rule across all data classes.

Default rule:
- use MusicBrainz for IDs, aliases, and entity relationships
- do not assume every MusicBrainz-adjacent asset is equally safe for commercial
  downstream use
- review supplementary data and cover-art sources separately

Engineering implications:
- store MusicBrainz IDs in canonical records now
- keep image and cover-art sourcing decoupled from core metadata sourcing

### 3. Treat Wikipedia text and Commons images as different policy surfaces

Do not collapse "Wikipedia" into one reuse rule.

Default rule:
- text summaries/articles: attribution required; share-alike implications matter
- images: file-by-file review required

Engineering implications:
- store source page URLs for text excerpts/summaries
- store retrieved Wikipedia page title and canonical URL with every saved
  summary so downstream attribution can be attached later
- if an image is ever ingested, also store:
  - source file page
  - creator/author
  - license
  - attribution text
- do not ingest images without this metadata

### 4. Treat setlist.fm as non-commercial unless separately approved

Default rule:
- current API use is fine for personal tooling and note assistance
- commercial downstream use is not safe by default

Engineering implications:
- preserve the setlist.fm source link in any rendered output
- avoid building a downstream commercial setlist product on top of the free API

## Downstream policy

If a downstream site or product is:

- private/personal: lowest risk, but still preserve provenance
- public/non-commercial: attribution and license display become necessary
- public/commercial: source review is mandatory before launch

For this repo, assume:
- metadata can flow into the personal garden
- publication rights must be evaluated per source
- images are the highest-risk content class

## Image policy

If the repo starts storing images, require per-image metadata.

Minimum fields:

- `source`
- `source_url`
- `license`
- `author`
- `attribution_text`
- `commercial_use_ok` as an explicit reviewed decision, not an assumption

Default assumptions:

- Spotify images: no
- Wikimedia Commons images: maybe, after file review
- MusicBrainz-related cover art: separate review required

For the current genre-enrichment pipeline:
- Wikipedia summaries are editorial seed data, not automatically approved
  publishable copy
- Commons image metadata is useful for review, but downstream use still needs a
  file-level decision before public publication

## Safe defaults for implementation

When rights are unclear:

- keep the content as source-linked metadata only
- avoid copying long text
- avoid ingesting images
- prefer storing identifiers, URLs, and short factual fields
- document the uncertainty in code/docs instead of guessing

## Primary references

- Spotify Developer Policy: <https://developer.spotify.com/policy>
- Spotify Design & Branding Guidelines: <https://developer.spotify.com/documentation/design>
- MusicBrainz Data License: <https://musicbrainz.org/doc/About/Data_License>
- MusicBrainz Database licensing notes: <https://musicbrainz.org/doc/MusicBrainz_Database>
- Wikimedia Commons reuse guidance: <https://commons.wikimedia.org/wiki/Commons:Reusing_content_outside_Wikimedia>
- setlist.fm API docs: <https://api.setlist.fm/>
- setlist.fm Terms of Use: <https://www.setlist.fm/help/terms>
