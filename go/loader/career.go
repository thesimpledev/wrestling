package loader

import (
	"encoding/json"
	"wrestling/engine"
	"wrestling/storage"
)

// LoadFederations reads federation save data from the store.
// Returns nil if no data exists.
func LoadFederations(store storage.Store) *engine.FederationSave {
	data, err := store.LoadCareerJSON()
	if err != nil || data == nil {
		return nil
	}

	var fs engine.FederationSave
	if err := json.Unmarshal(data, &fs); err != nil {
		return nil
	}
	if len(fs.Federations) == 0 {
		return nil
	}
	return &fs
}

// SaveFederations writes federation data to the store.
func SaveFederations(store storage.Store, save *engine.FederationSave) error {
	data, err := json.MarshalIndent(save, "", "  ")
	if err != nil {
		return err
	}
	return store.SaveCareerJSON(data)
}
