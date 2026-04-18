package pogopvp

import (
	"errors"
	"fmt"
	"math"
)

// Form distinguishes the release mode of a specific Pokémon instance.
// Shadow and Purified are mutually exclusive alternatives to Regular; the XL
// axis is orthogonal and is carried on [Pokemon] directly rather than on the
// form.
type Form uint8

// Form constants. FormRegular is the zero value so plain Pokemon{} structs
// default to the common case.
const (
	FormRegular Form = iota
	FormShadow
	FormPurified
)

// Canonical lower-case labels used by [Form.String] and [ParseForm]. They
// match the strings produced by the pvpoke gamemaster and the MCP JSON
// payloads.
const (
	formLabelRegular  = "regular"
	formLabelShadow   = "shadow"
	formLabelPurified = "purified"
)

// ErrInvalidForm is returned when a form identifier fails to parse or when a
// Form value falls outside the known enum range. Matches with [errors.Is].
var ErrInvalidForm = errors.New("invalid form")

// ErrEmptySpeciesID is returned by [NewPokemon] when SpeciesID is blank.
var ErrEmptySpeciesID = errors.New("empty species id")

// ErrInvalidLevel is returned by [NewPokemon] when the level is outside
// [1.0, MaxLevel], is not a multiple of 0.5, or exceeds NoXLMaxLevel without
// XL candy.
var ErrInvalidLevel = errors.New("invalid level")

// String returns the canonical lower-case label for the form. Unknown values
// fall back to a diagnostic placeholder so String is safe on any uint8 input.
func (form Form) String() string {
	switch form {
	case FormRegular:
		return formLabelRegular
	case FormShadow:
		return formLabelShadow
	case FormPurified:
		return formLabelPurified
	default:
		return fmt.Sprintf("Form(%d)", uint8(form))
	}
}

// ParseForm decodes a canonical form label (case-sensitive lower-case) into
// the corresponding Form value. Unknown labels report ErrInvalidForm.
func ParseForm(label string) (Form, error) {
	switch label {
	case formLabelRegular:
		return FormRegular, nil
	case formLabelShadow:
		return FormShadow, nil
	case formLabelPurified:
		return FormPurified, nil
	default:
		return 0, fmt.Errorf("%w: %q", ErrInvalidForm, label)
	}
}

// isKnownForm reports whether the Form value is one of the three defined
// variants. Callers use it to reject bogus uint8 casts.
func isKnownForm(form Form) bool {
	switch form {
	case FormRegular, FormShadow, FormPurified:
		return true
	default:
		return false
	}
}

// MinLevel is the lowest legal Pokémon level. Level 1.0 is the hatch floor.
const MinLevel = 1.0

// MaxLevel is the highest legal Pokémon level; reachable only with XL candy.
const MaxLevel = 51.0

// NoXLMaxLevel is the pre-XL level cap. Levels above this require XL candy.
const NoXLMaxLevel = 40.0

// Pokemon identifies a single PvP combatant: the species the moves apply to,
// the IV triple, the level (in 0.5 increments), the release form, and
// whether XL candy has been used to push past NoXLMaxLevel.
type Pokemon struct {
	SpeciesID string
	Form      Form
	IV        IV
	Level     float64
	XL        bool
}

// NewPokemon returns a validated Pokemon. It enforces non-empty SpeciesID, a
// known Form value, and a level that is on a 0.5 grid inside [MinLevel,
// MaxLevel] (with XL required beyond NoXLMaxLevel).
func NewPokemon(speciesID string, form Form, iv IV, level float64, hasXL bool) (Pokemon, error) {
	if speciesID == "" {
		return Pokemon{}, ErrEmptySpeciesID
	}

	if !isKnownForm(form) {
		return Pokemon{}, fmt.Errorf("%w: %d", ErrInvalidForm, uint8(form))
	}

	err := validateLevel(level, hasXL)
	if err != nil {
		return Pokemon{}, err
	}

	return Pokemon{
		SpeciesID: speciesID,
		Form:      form,
		IV:        iv,
		Level:     level,
		XL:        hasXL,
	}, nil
}

// validateLevel checks that level lies on the half-integer grid inside the
// allowed range and that levels above NoXLMaxLevel are accompanied by XL.
func validateLevel(level float64, hasXL bool) error {
	if level < MinLevel || level > MaxLevel {
		return fmt.Errorf("%w: %.2f outside [%.1f, %.1f]", ErrInvalidLevel, level, MinLevel, MaxLevel)
	}

	doubled := level * 2
	if doubled != math.Trunc(doubled) {
		return fmt.Errorf("%w: %.2f is not on the 0.5 grid", ErrInvalidLevel, level)
	}

	if !hasXL && level > NoXLMaxLevel {
		return fmt.Errorf("%w: level %.1f requires XL candy", ErrInvalidLevel, level)
	}

	return nil
}
