package pogopvp_test

import (
	"errors"
	"math"
	"testing"

	pogopvp "github.com/lexfrei/pogo-pvp-engine"
)

func newTestCombatant(atk, def, hp int, types []string, move pogopvp.Move, level float64) pogopvp.Combatant {
	return pogopvp.Combatant{
		Species: pogopvp.Species{
			ID:        "test",
			BaseStats: pogopvp.BaseStats{Atk: atk, Def: def, HP: hp},
			Types:     types,
		},
		IV:       pogopvp.MustNewIV(0, 0, 0),
		Level:    level,
		FastMove: move,
	}
}

func mustSimulate(t *testing.T, a, b *pogopvp.Combatant, opts pogopvp.BattleOptions) pogopvp.BattleResult {
	t.Helper()

	result, err := pogopvp.Simulate(a, b, opts)
	if err != nil {
		t.Fatalf("Simulate: %v", err)
	}

	return result
}

// In a symmetric mirror match with a 1-turn fast move and equal stats,
// both sides deal the same damage on the same tick and must faint on
// the same turn — a deterministic tie.
func TestSimulate_SymmetricTie(t *testing.T) {
	t.Parallel()

	move := pogopvp.Move{
		ID: "TACKLE", Type: "normal",
		Power: 3, EnergyGain: 3, Turns: 1,
		Category: pogopvp.MoveCategoryFast,
	}
	combatant := newTestCombatant(150, 100, 100, []string{"normal"}, move, 50.0)

	result := mustSimulate(t, &combatant, &combatant, pogopvp.BattleOptions{MaxTurns: 200})

	if result.Winner != pogopvp.BattleTie {
		t.Errorf("Winner = %d, want tie (-1)", result.Winner)
	}
	if result.HPRemaining[0] > 0 || result.HPRemaining[1] > 0 {
		t.Errorf("HPRemaining = %v, want both zero for tie", result.HPRemaining)
	}
}

// When the attacker's fast move has a 1-turn cooldown and the defender's
// is 2 turns, the attacker deals roughly double the damage per game tick
// and must win the exchange.
func TestSimulate_FasterWins(t *testing.T) {
	t.Parallel()

	fast := pogopvp.Move{ID: "FAST", Type: "normal", Power: 3, EnergyGain: 3, Turns: 1, Category: pogopvp.MoveCategoryFast}
	slow := pogopvp.Move{ID: "SLOW", Type: "normal", Power: 3, EnergyGain: 3, Turns: 2, Category: pogopvp.MoveCategoryFast}

	attacker := newTestCombatant(150, 100, 100, []string{"normal"}, fast, 50.0)
	defender := newTestCombatant(150, 100, 100, []string{"normal"}, slow, 50.0)

	result := mustSimulate(t, &attacker, &defender, pogopvp.BattleOptions{MaxTurns: 500})

	if result.Winner != 0 {
		t.Errorf("Winner = %d, want 0 (attacker with 1-turn move)", result.Winner)
	}
	if result.HPRemaining[0] <= 0 {
		t.Errorf("attacker HP = %d, want > 0", result.HPRemaining[0])
	}
	if result.HPRemaining[1] != 0 {
		t.Errorf("defender HP = %d, want 0", result.HPRemaining[1])
	}
}

// Energy must accumulate every fast move. A 1-turn fast move with 5
// energy gain used N times should leave attacker with exactly N*5 energy
// (capped at MaxEnergy = 100).
func TestSimulate_EnergyAccumulates(t *testing.T) {
	t.Parallel()

	move := pogopvp.Move{
		ID: "GAIN", Type: "normal",
		Power: 1, EnergyGain: 5, Turns: 1,
		Category: pogopvp.MoveCategoryFast,
	}

	// Tanky defender so neither side faints within 10 turns.
	attacker := newTestCombatant(150, 100, 300, []string{"normal"}, move, 50.0)
	defender := newTestCombatant(80, 300, 300, []string{"normal"}, move, 50.0)

	result := mustSimulate(t, &attacker, &defender, pogopvp.BattleOptions{MaxTurns: 10})

	if result.Turns != 10 {
		t.Errorf("Turns = %d, want 10 (both survive)", result.Turns)
	}
	// Both sides hit 10 times, 5 energy each → 50 energy each, no cap reached.
	if result.EnergyAtEnd[0] != 50 {
		t.Errorf("EnergyAtEnd[0] = %d, want 50", result.EnergyAtEnd[0])
	}
	if result.EnergyAtEnd[1] != 50 {
		t.Errorf("EnergyAtEnd[1] = %d, want 50", result.EnergyAtEnd[1])
	}
}

// MaxEnergy caps the energy pool at 100.
func TestSimulate_EnergyCaps(t *testing.T) {
	t.Parallel()

	move := pogopvp.Move{
		ID: "GAIN", Type: "normal",
		Power: 1, EnergyGain: 20, Turns: 1,
		Category: pogopvp.MoveCategoryFast,
	}

	// Enough HP to survive 20 turns against each other (damage=1 per turn).
	a := newTestCombatant(100, 200, 300, []string{"normal"}, move, 50.0)
	b := newTestCombatant(100, 200, 300, []string{"normal"}, move, 50.0)

	result := mustSimulate(t, &a, &b, pogopvp.BattleOptions{MaxTurns: 20})

	if result.EnergyAtEnd[0] > pogopvp.MaxEnergy {
		t.Errorf("EnergyAtEnd[0] = %d, exceeds MaxEnergy=%d", result.EnergyAtEnd[0], pogopvp.MaxEnergy)
	}
	if result.EnergyAtEnd[0] != pogopvp.MaxEnergy {
		t.Errorf("EnergyAtEnd[0] = %d, want %d (cap reached)", result.EnergyAtEnd[0], pogopvp.MaxEnergy)
	}
}

// TestSimulate_ChargedMoveFires verifies that once the attacker's energy
// reaches the cost of their cheapest charged move, the move fires and
// lands full damage against a shield-less defender.
func TestSimulate_ChargedMoveFires(t *testing.T) {
	t.Parallel()

	fast := pogopvp.Move{
		ID: "JOLT", Type: "normal",
		Power: 1, EnergyGain: 10, Turns: 1,
		Category: pogopvp.MoveCategoryFast,
	}
	charged := pogopvp.Move{
		ID: "BOOM", Type: "normal",
		Power: 100, Energy: 30, Turns: 1,
		Category: pogopvp.MoveCategoryCharged,
	}

	attacker := newTestCombatant(200, 200, 300, []string{"normal"}, fast, 50.0)
	attacker.ChargedMoves = []pogopvp.Move{charged}

	defender := newTestCombatant(50, 200, 500, []string{"normal"}, fast, 50.0)
	defender.Shields = 0

	defenderInitialHP := pogopvp.ComputeStats(
		defender.Species.BaseStats, defender.IV, mustCPM(t, 50.0),
	).HP

	result := mustSimulate(t, &attacker, &defender, pogopvp.BattleOptions{MaxTurns: 5})

	if result.ChargedFired[0] < 1 {
		t.Errorf("ChargedFired[0] = %d, want >= 1", result.ChargedFired[0])
	}
	// 3 fast hits (3 dmg) + 1 charged (~66 dmg) in 3 turns, plus 2 more fast.
	// Defender HP must drop by at least 60.
	if dropped := defenderInitialHP - result.HPRemaining[1]; dropped < 60 {
		t.Errorf("defender HP drop = %d, want >= 60 (charged fired)", dropped)
	}
}

// TestSimulate_ShieldBlocksCharged verifies that a defender with a shield
// reduces the incoming charged damage to 1 and burns one shield.
func TestSimulate_ShieldBlocksCharged(t *testing.T) {
	t.Parallel()

	fast := pogopvp.Move{
		ID: "JOLT", Type: "normal",
		Power: 1, EnergyGain: 10, Turns: 1,
		Category: pogopvp.MoveCategoryFast,
	}
	charged := pogopvp.Move{
		ID: "BOOM", Type: "normal",
		Power: 100, Energy: 30, Turns: 1,
		Category: pogopvp.MoveCategoryCharged,
	}

	attacker := newTestCombatant(200, 200, 300, []string{"normal"}, fast, 50.0)
	attacker.ChargedMoves = []pogopvp.Move{charged}

	defender := newTestCombatant(50, 200, 500, []string{"normal"}, fast, 50.0)
	defender.Shields = 1

	cpm, _ := pogopvp.CPMAt(50.0)
	defenderInitialHP := pogopvp.ComputeStats(defender.Species.BaseStats, defender.IV, cpm).HP

	result := mustSimulate(t, &attacker, &defender, pogopvp.BattleOptions{MaxTurns: 3})

	if result.ChargedFired[0] != 1 {
		t.Fatalf("ChargedFired[0] = %d, want exactly 1", result.ChargedFired[0])
	}
	if result.ShieldsUsed[1] != 1 {
		t.Errorf("ShieldsUsed[1] = %d, want 1", result.ShieldsUsed[1])
	}
	// Defender took 3 fast hits (1 dmg each) + 1 shielded charged (1 dmg) = 4 damage.
	expected := defenderInitialHP - 4
	if result.HPRemaining[1] != expected {
		t.Errorf("HPRemaining[1] = %d, want %d (3 fast + 1 shielded)", result.HPRemaining[1], expected)
	}
}

// TestSimulate_ShieldsDepleteOverMultipleThrows verifies that shields are
// consumed one per incoming charged move and the second throw lands
// without a shield.
func TestSimulate_ShieldsDepleteOverMultipleThrows(t *testing.T) {
	t.Parallel()

	fast := pogopvp.Move{
		ID: "JOLT", Type: "normal",
		Power: 1, EnergyGain: 20, Turns: 1,
		Category: pogopvp.MoveCategoryFast,
	}
	charged := pogopvp.Move{
		ID: "BOOM", Type: "normal",
		Power: 50, Energy: 20, Turns: 1,
		Category: pogopvp.MoveCategoryCharged,
	}

	attacker := newTestCombatant(200, 200, 300, []string{"normal"}, fast, 50.0)
	attacker.ChargedMoves = []pogopvp.Move{charged}

	defender := newTestCombatant(50, 200, 500, []string{"normal"}, fast, 50.0)
	defender.Shields = 1

	result := mustSimulate(t, &attacker, &defender, pogopvp.BattleOptions{MaxTurns: 5})

	if result.ChargedFired[0] < 2 {
		t.Fatalf("ChargedFired[0] = %d, want >= 2", result.ChargedFired[0])
	}
	if result.ShieldsUsed[1] != 1 {
		t.Errorf("ShieldsUsed[1] = %d, want 1 (shield burns once, rest land)", result.ShieldsUsed[1])
	}
}

// TestSimulate_MultipleChargedPicksCheapest verifies that when the
// attacker carries two charged moves the cheapest affordable one fires
// first, not the first-in-slice.
func TestSimulate_MultipleChargedPicksCheapest(t *testing.T) {
	t.Parallel()

	fast := pogopvp.Move{
		ID: "JOLT", Type: "normal",
		Power: 1, EnergyGain: 10, Turns: 1,
		Category: pogopvp.MoveCategoryFast,
	}
	expensive := pogopvp.Move{
		ID: "BIG", Type: "normal",
		Power: 200, Energy: 60, Turns: 1,
		Category: pogopvp.MoveCategoryCharged,
	}
	cheap := pogopvp.Move{
		ID: "CHEAP", Type: "normal",
		Power: 30, Energy: 30, Turns: 1,
		Category: pogopvp.MoveCategoryCharged,
	}

	// Big first in slice, cheap second — the simulator must still pick
	// cheap when only 30 energy is available after turn 3.
	attacker := newTestCombatant(200, 200, 300, []string{"normal"}, fast, 50.0)
	attacker.ChargedMoves = []pogopvp.Move{expensive, cheap}

	defender := newTestCombatant(50, 200, 500, []string{"normal"}, fast, 50.0)
	defender.Shields = 0

	result := mustSimulate(t, &attacker, &defender, pogopvp.BattleOptions{MaxTurns: 3})

	if result.ChargedFired[0] != 1 {
		t.Fatalf("ChargedFired[0] = %d, want 1 (only cheap fits in 30 energy)", result.ChargedFired[0])
	}
	// If the expensive move had fired instead, the hit would have been
	// much bigger. Confirm the cheap-move signature: defender HP dropped
	// by roughly cheap's damage + 3 fast hits, not expensive's.
	dropped := pogopvp.ComputeStats(defender.Species.BaseStats, defender.IV, mustCPM(t, 50.0)).HP - result.HPRemaining[1]
	if dropped > 80 {
		t.Errorf("defender HP drop = %d, too large — expensive move likely fired", dropped)
	}
}

// TestSimulate_DefenderDeathCancelsThrowback checks the death-cancels
// rule: a side that faints from fast-move damage this tick does not get
// to fire its own charged move the same tick.
func TestSimulate_DefenderDeathCancelsThrowback(t *testing.T) {
	t.Parallel()

	lethalFast := pogopvp.Move{
		ID: "LETHAL", Type: "normal",
		Power: 200, EnergyGain: 50, Turns: 1,
		Category: pogopvp.MoveCategoryFast,
	}
	weakFast := pogopvp.Move{
		ID: "WEAK", Type: "normal",
		Power: 1, EnergyGain: 50, Turns: 1,
		Category: pogopvp.MoveCategoryFast,
	}
	charged := pogopvp.Move{
		ID: "NUKE", Type: "normal",
		Power: 100, Energy: 50, Turns: 1,
		Category: pogopvp.MoveCategoryCharged,
	}

	// Attacker's fast move will one-shot the defender on turn 1. Even
	// though defender reaches 50 energy the same tick (enough for its
	// charged move), it is already dead and must not throw back.
	attacker := newTestCombatant(250, 250, 250, []string{"normal"}, lethalFast, 50.0)

	defender := newTestCombatant(50, 20, 20, []string{"normal"}, weakFast, 10.0)
	defender.ChargedMoves = []pogopvp.Move{charged}

	result := mustSimulate(t, &attacker, &defender, pogopvp.BattleOptions{MaxTurns: 5})

	if result.Winner != 0 {
		t.Errorf("Winner = %d, want 0 (attacker)", result.Winner)
	}
	if result.ChargedFired[1] != 0 {
		t.Errorf("ChargedFired[1] = %d, want 0 (defender died same tick)", result.ChargedFired[1])
	}
}

// TestSimulate_InvalidCombatantReturnsError tests every branch of
// Combatant.Valid via the Simulate entry point.
func TestSimulate_InvalidCombatantReturnsError(t *testing.T) {
	t.Parallel()

	valid := pogopvp.Combatant{
		Species: pogopvp.Species{
			ID:        "ok",
			BaseStats: pogopvp.BaseStats{Atk: 100, Def: 100, HP: 100},
			Types:     []string{"normal"},
		},
		IV:    pogopvp.MustNewIV(0, 0, 0),
		Level: 30.0,
		FastMove: pogopvp.Move{
			ID: "OK", Type: "normal", Power: 1, EnergyGain: 3, Turns: 1,
			Category: pogopvp.MoveCategoryFast,
		},
	}

	cases := []struct {
		name  string
		tweak func(*pogopvp.Combatant)
	}{
		{"level nan", func(c *pogopvp.Combatant) { c.Level = math.NaN() }},
		{"level below min", func(c *pogopvp.Combatant) { c.Level = 0.5 }},
		{"level above max", func(c *pogopvp.Combatant) { c.Level = 99.0 }},
		{"level off grid", func(c *pogopvp.Combatant) { c.Level = 30.25 }},
		{"iv out of range", func(c *pogopvp.Combatant) { c.IV = pogopvp.IV{Atk: 200, Def: 0, Sta: 0} }},
		{"fast move no energy gain", func(c *pogopvp.Combatant) { c.FastMove.EnergyGain = 0 }},
		{"fast move zero turns", func(c *pogopvp.Combatant) { c.FastMove.Turns = 0 }},
		{"negative shields", func(c *pogopvp.Combatant) { c.Shields = -1 }},
		{"charged move zero energy", func(c *pogopvp.Combatant) {
			c.ChargedMoves = []pogopvp.Move{{ID: "BAD", Energy: 0}}
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			broken := valid
			tc.tweak(&broken)

			_, err := pogopvp.Simulate(&broken, &valid, pogopvp.BattleOptions{MaxTurns: 10})
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !errors.Is(err, pogopvp.ErrInvalidCombatant) {
				t.Errorf("error = %v, want wrapping ErrInvalidCombatant", err)
			}
		})
	}
}

// mustCPM is a test helper that fetches CPM for a level, failing the
// test on unexpected CPMAt errors (every call site uses a known-valid
// level).
func mustCPM(t *testing.T, level float64) float64 {
	t.Helper()

	cpm, err := pogopvp.CPMAt(level)
	if err != nil {
		t.Fatalf("CPMAt(%v): %v", level, err)
	}

	return cpm
}

func TestSimulate_MaxTurnsStops(t *testing.T) {
	t.Parallel()

	// Attacker deals 1 damage per turn (min floor). Both sides have
	// bulky base stats; level 20 produces roughly 200 HP after CPM
	// flooring, so MaxTurns=10 stops well before a faint.
	move := pogopvp.Move{
		ID: "PUFF", Type: "normal",
		Power: 1, Turns: 1,
		Category: pogopvp.MoveCategoryFast,
	}

	// The simulator's Combatant.Valid requires EnergyGain > 0; set it to
	// 1 so the move is legal while keeping per-hit damage near the +1
	// minimum floor to guarantee the MaxTurns path triggers.
	move.EnergyGain = 1

	a := newTestCombatant(50, 200, 200, []string{"normal"}, move, 20.0)
	b := newTestCombatant(50, 200, 500, []string{"normal"}, move, 20.0)

	result := mustSimulate(t, &a, &b, pogopvp.BattleOptions{MaxTurns: 10})

	if result.Turns != 10 {
		t.Errorf("Turns = %d, want 10 (capped)", result.Turns)
	}
	if result.Winner != pogopvp.BattleTimeout {
		t.Errorf("Winner = %d, want BattleTimeout (-2)", result.Winner)
	}
}
