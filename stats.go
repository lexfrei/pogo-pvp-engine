package pogopvp

import "math"

// cpFloor is the minimum Combat Power any Pokémon can show. The in-game UI
// never displays a value below 10, and pvpoke's calculateCP clamps to the
// same floor; matching that behaviour keeps our output bit-for-bit
// compatible with upstream rankings.
const cpFloor = 10

// BaseStats holds the species-level attack, defense, and stamina values as
// published in the game master. Values are raw integers; no validation is
// applied here because the gamemaster parser is the single source.
type BaseStats struct {
	Atk int
	Def int
	HP  int
}

// Stats is the post-CPM combatant profile used by the battle simulator and
// the ranking code. Attack and Defense are fractional floats; HP is always
// floored to an integer because the battle engine subtracts whole-point
// damage each turn.
type Stats struct {
	Atk float64
	Def float64
	HP  int
}

// ComputeStats applies the standard PvP formula from pvpoke's Pokemon.js:
// each base stat is offset by its IV component, then multiplied by the
// level's CPM. HP is floored per the in-game rule; attack and defense keep
// their fractional precision.
func ComputeStats(base BaseStats, iv IV, cpm float64) Stats {
	return Stats{
		Atk: float64(base.Atk+int(iv.Atk)) * cpm,
		Def: float64(base.Def+int(iv.Def)) * cpm,
		HP:  int(math.Floor(float64(base.HP+int(iv.Sta)) * cpm)),
	}
}

// ComputeCP returns the displayed Combat Power for a Stats profile. The
// formula mirrors pvpoke: floor(atk * sqrt(def) * sqrt(hp) / 10), clamped
// upwards to [cpFloor].
func ComputeCP(stats Stats) int {
	raw := stats.Atk * math.Sqrt(stats.Def) * math.Sqrt(float64(stats.HP)) / 10
	floored := int(math.Floor(raw))

	if floored < cpFloor {
		return cpFloor
	}

	return floored
}

// ComputeStatProduct returns the stat product used to rank bulk-weighted
// PvP performance: Atk * Def * HP. Larger values indicate a more efficient
// use of the CP budget.
func ComputeStatProduct(stats Stats) float64 {
	return stats.Atk * stats.Def * float64(stats.HP)
}
