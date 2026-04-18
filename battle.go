package pogopvp

import (
	"errors"
	"fmt"
	"math"
)

// ErrInvalidCombatant is returned by [Simulate] when either combatant has
// invalid state — level outside the valid grid, out-of-range IV, or a
// fast move that cannot generate energy.
var ErrInvalidCombatant = errors.New("invalid combatant")

// MaxEnergy is the upper bound on the energy pool. Both fast-move gains
// and unused charged-move reservoirs clamp here.
const MaxEnergy = 100

// MaxShields is the starting shield count pvpoke uses for a 1v1 match.
const MaxShields = 2

// Battle outcome sentinels exposed alongside the 0 / 1 winner indices.
// BattleTie means both combatants fainted on the same tick; BattleTimeout
// means MaxTurns elapsed with both still alive.
const (
	BattleTie     = -1
	BattleTimeout = -2
)

// defaultMaxTurns is a generous ceiling (~4 minutes of game time) that
// lets full-HP stall matches resolve without capping prematurely when the
// caller does not set [BattleOptions.MaxTurns].
const defaultMaxTurns = 500

// shieldedDamage is the damage a successfully shielded charged move deals.
// In-game a shield blocks the whole hit and the attacker gets 1 damage of
// chip through, matching the pvpoke rule.
const shieldedDamage = 1

// Combatant is the data the simulator needs to run a Pokémon through a
// single battle: base stats + IV + level, types for STAB and effectiveness,
// the fast move driving the cooldown loop, optional charged moves the AI
// can fire when energy is sufficient, and the starting shield count.
type Combatant struct {
	Species      Species
	IV           IV
	Level        float64
	FastMove     Move
	ChargedMoves []Move
	Shields      int
}

// Valid performs the invariant checks [Simulate] applies on entry: level
// on the 0.5 grid in [MinLevel, MaxLevel], IV inside [0, MaxIV], a fast
// move that produces energy, and non-negative shields.
func (c *Combatant) Valid() error {
	err := validateCombatantLevel(c.Level)
	if err != nil {
		return err
	}

	if !c.IV.Valid() {
		return fmt.Errorf("%w: IV out of range: %+v", ErrInvalidCombatant, c.IV)
	}

	err = validateCombatantMoves(c)
	if err != nil {
		return err
	}

	if c.Shields < 0 || c.Shields > MaxShields {
		return fmt.Errorf("%w: shield count %d outside [0, %d]",
			ErrInvalidCombatant, c.Shields, MaxShields)
	}

	return nil
}

func validateCombatantLevel(level float64) error {
	if math.IsNaN(level) || level < MinLevel || level > MaxLevel {
		return fmt.Errorf("%w: level %v outside [%.1f, %.1f]",
			ErrInvalidCombatant, level, MinLevel, MaxLevel)
	}

	if doubled := level * 2; doubled != math.Trunc(doubled) {
		return fmt.Errorf("%w: level %v is not on the 0.5 grid", ErrInvalidCombatant, level)
	}

	return nil
}

func validateCombatantMoves(c *Combatant) error {
	if c.FastMove.EnergyGain <= 0 {
		return fmt.Errorf("%w: fast move %q has non-positive EnergyGain=%d",
			ErrInvalidCombatant, c.FastMove.ID, c.FastMove.EnergyGain)
	}

	if c.FastMove.Turns <= 0 {
		return fmt.Errorf("%w: fast move %q has non-positive Turns=%d",
			ErrInvalidCombatant, c.FastMove.ID, c.FastMove.Turns)
	}

	for _, move := range c.ChargedMoves {
		if move.Energy <= 0 {
			return fmt.Errorf("%w: charged move %q has non-positive Energy=%d",
				ErrInvalidCombatant, move.ID, move.Energy)
		}
	}

	return nil
}

// BattleOptions tunes non-combatant knobs. MaxTurns is an upper bound on
// simulation length; 0 defaults to [defaultMaxTurns]. Pass a very large
// number (math.MaxInt32) for an effectively unbounded simulation. One
// turn corresponds to 500 ms of game time.
type BattleOptions struct {
	MaxTurns int
}

// BattleResult reports the outcome of [Simulate]. Winner is 0 or 1 for a
// decisive match, [BattleTie] for a simultaneous faint, and
// [BattleTimeout] if the MaxTurns cap kicked in with both still alive.
type BattleResult struct {
	Winner       int
	Turns        int
	HPRemaining  [2]int
	EnergyAtEnd  [2]int
	ShieldsUsed  [2]int
	ChargedFired [2]int
}

// combatantState is the per-side mutable state carried across turns.
type combatantState struct {
	hp            int
	energy        int
	cooldown      int // turns remaining until next fast move fires
	attack        float64
	defense       float64
	fastStab      float64
	fastEffect    float64
	chargedStab   []float64 // parallel to [Combatant.ChargedMoves]
	chargedEffect []float64
	shields       int
	chargedFired  int
	shieldsUsed   int
}

// Simulate plays a match between two combatants using a simplified
// pvpoke-inspired tick model: each turn is 500 ms of game time; fast
// moves fire when their cooldown reaches zero, add energy (capped at
// [MaxEnergy]), and reset the cooldown. Whenever a side's energy covers
// the cheapest affordable charged move it fires — the defender shields
// if any remain, otherwise the full damage lands. A side that faints
// from the fast-damage round of a tick does not get to throw a charged
// move that same tick.
//
// The simulator is intentionally simpler than upstream pvpoke Battle.js:
// it does not model Charge-Move-Priority (simultaneous throws resolve
// in index order), does not apply shadow Atk/Def factors, and resolves
// fast damage before charged throws on the shared tick. These are
// known gaps that the ranker will need to address before full parity.
//
// Returns [ErrInvalidCombatant] wrapping the specific field that failed
// validation if either combatant carries out-of-range state.
func Simulate(attacker, defender *Combatant, opts BattleOptions) (BattleResult, error) {
	err := attacker.Valid()
	if err != nil {
		return BattleResult{}, fmt.Errorf("attacker: %w", err)
	}

	err = defender.Valid()
	if err != nil {
		return BattleResult{}, fmt.Errorf("defender: %w", err)
	}

	cpmA, err := CPMAt(attacker.Level)
	if err != nil {
		return BattleResult{}, fmt.Errorf("attacker: %w", err)
	}

	cpmB, err := CPMAt(defender.Level)
	if err != nil {
		return BattleResult{}, fmt.Errorf("defender: %w", err)
	}

	statsA := ComputeStats(attacker.Species.BaseStats, attacker.IV, cpmA)
	statsB := ComputeStats(defender.Species.BaseStats, defender.IV, cpmB)

	state := [2]combatantState{
		initState(attacker, statsA, defender.Species.Types),
		initState(defender, statsB, attacker.Species.Types),
	}
	combatants := [2]Combatant{*attacker, *defender}

	maxTurns := opts.MaxTurns
	if maxTurns <= 0 {
		maxTurns = defaultMaxTurns
	}

	turn := 0
	for turn < maxTurns && state[0].hp > 0 && state[1].hp > 0 {
		turn++

		advanceTick(&state, &combatants)
	}

	return buildResult(&state, turn, maxTurns), nil
}

// buildResult collects the final counters into a [BattleResult]. HPs are
// clamped to zero so callers don't see negative numbers when a move
// over-killed its target.
func buildResult(state *[2]combatantState, turn, maxTurns int) BattleResult {
	return BattleResult{
		Winner:       decideWinner(state, turn, maxTurns),
		Turns:        turn,
		HPRemaining:  [2]int{max(state[0].hp, 0), max(state[1].hp, 0)},
		EnergyAtEnd:  [2]int{state[0].energy, state[1].energy},
		ShieldsUsed:  [2]int{state[0].shieldsUsed, state[1].shieldsUsed},
		ChargedFired: [2]int{state[0].chargedFired, state[1].chargedFired},
	}
}

// initState seeds one combatant's simulation state from its computed
// fight stats and the opponent's defensive types. STAB and effectiveness
// for every move are precomputed to keep the tick-loop branch-free.
func initState(combatant *Combatant, stats Stats, opponentTypes []string) combatantState {
	stab := StabFactor(combatant.FastMove.Type, combatant.Species.Types)
	effect := TypeEffectiveness(combatant.FastMove.Type, opponentTypes)

	chargedStab := make([]float64, len(combatant.ChargedMoves))
	chargedEffect := make([]float64, len(combatant.ChargedMoves))

	for i, move := range combatant.ChargedMoves {
		chargedStab[i] = StabFactor(move.Type, combatant.Species.Types)
		chargedEffect[i] = TypeEffectiveness(move.Type, opponentTypes)
	}

	return combatantState{
		hp:            stats.HP,
		energy:        0,
		cooldown:      combatant.FastMove.Turns,
		attack:        stats.Atk,
		defense:       stats.Def,
		fastStab:      stab,
		fastEffect:    effect,
		chargedStab:   chargedStab,
		chargedEffect: chargedEffect,
		shields:       combatant.Shields,
	}
}

// advanceTick decrements both cooldowns, fires any fast moves whose
// cooldown hits zero, and resolves charged-move throws. A side that has
// already fainted to fast-move damage this tick does not throw back —
// the death-cancels-throwback rule matches in-game behaviour for
// simultaneous fast/charged resolution.
func advanceTick(state *[2]combatantState, combatants *[2]Combatant) {
	var actedFast [2]bool

	for i := range state {
		state[i].cooldown--
		if state[i].cooldown <= 0 {
			actedFast[i] = true
			state[i].cooldown = combatants[i].FastMove.Turns
		}
	}

	fastDamage := fastMoveDamage(state, combatants, &actedFast)
	for i := range state {
		state[i].hp -= fastDamage[i]
	}

	for i := range state {
		if !actedFast[i] || state[i].hp <= 0 {
			continue
		}

		tryChargedMove(state, combatants, i)
	}
}

// fastMoveDamage computes the damage each side will take from the other's
// fast move this tick, accumulating energy for the acting side. Damage is
// reported symmetrically so callers can apply it simultaneously.
func fastMoveDamage(state *[2]combatantState, combatants *[2]Combatant, acted *[2]bool) [2]int {
	var damage [2]int

	for i := range state {
		if !acted[i] {
			continue
		}

		opponent := 1 - i
		move := combatants[i].FastMove

		damage[opponent] = CalcDamage(
			move.Power, state[i].fastStab, state[i].fastEffect,
			state[i].attack, state[opponent].defense,
		)
		state[i].energy = min(state[i].energy+move.EnergyGain, MaxEnergy)
	}

	return damage
}

// tryChargedMove fires the cheapest affordable charged move for side i.
// The opponent shields it if any shields remain, otherwise the full
// damage lands. Energy is deducted up-front so repeated throws cannot
// leak budget.
func tryChargedMove(state *[2]combatantState, combatants *[2]Combatant, i int) {
	moves := combatants[i].ChargedMoves
	if len(moves) == 0 {
		return
	}

	choice := cheapestAffordable(moves, state[i].energy)
	if choice.Index < 0 {
		return
	}

	opponent := 1 - i
	move := moves[choice.Index]

	state[i].energy -= choice.Cost
	state[i].chargedFired++

	damage := CalcDamage(
		move.Power, state[i].chargedStab[choice.Index], state[i].chargedEffect[choice.Index],
		state[i].attack, state[opponent].defense,
	)

	if state[opponent].shields > 0 {
		state[opponent].shields--
		state[opponent].shieldsUsed++
		state[opponent].hp -= shieldedDamage

		return
	}

	state[opponent].hp -= damage
}

// chargedMoveChoice reports the cheapest charged-move selection for a
// given energy pool. A negative Index means no charged move is affordable.
type chargedMoveChoice struct {
	Index int
	Cost  int
}

// cheapestAffordable picks the cheapest charged move whose cost fits in
// the given energy pool. An Index < 0 signals nothing is affordable.
// Cost must be positive to prevent a zero-cost infinite-fire loop — the
// gamemaster parser rejects such moves but this function is exported and
// guards itself. Ties on cost resolve to the lower index.
func cheapestAffordable(moves []Move, energy int) chargedMoveChoice {
	best := chargedMoveChoice{Index: -1}

	for i := range moves {
		cost := moves[i].Energy
		if cost <= 0 || cost > energy {
			continue
		}

		if best.Index < 0 || cost < best.Cost {
			best.Index = i
			best.Cost = cost
		}
	}

	return best
}

// decideWinner maps final HP values to the canonical winner code. The
// timeout branch fires only when MaxTurns elapsed with both above zero.
func decideWinner(state *[2]combatantState, turn, maxTurns int) int {
	hpA, hpB := state[0].hp, state[1].hp

	switch {
	case hpA <= 0 && hpB <= 0:
		return BattleTie
	case hpA <= 0:
		return 1
	case hpB <= 0:
		return 0
	case turn >= maxTurns:
		return BattleTimeout
	default:
		return BattleTie
	}
}
