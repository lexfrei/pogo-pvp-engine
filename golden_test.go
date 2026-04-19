package pogopvp_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	pogopvp "github.com/lexfrei/pogo-pvp-engine"
)

// goldenSpec is the on-disk shape of a golden snapshot entry. The
// engine's Combatant is rehydrated at runtime from the fixture
// gamemaster (testdata/gamemaster_sample.json) using speciesId +
// move ids as indices. Field tags use camelCase to match pvpoke's
// own JSON conventions.
type goldenSpec struct {
	SpeciesID    string   `json:"speciesId"`
	IV           [3]int   `json:"iv"`
	Level        float64  `json:"level"`
	FastMove     string   `json:"fastMove"`
	ChargedMoves []string `json:"chargedMoves"`
	Shields      int      `json:"shields"`
}

// goldenOptions mirrors pogopvp.BattleOptions for JSON.
type goldenOptions struct {
	MaxTurns int `json:"maxTurns"`
}

// goldenExpected mirrors the fields of [pogopvp.BattleResult] that
// the harness compares against.
type goldenExpected struct {
	Winner       int    `json:"winner"`
	Turns        int    `json:"turns"`
	HPRemaining  [2]int `json:"hpRemaining"`
	EnergyAtEnd  [2]int `json:"energyAtEnd"`
	ShieldsUsed  [2]int `json:"shieldsUsed"`
	ChargedFired [2]int `json:"chargedFired"`
}

// goldenSnapshot is one row of the corpus.
type goldenSnapshot struct {
	Name     string         `json:"name"`
	Attacker goldenSpec     `json:"attacker"`
	Defender goldenSpec     `json:"defender"`
	Options  goldenOptions  `json:"options"`
	Expected goldenExpected `json:"expected"`
}

// TestGolden runs every snapshot under testdata/golden/ through
// Simulate and asserts the output exactly matches the stored
// expected values. Set GOLDEN_UPDATE=1 to refresh snapshots in
// place instead — review the diff before committing.
func TestGolden(t *testing.T) {
	t.Parallel()

	gm := loadGoldenGamemaster(t)

	files, err := filepath.Glob(filepath.Join("testdata", "golden", "*.json"))
	if err != nil {
		t.Fatalf("Glob: %v", err)
	}

	if len(files) == 0 {
		t.Skip("no golden snapshots present under testdata/golden/")
	}

	update := os.Getenv("GOLDEN_UPDATE") != ""

	for _, path := range files {
		t.Run(filepath.Base(path), func(t *testing.T) {
			t.Parallel()

			runGoldenCase(t, gm, path, update)
		})
	}
}

// runGoldenCase loads one snapshot file, runs Simulate, and either
// asserts (default) or overwrites (GOLDEN_UPDATE=1) the expected
// block. Kept short so TestGolden stays under funlen.
func runGoldenCase(t *testing.T, gm *pogopvp.Gamemaster, path string, update bool) {
	t.Helper()

	snapshot := readGoldenSnapshot(t, path)

	attacker := buildGoldenCombatant(t, gm, &snapshot.Attacker, "attacker")
	defender := buildGoldenCombatant(t, gm, &snapshot.Defender, "defender")

	result, err := pogopvp.Simulate(&attacker, &defender, pogopvp.BattleOptions{
		MaxTurns: snapshot.Options.MaxTurns,
	})
	if err != nil {
		t.Fatalf("Simulate: %v", err)
	}

	actual := goldenExpected{
		Winner:       result.Winner,
		Turns:        result.Turns,
		HPRemaining:  result.HPRemaining,
		EnergyAtEnd:  result.EnergyAtEnd,
		ShieldsUsed:  result.ShieldsUsed,
		ChargedFired: result.ChargedFired,
	}

	if update {
		snapshot.Expected = actual
		writeGoldenSnapshot(t, path, &snapshot)

		return
	}

	assertGoldenEqual(t, &snapshot.Expected, &actual)
}

// readGoldenSnapshot parses one JSON file into a goldenSnapshot.
func readGoldenSnapshot(t *testing.T, path string) goldenSnapshot {
	t.Helper()

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	var snapshot goldenSnapshot

	err = json.Unmarshal(body, &snapshot)
	if err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}

	return snapshot
}

// writeGoldenSnapshot serialises a snapshot back to disk with
// stable two-space indentation so git diffs stay reviewable.
func writeGoldenSnapshot(t *testing.T, path string, snapshot *goldenSnapshot) {
	t.Helper()

	body, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	body = append(body, '\n')

	err = os.WriteFile(path, body, 0o600)
	if err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// assertGoldenEqual compares the stored expected block against the
// current Simulate output field-by-field. Any mismatch fails the
// test and names which field drifted so the diff is actionable.
func assertGoldenEqual(t *testing.T, expected, actual *goldenExpected) {
	t.Helper()

	if expected.Winner != actual.Winner {
		t.Errorf("Winner = %d, want %d", actual.Winner, expected.Winner)
	}
	if expected.Turns != actual.Turns {
		t.Errorf("Turns = %d, want %d", actual.Turns, expected.Turns)
	}
	if expected.HPRemaining != actual.HPRemaining {
		t.Errorf("HPRemaining = %v, want %v", actual.HPRemaining, expected.HPRemaining)
	}
	if expected.EnergyAtEnd != actual.EnergyAtEnd {
		t.Errorf("EnergyAtEnd = %v, want %v", actual.EnergyAtEnd, expected.EnergyAtEnd)
	}
	if expected.ShieldsUsed != actual.ShieldsUsed {
		t.Errorf("ShieldsUsed = %v, want %v", actual.ShieldsUsed, expected.ShieldsUsed)
	}
	if expected.ChargedFired != actual.ChargedFired {
		t.Errorf("ChargedFired = %v, want %v", actual.ChargedFired, expected.ChargedFired)
	}
}

// loadGoldenGamemaster parses the shared test fixture once per
// TestGolden invocation.
func loadGoldenGamemaster(t *testing.T) *pogopvp.Gamemaster {
	t.Helper()

	body, err := os.ReadFile(filepath.Join("testdata", "gamemaster_sample.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	gm, err := pogopvp.ParseGamemaster(strings.NewReader(string(body)))
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}

	return gm
}

// buildGoldenCombatant rehydrates a snapshot spec into an engine
// Combatant by looking up species and moves in the fixture.
func buildGoldenCombatant(
	t *testing.T, gm *pogopvp.Gamemaster, spec *goldenSpec, side string,
) pogopvp.Combatant {
	t.Helper()

	species, ok := gm.Pokemon[spec.SpeciesID]
	if !ok {
		t.Fatalf("%s: unknown species %q in fixture", side, spec.SpeciesID)
	}

	iv, err := pogopvp.NewIV(spec.IV[0], spec.IV[1], spec.IV[2])
	if err != nil {
		t.Fatalf("%s: NewIV: %v", side, err)
	}

	fast, ok := gm.Moves[spec.FastMove]
	if !ok {
		t.Fatalf("%s: unknown fast move %q", side, spec.FastMove)
	}

	charged := make([]pogopvp.Move, 0, len(spec.ChargedMoves))

	for _, moveID := range spec.ChargedMoves {
		move, moveOK := gm.Moves[moveID]
		if !moveOK {
			t.Fatalf("%s: unknown charged move %q", side, moveID)
		}

		charged = append(charged, move)
	}

	return pogopvp.Combatant{
		Species:      species,
		IV:           iv,
		Level:        spec.Level,
		FastMove:     fast,
		ChargedMoves: charged,
		Shields:      spec.Shields,
	}
}
