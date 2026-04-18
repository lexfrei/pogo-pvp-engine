package pogopvp

import "math"

// cpFloor is the minimum CP the in-game display can show. pvpoke mirrors
// this floor in its calculateCP for parity with the UI; we do the same so
// ComputeCP and the in-game display never disagree in edge cases.
const cpFloor = 10

// hpFloor is the minimum HP a Pokémon can have. pvpoke clamps this in
// Pokemon.js (`Math.max(..., 10)`) and the in-game HP bar respects the same
// floor. Reachable only for very low base stamina at level 1.
const hpFloor = 10

// BaseStats holds the species-level attack, defense, and stamina values as
// published in the game master. Values are raw integers; no validation is
// applied here because the gamemaster parser is the single source.
type BaseStats struct {
	Atk int
	Def int
	HP  int
}

// Stats is the post-CPM combatant profile used by the battle simulator and
// the ranking code. Attack and Defense are fractional floats; HP is floored
// and clamped upwards to [hpFloor] so the battle engine always subtracts
// against a valid integer HP pool.
type Stats struct {
	Atk float64
	Def float64
	HP  int
}

// ComputeStats applies the standard PvP formula from pvpoke's Pokemon.js:
// each base stat is offset by its IV component, then multiplied by the
// level's CPM. Attack and defense keep their fractional precision; HP is
// floored per the in-game rule and clamped upwards to [hpFloor].
func ComputeStats(base BaseStats, iv IV, cpm float64) Stats {
	hpRaw := float64(base.HP+int(iv.Sta)) * cpm
	stamina := max(int(math.Floor(hpRaw)), hpFloor)

	return Stats{
		Atk: float64(base.Atk+int(iv.Atk)) * cpm,
		Def: float64(base.Def+int(iv.Def)) * cpm,
		HP:  stamina,
	}
}

// ComputeCP returns the displayed Combat Power as computed by pvpoke's
// Pokemon.js calculateCP:
//
//	floor((baseAtk+ivAtk)*cpm * sqrt((baseDef+ivDef)*cpm) * sqrt((baseHp+ivSta)*cpm) / 10)
//
// This is algebraically equivalent to
//
//	floor((baseAtk+ivAtk) * sqrt(baseDef+ivDef) * sqrt(baseHp+ivSta) * cpm^2 / 10)
//
// which is the form most commonly cited for the Niantic formula. HP is NOT
// floored before entering the sqrt — that's why ComputeCP takes raw base
// stats + IV + cpm rather than a [Stats] value. The result is clamped
// upwards to [cpFloor] for parity with the in-game display.
func ComputeCP(base BaseStats, iv IV, cpm float64) int {
	atk := float64(base.Atk+int(iv.Atk)) * cpm
	def := float64(base.Def+int(iv.Def)) * cpm
	sta := float64(base.HP+int(iv.Sta)) * cpm

	raw := atk * math.Sqrt(def) * math.Sqrt(sta) / 10

	return max(int(math.Floor(raw)), cpFloor)
}

// ComputeStatProduct returns the stat product used to rank bulk-weighted
// PvP performance: Atk * Def * HP (with HP floored, matching pvpoke's
// ranker). Larger values indicate a more efficient use of the CP budget.
func ComputeStatProduct(stats Stats) float64 {
	return stats.Atk * stats.Def * float64(stats.HP)
}
