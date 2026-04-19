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

func TestFindOptimalSpread_InvalidOpts(t *testing.T) {
	t.Parallel()

	base := pogopvp.BaseStats{Atk: 150, Def: 150, HP: 150}

	cases := []struct {
		name string
		cap  int
		opts pogopvp.FindSpreadOpts
	}{
		{"negative cap", -1, pogopvp.FindSpreadOpts{XLAllowed: true}},
		{"zero cap", 0, pogopvp.FindSpreadOpts{XLAllowed: true}},
		{"min above max", 1500, pogopvp.FindSpreadOpts{
			XLAllowed: true, MinLevelCap: 40.0, MaxLevelCap: 20.0,
		}},
		{"max above engine max", 1500, pogopvp.FindSpreadOpts{
			XLAllowed: true, MaxLevelCap: 60.0,
		}},
		{"min below engine min", 1500, pogopvp.FindSpreadOpts{
			XLAllowed: true, MinLevelCap: 0.5,
		}},
		{"off-grid max", 1500, pogopvp.FindSpreadOpts{
			XLAllowed: true, MaxLevelCap: 40.25,
		}},
		{"xl required but disallowed", 1500, pogopvp.FindSpreadOpts{
			XLAllowed: false, MaxLevelCap: 50.0,
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := pogopvp.FindOptimalSpread(base, tc.cap, tc.opts)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !errors.Is(err, pogopvp.ErrInvalidSpreadOpts) {
				t.Errorf("error = %v, want wrapping ErrInvalidSpreadOpts", err)
			}
		})
	}
}

// TestFindOptimalSpread_DefaultBoundsApply verifies that leaving
// MinLevelCap / MaxLevelCap at zero falls back to [MinLevel,
// NoXLMaxLevel] when XLAllowed=false and [MinLevel, MaxLevel] when it
// is true. This pins the zero-as-sentinel contract documented on
// FindSpreadOpts.
func TestFindOptimalSpread_DefaultBoundsApply(t *testing.T) {
	t.Parallel()

	base := pogopvp.BaseStats{Atk: 143, Def: 285, HP: 190}

	noXL, err := pogopvp.FindOptimalSpread(base, 2500, pogopvp.FindSpreadOpts{XLAllowed: false})
	if err != nil {
		t.Fatalf("no-XL default: %v", err)
	}
	if noXL.Level > pogopvp.NoXLMaxLevel {
		t.Errorf("no-XL default level=%f > NoXLMaxLevel=%f", noXL.Level, pogopvp.NoXLMaxLevel)
	}

	withXL, err := pogopvp.FindOptimalSpread(base, 2500, pogopvp.FindSpreadOpts{XLAllowed: true})
	if err != nil {
		t.Fatalf("with-XL default: %v", err)
	}
	if withXL.Level > pogopvp.MaxLevel {
		t.Errorf("with-XL default level=%f > MaxLevel=%f", withXL.Level, pogopvp.MaxLevel)
	}
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

// TestLevelForCP_FitsUnderCap pins the core contract: the returned
// level's CP must be ≤ targetCP. Walks a realistic meta species
// (medicham) at 0/15/15 under the three standard league caps and
// confirms the result is under each cap.
func TestLevelForCP_FitsUnderCap(t *testing.T) {
	t.Parallel()

	base := pogopvp.BaseStats{Atk: 121, Def: 152, HP: 155} // medicham

	ivs, err := pogopvp.NewIV(0, 15, 15)
	if err != nil {
		t.Fatalf("NewIV: %v", err)
	}

	for _, cap := range []int{500, 1500, 2500} {
		result, err := pogopvp.LevelForCP(base, ivs, cap,
			pogopvp.FindSpreadOpts{XLAllowed: true})
		if err != nil {
			t.Fatalf("cap=%d: LevelForCP: %v", cap, err)
		}

		if result.CP > cap {
			t.Errorf("cap=%d: result.CP=%d exceeds cap", cap, result.CP)
		}
		if result.Level < pogopvp.MinLevel || result.Level > pogopvp.MaxLevel {
			t.Errorf("cap=%d: Level=%.1f outside [%.1f, %.1f]",
				cap, result.Level, pogopvp.MinLevel, pogopvp.MaxLevel)
		}
	}
}

// TestLevelForCP_ExactHit pins that Exact=true when the caller's
// target matches some level's actual CP exactly. Take a known
// (base, iv, level) triple, compute its CP via ComputeCP, pass that
// CP back into LevelForCP, and assert the round-trip recovers the
// same level with Exact=true. Guards against off-by-one in the
// Exact flag computation.
func TestLevelForCP_ExactHit(t *testing.T) {
	t.Parallel()

	base := pogopvp.BaseStats{Atk: 121, Def: 152, HP: 155} // medicham

	ivs, err := pogopvp.NewIV(0, 15, 15)
	if err != nil {
		t.Fatalf("NewIV: %v", err)
	}

	const level = 40.0

	cpm, err := pogopvp.CPMAt(level)
	if err != nil {
		t.Fatalf("CPMAt: %v", err)
	}

	targetCP := pogopvp.ComputeCP(base, ivs, cpm)

	result, err := pogopvp.LevelForCP(base, ivs, targetCP,
		pogopvp.FindSpreadOpts{XLAllowed: true})
	if err != nil {
		t.Fatalf("LevelForCP: %v", err)
	}

	if !result.Exact {
		t.Errorf("Exact=false, want true (round-trip should hit exactly)")
	}
	if result.CP != targetCP {
		t.Errorf("CP=%d, want %d", result.CP, targetCP)
	}
	if result.Level < level {
		t.Errorf("Level=%.1f, want ≥ %.1f (max level with CP=%d)",
			result.Level, level, targetCP)
	}
}

// TestLevelForCP_TooLow confirms the ErrCPTooLow branch fires when
// the target is below the CP floor — pathological but the guard
// must be in place.
func TestLevelForCP_TooLow(t *testing.T) {
	t.Parallel()

	base := pogopvp.BaseStats{Atk: 121, Def: 152, HP: 155}

	ivs, err := pogopvp.NewIV(15, 15, 15)
	if err != nil {
		t.Fatalf("NewIV: %v", err)
	}

	_, err = pogopvp.LevelForCP(base, ivs, 5,
		pogopvp.FindSpreadOpts{XLAllowed: true})
	if !errors.Is(err, pogopvp.ErrCPTooLow) {
		t.Errorf("error = %v, want wrapping ErrCPTooLow", err)
	}
}

// TestLevelForCP_InvalidTargetCP rejects non-positive targets as
// ErrInvalidSpreadOpts (reusing the existing sentinel since the
// validation shape matches).
func TestLevelForCP_InvalidTargetCP(t *testing.T) {
	t.Parallel()

	base := pogopvp.BaseStats{Atk: 121, Def: 152, HP: 155}

	ivs, err := pogopvp.NewIV(15, 15, 15)
	if err != nil {
		t.Fatalf("NewIV: %v", err)
	}

	_, err = pogopvp.LevelForCP(base, ivs, 0, pogopvp.FindSpreadOpts{XLAllowed: true})
	if !errors.Is(err, pogopvp.ErrInvalidSpreadOpts) {
		t.Errorf("error = %v, want wrapping ErrInvalidSpreadOpts", err)
	}
}

// TestLevelForCP_InvalidIV pins the IV-validation guard. An out-of
// range IV component (here Atk=200, which overflows past MaxIV=15
// while still fitting in uint8) must surface ErrInvalidSpreadOpts
// before ComputeCP silently produces garbage.
func TestLevelForCP_InvalidIV(t *testing.T) {
	t.Parallel()

	base := pogopvp.BaseStats{Atk: 121, Def: 152, HP: 155}

	// Bypasses NewIV so we can construct an invalid IV that uint8
	// happily holds but pogopvp.IV.Valid() rejects.
	ivs := pogopvp.IV{Atk: 200, Def: 0, Sta: 0}

	_, err := pogopvp.LevelForCP(base, ivs, 1500,
		pogopvp.FindSpreadOpts{XLAllowed: true})
	if !errors.Is(err, pogopvp.ErrInvalidSpreadOpts) {
		t.Errorf("error = %v, want wrapping ErrInvalidSpreadOpts", err)
	}
}

// TestLevelForCP_TooLowWithMinLevelCap exercises the ErrCPTooLow
// branch where the floor is not the default level 1 but a
// caller-supplied MinLevelCap. A high MinLevelCap against a small
// targetCP must surface the error — otherwise the godoc on
// ErrCPTooLow that promises "min-level-bound CP" semantics would
// drift from reality.
func TestLevelForCP_TooLowWithMinLevelCap(t *testing.T) {
	t.Parallel()

	base := pogopvp.BaseStats{Atk: 121, Def: 152, HP: 155}

	ivs, err := pogopvp.NewIV(15, 15, 15)
	if err != nil {
		t.Fatalf("NewIV: %v", err)
	}

	// medicham 15/15/15 at level 40 easily clears CP 100; the
	// MinLevelCap=40 floor therefore blocks any level with CP ≤ 100.
	_, err = pogopvp.LevelForCP(base, ivs, 100, pogopvp.FindSpreadOpts{
		XLAllowed:   true,
		MinLevelCap: 40,
	})
	if !errors.Is(err, pogopvp.ErrCPTooLow) {
		t.Errorf("error = %v, want wrapping ErrCPTooLow", err)
	}
}
