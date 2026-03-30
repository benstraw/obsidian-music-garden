package genres

import (
	"encoding/json"
	"os"
)

const (
	artistStoreVersion  = 1
	releaseStoreVersion = 1
)

type ArtistStore struct {
	Version           int                     `json:"version"`
	Artists           map[string]ArtistRecord `json:"artists,omitempty"`
	ArtistSlugAliases map[string]string       `json:"artist_slug_aliases,omitempty"`
	ArtistSourceIndex map[string]string       `json:"artist_source_index,omitempty"`
}

type ReleaseStore struct {
	Version            int                      `json:"version"`
	Releases           map[string]ReleaseRecord `json:"releases,omitempty"`
	ReleaseSourceIndex map[string]string        `json:"release_source_index,omitempty"`
}

type Catalog struct {
	Genres   *Store
	Artists  *ArtistStore
	Releases *ReleaseStore
}

func NewArtistStore() *ArtistStore {
	return &ArtistStore{
		Version:           artistStoreVersion,
		Artists:           map[string]ArtistRecord{},
		ArtistSlugAliases: map[string]string{},
		ArtistSourceIndex: map[string]string{},
	}
}

func NewReleaseStore() *ReleaseStore {
	return &ReleaseStore{
		Version:            releaseStoreVersion,
		Releases:           map[string]ReleaseRecord{},
		ReleaseSourceIndex: map[string]string{},
	}
}

func NewCatalog() *Catalog {
	return &Catalog{
		Genres:   NewStore(),
		Artists:  NewArtistStore(),
		Releases: NewReleaseStore(),
	}
}

func LoadArtistStore(path string) (*ArtistStore, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return NewArtistStore(), nil
	}
	if err != nil {
		return nil, err
	}
	return LoadArtistStoreBytes(data)
}

func LoadArtistStoreBytes(data []byte) (*ArtistStore, error) {
	raw := map[string]json.RawMessage{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	store := NewArtistStore()
	if raw["artists"] != nil || raw["artist_source_index"] != nil || raw["artist_slug_aliases"] != nil || raw["version"] != nil {
		var persisted ArtistStore
		if err := json.Unmarshal(data, &persisted); err != nil {
			return nil, err
		}
		normalizeArtistStore(&persisted)
		return &persisted, nil
	}
	if _, ok := raw["genre_aliases"]; ok || raw["releases"] != nil || raw["genre_records"] != nil {
		var mixed Store
		if err := json.Unmarshal(data, &mixed); err != nil {
			return nil, err
		}
		normalizeStore(&mixed)
		store.Artists = mixed.Artists
		store.ArtistSlugAliases = mixed.ArtistSlugAliases
		store.ArtistSourceIndex = mixed.ArtistSourceIndex
		normalizeArtistStore(store)
		return store, nil
	}

	legacy := map[string]legacyEntry{}
	if err := json.Unmarshal(data, &legacy); err != nil {
		return nil, err
	}
	catalog := NewCatalog()
	for spotifyID, entry := range legacy {
		rec := Update(catalog, spotifyID, entry.Name, "", entry.Genres, entry.Images)
		if entry.LastUpdated != "" {
			rec.LastUpdated = entry.LastUpdated
			catalog.Artists.Artists[rec.Slug] = rec
		}
	}
	return catalog.Artists, nil
}

func SaveArtistStore(path string, store *ArtistStore) error {
	normalizeArtistStore(store)
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func LoadReleaseStore(path string) (*ReleaseStore, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return NewReleaseStore(), nil
	}
	if err != nil {
		return nil, err
	}
	return LoadReleaseStoreBytes(data)
}

func LoadReleaseStoreBytes(data []byte) (*ReleaseStore, error) {
	raw := map[string]json.RawMessage{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	store := NewReleaseStore()
	if raw["releases"] != nil || raw["release_source_index"] != nil || raw["version"] != nil {
		var persisted ReleaseStore
		if err := json.Unmarshal(data, &persisted); err != nil {
			return nil, err
		}
		normalizeReleaseStore(&persisted)
		return &persisted, nil
	}
	if _, ok := raw["genre_aliases"]; ok || raw["artists"] != nil || raw["genre_records"] != nil {
		var mixed Store
		if err := json.Unmarshal(data, &mixed); err != nil {
			return nil, err
		}
		normalizeStore(&mixed)
		store.Releases = mixed.Releases
		store.ReleaseSourceIndex = mixed.ReleaseSourceIndex
		normalizeReleaseStore(store)
		return store, nil
	}
	return store, nil
}

func SaveReleaseStore(path string, store *ReleaseStore) error {
	normalizeReleaseStore(store)
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func LoadCatalog(genresPath, artistsPath, releasesPath string) (*Catalog, error) {
	catalog := NewCatalog()

	genreBytes, err := os.ReadFile(genresPath)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if len(genreBytes) > 0 {
		genresStore, err := LoadBytes(genreBytes)
		if err != nil {
			return nil, err
		}
		catalog.Genres = genresStore
		// Mixed-store compatibility: use embedded entities if split files are absent.
		if len(genresStore.Artists) > 0 {
			catalog.Artists.Artists = genresStore.Artists
		}
		if len(genresStore.ArtistSlugAliases) > 0 {
			catalog.Artists.ArtistSlugAliases = genresStore.ArtistSlugAliases
		}
		if len(genresStore.ArtistSourceIndex) > 0 {
			catalog.Artists.ArtistSourceIndex = genresStore.ArtistSourceIndex
		}
		if len(genresStore.Releases) > 0 {
			catalog.Releases.Releases = genresStore.Releases
		}
		if len(genresStore.ReleaseSourceIndex) > 0 {
			catalog.Releases.ReleaseSourceIndex = genresStore.ReleaseSourceIndex
		}
	}

	if _, err := os.Stat(artistsPath); err == nil {
		store, err := LoadArtistStore(artistsPath)
		if err != nil {
			return nil, err
		}
		catalog.Artists = store
	} else if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	if _, err := os.Stat(releasesPath); err == nil {
		store, err := LoadReleaseStore(releasesPath)
		if err != nil {
			return nil, err
		}
		catalog.Releases = store
	} else if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	normalizeCatalog(catalog)
	return catalog, nil
}

func SaveCatalog(genresPath, artistsPath, releasesPath string, catalog *Catalog) error {
	normalizeCatalog(catalog)
	if err := Save(genresPath, catalog.Genres); err != nil {
		return err
	}
	if err := SaveArtistStore(artistsPath, catalog.Artists); err != nil {
		return err
	}
	if err := SaveReleaseStore(releasesPath, catalog.Releases); err != nil {
		return err
	}
	return nil
}

func normalizeArtistStore(store *ArtistStore) {
	if store.Version == 0 {
		store.Version = artistStoreVersion
	}
	if store.Artists == nil {
		store.Artists = map[string]ArtistRecord{}
	}
	if store.ArtistSlugAliases == nil {
		store.ArtistSlugAliases = map[string]string{}
	}
	if store.ArtistSourceIndex == nil {
		store.ArtistSourceIndex = map[string]string{}
	}
}

func normalizeReleaseStore(store *ReleaseStore) {
	if store.Version == 0 {
		store.Version = releaseStoreVersion
	}
	if store.Releases == nil {
		store.Releases = map[string]ReleaseRecord{}
	}
	if store.ReleaseSourceIndex == nil {
		store.ReleaseSourceIndex = map[string]string{}
	}
}

func normalizeCatalog(catalog *Catalog) {
	if catalog.Genres == nil {
		catalog.Genres = NewStore()
	}
	if catalog.Artists == nil {
		catalog.Artists = NewArtistStore()
	}
	if catalog.Releases == nil {
		catalog.Releases = NewReleaseStore()
	}
	normalizeStore(catalog.Genres)
	normalizeArtistStore(catalog.Artists)
	normalizeReleaseStore(catalog.Releases)
	catalog.Genres.Artists = catalog.Artists.Artists
	catalog.Genres.ArtistSlugAliases = catalog.Artists.ArtistSlugAliases
	catalog.Genres.ArtistSourceIndex = catalog.Artists.ArtistSourceIndex
	catalog.Genres.Releases = catalog.Releases.Releases
	catalog.Genres.ReleaseSourceIndex = catalog.Releases.ReleaseSourceIndex
}

func catalogFor(source any) *Catalog {
	switch v := source.(type) {
	case *Catalog:
		normalizeCatalog(v)
		return v
	case *Store:
		normalizeStore(v)
		catalog := NewCatalog()
		catalog.Genres = v
		catalog.Artists.Artists = v.Artists
		catalog.Artists.ArtistSlugAliases = v.ArtistSlugAliases
		catalog.Artists.ArtistSourceIndex = v.ArtistSourceIndex
		catalog.Releases.Releases = v.Releases
		catalog.Releases.ReleaseSourceIndex = v.ReleaseSourceIndex
		normalizeCatalog(catalog)
		return catalog
	default:
		panic("unsupported catalog source")
	}
}
