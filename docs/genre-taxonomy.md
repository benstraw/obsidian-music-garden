# Genre Taxonomy

`data/genre-taxonomy.json` is the canonical source of truth for genre identity.

It defines:

- `slug`
- `display_name`
- `aliases`
- `parent_slug`
- `notes`

## Rules

1. Prefer alias additions over new slugs when a label is clearly the same concept.
2. Keep slugs stable once they are in use.
3. Use `parent_slug` for broad structure only.
4. Use `notes` to explain curation decisions, not to store encyclopedic text.

## Workflow

1. Edit `data/genre-taxonomy.json`
2. Run:

```bash
./music-garden genre-report
```

3. Review:
   - known mappings
   - unknown labels
   - collisions where one normalized alias resolves to multiple slugs

4. Curate the file until the unknown labels and collisions are intentional.

## TODO

- Keep working down the remaining unknown-label queue from `genre-report`.
- Split non-genre junk and mood tags from real genre candidates so the report
  stays focused on curation work that matters.
