# Licensing Notes

This document translates source-policy research into working implementation
rules for `obsidian-music-garden`.

It is not legal advice. It is a practical engineering posture for an open
source, non-commercial garden repository that may later feed a separate public
website project.

## Core Boundary

Keep three situations distinct:

- personal garden use inside this repo
- public downstream publishing
- commercial downstream publishing

Those are different risk levels, and they should not be collapsed into one
assumption.

## Default Posture

### 1. This repo is non-commercial and source-aware

Default assumption:

- storing source-linked metadata and summaries for personal knowledge gardening
  is the main purpose here
- this repo should remain cautious, neutral, and non-commercial in itself

Implementation consequence:

- preserve provenance
- keep source responses in ignored runtime caches and retain provenance in the
  canonical catalog
- avoid burying source origin when records are canonicalized or published

### 2. Downstream website use is a separate review step

Default assumption:

- a future public website may consume selected outputs from this repo
- that does not automatically mean every collected field is ready for public or
  commercial reuse

Implementation consequence:

- canonical and published records should keep source references
- image and text fields should remain reviewable
- contract- or policy-sensitive sources should be clearly labeled

## Source-by-Source Notes

### Spotify

Working posture:

- fine for personal garden collection and metadata enrichment
- conservative for downstream public reuse
- review required before commercial downstream launch

Implementation notes:

- keep Spotify IDs, URLs, and images as source-linked metadata
- do not treat Spotify as the garden's canonical authority
- if Spotify-derived metadata or artwork is displayed publicly, preserve
  required link-back / branding treatment

### MusicBrainz

Working posture:

- very useful for canonical identity and metadata enrichment
- generally more downstream-friendly than Spotify
- still review exact field classes before public or commercial reuse

Implementation notes:

- use for artist/release IDs, aliases, and metadata enrichment
- keep image rights separate from metadata rights
- where uncertainty exists, prefer storing identifiers and provenance rather
  than making broader reuse assumptions

### Wikipedia text

Working posture:

- useful as editorial seed data inside the garden
- suitable for downstream use only if attribution and other obligations are
  handled correctly

Implementation notes:

- store page title, URL, and retrieval context with summaries
- treat summary reuse as something to review, not as automatically settled
- keep ambiguity and not-found statuses explicit

### Wikimedia Commons images

Working posture:

- candidate metadata is useful
- public reuse requires file-by-file review

Implementation notes:

- never treat Commons as one blanket image license
- keep file page URL, author, license, and attribution metadata attached
- avoid implying `commercial_use_ok` unless a human has reviewed that file

### Soundcharts (planned / optional)

Working posture:

- assume restricted or contract-governed use unless specific terms say otherwise

Implementation notes:

- do not treat Soundcharts as an open data source
- if implemented, keep fields clearly source-scoped
- review redistribution rights before any downstream website use

## Review Triggers

Stop and review before downstream website use when:

- the field came from Spotify
- the field is reused editorial text from Wikipedia
- the field is image metadata that might lead to image publication
- the source is contractual or gated, such as a future Soundcharts integration
- the planned site is public or commercial

## Safe Implementation Defaults

When rights or downstream suitability are unclear:

- keep the data as metadata, not as promoted published content
- store identifiers, URLs, timestamps, and provenance first
- keep text excerpts short and source-linked
- keep image metadata reviewable but separate from publication approval
- document uncertainty rather than hiding it

## Public-v1 source-policy review (2026-08-08)

The revival implementation was checked against the official source policies
before production cutover:

- MusicBrainz core database data is CC0. Supplementary data is CC BY-NC-SA
  3.0, so public-v1 must classify fields and avoid assuming every MusicBrainz
  response is core data.
- Wikimedia Commons requires file-by-file verification and compliance with
  author credit, license identification/linking, and ShareAlike where
  applicable. Public-v1 therefore accepts only PD, CC0, CC BY, and CC BY-SA
  candidates with preserved provenance and tracked approval.
- Cover Art Archive identifies art by exact MusicBrainz release ID, but its
  policy says reuse is at the user's risk; CAA origin is not a blanket copyright
  license. Artwork must remain edition-specific and approved before display.
- Spotify requires attribution and a link back whenever Spotify metadata or
  cover art is shown. More importantly, the Developer Policy effective
  2025-05-15 prohibits analyzing Spotify Content or the service to create
  derived listenership metrics, usage statistics, or user profiles. The atlas's
  derived play counts and listening pages may fall within that restriction.

**Cutover blocker:** do not deploy public-v1 or create the `v0.5.0` release tag
until the Spotify-derived public metrics use case receives written policy
clarification or a separate legal review. The implementation may remain on
feature branches and its privacy/contract mechanics may continue to be tested.

Official references:

- <https://musicbrainz.org/doc/About/Data_License>
- <https://commons.wikimedia.org/wiki/Commons:Reusing_content_outside_Wikimedia/en>
- <https://musicbrainz.org/doc/Cover_Art_Archive>
- <https://developer.spotify.com/policy>

## Reminder

These notes are intentionally cautious. They are meant to help with design and
implementation choices, not to replace source-term review at launch time.
