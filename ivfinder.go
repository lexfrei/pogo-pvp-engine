package pogopvp

import (
	"errors"
	"fmt"
	"math"
)

// ErrCPCapUnreachable is returned by [FindOptimalSpread] when the CP cap
// is lower than the minimum-IV, minimum-level CP of the species — no
// legal spread can fit.
var ErrCPCapUnreachable = errors.New("cp cap unreachable")

// ErrInvalidSpreadOpts is returned by [FindOptimalSpread] when the
// requested search envelope is self-contradictory: negative caps, min
// above max, bounds outside [MinLevel, MaxLevel], off-grid levels, or
// XL-disallowed plus a max level that requires XL candy.
var ErrInvalidSpreadOpts = errors.New("invalid spread opts")

// FindSpreadOpts tunes the IV search envelope. XLAllowed raises the
// upper level bound from NoXLMaxLevel to MaxLevel. Leaving both level
// overrides at zero uses the default bounds.
type FindSpreadOpts struct {
	XLAllowed   bool
	MinLevelCap float64
	MaxLevelCap float64
}

// OptimalSpread reports the best IV / level combination for a given base
// stat line under a CP cap: the IV triple, the level reached on the 0.5
// grid, the resulting CP, and the stat product used to rank it.
type OptimalSpread struct {
	IV          IV
	Level       float64
	CP          int
	StatProduct float64
}

// levelBounds is the resolved [minLevel, maxLevel] envelope used by the
// search; returning a struct keeps the function signature unnamed.
type levelBounds struct {
	Min float64
	Max float64
}

// FindOptimalSpread enumerates every IV triple in [0, MaxIV] and, for
// each, picks the highest legal level under the cap. The winner is the
// (IV, level) pair with the largest stat product; on ties, the first
// encountered wins (iteration order is lexicographic over atk, def, sta).
//
// The function runs ~4096 * 51 CP evaluations — ≪1 ms in practice — so
// it is fine to call on the hot path. Returns [ErrCPCapUnreachable] if
// the cap cannot be hit at all, and [ErrInvalidSpreadOpts] if the
// requested search envelope is self-contradictory.
func FindOptimalSpread(base BaseStats, cpCap int, opts FindSpreadOpts) (OptimalSpread, error) {
	if cpCap <= 0 {
		return OptimalSpread{}, fmt.Errorf("%w: cpCap %d must be positive", ErrInvalidSpreadOpts, cpCap)
	}

	bounds, err := resolveLevelBounds(&opts)
	if err != nil {
		return OptimalSpread{}, err
	}

	best, found := searchAllIVs(base, cpCap, bounds)
	if !found {
		return OptimalSpread{}, ErrCPCapUnreachable
	}

	return best, nil
}

// searchAllIVs iterates the 16^3 IV grid, keeping the (IV, level) pair
// with the largest stat product under the CP cap.
func searchAllIVs(base BaseStats, cpCap int, bounds levelBounds) (OptimalSpread, bool) {
	var best OptimalSpread

	found := false

	for atk := 0; atk <= MaxIV; atk++ {
		for def := 0; def <= MaxIV; def++ {
			for sta := 0; sta <= MaxIV; sta++ {
				iv := IV{Atk: uint8(atk), Def: uint8(def), Sta: uint8(sta)}

				candidate, ok := bestLevelForIV(base, iv, cpCap, bounds)
				if !ok {
					continue
				}

				if !found || candidate.StatProduct > best.StatProduct {
					best = candidate
					found = true
				}
			}
		}
	}

	return best, found
}

// resolveLevelBounds applies defaults when the caller left the level
// envelope zeroed and validates the envelope against the engine-wide
// level rules: NaN rejected, bounds inside [MinLevel, MaxLevel], on the
// 0.5 grid, min <= max, and max above NoXLMaxLevel only when XL is
// allowed. Returns [ErrInvalidSpreadOpts] on any violation.
func resolveLevelBounds(opts *FindSpreadOpts) (levelBounds, error) {
	minBound := opts.MinLevelCap
	if minBound == 0 {
		minBound = MinLevel
	}

	maxBound := opts.MaxLevelCap
	if maxBound == 0 {
		if opts.XLAllowed {
			maxBound = MaxLevel
		} else {
			maxBound = NoXLMaxLevel
		}
	}

	err := validateLevelBound("MinLevelCap", minBound)
	if err != nil {
		return levelBounds{}, err
	}

	err = validateLevelBound("MaxLevelCap", maxBound)
	if err != nil {
		return levelBounds{}, err
	}

	if minBound > maxBound {
		return levelBounds{}, fmt.Errorf("%w: MinLevelCap %.1f exceeds MaxLevelCap %.1f",
			ErrInvalidSpreadOpts, minBound, maxBound)
	}

	if !opts.XLAllowed && maxBound > NoXLMaxLevel {
		return levelBounds{}, fmt.Errorf(
			"%w: MaxLevelCap %.1f requires XL candy but XLAllowed=false",
			ErrInvalidSpreadOpts, maxBound)
	}

	return levelBounds{Min: minBound, Max: maxBound}, nil
}

// validateLevelBound rejects NaN, off-grid, and out-of-range envelope
// values with a descriptive error.
func validateLevelBound(name string, value float64) error {
	if math.IsNaN(value) {
		return fmt.Errorf("%w: %s is NaN", ErrInvalidSpreadOpts, name)
	}

	if value < MinLevel || value > MaxLevel {
		return fmt.Errorf("%w: %s %.2f outside [%.1f, %.1f]",
			ErrInvalidSpreadOpts, name, value, MinLevel, MaxLevel)
	}

	if doubled := value * 2; doubled != math.Trunc(doubled) {
		return fmt.Errorf("%w: %s %.2f not on the 0.5 grid",
			ErrInvalidSpreadOpts, name, value)
	}

	return nil
}

// bestLevelForIV walks the half-level grid downward from maxLevel and
// returns the first level whose CP fits inside [cpCap]. The stat product
// for that (IV, level) is included so the caller can rank directly.
func bestLevelForIV(base BaseStats, iv IV, cpCap int, bounds levelBounds) (OptimalSpread, bool) {
	for doubled := int(bounds.Max * 2); doubled >= int(bounds.Min*2); doubled-- {
		level := float64(doubled) / 2

		cpm, err := CPMAt(level)
		if err != nil {
			continue
		}

		combatPower := ComputeCP(base, iv, cpm)
		if combatPower > cpCap {
			continue
		}

		stats := ComputeStats(base, iv, cpm)

		return OptimalSpread{
			IV:          iv,
			Level:       level,
			CP:          combatPower,
			StatProduct: ComputeStatProduct(stats),
		}, true
	}

	return OptimalSpread{}, false
}
