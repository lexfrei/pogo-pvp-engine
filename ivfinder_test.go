package pogopvp_test

import (
	"errors"
	"testing"

	pogopvp "github.com/lexfrei/pogo-pvp-engine"
)

func TestFindOptimalSpread_CPNeverExceedsCap(t *testing.T) {
	t.Parallel()

	bases := []pogopvp.BaseStats{
		{Atk: 152, Def: 143, HP: 216}, // quagsire
		{Atk: 121, Def: 152, HP: 155}, // medicham
		{Atk: 234, Def: 159, HP: 207}, // machamp
		{Atk: 112, Def: 152, HP: 225}, // azumarill
	}

	for _, base := range bases {
		for _, cpCap := range []int{500, 1500, 2500} {
			result, err := pogopvp.FindOptimalSpread(base, cpCap, pogopvp.FindSpreadOpts{XLAllowed: true})
			if err != nil {
				t.Fatalf("FindOptimalSpread(cap=%d) error: %v", cpCap, err)
			}
			if result.CP > cpCap {
				t.Errorf("CP=%d exceeds cap=%d for base %+v", result.CP, cpCap, base)
			}
			if result.StatProduct <= 0 {
				t.Errorf("StatProduct=%f is not positive", result.StatProduct)
			}
		}
	}
}

func TestFindOptimalSpread_Deterministic(t *testing.T) {
	t.Parallel()

	base := pogopvp.BaseStats{Atk: 152, Def: 143, HP: 216}
	cpCap := 1500

	first, err := pogopvp.FindOptimalSpread(base, cpCap, pogopvp.FindSpreadOpts{XLAllowed: true})
	if err != nil {
		t.Fatalf("first call: %v", err)
	}

	second, err := pogopvp.FindOptimalSpread(base, cpCap, pogopvp.FindSpreadOpts{XLAllowed: true})
	if err != nil {
		t.Fatalf("second call: %v", err)
	}

	if first != second {
		t.Errorf("non-deterministic: first=%+v second=%+v", first, second)
	}
}

func TestFindOptimalSpread_XLRaisesSP(t *testing.T) {
	t.Parallel()

	// A bulky low-attacker like Registeel caps out under 1500 at level 50+
	// when XL is allowed but level 40 with XL not allowed; SP must not
	// decrease when XL is permitted.
	base := pogopvp.BaseStats{Atk: 143, Def: 285, HP: 190}
	cpCap := 1500

	withoutXL, err := pogopvp.FindOptimalSpread(base, cpCap, pogopvp.FindSpreadOpts{XLAllowed: false})
	if err != nil {
		t.Fatalf("without XL: %v", err)
	}

	withXL, err := pogopvp.FindOptimalSpread(base, cpCap, pogopvp.FindSpreadOpts{XLAllowed: true})
	if err != nil {
		t.Fatalf("with XL: %v", err)
	}

	if withXL.StatProduct < withoutXL.StatProduct {
		t.Errorf("XL allowed SP=%f, not-allowed SP=%f; should be >=",
			withXL.StatProduct, withoutXL.StatProduct)
	}
}

func TestFindOptimalSpread_NoXLStaysUnder41(t *testing.T) {
	t.Parallel()

	base := pogopvp.BaseStats{Atk: 143, Def: 285, HP: 190}
	result, err := pogopvp.FindOptimalSpread(base, 1500, pogopvp.FindSpreadOpts{XLAllowed: false})
	if err != nil {
		t.Fatalf("FindOptimalSpread: %v", err)
	}

	if result.Level > pogopvp.NoXLMaxLevel {
		t.Errorf("Level=%f exceeds NoXLMaxLevel=%f without XL", result.Level, pogopvp.NoXLMaxLevel)
	}
}

// TestFindOptimalSpread_IsGlobalMax asserts that the returned (IV, level)
// pair is truly the stat-product maximum — no other IV triple anywhere
// on the 16^3 grid can beat it under the same cap and level cap. This is
// the finder's whole contract; sampling against external rankings would
// not work because those are matchup-weighted rather than pure SP-optimal.
func TestFindOptimalSpread_IsGlobalMax(t *testing.T) {
	t.Parallel()

	bases := []pogopvp.BaseStats{
		{Atk: 121, Def: 152, HP: 155}, // medicham
		{Atk: 152, Def: 143, HP: 216}, // quagsire
		{Atk: 234, Def: 159, HP: 207}, // machamp
	}

	for _, base := range bases {
		best, err := pogopvp.FindOptimalSpread(base, 1500, pogopvp.FindSpreadOpts{
			XLAllowed:   true,
			MaxLevelCap: 50.0,
		})
		if err != nil {
			t.Fatalf("FindOptimalSpread: %v", err)
		}

		for atk := 0; atk <= pogopvp.MaxIV; atk++ {
			for def := 0; def <= pogopvp.MaxIV; def++ {
				for sta := 0; sta <= pogopvp.MaxIV; sta++ {
					iv := pogopvp.MustNewIV(atk, def, sta)
					if sp := bestSPForIV(base, iv, 1500, 50.0); sp > best.StatProduct {
						t.Errorf("base=%+v: IV %d/%d/%d has SP=%f > best.SP=%f",
							base, atk, def, sta, sp, best.StatProduct)
					}
				}
			}
		}
	}
}

// bestSPForIV mirrors what the finder's inner loop does for one IV, used
// by TestFindOptimalSpread_IsGlobalMax as an independent cross-check.
func bestSPForIV(base pogopvp.BaseStats, iv pogopvp.IV, cpCap int, maxLevel float64) float64 {
	var best float64

	for doubled := int(maxLevel * 2); doubled >= 2; doubled-- {
		level := float64(doubled) / 2

		cpm, err := pogopvp.CPMAt(level)
		if err != nil {
			continue
		}

		if pogopvp.ComputeCP(base, iv, cpm) > cpCap {
			continue
		}

		sp := pogopvp.ComputeStatProduct(pogopvp.ComputeStats(base, iv, cpm))
		if sp > best {
			best = sp
		}
	}

	return best
}

func TestFindOptimalSpread_UnreachableCap(t *testing.T) {
	t.Parallel()

	// Tiny CP cap below the level-1 0/0/0 minimum is unreachable for most
	// species. The finder should report ErrCPCapUnreachable.
	base := pogopvp.BaseStats{Atk: 500, Def: 500, HP: 500}
	_, err := pogopvp.FindOptimalSpread(base, 5, pogopvp.FindSpreadOpts{XLAllowed: true})
	if !errors.Is(err, pogopvp.ErrCPCapUnreachable) {
		t.Errorf("error = %v, want wrapping ErrCPCapUnreachable", err)
	}
}
