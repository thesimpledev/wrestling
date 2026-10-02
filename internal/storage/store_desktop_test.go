//go:build !js

package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func newTestStore(t *testing.T) (*DesktopStore, string) {
	t.Helper()
	root := t.TempDir()
	cards := filepath.Join(root, "wrestlers")
	if err := os.Mkdir(cards, 0o700); err != nil {
		t.Fatalf("creating card directory: %v", err)
	}
	return NewDesktopStore(cards), root
}

func TestDesktopStoreCards(t *testing.T) {
	store, root := newTestStore(t)
	if err := store.SaveCardBytes("some_wrestler.yaml", []byte("name: Some Wrestler")); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "wrestlers", "notes.txt"), []byte("ignore me"), 0o600); err != nil {
		t.Fatalf("writing extra file: %v", err)
	}
	cards, err := store.LoadAllCardBytes()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(cards) != 1 || string(cards["some_wrestler.yaml"]) != "name: Some Wrestler" {
		t.Fatalf("cards: %v", cards)
	}
}

func TestDesktopStoreKeepsCardWritesInsideTheCardDirectory(t *testing.T) {
	store, root := newTestStore(t)
	if err := store.SaveCardBytes("../escape.yaml", []byte("x")); err != nil {
		t.Fatalf("save: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "escape.yaml")); !os.IsNotExist(err) {
		t.Fatalf("card was written outside the card directory (stat error: %v)", err)
	}
	if _, err := os.Stat(filepath.Join(root, "wrestlers", "escape.yaml")); err != nil {
		t.Fatalf("card not written inside the card directory: %v", err)
	}
}

func TestDesktopStoreJSONFiles(t *testing.T) {
	type jsonFile struct {
		name string
		load func(*DesktopStore) ([]byte, error)
		save func(*DesktopStore, []byte) error
		file string
	}
	files := []jsonFile{
		{"injuries", (*DesktopStore).LoadInjuriesJSON, (*DesktopStore).SaveInjuriesJSON, "injuries.json"},
		{"career", (*DesktopStore).LoadCareerJSON, (*DesktopStore).SaveCareerJSON, "career.json"},
		{"settings", (*DesktopStore).LoadSettingsJSON, (*DesktopStore).SaveSettingsJSON, "settings.json"},
	}
	for _, f := range files {
		t.Run(f.name, func(t *testing.T) {
			store, root := newTestStore(t)
			data, err := f.load(store)
			if err != nil || data != nil {
				t.Fatalf("before any save: got %q, %v, want nil, nil", data, err)
			}
			if err := f.save(store, []byte(`{"k":1}`)); err != nil {
				t.Fatalf("save: %v", err)
			}
			data, err = f.load(store)
			if err != nil || string(data) != `{"k":1}` {
				t.Fatalf("after save: got %q, %v", data, err)
			}
			if _, err := os.Stat(filepath.Join(root, f.file)); err != nil {
				t.Fatalf("expected %s next to the card directory: %v", f.file, err)
			}
		})
	}
}
