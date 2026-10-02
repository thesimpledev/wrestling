package loader

import (
	"encoding/json"
	"fmt"

	"wrestling/internal/engine"
	"wrestling/internal/storage"
)

// LoadRules reads the optional-rule settings from the store. A setting that
// was never saved keeps its default. It always returns usable rules; the
// error reports saved settings that could not be read.
func LoadRules(store storage.Store) (engine.Rules, error) {
	rules := engine.DefaultRules()
	data, err := store.LoadSettingsJSON()
	if err != nil {
		return rules, fmt.Errorf("reading settings: %w", err)
	}
	if data == nil {
		return rules, nil
	}
	if err := json.Unmarshal(data, &rules); err != nil {
		return engine.DefaultRules(), fmt.Errorf("parsing settings: %w", err)
	}
	return rules, nil
}

// SaveRules writes the optional-rule settings to the store.
func SaveRules(store storage.Store, rules engine.Rules) error {
	data, err := json.MarshalIndent(rules, "", "  ")
	if err != nil {
		return err
	}
	return store.SaveSettingsJSON(data)
}
