package pogopvp

import (
	"slices"
	"strings"
)

// Type effectiveness multipliers, reproduced verbatim from pvpoke's
// src/js/battle/DamageCalculator.js (DamageMultiplier class). Keeping the
// same exact float literals as the JS source ensures bit-for-bit damage
// equality with pvpoke — the 1.60000002... tail is not a typo, it is the
// rounded float32 representation Niantic uses in the client.
const (
	SuperEffective  = 1.60000002384185791015625
	Resisted        = 0.625
	DoubleResisted  = 0.390625
	NeutralMatchup  = 1.0
	StabMultiplier  = 1.2000000476837158203125
	DamageBonus     = 1.2999999523162841796875
	ShadowAtkFactor = 1.2
	ShadowDefFactor = 0.83333331
)

// typeTraits bundles the three lists that describe a defending type's
// interaction with the 18 attacking types. Weaknesses contribute
// SuperEffective, resistances contribute Resisted, and immunities
// contribute DoubleResisted.
type typeTraits struct {
	weaknesses  []string
	resistances []string
	immunities  []string
}

// typeTraitsTable is a domain-constant mapping from defending type name to
// its effectiveness profile. Mirrors the getTypeTraits switch in
// pvpoke's DamageCalculator.js; identical entries yield identical results.
//
//nolint:gochecknoglobals // immutable effectiveness table driven from the Niantic type chart
var typeTraitsTable = map[string]typeTraits{
	"normal":   {weaknesses: []string{"fighting"}, immunities: []string{"ghost"}},
	"fighting": {resistances: []string{"rock", "bug", "dark"}, weaknesses: []string{"flying", "psychic", "fairy"}},
	"flying": {
		resistances: []string{"fighting", "bug", "grass"},
		weaknesses:  []string{"rock", "electric", "ice"},
		immunities:  []string{"ground"},
	},
	"poison": {
		resistances: []string{"fighting", "poison", "bug", "fairy", "grass"},
		weaknesses:  []string{"ground", "psychic"},
	},
	"ground": {
		resistances: []string{"poison", "rock"},
		weaknesses:  []string{"water", "grass", "ice"},
		immunities:  []string{"electric"},
	},
	"rock": {
		resistances: []string{"normal", "flying", "poison", "fire"},
		weaknesses:  []string{"fighting", "ground", "steel", "water", "grass"},
	},
	"bug": {
		resistances: []string{"fighting", "ground", "grass"},
		weaknesses:  []string{"flying", "rock", "fire"},
	},
	"ghost": {
		resistances: []string{"poison", "bug"},
		weaknesses:  []string{"ghost", "dark"},
		immunities:  []string{"normal", "fighting"},
	},
	"steel": {
		resistances: []string{"normal", "flying", "rock", "bug", "steel", "grass", "psychic", "ice", "dragon", "fairy"},
		weaknesses:  []string{"fighting", "ground", "fire"},
		immunities:  []string{"poison"},
	},
	"fire": {
		resistances: []string{"bug", "steel", "fire", "grass", "ice", "fairy"},
		weaknesses:  []string{"ground", "rock", "water"},
	},
	"water": {
		resistances: []string{"steel", "fire", "water", "ice"},
		weaknesses:  []string{"grass", "electric"},
	},
	"grass": {
		resistances: []string{"ground", "water", "grass", "electric"},
		weaknesses:  []string{"flying", "poison", "bug", "fire", "ice"},
	},
	"electric": {
		resistances: []string{"flying", "steel", "electric"},
		weaknesses:  []string{"ground"},
	},
	"psychic": {
		resistances: []string{"fighting", "psychic"},
		weaknesses:  []string{"bug", "ghost", "dark"},
	},
	"ice": {
		resistances: []string{"ice"},
		weaknesses:  []string{"fighting", "fire", "steel", "rock"},
	},
	"dragon": {
		resistances: []string{"fire", "water", "grass", "electric"},
		weaknesses:  []string{"dragon", "ice", "fairy"},
	},
	"dark": {
		resistances: []string{"ghost", "dark"},
		weaknesses:  []string{"fighting", "fairy", "bug"},
		immunities:  []string{"psychic"},
	},
	"fairy": {
		resistances: []string{"fighting", "bug", "dark"},
		weaknesses:  []string{"poison", "steel"},
		immunities:  []string{"dragon"},
	},
}

// TypeEffectiveness returns the composite damage multiplier a move of the
// given type deals to a defender with the given type list. Comparisons are
// case-insensitive; unknown types are treated as neutral. A nil or empty
// defender list yields 1.0.
func TypeEffectiveness(moveType string, defenderTypes []string) float64 {
	attacking := strings.ToLower(moveType)
	result := NeutralMatchup

	for _, defenderType := range defenderTypes {
		traits := typeTraitsTable[strings.ToLower(defenderType)]
		result *= singleTypeMultiplier(attacking, traits)
	}

	return result
}

// singleTypeMultiplier applies the effectiveness rules for one defending
// type against one attacking move type.
func singleTypeMultiplier(attacking string, traits typeTraits) float64 {
	switch {
	case slices.Contains(traits.weaknesses, attacking):
		return SuperEffective
	case slices.Contains(traits.resistances, attacking):
		return Resisted
	case slices.Contains(traits.immunities, attacking):
		return DoubleResisted
	default:
		return NeutralMatchup
	}
}
