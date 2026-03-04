package storage

// Store abstracts filesystem I/O so the game works on both desktop (os) and web (localStorage).
type Store interface {
	// LoadAllCardBytes returns a map of filename -> YAML bytes for all wrestler cards.
	LoadAllCardBytes() (map[string][]byte, error)

	// SaveCardBytes saves a wrestler card's YAML data under the given filename.
	SaveCardBytes(filename string, data []byte) error

	// LoadInjuriesJSON returns the raw JSON bytes for the injury store.
	// Returns nil, nil if no injury data exists yet.
	LoadInjuriesJSON() ([]byte, error)

	// SaveInjuriesJSON persists the injury store as JSON bytes.
	SaveInjuriesJSON(data []byte) error
}
