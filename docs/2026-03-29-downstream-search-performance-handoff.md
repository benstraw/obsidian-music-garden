# Downstream Handoff: Search Performance and Legacy Genre URLs

## Purpose

This note is for the downstream Hugo/site repo.

It summarizes the decision we made after reviewing:

- the Search Console `Pages.csv` export for `/musical-genres/` pages
- the older 2024 Spotify-era artist dump in `../benstrawbridge.com/data/spotify/artists.json`
- the current `music-garden` canonical genre taxonomy

The goal is to preserve search-performing legacy URLs on the website **without** forcing `music-garden` to treat every old search-capture label as a canonical genre.

## Final Boundary Between Repos

`music-garden` owns:

- canonical music genre taxonomy
- genre aliases and pending taxonomy cleanup
- canonical editorial enrichment for real genres
- artist and release identity
- play-derived aggregation

The Hugo/site repo owns:

- preserving legacy URL equity
- rendering or redirecting search-performing legacy `/musical-genres/<slug>/` pages
- deciding whether a legacy slug should:
  - stay as a page
  - redirect to a canonical genre page
  - become a lighter “related music topic” page

We explicitly decided **not** to extend `music-garden` with downstream search-performance logic.

## Why

We found that many old `/musical-genres/` URLs were driven by raw Spotify artist genres from the older site-era data. Some of those are:

- real microgenres
- weak scene/style labels
- instrument or period labels
- outright non-genres

Examples:

- valid or defensible microgenre/style labels: `gbvfi`, `hyphy`, `crunk`
- non-genre or weak genre labels: `victorian-britain`, `water`, `ocean`, `supergroup`

Those labels may still have impressions or clicks in Search Console, but that does **not** mean they should all live in the canonical taxonomy.

## Key Decision

If a legacy slug has search value, preserve it in the website repo.

Do **not** force it back into `music-garden` unless it is genuinely a real genre worth keeping in the canonical taxonomy.

## Useful Inputs for the Hugo Repo

### 1. Search Console preserve set

Use the Search Console `Pages.csv` export to determine which legacy `/musical-genres/` URLs still have search value.

Operational rule:

- if a legacy genre URL has impressions or clicks, preserve that URL intentionally
- then decide whether to render, redirect, or lightly repurpose it

### 2. Old raw Spotify genre source

The older file:

- `../benstrawbridge.com/data/spotify/artists.json`

looked like a raw Spotify-era artist dump. It contains:

- artist identity
- Spotify URL and IDs
- play totals
- raw Spotify genre labels

This file is useful in the Hugo repo as a historical source for:

- which artists previously contributed to a legacy slug
- whether a weird slug was a real old Spotify genre label vs. accidental noise

Example:

- `gbvfi` was present there and attached to real artists, so it is not just garbage text

### 3. Current canonical preserve set from `music-garden`

For real genres that are still part of the clean taxonomy, the Hugo repo should prefer the current canonical output from `music-garden`.

Use canonical genre pages when available.

## Recommended Hugo-Side Policy

For each legacy `/musical-genres/<slug>/` URL with search value, classify it into one of three buckets.

### A. Canonical genre exists in `music-garden`

Example: `jazz`, `classical`, `post-punk`, `reggae`, `folk`

Recommendation:

- keep the URL live
- render from current canonical genre data
- if the slug changed, use a permanent redirect or alias to the canonical slug

### B. Not canonical in `music-garden`, but worth preserving for search

Examples may include:

- `gbvfi`
- `victorian-britain`
- other legacy search-performing slugs from Search Console

Recommendation:

- keep the URL in the Hugo repo
- do **not** push it back into `music-garden` unless you intentionally decide it belongs in the canonical genre taxonomy
- generate the page from legacy/raw sources or a lightweight manually curated template
- clearly link to the nearest canonical genre page where appropriate

This allows search preservation without polluting the canonical taxonomy.

### C. No canonical value and no meaningful search value

Recommendation:

- retire or redirect these over time

## Suggested Downstream Implementation

### Option 1: Legacy preserve manifest

Create a small Hugo-side data file, for example:

- `data/legacy-musical-genre-pages.json`

Each record can include:

- `slug`
- `mode`: `canonical`, `legacy`, or `redirect`
- `canonical_slug` if applicable
- `legacy_source`: `search_console`, `spotify_2024`, or both
- optional notes

Example shape:

```json
[
  {
    "slug": "gbvfi",
    "mode": "legacy",
    "legacy_source": ["search_console", "spotify_2024"],
    "notes": "Guided by Voices-influenced microgenre; preserve URL without adding to canonical taxonomy."
  },
  {
    "slug": "post-punk",
    "mode": "canonical",
    "canonical_slug": "post-punk"
  },
  {
    "slug": "victorian-britain",
    "mode": "legacy",
    "legacy_source": ["search_console"],
    "notes": "Search-preserve page, not a canonical genre."
  }
]
```

### Option 2: Redirect map

For legacy slugs that should resolve to a canonical genre:

- maintain a redirect/alias table in the Hugo repo
- do not add those aliases back into `music-garden` unless they are true genre aliases

### Option 3: Lightweight legacy page template

For preserved non-canonical slugs:

- render a simpler page
- state that it is a legacy music-topic / microtag page
- optionally show:
  - artists historically associated with the label
  - related canonical genres
  - a short editorial note

This keeps the URL useful without pretending it is a first-class canonical genre.

## Explicit Non-Goals for `music-garden`

Do not add Search Console preservation logic to `music-garden`.

Do not re-inflate canonical genre storage with old raw Spotify genre labels just because they ranked historically.

Do not make canonical taxonomy decisions based on downstream SEO alone.

## Practical Rule of Thumb

When updating Hugo generation:

- prefer `music-garden` canonical output for real genres
- preserve search-performing legacy URLs in Hugo
- treat preservation as a website concern, not a canonical taxonomy concern

## Short Version

`music-garden` should stay clean.

The Hugo repo should preserve legacy `/musical-genres/` URLs that still matter in search, even when those slugs are not part of the clean canonical genre taxonomy anymore.
