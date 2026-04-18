package pogopvp_test

import (
	"errors"
	"os"
	"slices"
	"strings"
	"testing"

	pogopvp "github.com/lexfrei/pogo-pvp-engine"
)

func loadSampleGamemaster(t *testing.T) *pogopvp.Gamemaster {
	t.Helper()

	f, err := os.Open("testdata/gamemaster_sample.json")
	if err != nil {
		t.Fatalf("open testdata: %v", err)
	}
	defer f.Close()

	gm, err := pogopvp.ParseGamemaster(f)
	if err != nil {
		t.Fatalf("ParseGamemaster: %v", err)
	}
	return gm
}

// The testdata sample is a fixed subset (5 Pokémon — bulbasaur, medicham,
// whiscash, machamp, azumarill — plus all the moves they reference). These
// counts are exact, not floors; if the fixture is regenerated the counts
// move with it.
const (
	sampleExpectedPokemonCount = 5
	sampleExpectedMovesCount   = 31
)

func TestParseGamemaster_SampleCounts(t *testing.T) {
	t.Parallel()

	gm := loadSampleGamemaster(t)

	if len(gm.Pokemon) != sampleExpectedPokemonCount {
		t.Errorf("Pokemon count = %d, want %d", len(gm.Pokemon), sampleExpectedPokemonCount)
	}
	if len(gm.Moves) != sampleExpectedMovesCount {
		t.Errorf("Moves count = %d, want %d", len(gm.Moves), sampleExpectedMovesCount)
	}
}

func TestParseGamemaster_PokemonFields(t *testing.T) {
	t.Parallel()

	gm := loadSampleGamemaster(t)

	bulb, ok := gm.Pokemon["bulbasaur"]
	if !ok {
		t.Fatal("bulbasaur missing from parsed Pokemon map")
	}

	if bulb.Dex != 1 {
		t.Errorf("bulbasaur Dex = %d, want 1", bulb.Dex)
	}
	if bulb.Name != "Bulbasaur" {
		t.Errorf("bulbasaur Name = %q, want \"Bulbasaur\"", bulb.Name)
	}
	if bulb.BaseStats.Atk != 118 || bulb.BaseStats.Def != 111 || bulb.BaseStats.HP != 128 {
		t.Errorf("bulbasaur BaseStats = %+v, want {118, 111, 128}", bulb.BaseStats)
	}
	wantTypes := []string{"grass", "poison"}
	if !slices.Equal(bulb.Types, wantTypes) {
		t.Errorf("bulbasaur Types = %v, want %v", bulb.Types, wantTypes)
	}
	if !bulb.Released {
		t.Error("bulbasaur Released = false, want true")
	}
	if !slices.Contains(bulb.FastMoves, "VINE_WHIP") {
		t.Errorf("bulbasaur FastMoves = %v, want to contain VINE_WHIP", bulb.FastMoves)
	}
	if !slices.Contains(bulb.ChargedMoves, "SLUDGE_BOMB") {
		t.Errorf("bulbasaur ChargedMoves = %v, want to contain SLUDGE_BOMB", bulb.ChargedMoves)
	}
}

func TestParseGamemaster_FastMove(t *testing.T) {
	t.Parallel()

	gm := loadSampleGamemaster(t)

	vineWhip, ok := gm.Moves["VINE_WHIP"]
	if !ok {
		t.Fatal("VINE_WHIP missing from parsed Moves map")
	}

	if vineWhip.Category != pogopvp.MoveCategoryFast {
		t.Errorf("VINE_WHIP Category = %v, want Fast", vineWhip.Category)
	}
	if vineWhip.Power != 5 {
		t.Errorf("VINE_WHIP Power = %d, want 5", vineWhip.Power)
	}
	if vineWhip.EnergyGain != 8 {
		t.Errorf("VINE_WHIP EnergyGain = %d, want 8", vineWhip.EnergyGain)
	}
	if vineWhip.Energy != 0 {
		t.Errorf("VINE_WHIP Energy = %d, want 0", vineWhip.Energy)
	}
	if vineWhip.Turns != 2 {
		t.Errorf("VINE_WHIP Turns = %d, want 2", vineWhip.Turns)
	}
	if vineWhip.Type != "grass" {
		t.Errorf("VINE_WHIP Type = %q, want grass", vineWhip.Type)
	}
}

func TestParseGamemaster_ChargedMove(t *testing.T) {
	t.Parallel()

	gm := loadSampleGamemaster(t)

	sb, ok := gm.Moves["SLUDGE_BOMB"]
	if !ok {
		t.Fatal("SLUDGE_BOMB missing from parsed Moves map")
	}

	if sb.Category != pogopvp.MoveCategoryCharged {
		t.Errorf("SLUDGE_BOMB Category = %v, want Charged", sb.Category)
	}
	if sb.Power != 80 {
		t.Errorf("SLUDGE_BOMB Power = %d, want 80", sb.Power)
	}
	if sb.Energy != 50 {
		t.Errorf("SLUDGE_BOMB Energy = %d, want 50", sb.Energy)
	}
	if sb.EnergyGain != 0 {
		t.Errorf("SLUDGE_BOMB EnergyGain = %d, want 0", sb.EnergyGain)
	}
}

func TestParseGamemaster_EmptyInput(t *testing.T) {
	t.Parallel()

	_, err := pogopvp.ParseGamemaster(strings.NewReader(""))
	if err == nil {
		t.Fatal("ParseGamemaster on empty input expected error")
	}
	if !errors.Is(err, pogopvp.ErrGamemasterDecode) {
		t.Errorf("error = %v, want wrapping ErrGamemasterDecode", err)
	}
}

func TestParseGamemaster_MalformedJSON(t *testing.T) {
	t.Parallel()

	_, err := pogopvp.ParseGamemaster(strings.NewReader("{not json"))
	if err == nil {
		t.Fatal("ParseGamemaster on malformed JSON expected error")
	}
	if !errors.Is(err, pogopvp.ErrGamemasterDecode) {
		t.Errorf("error = %v, want wrapping ErrGamemasterDecode", err)
	}
}

func TestParseGamemaster_MissingPokemonID(t *testing.T) {
	t.Parallel()

	raw := `{"id":"gamemaster","pokemon":[` +
		`{"dex":1,"speciesName":"x","baseStats":{"atk":1,"def":1,"hp":1},"types":["fire","fire"]}` +
		`],"moves":[]}`
	_, err := pogopvp.ParseGamemaster(strings.NewReader(raw))
	if err == nil {
		t.Fatal("ParseGamemaster with missing speciesId expected error")
	}
	if !errors.Is(err, pogopvp.ErrGamemasterInvalid) {
		t.Errorf("error = %v, want wrapping ErrGamemasterInvalid", err)
	}
}

func TestParseGamemaster_DuplicatePokemon(t *testing.T) {
	t.Parallel()

	raw := `{"id":"gamemaster","pokemon":[` +
		`{"dex":1,"speciesId":"foo","speciesName":"Foo","baseStats":{"atk":1,"def":1,"hp":1},"types":["fire","fire"],"released":true},` +
		`{"dex":2,"speciesId":"foo","speciesName":"FooDup","baseStats":{"atk":2,"def":2,"hp":2},"types":["water","water"],"released":true}` +
		`],"moves":[]}`
	_, err := pogopvp.ParseGamemaster(strings.NewReader(raw))
	if err == nil {
		t.Fatal("ParseGamemaster with duplicate speciesId expected error")
	}
	if !errors.Is(err, pogopvp.ErrGamemasterInvalid) {
		t.Errorf("error = %v, want wrapping ErrGamemasterInvalid", err)
	}
}

func TestParseGamemaster_WrongDocumentID(t *testing.T) {
	t.Parallel()

	raw := `{"id":"rankings","pokemon":[],"moves":[]}`
	_, err := pogopvp.ParseGamemaster(strings.NewReader(raw))
	if err == nil {
		t.Fatal("ParseGamemaster with wrong document id expected error")
	}
	if !errors.Is(err, pogopvp.ErrGamemasterInvalid) {
		t.Errorf("error = %v, want wrapping ErrGamemasterInvalid", err)
	}
}

func TestParseGamemaster_MissingDocumentID(t *testing.T) {
	t.Parallel()

	raw := `{"pokemon":[],"moves":[]}`
	_, err := pogopvp.ParseGamemaster(strings.NewReader(raw))
	if err == nil {
		t.Fatal("ParseGamemaster with missing document id expected error")
	}
	if !errors.Is(err, pogopvp.ErrGamemasterInvalid) {
		t.Errorf("error = %v, want wrapping ErrGamemasterInvalid", err)
	}
}

func TestParseGamemaster_ZeroBaseStats(t *testing.T) {
	t.Parallel()

	raw := `{"id":"gamemaster","pokemon":[` +
		`{"dex":1,"speciesId":"foo","baseStats":{"atk":0,"def":1,"hp":1},"types":["fire","fire"]}` +
		`],"moves":[]}`
	_, err := pogopvp.ParseGamemaster(strings.NewReader(raw))
	if err == nil {
		t.Fatal("ParseGamemaster with zero baseStats expected error")
	}
	if !errors.Is(err, pogopvp.ErrGamemasterInvalid) {
		t.Errorf("error = %v, want wrapping ErrGamemasterInvalid", err)
	}
}

func TestParseGamemaster_NegativeBaseStats(t *testing.T) {
	t.Parallel()

	raw := `{"id":"gamemaster","pokemon":[` +
		`{"dex":1,"speciesId":"foo","baseStats":{"atk":1,"def":-5,"hp":1},"types":["fire","fire"]}` +
		`],"moves":[]}`
	_, err := pogopvp.ParseGamemaster(strings.NewReader(raw))
	if err == nil {
		t.Fatal("ParseGamemaster with negative baseStats expected error")
	}
	if !errors.Is(err, pogopvp.ErrGamemasterInvalid) {
		t.Errorf("error = %v, want wrapping ErrGamemasterInvalid", err)
	}
}

func TestParseGamemaster_ZeroDex(t *testing.T) {
	t.Parallel()

	raw := `{"id":"gamemaster","pokemon":[` +
		`{"dex":0,"speciesId":"foo","baseStats":{"atk":1,"def":1,"hp":1},"types":["fire","fire"]}` +
		`],"moves":[]}`
	_, err := pogopvp.ParseGamemaster(strings.NewReader(raw))
	if err == nil {
		t.Fatal("ParseGamemaster with dex<1 expected error")
	}
	if !errors.Is(err, pogopvp.ErrGamemasterInvalid) {
		t.Errorf("error = %v, want wrapping ErrGamemasterInvalid", err)
	}
}

// TestParseGamemaster_MonotypeNormalisation verifies that the pvpoke
// placeholder "none" used for monotype Pokemon is stripped from the parsed
// Species.Types so consumers only see real type identifiers.
func TestParseGamemaster_MonotypeNormalisation(t *testing.T) {
	t.Parallel()

	gm := loadSampleGamemaster(t)

	machamp, ok := gm.Pokemon["machamp"]
	if !ok {
		t.Fatal("machamp missing from parsed Pokemon map")
	}

	wantTypes := []string{"fighting"}
	if !slices.Equal(machamp.Types, wantTypes) {
		t.Errorf("machamp Types = %v, want %v", machamp.Types, wantTypes)
	}
}

func TestParseGamemaster_EmptyTypesAfterNormalise(t *testing.T) {
	t.Parallel()

	raw := `{"id":"gamemaster","pokemon":[` +
		`{"dex":1,"speciesId":"foo","baseStats":{"atk":1,"def":1,"hp":1},"types":["none","none"]}` +
		`],"moves":[]}`
	_, err := pogopvp.ParseGamemaster(strings.NewReader(raw))
	if err == nil {
		t.Fatal("ParseGamemaster with all-placeholder types expected error")
	}
	if !errors.Is(err, pogopvp.ErrGamemasterInvalid) {
		t.Errorf("error = %v, want wrapping ErrGamemasterInvalid", err)
	}
}

// TestParseGamemaster_MoveWithNoEnergy pins the TRANSFORM-style skip
// behaviour: a move with neither energy nor energyGain (Ditto's
// signature move in the upstream pvpoke gamemaster) is silently
// dropped from the indexed output rather than failing the whole load.
// Rejecting would block every pvpoke-sourced gamemaster from parsing.
func TestParseGamemaster_MoveWithNoEnergy(t *testing.T) {
	t.Parallel()

	raw := `{"id":"gamemaster","pokemon":[],"moves":[` +
		`{"moveId":"TRANSFORM","name":"Transform","type":"normal","power":0,"energy":0,"energyGain":0,"turns":0},` +
		`{"moveId":"COUNTER","name":"Counter","type":"fighting","power":8,"energy":0,"energyGain":7,` +
		`"cooldown":1000,"turns":2}` +
		`]}`
	gm, err := pogopvp.ParseGamemaster(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("ParseGamemaster with TRANSFORM-style move: unexpected error %v", err)
	}

	_, transformPresent := gm.Moves["TRANSFORM"]
	if transformPresent {
		t.Errorf("TRANSFORM-style move leaked into gamemaster.Moves, expected silent drop")
	}

	_, counterPresent := gm.Moves["COUNTER"]
	if !counterPresent {
		t.Errorf("COUNTER missing from gamemaster.Moves, regular moves must still load")
	}
}

func TestParseGamemaster_MoveWithBothEnergyAndGain(t *testing.T) {
	t.Parallel()

	raw := `{"id":"gamemaster","pokemon":[],"moves":[` +
		`{"moveId":"BROKEN","name":"Broken","type":"normal","power":1,"energy":50,"energyGain":8,"turns":2}` +
		`]}`
	_, err := pogopvp.ParseGamemaster(strings.NewReader(raw))
	if err == nil {
		t.Fatal("ParseGamemaster with conflicting energy fields expected error")
	}
	if !errors.Is(err, pogopvp.ErrGamemasterInvalid) {
		t.Errorf("error = %v, want wrapping ErrGamemasterInvalid", err)
	}
}
