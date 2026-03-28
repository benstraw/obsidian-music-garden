# Genre Markdown Pages

Date: 2026-03-27

## Goal

Render deterministic Obsidian-friendly markdown genre pages from aggregated
genre JSON records.

## Inputs

- `data/aggregated/genres/*.json`

## Output

- repo-local `content/genres/*.md`
- optionally a sandbox output directory via `--out-dir`

## Scope

- front matter suitable for Obsidian and future static-site compatibility
- concise summary
- listening stats
- top artists, albums, and tracks
- source notes
- related genre links
- deterministic and idempotent rewrites
