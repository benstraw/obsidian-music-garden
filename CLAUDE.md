# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

Go CLI for a multi-source music knowledge garden that currently ships with a
Spotify collector and setlist.fm lookup support. It generates Obsidian markdown
notes (weekly summaries, daily notes, artist stubs, and a "Music Taste"
persona context pack). Zero external Go dependencies — stdlib only.

## Build & Test

```sh
go build -o music-garden .           # build binary
go vet ./...                           # static checks (used in CI)
go test ./...                          # all tests
go test ./internal/render/ -run TestWeekStr  # single test
```

Pre-commit gate (CI mirrors this): `go vet ./... && go test ./... && go build -o music-garden .`

Version injection on release: `go build -ldflags "-X main.version=vX.Y.Z" -o music-garden .`

## Architecture

**main.go** — CLI dispatch, `.env` loading, runtime path resolution. Commands: `auth`, `collect`, `daily`, `weekly`, `catch-up`, `persona`, `setlist`, `doctor`, `version`.

**internal/** packages (each small, single-responsibility):
- `auth` — OAuth2 authorization code flow, token save/load/refresh. Local HTTP callback server on port 8888 or manual paste for external redirect URIs.
- `client` — Authenticated HTTP GET with exponential backoff on 429 (1s → 2s → 4s → fail). Bearer token, 30s timeout.
- `fetch` — source adapters. Today that means Spotify API + setlist.fm. Silently filters podcast episodes.
- `genres` — canonical artist/release metadata store, curated genre aliases, and source→canonical resolution helpers.
- `models` — Data structs: `Play`, `TopTrack`, `TopArtist`, `Setlist`.
- `plays` — sharded play storage under `data/plays/YYYY/YYYY-WNN.json`, merge/dedup, and canonical-field migration helpers.
- `render` — Weekly/daily note generation, artist stub creation (never overwrites existing), persona template rendering. ISO week math lives here (`WeekBounds`, `WeekStr`).

**templates/** — Go `text/template` files for persona and weekly reference. Template dir resolved: `MUSIC_TEMPLATES_DIR` env → `./templates` → relative to executable.

## Runtime Path Resolution

Precedence: CLI flags → env vars → `MUSIC_STATE_DIR` subdirectories → CWD fallback (with warning). This applies to `.env`, `tokens.json`, sharded `data/plays/`, and canonical `data/genres.json`. See `resolveRuntimePaths()` in main.go.

## Testing Patterns

- Stdlib `testing` only. Name pattern: `Test<Function>_<scenario>`.
- Date/time tests use `localNoon(year, month, day)` helper to avoid UTC→local day-shift bugs.
- File I/O tests use `t.TempDir()` + `t.Setenv()` for isolated vault simulation.
- Always test ISO week edge cases (Mon/Sun boundaries) when touching date logic.

## Conventions

- Errors: always wrap with context — `fmt.Errorf("context: %w", err)`.
- Business logic belongs in `internal/`, not `main.go`.
- `gofmt` before committing.
- Commits: concise, imperative, scoped.
- Major features: write a plan under `docs/plans/` with a dated filename before or alongside implementation. Bump minor version on completion.

## Environment Variables

Defined in `.env` (git-ignored); see `.env.example` for required keys:
- `SPOTIFY_CLIENT_ID`, `SPOTIFY_CLIENT_SECRET`, `SPOTIFY_REDIRECT_URI`
- `OBSIDIAN_VAULT_PATH` — target vault root
- `MUSIC_TEMPLATES_DIR` — override template location
- `MUSIC_STATE_DIR` — preferred location for tokens/data
- `MUSIC_AUTO_DAILY_ON_COLLECT_SPOTIFY` — when truthy, `collect` also regenerates today's daily note
- `SETLISTFM_API_KEY` — for `setlist` command
