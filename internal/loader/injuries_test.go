package loader

import "testing"

type memStore struct {
	cards    map[string][]byte
	injuries []byte
	career   []byte
	settings []byte
}

func (s *memStore) LoadAllCardBytes() (map[string][]byte, error) { return s.cards, nil }
func (s *memStore) SaveCardBytes(name string, data []byte) error {
	if s.cards == nil {
		s.cards = map[string][]byte{}
	}
	s.cards[name] = data
	return nil
}
func (s *memStore) LoadInjuriesJSON() ([]byte, error) { return s.injuries, nil }
func (s *memStore) SaveInjuriesJSON(data []byte) error {
	s.injuries = data
	return nil
}
func (s *memStore) LoadCareerJSON() ([]byte, error) { return s.career, nil }
func (s *memStore) SaveCareerJSON(data []byte) error {
	s.career = data
	return nil
}

func TestInjuryCountsDownAndHeals(t *testing.T) {
	s := make(InjuryStore)
	s.RecordInjury("A", 2)
	if !s.IsInjured("A") || s.InjuryCards("A") != 2 {
		t.Fatalf("after recording: injured %v cards %d", s.IsInjured("A"), s.InjuryCards("A"))
	}
	s.DecrementAll()
	if s.InjuryCards("A") != 1 {
		t.Fatalf("after one card: %d cards left, want 1", s.InjuryCards("A"))
	}
	s.DecrementAll()
	if s.IsInjured("A") {
		t.Fatal("still injured after two cards")
	}
	if _, kept := s["A"]; kept {
		t.Fatal("healed wrestler still in the store")
	}
}

func TestSuspensionCountsDownSeparatelyFromInjury(t *testing.T) {
	s := make(InjuryStore)
	s.RecordSuspension("A", 3)
	s.RecordInjury("A", 1)
	if !s.IsSuspended("A") || s.SuspensionCards("A") != 3 || s.InjuryCards("A") != 1 {
		t.Fatalf("record: %+v", s["A"])
	}
	s.DecrementAll()
	if s.IsInjured("A") {
		t.Fatal("injury of 1 card should be over after one card")
	}
	if s.SuspensionCards("A") != 2 {
		t.Fatalf("suspension: %d cards left, want 2", s.SuspensionCards("A"))
	}
	s.DecrementAll()
	s.DecrementAll()
	if s.IsSuspended("A") {
		t.Fatal("still suspended after three cards")
	}
	if _, kept := s["A"]; kept {
		t.Fatal("cleared wrestler still in the store")
	}
}

func TestRecordingZeroOrNegativeCardsDoesNothing(t *testing.T) {
	s := make(InjuryStore)
	s.RecordInjury("A", 0)
	s.RecordSuspension("A", -1)
	if len(s) != 0 {
		t.Fatalf("store should be empty, got %+v", s)
	}
}

func TestInjuriesSaveAndLoad(t *testing.T) {
	store := &memStore{}
	s := make(InjuryStore)
	s.RecordInjury("A", 2)
	s.RecordSuspension("B", 4)
	if err := SaveInjuries(store, s); err != nil {
		t.Fatalf("save: %v", err)
	}
	loaded, err := LoadInjuries(store)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.InjuryCards("A") != 2 || loaded.SuspensionCards("B") != 4 {
		t.Fatalf("loaded: %+v", loaded)
	}
}

func TestLoadInjuriesReadsTheOlderFormat(t *testing.T) {
	store := &memStore{injuries: []byte(`{"A": {"cards_remaining": 3}}`)}
	loaded, err := LoadInjuries(store)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.InjuryCards("A") != 3 || loaded.IsSuspended("A") {
		t.Fatalf("loaded: %+v", loaded)
	}
}

func TestLoadInjuriesReportsUnreadableData(t *testing.T) {
	store := &memStore{injuries: []byte(`[[[broken`)}
	loaded, err := LoadInjuries(store)
	if err == nil {
		t.Fatal("want an error for unreadable injury data")
	}
	if loaded == nil || len(loaded) != 0 {
		t.Fatalf("want an empty usable store, got %+v", loaded)
	}
}

func TestLoadInjuriesWithNoSavedData(t *testing.T) {
	loaded, err := LoadInjuries(&memStore{})
	if err != nil || loaded == nil || len(loaded) != 0 {
		t.Fatalf("got %+v, %v", loaded, err)
	}
}
