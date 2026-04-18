package pogopvp_test

import (
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

	result := pogopvp.Simulate(&combatant, &combatant, pogopvp.BattleOptions{MaxTurns: 200})

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

	result := pogopvp.Simulate(&attacker, &defender, pogopvp.BattleOptions{MaxTurns: 500})

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

	result := pogopvp.Simulate(&attacker, &defender, pogopvp.BattleOptions{MaxTurns: 10})

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

	result := pogopvp.Simulate(&a, &b, pogopvp.BattleOptions{MaxTurns: 20})

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

	result := pogopvp.Simulate(&attacker, &defender, pogopvp.BattleOptions{MaxTurns: 5})

	if result.ChargedFired[0] < 1 {
		t.Errorf("ChargedFired[0] = %d, want >= 1", result.ChargedFired[0])
	}
	// 3 fast hits (3 dmg) + 1 charged (~66 dmg) in 3 turns, plus 2 more fast.
	// Defender HP must drop by at least 60.
	if dropped := 500 - result.HPRemaining[1]; dropped < 60 {
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

	result := pogopvp.Simulate(&attacker, &defender, pogopvp.BattleOptions{MaxTurns: 3})

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

	result := pogopvp.Simulate(&attacker, &defender, pogopvp.BattleOptions{MaxTurns: 5})

	if result.ChargedFired[0] < 2 {
		t.Fatalf("ChargedFired[0] = %d, want >= 2", result.ChargedFired[0])
	}
	if result.ShieldsUsed[1] != 1 {
		t.Errorf("ShieldsUsed[1] = %d, want 1 (shield burns once, rest land)", result.ShieldsUsed[1])
	}
}

func TestSimulate_MaxTurnsStops(t *testing.T) {
	t.Parallel()

	// Attacker deals 1 damage per turn (min floor). Defender has 500 HP
	// → would take 500 turns, so MaxTurns=10 stops early.
	move := pogopvp.Move{
		ID: "PUFF", Type: "normal",
		Power: 1, EnergyGain: 0, Turns: 1,
		Category: pogopvp.MoveCategoryFast,
	}

	// EnergyGain=0 is an invalid fast move per parser, but we construct the Move
	// literal directly here to test the simulator's time cap. Power 1 + tiny
	// atk/def spread means ~1 damage per hit.
	move.EnergyGain = 1

	a := newTestCombatant(50, 200, 200, []string{"normal"}, move, 20.0)
	b := newTestCombatant(50, 200, 500, []string{"normal"}, move, 20.0)

	result := pogopvp.Simulate(&a, &b, pogopvp.BattleOptions{MaxTurns: 10})

	if result.Turns != 10 {
		t.Errorf("Turns = %d, want 10 (capped)", result.Turns)
	}
	if result.Winner != pogopvp.BattleTimeout {
		t.Errorf("Winner = %d, want BattleTimeout (-2)", result.Winner)
	}
}
