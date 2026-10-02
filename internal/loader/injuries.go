package loader

import (
	"encoding/json"
	"fmt"

	"wrestling/internal/storage"
)

// InjuryRecord tracks how many fight cards a wrestler is still injured or
// suspended for.
type InjuryRecord struct {
	CardsRemaining int `json:"cards_remaining"`
	SuspendedCards int `json:"suspended_cards,omitempty"`
}

func (r InjuryRecord) cleared() bool {
	return r.CardsRemaining <= 0 && r.SuspendedCards <= 0
}

// InjuryStore maps wrestler names to their injury and suspension status.
type InjuryStore map[string]InjuryRecord

// LoadInjuries reads injury data from the store. It always returns a usable
// store; the error reports saved data that could not be read.
func LoadInjuries(store storage.Store) (InjuryStore, error) {
	data, err := store.LoadInjuriesJSON()
	if err != nil {
		return make(InjuryStore), fmt.Errorf("reading injuries: %w", err)
	}
	if data == nil {
		return make(InjuryStore), nil
	}
	var s InjuryStore
	if err := json.Unmarshal(data, &s); err != nil {
		return make(InjuryStore), fmt.Errorf("parsing injuries: %w", err)
	}
	if s == nil {
		s = make(InjuryStore)
	}
	return s, nil
}

// SaveInjuries writes injury data to the store.
func SaveInjuries(store storage.Store, s InjuryStore) error {
	s.dropCleared()
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return store.SaveInjuriesJSON(data)
}

func (s InjuryStore) dropCleared() {
	for name, rec := range s {
		if rec.cleared() {
			delete(s, name)
		}
	}
}

// RecordInjury sets a wrestler's injury to the given number of fight cards.
func (s InjuryStore) RecordInjury(name string, cards int) {
	if cards <= 0 {
		return
	}
	rec := s[name]
	rec.CardsRemaining = cards
	s[name] = rec
}

// RecordSuspension sets a wrestler's suspension to the given number of fight cards.
func (s InjuryStore) RecordSuspension(name string, cards int) {
	if cards <= 0 {
		return
	}
	rec := s[name]
	rec.SuspendedCards = cards
	s[name] = rec
}

// DecrementAll counts one fight card off every injury and suspension.
func (s InjuryStore) DecrementAll() {
	for name, rec := range s {
		if rec.CardsRemaining > 0 {
			rec.CardsRemaining--
		}
		if rec.SuspendedCards > 0 {
			rec.SuspendedCards--
		}
		s[name] = rec
	}
	s.dropCleared()
}

// IsInjured returns true if the wrestler is currently injured.
func (s InjuryStore) IsInjured(name string) bool {
	return s[name].CardsRemaining > 0
}

// InjuryCards returns the remaining injury cards for a wrestler.
func (s InjuryStore) InjuryCards(name string) int {
	return s[name].CardsRemaining
}

// IsSuspended returns true if the wrestler is currently suspended.
func (s InjuryStore) IsSuspended(name string) bool {
	return s[name].SuspendedCards > 0
}

// SuspensionCards returns the remaining suspension cards for a wrestler.
func (s InjuryStore) SuspensionCards(name string) int {
	return s[name].SuspendedCards
}
