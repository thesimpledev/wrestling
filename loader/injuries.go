package loader

import (
	"encoding/json"
	"wrestling/storage"
)

// InjuryRecord tracks an active injury for a wrestler.
type InjuryRecord struct {
	CardsRemaining int `json:"cards_remaining"`
}

// InjuryStore maps wrestler names to their injury status.
type InjuryStore map[string]InjuryRecord

// LoadInjuries reads injury data from the store.
// Returns an empty store if no data exists yet.
func LoadInjuries(store storage.Store) InjuryStore {
	data, err := store.LoadInjuriesJSON()
	if err != nil || data == nil {
		return make(InjuryStore)
	}
	var s InjuryStore
	if err := json.Unmarshal(data, &s); err != nil {
		return make(InjuryStore)
	}
	return s
}

// SaveInjuries writes injury data to the store.
func SaveInjuries(store storage.Store, s InjuryStore) error {
	// Clean up healed injuries
	for name, rec := range s {
		if rec.CardsRemaining <= 0 {
			delete(s, name)
		}
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return store.SaveInjuriesJSON(data)
}

// RecordInjury adds or updates an injury for a wrestler.
func (s InjuryStore) RecordInjury(name string, cards int) {
	if cards <= 0 {
		return
	}
	s[name] = InjuryRecord{CardsRemaining: cards}
}

// DecrementAll reduces all injury counters by 1 (called after each match card).
func (s InjuryStore) DecrementAll() {
	for name, rec := range s {
		rec.CardsRemaining--
		if rec.CardsRemaining <= 0 {
			delete(s, name)
		} else {
			s[name] = rec
		}
	}
}

// IsInjured returns true if the wrestler is currently injured.
func (s InjuryStore) IsInjured(name string) bool {
	rec, ok := s[name]
	return ok && rec.CardsRemaining > 0
}

// InjuryCards returns the remaining injury cards for a wrestler.
func (s InjuryStore) InjuryCards(name string) int {
	if rec, ok := s[name]; ok {
		return rec.CardsRemaining
	}
	return 0
}
