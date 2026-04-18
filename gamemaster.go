package pogopvp

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// MoveCategory distinguishes fast moves (generate energy, repeat on cooldown)
// from charged moves (consume energy, trigger shields).
type MoveCategory uint8

// MoveCategory constants. Zero value is Fast so that Move{}.Category defaults
// to the more common case.
const (
	MoveCategoryFast MoveCategory = iota
	MoveCategoryCharged
)

// Move describes a PvP move as it appears in the gamemaster. Fast moves
// carry a non-zero EnergyGain and Turns count; charged moves carry a
// non-zero Energy cost. Category is inferred during parsing.
type Move struct {
	ID         string
	Name       string
	Type       string
	Power      int
	Energy     int
	EnergyGain int
	Cooldown   int
	Turns      int
	Category   MoveCategory
}

// Species is one Pokémon entry from the gamemaster. Base stats and types
// are authoritative for CP / stat-product math; the move slices list legal
// choices keyed into [Gamemaster.Moves].
type Species struct {
	Dex          int
	ID           string
	Name         string
	BaseStats    BaseStats
	Types        []string
	FastMoves    []string
	ChargedMoves []string
	Tags         []string
	Released     bool
}

// Gamemaster is the parsed and indexed view of the pvpoke gamemaster file.
// Pokemon and Moves are keyed by their canonical ID so lookups are O(1).
// Version captures the source timestamp for cache invalidation.
type Gamemaster struct {
	Version string
	Pokemon map[string]Species
	Moves   map[string]Move
}

// ErrGamemasterDecode wraps JSON-syntax errors from the underlying decoder.
var ErrGamemasterDecode = errors.New("gamemaster decode error")

// ErrGamemasterInvalid flags semantic violations (missing required fields,
// duplicate IDs) in an otherwise syntactically valid payload.
var ErrGamemasterInvalid = errors.New("gamemaster invalid")

// gamemasterRaw mirrors the on-disk gamemaster JSON layout; only the fields
// the engine actually reads are represented. Unknown fields are ignored.
// Timestamp is a human-readable string in upstream payloads, not an int.
type gamemasterRaw struct {
	Timestamp string       `json:"timestamp"`
	ID        string       `json:"id"`
	Pokemon   []speciesRaw `json:"pokemon"`
	Moves     []moveRaw    `json:"moves"`
}

type speciesRaw struct {
	Dex          int          `json:"dex"`
	SpeciesID    string       `json:"speciesId"`
	SpeciesName  string       `json:"speciesName"`
	BaseStats    baseStatsRaw `json:"baseStats"`
	Types        []string     `json:"types"`
	FastMoves    []string     `json:"fastMoves"`
	ChargedMoves []string     `json:"chargedMoves"`
	Tags         []string     `json:"tags"`
	Released     bool         `json:"released"`
}

type baseStatsRaw struct {
	Atk int `json:"atk"`
	Def int `json:"def"`
	HP  int `json:"hp"`
}

type moveRaw struct {
	MoveID     string `json:"moveId"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Power      int    `json:"power"`
	Energy     int    `json:"energy"`
	EnergyGain int    `json:"energyGain"`
	Cooldown   int    `json:"cooldown"`
	Turns      int    `json:"turns"`
}

// ParseGamemaster reads a gamemaster JSON document from the reader and
// returns the indexed view. Decode errors wrap [ErrGamemasterDecode];
// schema violations wrap [ErrGamemasterInvalid].
func ParseGamemaster(reader io.Reader) (*Gamemaster, error) {
	var raw gamemasterRaw

	decoder := json.NewDecoder(reader)

	err := decoder.Decode(&raw)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrGamemasterDecode, err)
	}

	gamemaster := &Gamemaster{
		Version: raw.Timestamp,
		Pokemon: make(map[string]Species, len(raw.Pokemon)),
		Moves:   make(map[string]Move, len(raw.Moves)),
	}

	for index := range raw.Pokemon {
		species, err := convertSpecies(index, &raw.Pokemon[index])
		if err != nil {
			return nil, err
		}

		_, exists := gamemaster.Pokemon[species.ID]
		if exists {
			return nil, fmt.Errorf("%w: duplicate speciesId %q", ErrGamemasterInvalid, species.ID)
		}

		gamemaster.Pokemon[species.ID] = species
	}

	for index := range raw.Moves {
		move, err := convertMove(index, &raw.Moves[index])
		if err != nil {
			return nil, err
		}

		_, exists := gamemaster.Moves[move.ID]
		if exists {
			return nil, fmt.Errorf("%w: duplicate moveId %q", ErrGamemasterInvalid, move.ID)
		}

		gamemaster.Moves[move.ID] = move
	}

	return gamemaster, nil
}

// convertSpecies promotes one raw entry into a Species, reporting
// ErrGamemasterInvalid on missing required fields.
func convertSpecies(index int, raw *speciesRaw) (Species, error) {
	if raw.SpeciesID == "" {
		return Species{}, fmt.Errorf("%w: pokemon[%d] missing speciesId", ErrGamemasterInvalid, index)
	}

	return Species{
		Dex:          raw.Dex,
		ID:           raw.SpeciesID,
		Name:         raw.SpeciesName,
		BaseStats:    BaseStats{Atk: raw.BaseStats.Atk, Def: raw.BaseStats.Def, HP: raw.BaseStats.HP},
		Types:        raw.Types,
		FastMoves:    raw.FastMoves,
		ChargedMoves: raw.ChargedMoves,
		Tags:         raw.Tags,
		Released:     raw.Released,
	}, nil
}

// convertMove promotes one raw move, inferring Category from the energy
// fields. A move with a non-zero Energy cost is a charged move; anything
// else (including an explicit EnergyGain) is classified as fast.
func convertMove(index int, raw *moveRaw) (Move, error) {
	if raw.MoveID == "" {
		return Move{}, fmt.Errorf("%w: moves[%d] missing moveId", ErrGamemasterInvalid, index)
	}

	category := MoveCategoryFast
	if raw.Energy > 0 {
		category = MoveCategoryCharged
	}

	return Move{
		ID:         raw.MoveID,
		Name:       raw.Name,
		Type:       raw.Type,
		Power:      raw.Power,
		Energy:     raw.Energy,
		EnergyGain: raw.EnergyGain,
		Cooldown:   raw.Cooldown,
		Turns:      raw.Turns,
		Category:   category,
	}, nil
}
