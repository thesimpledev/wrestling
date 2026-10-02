package loader

import (
	"testing"

	"wrestling/internal/engine"
)

func TestLoadFederationsWithNothingSaved(t *testing.T) {
	save, err := LoadFederations(&memStore{})
	if err != nil || save != nil {
		t.Fatalf("got %+v, %v; want nil, nil", save, err)
	}
}

func TestLoadFederationsReportsAnUnreadableSave(t *testing.T) {
	save, err := LoadFederations(&memStore{career: []byte(`{"federations": [[[`)})
	if err == nil {
		t.Fatalf("no error for a corrupt save; got %+v", save)
	}
	if save != nil {
		t.Fatalf("a corrupt save should load as nothing, got %+v", save)
	}
}

func TestLoadFederationsReadsASaveFromBeforeTheTitleRuleChanged(t *testing.T) {
	old := `{"federations": [{"name": "Old Fed", "roster": ["A", "B"], "ppv_frequency": 4, "week": 6,
		"records": {"A": {"wins": 2}},
		"championships": [{"name": "World", "champion": "A", "defenses_left": 2, "history": []}],
		"rivalries": {}, "match_history": [{"week": 5, "winner": "A", "loser": "B", "method": "pinfall", "match_type": "", "is_title": false}],
		"ppv_names": ["BIG ONE"], "ppv_index": 1}], "active_index": 0}`
	save, err := LoadFederations(&memStore{career: []byte(old)})
	if err != nil {
		t.Fatal(err)
	}
	fed := save.ActiveFederation()
	if fed == nil || fed.Name != "Old Fed" || fed.MainChampion() != "A" || fed.Week != 6 {
		t.Fatalf("loaded federation: %+v", fed)
	}

	fed.AdvanceWeek()
	fed.AdvanceWeek()
	if fed.MainChampion() != "A" {
		t.Fatal("champion from an old save was stripped before the next PPV")
	}
}

func TestSaveFederationsRoundTrip(t *testing.T) {
	store := &memStore{}
	fed := engine.NewFederation(engine.FederationConfig{Name: "Fed", RosterNames: []string{"A", "B"}, ChampNames: []string{"World"}})
	fed.ChangeTitleHolder(0, "A", "", "tournament")
	if err := SaveFederations(store, &engine.FederationSave{Federations: []*engine.Federation{fed}}); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadFederations(store)
	if err != nil {
		t.Fatal(err)
	}
	got := loaded.ActiveFederation()
	if got.MainChampion() != "A" || !got.Championships[0].ContestedSincePPV {
		t.Fatalf("round trip lost the champion or the title match: %+v", got.Championships[0])
	}
}
