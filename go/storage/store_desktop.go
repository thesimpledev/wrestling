//go:build !js

package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DesktopStore reads/writes wrestler cards and injuries using the local filesystem.
type DesktopStore struct {
	dataDir string
}

func NewDesktopStore(dataDir string) *DesktopStore {
	return &DesktopStore{dataDir: dataDir}
}

func (s *DesktopStore) LoadAllCardBytes() (map[string][]byte, error) {
	entries, err := os.ReadDir(s.dataDir)
	if err != nil {
		return nil, fmt.Errorf("reading wrestlers directory: %w", err)
	}

	cards := make(map[string][]byte)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.dataDir, name))
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", name, err)
		}
		cards[name] = data
	}
	return cards, nil
}

func (s *DesktopStore) SaveCardBytes(filename string, data []byte) error {
	return os.WriteFile(filepath.Join(s.dataDir, filename), data, 0644)
}

func (s *DesktopStore) injuriesPath() string {
	return filepath.Join(s.dataDir, "..", "injuries.json")
}

func (s *DesktopStore) LoadInjuriesJSON() ([]byte, error) {
	data, err := os.ReadFile(s.injuriesPath())
	if os.IsNotExist(err) {
		return nil, nil
	}
	return data, err
}

func (s *DesktopStore) SaveInjuriesJSON(data []byte) error {
	return os.WriteFile(s.injuriesPath(), data, 0644)
}

func (s *DesktopStore) careerPath() string {
	return filepath.Join(s.dataDir, "..", "career.json")
}

func (s *DesktopStore) LoadCareerJSON() ([]byte, error) {
	data, err := os.ReadFile(s.careerPath())
	if os.IsNotExist(err) {
		return nil, nil
	}
	return data, err
}

func (s *DesktopStore) SaveCareerJSON(data []byte) error {
	return os.WriteFile(s.careerPath(), data, 0644)
}
