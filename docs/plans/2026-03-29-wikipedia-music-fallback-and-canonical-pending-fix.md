# Wikipedia "music" Fallback + Canonical Genre Pending-State Fix

## Summary
- Fix genre Wikipedia enrichment so ambiguous canonical genres can retry with a music-qualified title such as `classical music`.
- Fix aggregated canonical genres so they are not marked pending when taxonomy aliases already resolve the canonical slug.
- Add a single-genre refresh path to `wikipedia-backfill-genres` for targeted recovery of stuck canonical genres like `classical`.

## Implementation Changes
- Update `internal/enrich/enrich.go` so `WikipediaGenre` first prefers an exact `"<search term> music"` candidate from the initial search results, then does one fallback search for that term when the first pass stays ambiguous or lands on a disambiguation page.
- Keep the fallback genre-only and skip it when `seed.PageTitle` is explicitly configured.
- Update `internal/datalayer/datalayer.go` so `genreAliasesForSlug` only inherits pending state from raw labels when the canonical slug has no resolved taxonomy aliases.
- Update `genre_commands.go` so `wikipedia-backfill-genres` accepts `--slug` and processes only that canonical slug when present.
- Update `docs/commands.md` to document the new `--slug` option.

## Test Plan
- Add pure-function tests for `ChooseWikipediaPageTitle` covering music-qualified exact matches, explicit page titles, and unresolved ambiguous cases.
- Add HTTP-mock tests for genre enrichment covering:
  - ambiguous initial search resolved by `"<term> music"` fallback
  - disambiguation summary resolved by `"<term> music"` fallback
- Add a datalayer regression test proving a canonical mapped genre with matching raw pending labels is still aggregated with `Pending == false`.
- Add a workflow regression test proving a mapped, non-pending, enriched genre with plays is eligible for `ready-basic`.

## Operational Validation
- Run `./music-garden wikipedia-backfill-genres --slug classical --refresh`
- Run `./music-garden aggregate-genre --slug classical`
- Confirm `./music-garden genre-review --slug classical` reports mapped taxonomy, `pending=false`, matched Wikipedia metadata, and retained plays.
- Promote `classical` to `publishable` and generate the page.

## Assumptions
- The fallback suffix is exactly `" music"` for v1.
- One extra Wikipedia search per unresolved genre is acceptable.
- Raw pending labels may remain in the store, but they must not override already mapped canonical genres.
