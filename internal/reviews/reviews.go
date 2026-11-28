package reviews

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const Version = 1

type Decision struct {
	Status       string `json:"status"`
	Candidate    string `json:"candidate,omitempty"`
	Reason       string `json:"reason,omitempty"`
	ForcePublish bool   `json:"force_publish,omitempty"`
	Suppress     bool   `json:"suppress,omitempty"`
}

type File struct {
	Version   int                 `json:"version"`
	Decisions map[string]Decision `json:"decisions"`
}

type Store struct {
	Kinds map[string]*File
}

func Load(dir string) (*Store, error) {
	store := &Store{Kinds: map[string]*File{}}
	for _, kind := range []string{"artist", "genre", "release", "media"} {
		path := filepath.Join(dir, kind+"s.json")
		file := &File{Version: Version, Decisions: map[string]Decision{}}
		data, err := os.ReadFile(path)
		if err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("read %s reviews: %w", kind, err)
		}
		if len(data) > 0 {
			if err := json.Unmarshal(data, file); err != nil {
				return nil, fmt.Errorf("decode %s reviews: %w", kind, err)
			}
		}
		if file.Version == 0 {
			file.Version = Version
		}
		if file.Decisions == nil {
			file.Decisions = map[string]Decision{}
		}
		store.Kinds[kind] = file
	}
	return store, nil
}

func (s *Store) Decision(kind, slug string) (Decision, bool) {
	file := s.Kinds[normalizeKind(kind)]
	if file == nil {
		return Decision{}, false
	}
	d, ok := file.Decisions[strings.TrimSpace(slug)]
	return d, ok
}

func (s *Store) Set(kind, slug string, decision Decision) error {
	kind = normalizeKind(kind)
	if kind == "" || strings.TrimSpace(slug) == "" {
		return fmt.Errorf("kind and slug are required")
	}
	file := s.Kinds[kind]
	if file == nil {
		return fmt.Errorf("unsupported review kind %q", kind)
	}
	decision.Status = strings.ToLower(strings.TrimSpace(decision.Status))
	if decision.Status != "approved" && decision.Status != "rejected" {
		return fmt.Errorf("status must be approved or rejected")
	}
	file.Decisions[strings.TrimSpace(slug)] = decision
	return nil
}

func (s *Store) SaveKind(dir, kind string) error {
	kind = normalizeKind(kind)
	file := s.Kinds[kind]
	if file == nil {
		return fmt.Errorf("unsupported review kind %q", kind)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create review directory: %w", err)
	}
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s reviews: %w", kind, err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(dir, kind+"s.json"), data, 0o644); err != nil {
		return fmt.Errorf("write %s reviews: %w", kind, err)
	}
	return nil
}

func (s *Store) Slugs(kind string) []string {
	file := s.Kinds[normalizeKind(kind)]
	if file == nil {
		return nil
	}
	slugs := make([]string, 0, len(file.Decisions))
	for slug := range file.Decisions {
		slugs = append(slugs, slug)
	}
	sort.Strings(slugs)
	return slugs
}

func normalizeKind(kind string) string {
	kind = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(kind)), "s")
	switch kind {
	case "artist", "genre", "release", "media":
		return kind
	default:
		return ""
	}
}
