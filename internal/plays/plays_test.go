package plays

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/benstraw/spotify-garden/internal/models"
)

func TestMerge_dedup(t *testing.T) {
	existing := []models.Play{
		{PlayedAt: "2026-02-21T10:00:00Z", TrackName: "Track A"},
		{PlayedAt: "2026-02-21T09:00:00Z", TrackName: "Track B"},
	}
	incoming := []models.Play{
		{PlayedAt: "2026-02-21T10:00:00Z", TrackName: "Track A"}, // duplicate
		{PlayedAt: "2026-02-21T11:00:00Z", TrackName: "Track C"}, // new
	}
	result := Merge(existing, incoming)
	if len(result) != 3 {
		t.Errorf("expected 3 plays after merge, got %d", len(result))
	}
}

func TestMerge_sortedDescending(t *testing.T) {
	existing := []models.Play{
		{PlayedAt: "2026-02-21T09:00:00Z"},
	}
	incoming := []models.Play{
		{PlayedAt: "2026-02-21T11:00:00Z"},
		{PlayedAt: "2026-02-21T10:00:00Z"},
	}
	result := Merge(existing, incoming)
	for i := 1; i < len(result); i++ {
		if result[i-1].PlayedAt < result[i].PlayedAt {
			t.Errorf("result not sorted descending at index %d: %s < %s",
				i, result[i-1].PlayedAt, result[i].PlayedAt)
		}
	}
}

func TestMerge_bothEmpty(t *testing.T) {
	result := Merge(nil, nil)
	if len(result) != 0 {
		t.Errorf("expected empty result, got %d plays", len(result))
	}
}

func TestMerge_incomingOnly(t *testing.T) {
	incoming := []models.Play{
		{PlayedAt: "2026-02-21T10:00:00Z", TrackName: "Track A"},
	}
	result := Merge(nil, incoming)
	if len(result) != 1 {
		t.Errorf("expected 1 play, got %d", len(result))
	}
}

func TestMerge_existingOnly(t *testing.T) {
	existing := []models.Play{
		{PlayedAt: "2026-02-21T10:00:00Z", TrackName: "Track A"},
	}
	result := Merge(existing, nil)
	if len(result) != 1 || result[0].TrackName != "Track A" {
		t.Errorf("unexpected result: %+v", result)
	}
}

func TestSaveLoad_roundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "plays.json")

	original := []models.Play{
		{PlayedAt: "2026-02-21T11:00:00Z", TrackName: "Track B", ArtistName: "Artist 2"},
		{PlayedAt: "2026-02-21T10:00:00Z", TrackName: "Track A", ArtistName: "Artist 1"},
	}

	if err := Save(path, original); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded) != len(original) {
		t.Errorf("expected %d plays, got %d", len(original), len(loaded))
	}
	for i, p := range loaded {
		if p.PlayedAt != original[i].PlayedAt || p.TrackName != original[i].TrackName {
			t.Errorf("play %d mismatch: got %+v, want %+v", i, p, original[i])
		}
	}
}

func TestLoad_missingFile(t *testing.T) {
	result, err := Load("/nonexistent/path/plays.json")
	if err != nil {
		t.Errorf("expected no error for missing file, got %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty slice for missing file, got %d plays", len(result))
	}
}

func TestSave_sortedDescending(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "plays.json")

	plays := []models.Play{
		{PlayedAt: "2026-02-21T09:00:00Z"},
		{PlayedAt: "2026-02-21T11:00:00Z"},
		{PlayedAt: "2026-02-21T10:00:00Z"},
	}

	if err := Save(path, plays); err != nil {
		t.Fatalf("Save: %v", err)
	}

	data, _ := os.ReadFile(path)
	var loaded []models.Play
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	for i := 1; i < len(loaded); i++ {
		if loaded[i-1].PlayedAt < loaded[i].PlayedAt {
			t.Errorf("not sorted descending at index %d", i)
		}
	}
}
