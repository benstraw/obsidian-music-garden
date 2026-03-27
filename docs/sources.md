# Sources

This document tracks the external data sources used, or planned for use, by
`obsidian-music-garden`.

It is an engineering reference, not legal advice. If the project starts
shipping public/commercial downstream products, review the linked source terms
again before launch.

## Summary Matrix

| Source | What data we use or plan to use | Attribution needed? | Commercial downstream use allowed? | Images need per-image attribution? | Notes |
|---|---|---|---|---|---|
| Spotify | User-authorized listening history, artist/album/track metadata, Spotify URLs, Spotify-provided artwork/images | Yes | Treat as restricted. Limited commercial use exists for some non-streaming apps, but the policy is narrow and subject to Spotify platform rules | Usually no separate creator credit flow from Spotify itself, but Spotify branding/link-back rules apply whenever Spotify content/artwork/metadata is displayed | Must use Spotify branding for Spotify metadata; metadata/artwork must link back to Spotify |
| MusicBrainz | Artist/release/work metadata, IDs, relationships, aliases | Sometimes | Core data: yes. Supplementary data: not safely by default without separate commercial licensing or narrower use decisions | MusicBrainz itself is metadata, not image-first. Cover art comes from the Cover Art Archive and needs separate review | Distinguish core vs supplementary data |
| Wikipedia / Wikimedia text | Encyclopedic text summaries, article titles/URLs, structured reference data | Yes | Yes, if license obligations are met | Not applicable for text | Text reuse generally requires attribution and share-alike handling |
| Wikimedia Commons images | Artist/release/genre images hosted on Commons | Yes | Often yes, but depends on the specific file license and any non-copyright restrictions | Yes | Must inspect each file page; license can vary per image |
| setlist.fm | Concert setlists, venues, dates, source links | Yes | No, not by default. Free API use is non-commercial only unless separately approved | Not a primary image source here | API responses include attribution requirements and caching/storage limits |

## Current Policy by Source

### Spotify

- Use for: Spotify collector data and Spotify-provided metadata/images.
- Attribution: required when displaying Spotify metadata or artwork.
- Commercial downstream: assume **not safe by default** unless the specific use
  clearly fits Spotify's permitted non-streaming commercial uses and all other
  platform rules are satisfied.
- Image handling: do not treat Spotify images as standalone assets. Keep
  Spotify branding/link-back requirements attached.

Operational rule for this repo:
- Internal storage of Spotify-derived metadata is acceptable for personal garden
  workflows.
- Any public site or monetized downstream use should be reviewed separately.

### MusicBrainz

- Use for: canonical IDs, aliases, artist/release metadata, relationship data.
- Attribution: depends on whether the data used is core or supplementary.
- Commercial downstream:
  - core data: generally yes
  - supplementary data: not safely by default under the public license
- Image handling: do not assume MusicBrainz gives image reuse rights. Review
  Cover Art Archive or any other image source separately.

Operational rule for this repo:
- Prefer MusicBrainz core metadata for canonical identifiers.
- Avoid depending on supplementary data unless the exact field and license
  treatment are documented.

### Wikipedia / Wikimedia text

- Use for: short text summaries and article references.
- Attribution: required.
- Commercial downstream: yes, if attribution and share-alike obligations are
  satisfied.
- Image handling: text rules do not automatically apply to images.

Operational rule for this repo:
- Keep Wikipedia/Wikimedia text clearly source-labeled.
- If text is redistributed in public downstream content, carry attribution and
  review share-alike implications.

### Wikimedia Commons images

- Use for: artist/release/genre imagery where file-level license allows it.
- Attribution: commonly required.
- Commercial downstream: often allowed, but file-by-file review is mandatory.
- Per-image attribution: yes, assume yes unless the specific file is truly
  public domain or CC0 and there are no extra restrictions.

Operational rule for this repo:
- Never treat Commons as one blanket image license.
- Store source page, author, and license metadata with each reused image.

### setlist.fm

- Use for: setlists and concert metadata via the API.
- Attribution: required wherever setlist.fm data is shown.
- Commercial downstream: free API use is non-commercial only unless separately
  approved by setlist.fm.
- Image handling: not applicable for the current integration.

Operational rule for this repo:
- Preserve and display the attribution link provided by the API response when
  setlist.fm data is rendered or copied into downstream output.
- Do not rely on long-term storage of setlist.fm data beyond short caching
  without another policy review.

## Primary References

- Spotify Developer Policy: <https://developer.spotify.com/policy>
- Spotify Design & Branding Guidelines: <https://developer.spotify.com/documentation/design>
- MusicBrainz Data License: <https://musicbrainz.org/doc/About/Data_License>
- MusicBrainz Database licensing breakdown: <https://musicbrainz.org/doc/MusicBrainz_Database>
- Wikimedia Commons reuse guidance: <https://commons.wikimedia.org/wiki/Commons:Reusing_content_outside_Wikimedia>
- Wikimedia Commons license summaries: <https://commons.wikimedia.org/wiki/Commons:Reusing_content_outside_Wikimedia/licenses>
- Wikimedia Commons reuse contact note: <https://commons.wikimedia.org/wiki/Commons:Contact_us/Reuse>
- setlist.fm API docs: <https://api.setlist.fm/>
- setlist.fm Terms of Use: <https://www.setlist.fm/help/terms>
