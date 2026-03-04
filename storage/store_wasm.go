//go:build js

package storage

import (
	"embed"
	"fmt"
	"io/fs"
	"strings"
	"syscall/js"
)

const (
	cardPrefix  = "wrestling_card:"
	injuriesKey = "wrestling_injuries"
	cardListKey = "wrestling_card_list"
)

// WASMStore reads/writes wrestler cards and injuries using browser localStorage.
// Default cards are loaded from an embedded filesystem and can be overridden by user saves.
type WASMStore struct {
	defaults embed.FS
}

func NewWASMStore(defaults embed.FS) *WASMStore {
	return &WASMStore{defaults: defaults}
}

func (s *WASMStore) localStorage() js.Value {
	return js.Global().Get("localStorage")
}

func (s *WASMStore) LoadAllCardBytes() (map[string][]byte, error) {
	cards := make(map[string][]byte)

	// Load embedded defaults
	err := fs.WalkDir(s.defaults, "data/wrestlers", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".yaml") && !strings.HasSuffix(d.Name(), ".yml") {
			return nil
		}
		data, err := s.defaults.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading embedded %s: %w", d.Name(), err)
		}
		cards[d.Name()] = data
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Override with user-saved cards from localStorage
	ls := s.localStorage()
	listVal := ls.Call("getItem", cardListKey)
	if !listVal.IsNull() {
		names := strings.Split(listVal.String(), "\n")
		for _, name := range names {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			val := ls.Call("getItem", cardPrefix+name)
			if !val.IsNull() {
				cards[name] = []byte(val.String())
			}
		}
	}

	return cards, nil
}

func (s *WASMStore) SaveCardBytes(filename string, data []byte) error {
	ls := s.localStorage()
	ls.Call("setItem", cardPrefix+filename, string(data))

	// Track saved card names so we can enumerate them later
	existing := make(map[string]bool)
	listVal := ls.Call("getItem", cardListKey)
	if !listVal.IsNull() {
		for _, name := range strings.Split(listVal.String(), "\n") {
			name = strings.TrimSpace(name)
			if name != "" {
				existing[name] = true
			}
		}
	}
	existing[filename] = true

	var names []string
	for name := range existing {
		names = append(names, name)
	}
	ls.Call("setItem", cardListKey, strings.Join(names, "\n"))
	return nil
}

func (s *WASMStore) LoadInjuriesJSON() ([]byte, error) {
	val := s.localStorage().Call("getItem", injuriesKey)
	if val.IsNull() {
		return nil, nil
	}
	return []byte(val.String()), nil
}

func (s *WASMStore) SaveInjuriesJSON(data []byte) error {
	s.localStorage().Call("setItem", injuriesKey, string(data))
	return nil
}
