package loader

import (
	"encoding/json"
	"wrestling/engine"
	"wrestling/storage"
)

// LoadCareer reads career save data from the store.
// Returns nil if no career data exists.
func LoadCareer(store storage.Store) *engine.CareerSave {
	data, err := store.LoadCareerJSON()
	if err != nil || data == nil {
		return nil
	}
	var save engine.CareerSave
	if err := json.Unmarshal(data, &save); err != nil {
		return nil
	}
	return &save
}

// SaveCareer writes career data to the store.
func SaveCareer(store storage.Store, save *engine.CareerSave) error {
	data, err := json.MarshalIndent(save, "", "  ")
	if err != nil {
		return err
	}
	return store.SaveCareerJSON(data)
}
