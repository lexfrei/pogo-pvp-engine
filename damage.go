package pogopvp

import (
	"math"
	"strings"
)

// damageHalfFactor is the flat 0.5 multiplier applied in pvpoke's Battle.js
// damage formula. It predates the PvP-specific BONUS of 1.3; together they
// give the rough ~0.65 scaling factor against raw (atk/def)*effectiveness.
const damageHalfFactor = 0.5

// StabFactor returns StabMultiplier when the move type matches one of the
// attacker's types (Same Type Attack Bonus) and NeutralMatchup otherwise.
// Comparisons are case-insensitive so callers do not need to normalise
// strings before passing them in.
func StabFactor(moveType string, attackerTypes []string) float64 {
	for _, t := range attackerTypes {
		if strings.EqualFold(t, moveType) {
			return StabMultiplier
		}
	}

	return NeutralMatchup
}

// CalcDamage computes the integer damage a move of the given power deals to
// a defender, using pvpoke's Battle.js formula:
//
//	damage = floor(power * stab * (attack/defense) * effectiveness * 0.5 * BONUS) + 1
//
// stab should be [StabMultiplier] when the move matches the attacker's
// type, [NeutralMatchup] otherwise. effectiveness should be the composite
// multiplier returned by [TypeEffectiveness]. The trailing +1 is Niantic's
// PvP minimum-damage rule — no move ever deals less than 1 HP.
func CalcDamage(power int, stab, effectiveness, attack, defense float64) int {
	ratio := attack / defense
	raw := float64(power) * stab * ratio * effectiveness * damageHalfFactor * DamageBonus

	return int(math.Floor(raw)) + 1
}
