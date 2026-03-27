# Bulk Enrichment Backfills

Date: 2026-03-27

## Goal

Add batch backfill commands for the source enrichers that already exist in
single-item form.

## Commands

- `musicbrainz-backfill-artists`
- `musicbrainz-backfill-albums`
- `wikipedia-backfill-genres`
- `wikipedia-backfill-artists`

## Behavior

- reuse the same enrichment logic as the single-item commands
- skip already matched records by default
- support `--refresh` and `--limit`
- continue on per-record errors with helpful logs
- save raw responses, normalized records, canonical store updates, and refreshed aggregates
