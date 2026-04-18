package pogopvp

import "errors"

// ErrCPCapUnreachable is returned by [FindOptimalSpread] when the CP cap
// is lower than the minimum-IV, minimum-level CP of the species — no
// legal spread can fit.
var ErrCPCapUnreachable = errors.New("cp cap unreachable")

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
// the cap cannot be hit at all.
func FindOptimalSpread(base BaseStats, cpCap int, opts FindSpreadOpts) (OptimalSpread, error) {
	bounds := resolveLevelBounds(&opts)

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
// envelope zeroed. XLAllowed toggles between the NoXLMaxLevel cap and the
// full MaxLevel cap.
func resolveLevelBounds(opts *FindSpreadOpts) levelBounds {
	minBound := opts.MinLevelCap
	if minBound <= 0 {
		minBound = MinLevel
	}

	maxBound := opts.MaxLevelCap
	if maxBound <= 0 {
		if opts.XLAllowed {
			maxBound = MaxLevel
		} else {
			maxBound = NoXLMaxLevel
		}
	}

	return levelBounds{Min: minBound, Max: maxBound}
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
