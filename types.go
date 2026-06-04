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

// Pokémon type names, used as keys and list entries in the effectiveness
// table. Defined as named constants so the table and its tests share a
// single source of truth for the 18 canonical type identifiers.
const (
	TypeNormal   = "normal"
	TypeFighting = "fighting"
	TypeFlying   = "flying"
	TypePoison   = "poison"
	TypeGround   = "ground"
	TypeRock     = "rock"
	TypeBug      = "bug"
	TypeGhost    = "ghost"
	TypeSteel    = "steel"
	TypeFire     = "fire"
	TypeWater    = "water"
	TypeGrass    = "grass"
	TypeElectric = "electric"
	TypePsychic  = "psychic"
	TypeIce      = "ice"
	TypeDragon   = "dragon"
	TypeDark     = "dark"
	TypeFairy    = "fairy"
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
	TypeNormal:   {weaknesses: []string{TypeFighting}, immunities: []string{TypeGhost}},
	TypeFighting: {resistances: []string{TypeRock, TypeBug, TypeDark}, weaknesses: []string{TypeFlying, TypePsychic, TypeFairy}},
	TypeFlying: {
		resistances: []string{TypeFighting, TypeBug, TypeGrass},
		weaknesses:  []string{TypeRock, TypeElectric, TypeIce},
		immunities:  []string{TypeGround},
	},
	TypePoison: {
		resistances: []string{TypeFighting, TypePoison, TypeBug, TypeFairy, TypeGrass},
		weaknesses:  []string{TypeGround, TypePsychic},
	},
	TypeGround: {
		resistances: []string{TypePoison, TypeRock},
		weaknesses:  []string{TypeWater, TypeGrass, TypeIce},
		immunities:  []string{TypeElectric},
	},
	TypeRock: {
		resistances: []string{TypeNormal, TypeFlying, TypePoison, TypeFire},
		weaknesses:  []string{TypeFighting, TypeGround, TypeSteel, TypeWater, TypeGrass},
	},
	TypeBug: {
		resistances: []string{TypeFighting, TypeGround, TypeGrass},
		weaknesses:  []string{TypeFlying, TypeRock, TypeFire},
	},
	TypeGhost: {
		resistances: []string{TypePoison, TypeBug},
		weaknesses:  []string{TypeGhost, TypeDark},
		immunities:  []string{TypeNormal, TypeFighting},
	},
	TypeSteel: {
		resistances: []string{TypeNormal, TypeFlying, TypeRock, TypeBug, TypeSteel, TypeGrass, TypePsychic, TypeIce, TypeDragon, TypeFairy},
		weaknesses:  []string{TypeFighting, TypeGround, TypeFire},
		immunities:  []string{TypePoison},
	},
	TypeFire: {
		resistances: []string{TypeBug, TypeSteel, TypeFire, TypeGrass, TypeIce, TypeFairy},
		weaknesses:  []string{TypeGround, TypeRock, TypeWater},
	},
	TypeWater: {
		resistances: []string{TypeSteel, TypeFire, TypeWater, TypeIce},
		weaknesses:  []string{TypeGrass, TypeElectric},
	},
	TypeGrass: {
		resistances: []string{TypeGround, TypeWater, TypeGrass, TypeElectric},
		weaknesses:  []string{TypeFlying, TypePoison, TypeBug, TypeFire, TypeIce},
	},
	TypeElectric: {
		resistances: []string{TypeFlying, TypeSteel, TypeElectric},
		weaknesses:  []string{TypeGround},
	},
	TypePsychic: {
		resistances: []string{TypeFighting, TypePsychic},
		weaknesses:  []string{TypeBug, TypeGhost, TypeDark},
	},
	TypeIce: {
		resistances: []string{TypeIce},
		weaknesses:  []string{TypeFighting, TypeFire, TypeSteel, TypeRock},
	},
	TypeDragon: {
		resistances: []string{TypeFire, TypeWater, TypeGrass, TypeElectric},
		weaknesses:  []string{TypeDragon, TypeIce, TypeFairy},
	},
	TypeDark: {
		resistances: []string{TypeGhost, TypeDark},
		weaknesses:  []string{TypeFighting, TypeFairy, TypeBug},
		immunities:  []string{TypePsychic},
	},
	TypeFairy: {
		resistances: []string{TypeFighting, TypeBug, TypeDark},
		weaknesses:  []string{TypePoison, TypeSteel},
		immunities:  []string{TypeDragon},
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
