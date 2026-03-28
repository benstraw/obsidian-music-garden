# MusicBrainz Field Inventory

This document is a working inventory for future MusicBrainz enrichment in
`obsidian-music-garden`.

It is not a promise to collect every field listed here. The point is to make
field decisions explicit before the collector grows.

Use this file to answer four questions for any candidate field:

- do we want to fetch it at all
- should it stay raw only, or move into normalized / aggregated records
- is it useful only inside the garden, or also a candidate for downstream
  publishing
- does it carry any review burden around attribution, images, or external-link
  policy

## Decision Labels

Use these labels while curating fields:

- `ignore` — not useful enough to collect
- `raw-only` — worth capturing for audit/debug, but not yet mapped downstream
- `normalize` — map into `data/normalized/...`
- `aggregate` — carry into canonical aggregated records
- `publish-review` — potentially useful downstream, but should not be treated
  as automatically publishable

## Working Rules

- prefer stable IDs and factual metadata first
- treat tags / genres as hints, not as canonical genre authority
- keep image rights separate from metadata rights
- treat external store / marketplace links as reviewable, not automatically
  desirable
- if a field feels editorial, ambiguous, or contract-sensitive, mark it
  `publish-review`

## Artist Fields

| Field | Example / MusicBrainz shape | Why we might want it | Likely canonical destination | Suggested status | Notes |
|---|---|---|---|---|---|
| artist id | artist MBID | Stable cross-source identity | `artists[].musicbrainz_artist_id` | `aggregate` | Already in use |
| name | `name` | Human display fallback | canonical artist record | `aggregate` | Already in use as fallback only |
| sort name | `sort-name` | Sorting / display normalization | normalized artist record | `normalize` | Useful for downstream indexing, not required for note generation |
| disambiguation | `disambiguation` | Avoid common-name collisions | normalized artist record | `normalize` | Good for review UIs and future matching |
| type | `type`, `type-id` | Person / group / orchestra / choir, etc. | normalized + aggregated artist record | `normalize` | Useful factual metadata |
| gender | `gender`, `gender-id` | Optional person metadata | normalized artist record | `publish-review` | Probably low priority |
| country | `country` | Useful factual context | normalized + aggregated artist record | `normalize` | Reasonable downstream field |
| area | `area`, `begin-area` | Geographic context | normalized artist record | `normalize` | Good candidate, but choose one shape intentionally |
| life span | `life-span.begin`, `life-span.end`, `ended` | Factual chronology | normalized + aggregated artist record | `normalize` | Useful for website enrichment later |
| aliases | `aliases[]` | Better matching across sources | normalized artist record | `normalize` | High value for resolver quality |
| tags / genres | `tags[]`, `genres[]` | Genre hints and scene vocabulary | normalized artist + genre alias inputs | `aggregate` | Already partly in use; keep as hints only |
| annotation | `annotation` | Long-form editorial note | raw or normalized only | `publish-review` | Likely too editorial for default aggregation |
| relations | `relations[]` | Labels, members, URLs, works, places | raw-only initially | `raw-only` | Huge surface area; inventory before implementation |
| rating | `rating` | Weak signal at best | none | `ignore` | Low-value for this project |

## Release / Release Group Fields

| Field | Example / MusicBrainz shape | Why we might want it | Likely canonical destination | Suggested status | Notes |
|---|---|---|---|---|---|
| release-group id | release-group MBID | Stable album-level identity | `releases[].musicbrainz_release_group_id` | `aggregate` | Already in use |
| release id | release MBID | Edition-specific identity | `releases[].musicbrainz_release_id` | `aggregate` | Already in use |
| title | `title` | Name fallback / verification | canonical release record | `aggregate` | Already in use |
| primary type | `primary-type` | Album / single / EP, etc. | normalized + aggregated release record | `normalize` | High-value factual field |
| secondary types | `secondary-types[]` | Live / soundtrack / compilation, etc. | normalized release record | `normalize` | Useful for filtering and future UI |
| first release date | `first-release-date` | Chronology | normalized + aggregated release record | `normalize` | Good downstream field |
| disambiguation | `disambiguation` | Distinguish editions / similarly named releases | normalized release record | `normalize` | Helpful for matching |
| artist credit | `artist-credit[]` | Better release attribution than a single seed artist | normalized release record | `normalize` | Important if we later support compilations / multiple primaries |
| release status | `status` | Official / bootleg / promo, etc. | normalized release record | `normalize` | Useful but not urgent |
| country | `country` | Edition-specific factual metadata | normalized release record | `normalize` | Candidate, but think about edition-vs-group semantics |
| label info | `label-info[]` | Label enrichment | normalized release record | `publish-review` | Likely useful later |
| barcode / catalog number | `barcode`, `label-info.catalog-number` | Edition detail | normalized release record | `raw-only` | Probably more detail than the garden needs |
| tags / genres | `tags[]`, `genres[]` | Release-specific genre hints | normalized release + genre alias inputs | `aggregate` | Already partly in use |
| annotation | `annotation` | Editorial note | raw-only initially | `publish-review` | Not a default display field |
| release events | `release-events[]` | Market-specific dates and countries | raw-only initially | `raw-only` | High detail, low immediate value |

## Track / Recording Fields

The current pipeline does not deeply enrich tracks from MusicBrainz yet, but if
we expand there, these are the likely candidates.

| Field | Example / MusicBrainz shape | Why we might want it | Likely canonical destination | Suggested status | Notes |
|---|---|---|---|---|---|
| recording id | recording MBID | Stable cross-source track identity | `tracks[].musicbrainz_track_id` | `aggregate` | High-value future field |
| title | `title` | Track name verification | canonical track record | `normalize` | Straightforward |
| length | `length` | Factual validation against Spotify | normalized track record | `normalize` | Helpful consistency check |
| artist credit | `artist-credit[]` | Better credit fidelity | normalized track record | `normalize` | Important for featured artists later |
| ISRC | `isrcs[]` | Strong matching signal | normalized track record | `raw-only` | Valuable, but more collector work |
| tags / genres | `tags[]`, `genres[]` | Weak track-level genre hints | normalized track + genre alias inputs | `raw-only` | Probably lower value than artist/release hints |

## Image / Cover Fields

MusicBrainz itself is primarily metadata, but some image data may be reachable
through linked services such as the Cover Art Archive.

This is worth tracking separately from ordinary metadata fields.

| Field | Example / source | Why we might want it | Likely canonical destination | Suggested status | Notes |
|---|---|---|---|---|---|
| cover art presence | Cover Art Archive has image / front cover | Decide whether a release has cover data available | normalized release record | `normalize` | Good low-risk signal |
| cover art URLs | Cover Art Archive image URLs / thumbnails | Later display or review workflows | raw + normalized release image metadata | `publish-review` | Keep provenance attached |
| front / back / booklet types | Cover image types | Choose best candidate image | normalized release image metadata | `normalize` | Useful if we later support more than one image candidate |
| dimensions | width / height | Rendering and asset decisions | normalized release image metadata | `normalize` | Factual and low-risk |
| image comment | image comment / description | Human review aid | normalized release image metadata | `raw-only` | Helpful but secondary |
| approved publication status | human-reviewed downstream flag | Distinguish metadata from approved published media | aggregated release image metadata | `publish-review` | This would be our own field, not a MusicBrainz field |

### Cover-Art Notes

- Cover Art Archive data should not be treated as “just another text field.”
- Keep image URLs, source URLs, and retrieval timestamps explicit.
- If we later download binaries, that should be a separate decision and
  probably a separate pipeline step.

## External Link Fields

Some MusicBrainz entities include external links under URL relations.
Releases often have commerce-adjacent links, including store identifiers.

For example, a release can expose an Amazon ASIN relation like:

- release MBID: `a4864e94-6d75-4ade-bc93-0dabf3521453`
- external links may include Discogs and a US Amazon / ASIN value such as
  `B00001P4TH`

These links can be useful, but they need stricter review than ordinary factual
metadata.

| Field | Example / source | Why we might want it | Likely canonical destination | Suggested status | Notes |
|---|---|---|---|---|---|
| Discogs relation URL | URL relation | Cross-source research and verification | normalized external links | `normalize` | Useful, generally non-problematic as metadata |
| Amazon ASIN / link | URL relation / external link | Lookup convenience, commerce identifier | raw or normalized external links | `publish-review` | Do not treat as a default public field in this repo |
| Wikidata / Wikipedia relation | URL relation | Better cross-source graph resolution | normalized external links | `normalize` | High-value if we later expand artist/release editorial data |
| official homepage / social links | URL relations | Context and verification | normalized external links | `publish-review` | Probably useful later, but low priority now |
| streaming / purchase links | URL relations | Convenience only | none or normalized external links | `ignore` or `publish-review` | Keep this repo non-commercial and neutral |

### External-Link Notes

- Do not automatically surface Amazon or other purchase links in aggregated
  public-facing output.
- If we store them, treat them as metadata only.
- This repo should not drift into affiliate or commerce logic.

## Recommended Next Pass

If we expand MusicBrainz incrementally, the highest-value next fields are:

1. Artist aliases
2. Artist disambiguation
3. Artist type / country / life span
4. Release primary type / secondary types / first release date
5. Release cover-art presence and source-linked image metadata
6. Selected external links with conservative handling:
   Discogs first, Amazon only as reviewable metadata

## Suggested Implementation Sequence

1. Add desired fields to `docs/musicbrainz-field-inventory.md` first.
2. Mark each field `ignore`, `raw-only`, `normalize`, `aggregate`, or
   `publish-review`.
3. Expand the raw collector only after the field decision is documented.
4. Keep image and external-link handling explicit in `docs/sources.md` and
   `docs/licensing-notes.md`.
5. Add tests for any field that affects canonical identity or public output.
