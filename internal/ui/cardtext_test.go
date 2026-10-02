package ui

import (
	"reflect"
	"strings"
	"testing"

	"wrestling/internal/engine"
)

func tags(list ...engine.MoveTag) []engine.MoveTag { return list }

func TestMoveLineRoundTrip(t *testing.T) {
	cases := []struct {
		line string
		move engine.Move
	}{
		{"Arm Drag,1,1", engine.Move{Name: "Arm Drag", Power: 1, DefLevel: 1}},
		{"Flying Elbow,2,2,ag", engine.Move{Name: "Flying Elbow", Power: 2, DefLevel: 2, Tags: tags(engine.TagAgility)}},
		{"Super Slam,3,3,pw", engine.Move{Name: "Super Slam", Power: 3, DefLevel: 3, Tags: tags(engine.TagPower)}},
		{"Into the Ropes,2,1,ropes", engine.Move{Name: "Into the Ropes", Power: 2, DefLevel: 1,
			Tags: tags(engine.TagChart), ChartType: "ropes"}},
		{"Into the Turnbuckle,2,2,turnbuckle c", engine.Move{Name: "Into the Turnbuckle", Power: 2, DefLevel: 2,
			Tags: tags(engine.TagChart, engine.TagChartChoice), ChartType: "turnbuckle"}},
		{"Out of the Ring,3,3,ring", engine.Move{Name: "Out of the Ring", Power: 3, DefLevel: 3,
			Tags: tags(engine.TagChart), ChartType: "ring"}},
		{"Deathjump,3,3,deathjump", engine.Move{Name: "Deathjump", Power: 3, DefLevel: 3,
			Tags: tags(engine.TagChart), ChartType: "deathjump"}},
		{"Choice,2,2,ch E", engine.Move{Name: "Choice", Power: 2, DefLevel: 2,
			Tags: tags(engine.TagChoice), ChoiceKey: "E"}},
		{"Low Blow,3,3,dis", engine.Move{Name: "Low Blow", Power: 3, DefLevel: 3, Tags: tags(engine.TagDQ)}},
		{"Chair Shot,3,3,dis 7", engine.Move{Name: "Chair Shot", Power: 3, DefLevel: 3,
			Tags: tags(engine.TagDQ), DQNumber: 7}},
		{"Belly to Back Suplex,3,3,add1", engine.Move{Name: "Belly to Back Suplex", Power: 3, DefLevel: 3,
			Tags: tags(engine.TagAdd1)}},
		{"Double Team,2,2,tag", engine.Move{Name: "Double Team", Power: 2, DefLevel: 2, Tags: tags(engine.TagTagTeam)}},
		{"Solo Special,2,2,singles", engine.Move{Name: "Solo Special", Power: 2, DefLevel: 2, Tags: tags(engine.TagSingles)}},
		{"Flying Chair,3,3,ag dis add1", engine.Move{Name: "Flying Chair", Power: 3, DefLevel: 3,
			Tags: tags(engine.TagAgility, engine.TagDQ, engine.TagAdd1)}},
	}
	for _, tc := range cases {
		t.Run(tc.line, func(t *testing.T) {
			got, err := parseMoveLine(tc.line)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if !reflect.DeepEqual(got, tc.move) {
				t.Fatalf("parsed %+v, want %+v", got, tc.move)
			}
			if line := formatMoveLine(tc.move); line != tc.line {
				t.Fatalf("formatted %q, want %q", line, tc.line)
			}
		})
	}
}

func TestMoveLineIsForgivingAboutSpacingAndCase(t *testing.T) {
	got, err := parseMoveLine("  Into the Ropes , 2 , 1 ,  ROPES   C ")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	want := engine.Move{Name: "Into the Ropes", Power: 2, DefLevel: 1,
		Tags: tags(engine.TagChart, engine.TagChartChoice), ChartType: "ropes"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parsed %+v, want %+v", got, want)
	}
}

func TestMoveLineErrors(t *testing.T) {
	cases := []struct {
		line string
		want string
	}{
		{"Arm Drag", "name,power,deflvl"},
		{"Arm Drag,1", "name,power,deflvl"},
		{",1,1", "name cannot be empty"},
		{"Arm Drag,x,1", "power must be a number"},
		{"Arm Drag,4,1", "power must be 1-3"},
		{"Arm Drag,0,1", "power must be 1-3"},
		{"Arm Drag,1,0", "def level must be 1-3"},
		{"Arm Drag,1,4", "def level must be 1-3"},
		{"Arm Drag,1,1,flying", `unknown instruction "flying"`},
		{"Choice,2,2,ch", "ch needs a letter"},
		{"Choice,2,2,ch Z", `unknown choice "Z"`},
		{"Chair Shot,3,3,dis 13", "dis number must be 2-12"},
		{"Jump,2,2,ropes ring", "only one chart"},
		{"Strike,2,2,c", "(c) only applies to a chart move"},
	}
	for _, tc := range cases {
		t.Run(tc.line, func(t *testing.T) {
			_, err := parseMoveLine(tc.line)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error %v, want one containing %q", err, tc.want)
			}
		})
	}
}

func TestChartAndChoiceMovesNeedNoNumbers(t *testing.T) {
	for _, line := range []string{"Into the Ropes,0,0,ropes", "Choice,0,0,ch C"} {
		if _, err := parseMoveLine(line); err != nil {
			t.Errorf("%q: %v", line, err)
		}
	}
}

func TestDefenseLineRoundTrip(t *testing.T) {
	cases := []struct {
		line    string
		outcome engine.DefenseOutcome
	}{
		{"dazed,1", engine.DefenseOutcome{Type: engine.DefDazed, Power: 1}},
		{"hurt,2", engine.DefenseOutcome{Type: engine.DefHurt, Power: 2}},
		{"down,3", engine.DefenseOutcome{Type: engine.DefDown, Power: 3}},
		{"down,3,lv", engine.DefenseOutcome{Type: engine.DefDown, Power: 3, Tags: tags(engine.TagLeave)}},
		{"dazed,1,tag", engine.DefenseOutcome{Type: engine.DefDazed, Power: 1, Tags: tags(engine.TagTagTeam)}},
		{"down,3,tag lv", engine.DefenseOutcome{Type: engine.DefDown, Power: 3,
			Tags: tags(engine.TagTagTeam, engine.TagLeave)}},
		{"reversal,2", engine.DefenseOutcome{Type: engine.DefReversal, Power: 2}},
		{"pin,0", engine.DefenseOutcome{Type: engine.DefPIN}},
	}
	for _, tc := range cases {
		t.Run(tc.line, func(t *testing.T) {
			got, err := parseDefenseLine(tc.line)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if !reflect.DeepEqual(got, tc.outcome) {
				t.Fatalf("parsed %+v, want %+v", got, tc.outcome)
			}
			if line := formatDefenseLine(tc.outcome); line != tc.line {
				t.Fatalf("formatted %q, want %q", line, tc.line)
			}
		})
	}
}

func TestDefenseLineErrors(t *testing.T) {
	cases := []struct {
		line string
		want string
	}{
		{"dazed", "type,power"},
		{"stunned,1", "type must be dazed/hurt/down/reversal/pin"},
		{"dazed,0", "power must be 1-3"},
		{"reversal,4", "power must be 1-3"},
		{"down,x", "power must be a number"},
		{"down,3,fly", `unknown instruction "fly"`},
	}
	for _, tc := range cases {
		t.Run(tc.line, func(t *testing.T) {
			_, err := parseDefenseLine(tc.line)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error %v, want one containing %q", err, tc.want)
			}
		})
	}
}

func TestPinNeedsNoNumber(t *testing.T) {
	for _, line := range []string{"pin,0", "pin,3", "pin"} {
		got, err := parseDefenseLine(line)
		if err != nil || got.Type != engine.DefPIN {
			t.Errorf("%q: %+v, %v", line, got, err)
		}
	}
}

func TestFinisherRollField(t *testing.T) {
	cases := []struct {
		text    string
		isRoll  bool
		min     int
		max     int
		wantErr string
	}{
		{"", false, 0, 0, ""},
		{"  ", false, 0, 0, ""},
		{"2-6", true, 2, 6, ""},
		{" 3 - 5 ", true, 3, 5, ""},
		{"6", false, 0, 0, "min-max"},
		{"0-6", false, 0, 0, "1 to 6"},
		{"2-7", false, 0, 0, "1 to 6"},
		{"5-2", false, 0, 0, "smaller number first"},
		{"a-b", false, 0, 0, "min-max"},
	}
	for _, tc := range cases {
		t.Run(tc.text, func(t *testing.T) {
			var f engine.Finisher
			err := applyFinisherRoll(&f, tc.text)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error %v, want one containing %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if f.IsRoll != tc.isRoll || f.RollMin != tc.min || f.RollMax != tc.max {
				t.Fatalf("finisher %+v", f)
			}
		})
	}
	if got := formatFinisherRoll(engine.Finisher{IsRoll: true, RollMin: 2, RollMax: 6}); got != "2-6" {
		t.Fatalf("format: %q", got)
	}
	if got := formatFinisherRoll(engine.Finisher{}); got != "" {
		t.Fatalf("format of a plain finisher: %q", got)
	}
}
