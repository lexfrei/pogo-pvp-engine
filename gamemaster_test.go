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

// The testdata sample is a fixed subset (7 Pokémon — bulbasaur,
// medicham, whiscash, machamp, azumarill, plus ditto and eevee
// added to exercise the family-block edge cases: ditto has no
// family block at all, eevee has branching evolutions with no
// parent). Move count stays at the 31 ids referenced by the
// original five species; ditto / eevee moves are not materialised
// in the Moves section. These counts are exact, not floors; if the
// fixture is regenerated the counts move with it.
const (
	sampleExpectedPokemonCount = 7
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

// TestParseGamemaster_FamilyEvolutions confirms the family block is
// decoded onto Species.Evolutions + PreEvolution: bulbasaur evolves
// into ivysaur (no parent), medicham has a parent (meditite) but no
// children.
func TestParseGamemaster_FamilyEvolutions(t *testing.T) {
	t.Parallel()

	gm := loadSampleGamemaster(t)

	bulb := gm.Pokemon["bulbasaur"]
	if !slices.Equal(bulb.Evolutions, []string{"ivysaur"}) {
		t.Errorf("bulbasaur Evolutions = %v, want [ivysaur]", bulb.Evolutions)
	}
	if bulb.PreEvolution != "" {
		t.Errorf("bulbasaur PreEvolution = %q, want empty (base form)", bulb.PreEvolution)
	}

	medi := gm.Pokemon["medicham"]
	if medi.PreEvolution != "meditite" {
		t.Errorf("medicham PreEvolution = %q, want meditite", medi.PreEvolution)
	}
	if len(medi.Evolutions) != 0 {
		t.Errorf("medicham Evolutions = %v, want empty (terminal form)", medi.Evolutions)
	}
}

// TestParseGamemaster_NoFamily confirms the nil-Family branch: a
// species with no family block (pvpoke has 300+ such entries in
// real data) must parse with Evolutions=nil, PreEvolution="" and
// no panic / error. Ditto is the canonical example — no family in
// pvpoke's gamemaster.
func TestParseGamemaster_NoFamily(t *testing.T) {
	t.Parallel()

	gm := loadSampleGamemaster(t)

	ditto, ok := gm.Pokemon["ditto"]
	if !ok {
		t.Fatal("ditto missing from parsed Pokemon map")
	}

	if ditto.PreEvolution != "" {
		t.Errorf("ditto PreEvolution = %q, want empty (no family block)",
			ditto.PreEvolution)
	}
	if ditto.Evolutions != nil {
		t.Errorf("ditto Evolutions = %v, want nil (no family block)",
			ditto.Evolutions)
	}
}

// TestParseGamemaster_BranchingEvolutions exercises species with more
// than one direct child (eevee has 5+ in pvpoke). The godoc on
// Species.Evolutions mentions branching families; this pins it.
func TestParseGamemaster_BranchingEvolutions(t *testing.T) {
	t.Parallel()

	gm := loadSampleGamemaster(t)

	eevee, ok := gm.Pokemon["eevee"]
	if !ok {
		t.Fatal("eevee missing from parsed Pokemon map")
	}

	if eevee.PreEvolution != "" {
		t.Errorf("eevee PreEvolution = %q, want empty (base form)",
			eevee.PreEvolution)
	}
	if len(eevee.Evolutions) < 3 {
		t.Errorf("eevee Evolutions = %v, want ≥ 3 branches", eevee.Evolutions)
	}
	if !slices.Contains(eevee.Evolutions, "vaporeon") {
		t.Errorf("eevee Evolutions = %v, want to contain vaporeon",
			eevee.Evolutions)
	}
}

// TestParseGamemaster_LegacyMoves confirms the legacyMoves top-level
// species field populates Species.LegacyMoves and IsLegacyMove answers
// correctly per (species, move) — the same move id can be regular on
// one species and legacy on another.
func TestParseGamemaster_LegacyMoves(t *testing.T) {
	t.Parallel()

	gm := loadSampleGamemaster(t)

	medi := gm.Pokemon["medicham"]
	wantLegacy := []string{"POWER_UP_PUNCH", "PSYCHIC"}

	if !slices.Equal(medi.LegacyMoves, wantLegacy) {
		t.Errorf("medicham LegacyMoves = %v, want %v", medi.LegacyMoves, wantLegacy)
	}

	if !pogopvp.IsLegacyMove(&medi, "POWER_UP_PUNCH") {
		t.Error("IsLegacyMove(medicham, POWER_UP_PUNCH) = false, want true")
	}
	if !pogopvp.IsLegacyMove(&medi, "PSYCHIC") {
		t.Error("IsLegacyMove(medicham, PSYCHIC) = false, want true")
	}
	if pogopvp.IsLegacyMove(&medi, "COUNTER") {
		t.Error("IsLegacyMove(medicham, COUNTER) = true, want false (regular)")
	}

	// Species without LegacyMoves must report false cleanly.
	bulb := gm.Pokemon["bulbasaur"]
	if pogopvp.IsLegacyMove(&bulb, "VINE_WHIP") {
		t.Error("IsLegacyMove(bulbasaur, VINE_WHIP) = true, want false")
	}

	// Defensive path: nil species returns false, not a panic.
	if pogopvp.IsLegacyMove(nil, "COUNTER") {
		t.Error("IsLegacyMove(nil, COUNTER) = true, want false")
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

// TestParseGamemaster_Cups pins the cup table: the fixture declares
// three cups (all, spring, little), and each one must round-trip
// through ParseGamemaster with the expected fields.
func TestParseGamemaster_Cups(t *testing.T) {
	t.Parallel()

	gm := loadSampleGamemaster(t)

	if len(gm.Cups) != 3 {
		t.Fatalf("Cups len = %d, want 3 (all, spring, little)", len(gm.Cups))
	}

	all, ok := gm.Cups["all"]
	if !ok {
		t.Fatal("Cups missing key \"all\"")
	}
	if all.Title != "All Pokemon" {
		t.Errorf("all.Title = %q, want \"All Pokemon\"", all.Title)
	}
	if len(all.Include) != 0 {
		t.Errorf("all.Include = %v, want empty", all.Include)
	}
	if len(all.Exclude) != 1 || all.Exclude[0].FilterType != "tag" {
		t.Errorf("all.Exclude = %+v, want single tag filter", all.Exclude)
	}

	spring, ok := gm.Cups["spring"]
	if !ok {
		t.Fatal("Cups missing key \"spring\"")
	}
	if spring.PartySize != 3 {
		t.Errorf("spring.PartySize = %d, want 3", spring.PartySize)
	}
	if len(spring.Include) != 1 {
		t.Fatalf("spring.Include len = %d, want 1", len(spring.Include))
	}
	wantTypes := []string{"water", "grass", "fairy"}
	if !slices.Equal(spring.Include[0].Values, wantTypes) {
		t.Errorf("spring.Include[0].Values = %v, want %v",
			spring.Include[0].Values, wantTypes)
	}
	if len(spring.Exclude) != 2 {
		t.Errorf("spring.Exclude len = %d, want 2 (tag + id)", len(spring.Exclude))
	}

	little, ok := gm.Cups["little"]
	if !ok {
		t.Fatal("Cups missing key \"little\"")
	}
	if little.LevelCap != 50 {
		t.Errorf("little.LevelCap = %d, want 50", little.LevelCap)
	}

	// Little Cup uses the "evolution" filter whose upstream payload
	// is a JSON integer array (`"values": [1]`). Round-trip the
	// numeric value through the string slice so the parser does not
	// fall over on pvpoke data — this was the bootstrap-crash hotfix.
	if len(little.Include) != 1 {
		t.Fatalf("little.Include len = %d, want 1 (evolution filter)", len(little.Include))
	}
	if little.Include[0].FilterType != "evolution" {
		t.Errorf("little.Include[0].FilterType = %q, want \"evolution\"",
			little.Include[0].FilterType)
	}
	if !slices.Equal(little.Include[0].Values, []string{"1"}) {
		t.Errorf("little.Include[0].Values = %v, want [\"1\"] (numeric stringified)",
			little.Include[0].Values)
	}
}

// TestParseGamemaster_ThirdMoveCostBooleanFalse pins the second
// bootstrap-crash hotfix: pvpoke emits `"thirdMoveCost": false`
// for species whose second charged move is never unlockable
// (smeargle in current payload). The parser must stringify-false
// to 0 instead of failing the whole decode.
func TestParseGamemaster_ThirdMoveCostBooleanFalse(t *testing.T) {
	t.Parallel()

	raw := `{"id":"gamemaster","pokemon":[` +
		`{"dex":235,"speciesId":"smeargle","speciesName":"Smeargle",` +
		`"baseStats":{"atk":40,"def":88,"hp":146},"types":["normal"],` +
		`"fastMoves":["TACKLE"],"chargedMoves":["STRUGGLE"],` +
		`"thirdMoveCost":false,"released":true}` +
		`],"moves":[` +
		`{"moveId":"TACKLE","name":"Tackle","type":"normal",` +
		`"power":3,"energy":0,"energyGain":3,"cooldown":500,"turns":1},` +
		`{"moveId":"STRUGGLE","name":"Struggle","type":"normal",` +
		`"power":35,"energy":100,"cooldown":500}` +
		`]}`

	gm, err := pogopvp.ParseGamemaster(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("ParseGamemaster: %v (boolean false must not fail decode)", err)
	}

	smeargle, ok := gm.Pokemon["smeargle"]
	if !ok {
		t.Fatal("smeargle missing")
	}
	if smeargle.ThirdMoveCost != 0 {
		t.Errorf("smeargle ThirdMoveCost = %d, want 0 (false → 0 normalisation)",
			smeargle.ThirdMoveCost)
	}
}

// TestParseGamemaster_CupFilterMixedValueTypes pins the normaliser:
// pvpoke may emit strings, integers, or floats inside `cups[].*.values`
// depending on filter type. The parser must stringify each element
// rather than fall over on the first non-string.
func TestParseGamemaster_CupFilterMixedValueTypes(t *testing.T) {
	t.Parallel()

	raw := `{"id":"gamemaster","pokemon":[],"moves":[],"cups":[` +
		`{"name":"mixed","title":"Mixed",` +
		`"include":[` +
		`{"filterType":"evolution","values":[1]},` +
		`{"filterType":"type","values":["water","grass"]},` +
		`{"filterType":"mystery","values":[2.5]}` +
		`],"exclude":[]}` +
		`]}`

	gm, err := pogopvp.ParseGamemaster(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("ParseGamemaster: %v", err)
	}

	cup, ok := gm.Cups["mixed"]
	if !ok {
		t.Fatal("\"mixed\" cup missing")
	}

	if len(cup.Include) != 3 {
		t.Fatalf("Include len = %d, want 3", len(cup.Include))
	}

	if !slices.Equal(cup.Include[0].Values, []string{"1"}) {
		t.Errorf("evolution filter: Values = %v, want [\"1\"]", cup.Include[0].Values)
	}

	if !slices.Equal(cup.Include[1].Values, []string{"water", "grass"}) {
		t.Errorf("type filter: Values = %v, want [\"water\",\"grass\"]",
			cup.Include[1].Values)
	}

	if !slices.Equal(cup.Include[2].Values, []string{"2.5"}) {
		t.Errorf("float filter: Values = %v, want [\"2.5\"]", cup.Include[2].Values)
	}
}

// TestParseGamemaster_CupFilterExoticValueTypes pins the "no data
// is lost" claim in normaliseCupFilterValues for JSON element shapes
// the current pvpoke payload does not produce but might in future:
// booleans, null, and nested objects/arrays should all survive as
// their raw trimmed JSON text instead of getting silently dropped.
func TestParseGamemaster_CupFilterExoticValueTypes(t *testing.T) {
	t.Parallel()

	raw := `{"id":"gamemaster","pokemon":[],"moves":[],"cups":[` +
		`{"name":"exotic","title":"Exotic",` +
		`"include":[{"filterType":"weird","values":[true,null,{"x":1}]}],` +
		`"exclude":[]}` +
		`]}`

	gm, err := pogopvp.ParseGamemaster(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("ParseGamemaster: %v (exotic values must not fail decode)", err)
	}

	values := gm.Cups["exotic"].Include[0].Values
	want := []string{"true", "null", `{"x":1}`}

	if !slices.Equal(values, want) {
		t.Errorf("Values = %v, want %v (raw JSON pass-through)", values, want)
	}
}

// TestParseGamemaster_BuddyDistance pins the buddyDistance field.
// Fixture values: bulbasaur=3, machamp=3, azumarill=3, medicham=3,
// whiscash=1 (from pvpoke payload).
func TestParseGamemaster_BuddyDistance(t *testing.T) {
	t.Parallel()

	gm := loadSampleGamemaster(t)

	whisc, ok := gm.Pokemon["whiscash"]
	if !ok {
		t.Fatal("whiscash missing")
	}
	if whisc.BuddyDistance != 1 {
		t.Errorf("whiscash BuddyDistance = %d, want 1", whisc.BuddyDistance)
	}

	medi, ok := gm.Pokemon["medicham"]
	if !ok {
		t.Fatal("medicham missing")
	}
	if medi.BuddyDistance != 3 {
		t.Errorf("medicham BuddyDistance = %d, want 3", medi.BuddyDistance)
	}

	ditto, ok := gm.Pokemon["ditto"]
	if !ok {
		t.Fatal("ditto missing")
	}
	if ditto.BuddyDistance != 0 {
		t.Errorf("ditto BuddyDistance = %d, want 0 (absent from payload)",
			ditto.BuddyDistance)
	}
}

// TestParseGamemaster_ThirdMoveCost pins the thirdMoveCost field:
// values in the fixture vary by species (bulbasaur=10000, medicham /
// machamp / azumarill=50000, whiscash=10000). Absence in the payload
// (ditto, eevee) surfaces as zero, not a decode failure.
func TestParseGamemaster_ThirdMoveCost(t *testing.T) {
	t.Parallel()

	gm := loadSampleGamemaster(t)

	bulb, ok := gm.Pokemon["bulbasaur"]
	if !ok {
		t.Fatal("bulbasaur missing")
	}
	if bulb.ThirdMoveCost != 10000 {
		t.Errorf("bulbasaur ThirdMoveCost = %d, want 10000", bulb.ThirdMoveCost)
	}

	champ, ok := gm.Pokemon["machamp"]
	if !ok {
		t.Fatal("machamp missing")
	}
	if champ.ThirdMoveCost != 50000 {
		t.Errorf("machamp ThirdMoveCost = %d, want 50000", champ.ThirdMoveCost)
	}

	ditto, ok := gm.Pokemon["ditto"]
	if !ok {
		t.Fatal("ditto missing")
	}
	if ditto.ThirdMoveCost != 0 {
		t.Errorf("ditto ThirdMoveCost = %d, want 0 (absent from payload)",
			ditto.ThirdMoveCost)
	}
}

// TestParseGamemaster_CupEmptyNameRejected pins the validation:
// a cup whose `name` is blank is rejected as ErrGamemasterInvalid
// rather than silently indexed under the empty string.
func TestParseGamemaster_CupEmptyNameRejected(t *testing.T) {
	t.Parallel()

	raw := `{"id":"gamemaster","pokemon":[],"moves":[],"cups":[` +
		`{"name":"","title":"broken","include":[],"exclude":[]}` +
		`]}`
	_, err := pogopvp.ParseGamemaster(strings.NewReader(raw))
	if err == nil {
		t.Fatal("ParseGamemaster with empty cup name expected error")
	}
	if !errors.Is(err, pogopvp.ErrGamemasterInvalid) {
		t.Errorf("error = %v, want wrapping ErrGamemasterInvalid", err)
	}
}

// TestParseGamemaster_CupDuplicateIDRejected pins the dedup check.
func TestParseGamemaster_CupDuplicateIDRejected(t *testing.T) {
	t.Parallel()

	raw := `{"id":"gamemaster","pokemon":[],"moves":[],"cups":[` +
		`{"name":"spring","title":"A","include":[],"exclude":[]},` +
		`{"name":"spring","title":"B","include":[],"exclude":[]}` +
		`]}`
	_, err := pogopvp.ParseGamemaster(strings.NewReader(raw))
	if err == nil {
		t.Fatal("ParseGamemaster with duplicate cup name expected error")
	}
	if !errors.Is(err, pogopvp.ErrGamemasterInvalid) {
		t.Errorf("error = %v, want wrapping ErrGamemasterInvalid", err)
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
