package genres

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

// Taxonomy is the human-edited canonical genre mapping file.
type Taxonomy struct {
	Version int               `json:"version"`
	Genres  []GenreDefinition `json:"genres"`
}

// GenreDefinition is one curated canonical genre entry.
type GenreDefinition struct {
	Slug        string   `json:"slug"`
	DisplayName string   `json:"display_name"`
	Aliases     []string `json:"aliases,omitempty"`
	ParentSlug  string   `json:"parent_slug,omitempty"`
	Notes       string   `json:"notes,omitempty"`
}

// GenreCollision reports an ambiguous normalized alias in the taxonomy.
type GenreCollision struct {
	NormalizedLabel string   `json:"normalized_label"`
	RawAliases      []string `json:"raw_aliases"`
	CanonicalSlugs  []string `json:"canonical_slugs"`
}

// GenreReport summarizes the taxonomy state against observed unknown labels.
type GenreReport struct {
	KnownMappings []GenreResolvedAlias `json:"known_mappings"`
	UnknownLabels []string             `json:"unknown_labels"`
	Collisions    []GenreCollision     `json:"collisions"`
}

// GenreResolvedAlias is one normalized alias -> canonical mapping.
type GenreResolvedAlias struct {
	NormalizedLabel string `json:"normalized_label"`
	RawAlias        string `json:"raw_alias"`
	CanonicalSlug   string `json:"canonical_slug"`
	DisplayName     string `json:"display_name"`
}

// LoadTaxonomy reads a curated taxonomy file from disk.
func LoadTaxonomy(path string) (*Taxonomy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var taxonomy Taxonomy
	if err := json.Unmarshal(data, &taxonomy); err != nil {
		return nil, err
	}
	if taxonomy.Version == 0 {
		taxonomy.Version = 1
	}
	return &taxonomy, nil
}

// ResolveGenreLabel maps an arbitrary input label to a canonical genre slug.
func ResolveGenreLabel(taxonomy *Taxonomy, label string) (string, bool, []string) {
	index := taxonomyIndex(taxonomy)
	normalized := normalizeLookup(label)
	if normalized == "" {
		return "", false, nil
	}
	entry := index[normalized]
	if len(entry.canonicalSlugs) > 1 {
		return "", false, append([]string(nil), entry.canonicalSlugs...)
	}
	if len(entry.canonicalSlugs) == 1 {
		return entry.canonicalSlugs[0], true, nil
	}
	return "", false, nil
}

// ApplyTaxonomy overlays the curated taxonomy onto the runtime store.
func ApplyTaxonomy(store *Store, taxonomy *Taxonomy) {
	normalizeStore(store)
	store.GenreAliases = map[string]string{}
	index := taxonomyIndex(taxonomy)
	for _, definition := range taxonomy.Genres {
		slug := strings.TrimSpace(definition.Slug)
		if slug == "" {
			continue
		}
		record := store.GenreRecords[slug]
		record.Slug = slug
		record.DisplayName = firstNonEmpty(definition.DisplayName, record.DisplayName, humanizeGenreSlug(slug))
		record.ParentSlug = firstNonEmpty(definition.ParentSlug, record.ParentSlug)
		record.Notes = firstNonEmpty(definition.Notes, record.Notes)
		store.GenreRecords[slug] = record

	}
	for normalized, entry := range index {
		if len(entry.canonicalSlugs) != 1 {
			continue
		}
		store.GenreAliases[normalized] = entry.canonicalSlugs[0]
	}
	for slug, artist := range store.Artists {
		artist.Genres = canonicalGenres(store, artist.SourceGenres)
		store.Artists[slug] = artist
	}
}

// BuildGenreReport summarizes curated mappings, unknown labels, and collisions.
func BuildGenreReport(taxonomy *Taxonomy, unknownLabels []string) GenreReport {
	index := taxonomyIndex(taxonomy)
	known := make([]GenreResolvedAlias, 0, len(index))
	collisions := make([]GenreCollision, 0)
	for normalized, entry := range index {
		if len(entry.canonicalSlugs) > 1 {
			collisions = append(collisions, GenreCollision{
				NormalizedLabel: normalized,
				RawAliases:      append([]string(nil), entry.rawAliases...),
				CanonicalSlugs:  append([]string(nil), entry.canonicalSlugs...),
			})
			continue
		}
		known = append(known, GenreResolvedAlias{
			NormalizedLabel: normalized,
			RawAlias:        entry.primaryAlias,
			CanonicalSlug:   entry.canonicalSlugs[0],
			DisplayName:     entry.displayName,
		})
	}
	sort.Slice(known, func(i, j int) bool {
		if known[i].CanonicalSlug != known[j].CanonicalSlug {
			return known[i].CanonicalSlug < known[j].CanonicalSlug
		}
		return known[i].NormalizedLabel < known[j].NormalizedLabel
	})
	sort.Slice(collisions, func(i, j int) bool {
		return collisions[i].NormalizedLabel < collisions[j].NormalizedLabel
	})

	unknown := make([]string, 0, len(unknownLabels))
	seen := map[string]bool{}
	for _, label := range unknownLabels {
		normalized := normalizeLookup(label)
		if normalized == "" || seen[normalized] {
			continue
		}
		seen[normalized] = true
		if _, ok, ambiguous := ResolveGenreLabel(taxonomy, label); ok || len(ambiguous) > 0 {
			continue
		}
		unknown = append(unknown, label)
	}
	sort.Strings(unknown)

	return GenreReport{
		KnownMappings: known,
		UnknownLabels: unknown,
		Collisions:    collisions,
	}
}

type taxonomyEntry struct {
	canonicalSlugs []string
	rawAliases     []string
	primaryAlias   string
	displayName    string
}

func taxonomyIndex(taxonomy *Taxonomy) map[string]taxonomyEntry {
	index := map[string]taxonomyEntry{}
	for _, definition := range taxonomy.Genres {
		slug := strings.TrimSpace(definition.Slug)
		if slug == "" {
			continue
		}
		labels := append([]string{definition.DisplayName, slug}, definition.Aliases...)
		for _, alias := range labels {
			normalized := normalizeLookup(alias)
			if normalized == "" {
				continue
			}
			entry := index[normalized]
			if !containsString(entry.canonicalSlugs, slug) {
				entry.canonicalSlugs = append(entry.canonicalSlugs, slug)
			}
			if !containsString(entry.rawAliases, alias) {
				entry.rawAliases = append(entry.rawAliases, alias)
			}
			if entry.primaryAlias == "" {
				entry.primaryAlias = alias
			}
			if entry.displayName == "" {
				entry.displayName = firstNonEmpty(definition.DisplayName, humanizeGenreSlug(slug))
			}
			sort.Strings(entry.canonicalSlugs)
			sort.Strings(entry.rawAliases)
			index[normalized] = entry
		}
	}
	return index
}

func humanizeGenreSlug(slug string) string {
	parts := strings.Split(strings.TrimSpace(slug), "-")
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

// ReportString formats a genre report for CLI output.
func (r GenreReport) ReportString() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Known mappings: %d\n", len(r.KnownMappings))
	for _, mapping := range r.KnownMappings {
		fmt.Fprintf(&b, "  %s -> %s (%s)\n", mapping.RawAlias, mapping.CanonicalSlug, mapping.DisplayName)
	}
	fmt.Fprintf(&b, "\nUnknown labels: %d\n", len(r.UnknownLabels))
	for _, label := range r.UnknownLabels {
		fmt.Fprintf(&b, "  %s\n", label)
	}
	fmt.Fprintf(&b, "\nCollisions: %d\n", len(r.Collisions))
	for _, collision := range r.Collisions {
		fmt.Fprintf(&b, "  %s -> %s\n", collision.NormalizedLabel, strings.Join(collision.CanonicalSlugs, ", "))
	}
	return b.String()
}
