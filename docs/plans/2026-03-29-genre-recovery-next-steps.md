# Genre Recovery Next Steps

## Summary

The `classical` and `jazz` recovery work is complete. The next pass should focus on two buckets:

- high-play canonical genres that are already enriched but still unpublished
- high-play canonical genres that are still unresolved on Wikipedia and should be validated against the new `"<term> music"` fallback

This note is intentionally narrow. It is not a full repo cleanup plan.

## Immediate Targets

### 1. Promote already-enriched high-play genres

These should be reviewed first because they already have canonical mapping and rich editorial metadata. They are the fastest path to expanding the published set.

- `hard-bop`
- `cool-jazz`
- `reggae`
- `east-coast-hip-hop`

Validation loop:

```sh
./music-garden genre-review --slug <slug>
./music-garden aggregate-genre --slug <slug>
./music-garden genre-review --slug <slug>
./music-garden genre-promote --slug <slug> --state publishable
./music-garden generate-genre-pages --slug <slug> --out-dir ./sandbox/music/genres
```

### 2. Validate the new Wikipedia fallback on ambiguous canonical genres

These are high-value because they are canonical, have plays, and were previously stuck on unresolved or ambiguous Wikipedia lookups.

- `folk`
- `pop`
- `singer-songwriter`
- `rap`

Validation loop:

```sh
./music-garden wikipedia-backfill-genres --slug <slug> --refresh
./music-garden aggregate-genre --slug <slug>
./music-garden genre-review --slug <slug>
```

Expected outcome:

- Wikipedia title resolves to the music-specific page when appropriate
- `Summary present: true`
- `Image present: true` when available
- taxonomy remains `mapped`

## Repo Hygiene Backlog

The working tree still contains a large amount of unrelated generated and in-progress data outside this fix series. The highest-signal follow-up cleanup is:

1. Decide whether `internal/importlegacy/` and its companion plan should ship next or be removed from the worktree until resumed.
2. Review untracked helper scripts under `scripts/aggregate/` and `scripts/resolve/` and either commit the ones that are now part of the workflow or delete them.
3. Decide which generated data under `data/raw/wikipedia/` and `data/aggregated/` is intended to be committed versus refreshed locally only.

## Acceptance

This recovery series is in a good stopping state when:

- `classical` and `jazz` remain `pending=false` after `sync-data-layer`
- at least one ambiguous canonical genre such as `folk` resolves correctly through the new fallback
- at least one already-enriched draft genre is promoted and generated cleanly
