# Genre Aggregation Pipeline

Date: 2026-03-27

## Goal

Build deterministic aggregated genre records that combine:

- canonical genre taxonomy
- local Spotify listening history
- MusicBrainz-enriched identifiers already merged into canonical artist and release records
- Wikipedia editorial summary and image metadata already merged into canonical genre records

## Scope

- define a richer aggregated genre record shape
- rebuild one genre or all genres from canonical upstream data
- keep markdown generation out of this step
- keep provenance explicit with source references

## Outcome

- `aggregate-genre` rebuilds one genre JSON record
- `aggregate-genres` batch rebuilds all canonical genre records
- `data/aggregated/genres/*.json` now carries listening stats, top entities, editorial fields, aliases, and source references
