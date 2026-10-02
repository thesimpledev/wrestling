package loader

import (
	"encoding/json"
	"fmt"

	"wrestling/internal/engine"
	"wrestling/internal/storage"
)

// LoadFederations reads federation save data from the store. It returns nil
// with no error when nothing has been saved, and nil with an error when the
// saved data cannot be read.
func LoadFederations(store storage.Store) (*engine.FederationSave, error) {
	data, err := store.LoadCareerJSON()
	if err != nil {
		return nil, fmt.Errorf("reading federations: %w", err)
	}
	if len(data) == 0 {
		return nil, nil
	}

	var fs engine.FederationSave
	if err := json.Unmarshal(data, &fs); err != nil {
		return nil, fmt.Errorf("parsing federations: %w", err)
	}
	if len(fs.Federations) == 0 {
		return nil, nil
	}
	return &fs, nil
}

// SaveFederations writes federation data to the store.
func SaveFederations(store storage.Store, save *engine.FederationSave) error {
	data, err := json.MarshalIndent(save, "", "  ")
	if err != nil {
		return err
	}
	return store.SaveCareerJSON(data)
}
