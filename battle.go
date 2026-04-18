package pogopvp

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
// lets full-HP stall matches resolve without capping prematurely.
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

// BattleOptions tunes non-combatant knobs. MaxTurns caps the simulation
// length; a value of 0 disables the cap (the simulation runs until one
// side faints). One turn corresponds to 500 ms of game time.
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

// Simulate plays a match between two combatants using the pvpoke Battle.js
// tick model. Each turn is 500 ms of game time; fast moves fire when their
// cooldown reaches zero, add energy (capped at [MaxEnergy]), and reset the
// cooldown. Whenever a side's energy reaches or exceeds the cheapest
// charged move's cost it fires that charged move — the defender shields
// it if any shields remain, otherwise the full damage lands. The first
// side to reach HP <= 0 loses; simultaneous faints produce [BattleTie];
// running out of turns produces [BattleTimeout].
func Simulate(attacker, defender *Combatant, opts BattleOptions) BattleResult {
	cpmA, _ := CPMAt(attacker.Level)
	cpmB, _ := CPMAt(defender.Level)

	statsA := ComputeStats(attacker.Species.BaseStats, attacker.IV, cpmA)
	statsB := ComputeStats(defender.Species.BaseStats, defender.IV, cpmB)

	state := [2]combatantState{
		initState(attacker, statsA, defender.Species.Types),
		initState(defender, statsB, attacker.Species.Types),
	}
	moves := [2]Combatant{*attacker, *defender}

	maxTurns := opts.MaxTurns
	if maxTurns <= 0 {
		maxTurns = defaultMaxTurns
	}

	turn := 0
	for turn < maxTurns && state[0].hp > 0 && state[1].hp > 0 {
		turn++

		advanceTick(&state, &moves)
	}

	return buildResult(&state, turn, maxTurns)
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
// cooldown hits zero, and resolves charged-move throws in sequence so
// the defender's shield state is consistent when the second throw lands.
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
		if !actedFast[i] {
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
func cheapestAffordable(moves []Move, energy int) chargedMoveChoice {
	best := chargedMoveChoice{Index: -1}

	for i := range moves {
		cost := moves[i].Energy
		if cost > energy {
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
