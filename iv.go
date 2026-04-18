// Package pogopvp provides Pokémon GO PvP battle simulation and ranking
// primitives. The package is factored so the core types (IV, Pokemon, Move)
// have zero external dependencies and can be embedded in servers, bots, or
// CLIs without pulling in transport or I/O machinery.
package pogopvp

import (
	"errors"
	"fmt"
)

// MaxIV is the inclusive upper bound for each component of an IV triple.
// Pokémon GO stores appraisal IVs on the closed interval [0, 15].
const MaxIV = 15

// ErrIVOutOfRange is returned by [NewIV] when any component of the requested
// IV triple falls outside [0, MaxIV]. Errors returned by [NewIV] wrap this
// sentinel so callers can match on it with [errors.Is].
var ErrIVOutOfRange = errors.New("iv out of range")

// IV is an individual-values triple for a single Pokémon. Each field is
// bounded to [0, MaxIV]; construct instances through [NewIV] or [MustNewIV]
// to enforce that invariant. Callers that build IV literals directly
// should run [IV.Valid] before handing the value to other engine APIs.
type IV struct {
	Atk uint8
	Def uint8
	Sta uint8
}

// NewIV returns a validated IV triple. It reports ErrIVOutOfRange when any
// component is outside [0, MaxIV].
func NewIV(atk, def, sta int) (IV, error) {
	atkU, err := toIVComponent("atk", atk)
	if err != nil {
		return IV{}, err
	}

	defU, err := toIVComponent("def", def)
	if err != nil {
		return IV{}, err
	}

	staU, err := toIVComponent("sta", sta)
	if err != nil {
		return IV{}, err
	}

	return IV{Atk: atkU, Def: defU, Sta: staU}, nil
}

// MustNewIV is the panicking companion to [NewIV], intended for test fixtures
// and compile-time constants where invalid input is a programmer error.
func MustNewIV(atk, def, sta int) IV {
	iv, err := NewIV(atk, def, sta)
	if err != nil {
		panic(err)
	}

	return iv
}

// Valid reports whether every component is inside [0, MaxIV]. Because the
// fields are uint8 the lower bound is automatic; the method guards against
// literal IV{Atk: 200} constructions that bypass [NewIV].
func (iv IV) Valid() bool {
	return iv.Atk <= MaxIV && iv.Def <= MaxIV && iv.Sta <= MaxIV
}

// toIVComponent validates that value is in [0, MaxIV] and narrows it to
// uint8. The narrowing is safe because the range check precedes the cast.
func toIVComponent(name string, value int) (uint8, error) {
	if value < 0 || value > MaxIV {
		return 0, fmt.Errorf("%w: %s=%d (want 0..%d)", ErrIVOutOfRange, name, value, MaxIV)
	}

	return uint8(value), nil
}
