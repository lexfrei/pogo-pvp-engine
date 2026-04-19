package pogopvp

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
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
// choices keyed into [Gamemaster.Moves]. LegacyMoves lists move ids that
// are legacy on THIS species specifically (community-day, event-exclusive,
// ETM-only). The same move id can be regular on one species and legacy
// on another — legacy is a per-species property, not per-move.
// Evolutions and PreEvolution map the pvpoke `family` block: Evolutions
// lists direct children (can branch, e.g. eevee), PreEvolution names the
// immediate parent ("" for base forms). Chain traversal is the caller's
// responsibility.
type Species struct {
	Dex          int
	ID           string
	Name         string
	BaseStats    BaseStats
	Types        []string
	FastMoves    []string
	ChargedMoves []string
	LegacyMoves  []string
	Evolutions   []string
	PreEvolution string
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
// duplicate IDs, wrong document id) in an otherwise syntactically valid
// payload.
var ErrGamemasterInvalid = errors.New("gamemaster invalid")

// gamemasterDocumentID is the expected value of the top-level "id" field in
// a pvpoke gamemaster.json. It guards against accidentally feeding an
// unrelated JSON document (rankings, format lists, cup configs) through the
// parser.
const gamemasterDocumentID = "gamemaster"

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
	LegacyMoves  []string     `json:"legacyMoves"`
	Family       *familyRaw   `json:"family"`
	Tags         []string     `json:"tags"`
	Released     bool         `json:"released"`
}

// familyRaw mirrors the pvpoke `family` block on each species:
// Parent is the direct pre-evolution id (absent for base forms),
// Evolutions is the direct-children list (can branch — eevee has
// 8+ entries). ID (e.g. "FAMILY_BULBASAUR") is parsed but not
// surfaced on Species — the string id is not useful for any lookup
// and adds noise to the public shape.
type familyRaw struct {
	ID         string   `json:"id"`
	Parent     string   `json:"parent"`
	Evolutions []string `json:"evolutions"`
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

	if raw.ID != gamemasterDocumentID {
		return nil, fmt.Errorf("%w: expected document id %q, got %q",
			ErrGamemasterInvalid, gamemasterDocumentID, raw.ID)
	}

	gamemaster := &Gamemaster{
		Version: raw.Timestamp,
		Pokemon: make(map[string]Species, len(raw.Pokemon)),
		Moves:   make(map[string]Move, len(raw.Moves)),
	}

	err = indexSpecies(gamemaster, raw.Pokemon)
	if err != nil {
		return nil, err
	}

	err = indexMoves(gamemaster, raw.Moves)
	if err != nil {
		return nil, err
	}

	return gamemaster, nil
}

// indexSpecies promotes every raw Pokémon entry and populates the
// gamemaster map, rejecting duplicates.
func indexSpecies(gamemaster *Gamemaster, entries []speciesRaw) error {
	for index := range entries {
		species, err := convertSpecies(index, &entries[index])
		if err != nil {
			return err
		}

		_, exists := gamemaster.Pokemon[species.ID]
		if exists {
			return fmt.Errorf("%w: duplicate speciesId %q", ErrGamemasterInvalid, species.ID)
		}

		gamemaster.Pokemon[species.ID] = species
	}

	return nil
}

// indexMoves promotes every raw move entry and populates the gamemaster
// map. Moves that convertMove marks as skipped (e.g. TRANSFORM) are
// silently dropped.
func indexMoves(gamemaster *Gamemaster, entries []moveRaw) error {
	for index := range entries {
		move, keep, err := convertMove(index, &entries[index])
		if err != nil {
			return err
		}

		if !keep {
			continue
		}

		_, exists := gamemaster.Moves[move.ID]
		if exists {
			return fmt.Errorf("%w: duplicate moveId %q", ErrGamemasterInvalid, move.ID)
		}

		gamemaster.Moves[move.ID] = move
	}

	return nil
}

// convertSpecies promotes one raw entry into a Species, reporting
// ErrGamemasterInvalid on missing required fields. Base stats must be
// strictly positive and the dex number must be at least 1 — the pvpoke
// gamemaster never violates this, so anything weaker is a corrupted row
// and we fail loudly rather than produce CP-10 ghost creatures downstream.
// The pvpoke "none" placeholder in the types slice is normalised away
// here so Species.Types contains only real type identifiers.
func convertSpecies(index int, raw *speciesRaw) (Species, error) {
	if raw.SpeciesID == "" {
		return Species{}, fmt.Errorf("%w: pokemon[%d] missing speciesId", ErrGamemasterInvalid, index)
	}

	if raw.Dex < 1 {
		return Species{}, fmt.Errorf("%w: pokemon[%d] (%s) dex=%d < 1",
			ErrGamemasterInvalid, index, raw.SpeciesID, raw.Dex)
	}

	if raw.BaseStats.Atk <= 0 || raw.BaseStats.Def <= 0 || raw.BaseStats.HP <= 0 {
		return Species{}, fmt.Errorf(
			"%w: pokemon[%d] (%s) non-positive baseStats atk=%d def=%d hp=%d",
			ErrGamemasterInvalid, index, raw.SpeciesID,
			raw.BaseStats.Atk, raw.BaseStats.Def, raw.BaseStats.HP,
		)
	}

	types := normaliseTypes(raw.Types)
	if len(types) == 0 {
		return Species{}, fmt.Errorf(
			"%w: pokemon[%d] (%s) has no real types after normalisation (raw=%v)",
			ErrGamemasterInvalid, index, raw.SpeciesID, raw.Types)
	}

	var (
		preEvolution string
		evolutions   []string
	)

	if raw.Family != nil {
		preEvolution = raw.Family.Parent
		evolutions = raw.Family.Evolutions
	}

	return Species{
		Dex:          raw.Dex,
		ID:           raw.SpeciesID,
		Name:         raw.SpeciesName,
		BaseStats:    BaseStats{Atk: raw.BaseStats.Atk, Def: raw.BaseStats.Def, HP: raw.BaseStats.HP},
		Types:        types,
		FastMoves:    raw.FastMoves,
		ChargedMoves: raw.ChargedMoves,
		LegacyMoves:  raw.LegacyMoves,
		Evolutions:   evolutions,
		PreEvolution: preEvolution,
		Tags:         raw.Tags,
		Released:     raw.Released,
	}, nil
}

// IsLegacyMove reports whether moveID is a legacy move for this species.
// Legacy in pvpoke semantics = community-day exclusive, event-exclusive,
// or ETM-only (not accessible via regular TM at the time of rollout).
// The same move id can be regular on one species and legacy on another —
// the lookup is scoped to the species passed in. A nil species returns
// false defensively.
func IsLegacyMove(species *Species, moveID string) bool {
	if species == nil {
		return false
	}

	return slices.Contains(species.LegacyMoves, moveID)
}

// normaliseTypes drops the pvpoke placeholder "none" and any empty strings,
// returning a slice that only holds real type identifiers.
func normaliseTypes(raw []string) []string {
	result := make([]string, 0, len(raw))

	for _, t := range raw {
		if t == "" || t == "none" {
			continue
		}

		result = append(result, t)
	}

	return result
}

// convertMove promotes one raw move, inferring Category from the energy
// fields. A move with a non-zero Energy cost is a charged move; a move
// with a non-zero EnergyGain is fast. An entry with both columns
// positive is rejected as malformed. A move with both columns zero is
// skipped — pvpoke's signature move TRANSFORM (Ditto) carries no energy
// numbers because Ditto copies its target instead of attacking. The
// third return value is false when the move must be silently dropped
// from the gamemaster; callers should `continue` rather than insert it.
func convertMove(index int, raw *moveRaw) (Move, bool, error) {
	if raw.MoveID == "" {
		return Move{}, false, fmt.Errorf("%w: moves[%d] missing moveId", ErrGamemasterInvalid, index)
	}

	if raw.Energy == 0 && raw.EnergyGain == 0 {
		return Move{}, false, nil
	}

	if raw.Energy > 0 && raw.EnergyGain > 0 {
		return Move{}, false, fmt.Errorf(
			"%w: moves[%d] (%s) sets both energy=%d and energyGain=%d",
			ErrGamemasterInvalid, index, raw.MoveID, raw.Energy, raw.EnergyGain,
		)
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
	}, true, nil
}
