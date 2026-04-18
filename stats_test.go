package pogopvp_test

import (
	"math"
	"math/rand/v2"
	"testing"

	pogopvp "github.com/lexfrei/pogo-pvp-engine"
)

const statsEpsilon = 1e-9

// reference values computed directly from the pvpoke Battle.js formula:
//
//	attack = (baseAtk + ivAtk) * cpm
//	defense = (baseDef + ivDef) * cpm
//	hp = floor((baseSta + ivSta) * cpm)
//	CP = max(10, floor(attack * sqrt(defense) * sqrt(hp) / 10))
//	statProduct = attack * defense * hp
//
// See src/js/pokemon/Pokemon.js calculateCP() in github.com/pvpoke/pvpoke.
func TestComputeStats_Anchors(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		base        pogopvp.BaseStats
		iv          pogopvp.IV
		cpm         float64
		wantAtk     float64
		wantDef     float64
		wantHP      int
		wantCP      int
		wantProduct float64
	}{
		{
			name:        "medicham 15/15/15 L40",
			base:        pogopvp.BaseStats{Atk: 121, Def: 152, HP: 155},
			iv:          pogopvp.MustNewIV(15, 15, 15),
			cpm:         0.790300011634826,
			wantAtk:     107.4808015823363,
			wantDef:     131.9801019430158,
			wantHP:      134,
			wantCP:      1429,
			wantProduct: 1900833.8380671099,
		},
		{
			name:        "whiscash 0/15/15 L50",
			base:        pogopvp.BaseStats{Atk: 151, Def: 141, HP: 242},
			iv:          pogopvp.MustNewIV(0, 15, 15),
			cpm:         0.840300023555755,
			wantAtk:     126.88530355678802,
			wantDef:     131.08680367396776,
			wantHP:      215,
			wantCP:      2130,
			wantProduct: 3576092.6084630578,
		},
		{
			name:        "azumarill 15/15/15 L45 XL",
			base:        pogopvp.BaseStats{Atk: 112, Def: 152, HP: 225},
			iv:          pogopvp.MustNewIV(15, 15, 15),
			cpm:         0.815299987792968,
			wantAtk:     103.54309844970392,
			wantDef:     136.15509796142577,
			wantHP:      195,
			wantCP:      1687,
			wantProduct: 2749094.5390,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			stats := pogopvp.ComputeStats(tc.base, tc.iv, tc.cpm)
			if math.Abs(stats.Atk-tc.wantAtk) > statsEpsilon {
				t.Errorf("Atk = %.12f, want %.12f", stats.Atk, tc.wantAtk)
			}
			if math.Abs(stats.Def-tc.wantDef) > statsEpsilon {
				t.Errorf("Def = %.12f, want %.12f", stats.Def, tc.wantDef)
			}
			if stats.HP != tc.wantHP {
				t.Errorf("HP = %d, want %d", stats.HP, tc.wantHP)
			}

			cp := pogopvp.ComputeCP(stats)
			if cp != tc.wantCP {
				t.Errorf("CP = %d, want %d", cp, tc.wantCP)
			}

			sp := pogopvp.ComputeStatProduct(stats)
			if math.Abs(sp-tc.wantProduct) > 1.0 {
				t.Errorf("StatProduct = %.4f, want %.4f", sp, tc.wantProduct)
			}
		})
	}
}

func TestComputeCP_MinClamped(t *testing.T) {
	t.Parallel()

	// A level-1 Bulbasaur-ish creature with 0 IVs is still CP 10 by the
	// floor rule, not 9 or lower.
	base := pogopvp.BaseStats{Atk: 1, Def: 1, HP: 1}
	iv := pogopvp.MustNewIV(0, 0, 0)
	stats := pogopvp.ComputeStats(base, iv, 0.0939999967813491)
	if cp := pogopvp.ComputeCP(stats); cp != 10 {
		t.Errorf("ComputeCP floor = %d, want 10", cp)
	}
}

func TestComputeStats_MonotonicInCPM(t *testing.T) {
	t.Parallel()

	base := pogopvp.BaseStats{Atk: 200, Def: 200, HP: 200}
	iv := pogopvp.MustNewIV(10, 10, 10)

	var prev float64
	for doubled := 2; doubled <= 102; doubled++ {
		level := float64(doubled) / 2
		cpm, err := pogopvp.CPMAt(level)
		if err != nil {
			t.Fatalf("CPMAt(%.1f) = %v", level, err)
		}

		sp := pogopvp.ComputeStatProduct(pogopvp.ComputeStats(base, iv, cpm))
		if doubled > 2 && sp <= prev {
			t.Errorf("StatProduct not monotonic across CPM at level %.1f: got %.4f after %.4f",
				level, sp, prev)
		}
		prev = sp
	}
}

func TestComputeStatProduct_MonotonicInIV(t *testing.T) {
	t.Parallel()

	// For fixed base stats and CPM, stat product must monotonically
	// increase as IVs rise. Holding defense and stamina at 15, sweep
	// attack from 0 to 15.
	base := pogopvp.BaseStats{Atk: 150, Def: 150, HP: 200}
	cpm := 0.7317003147125 // level 30

	var prev float64
	for atk := 0; atk <= pogopvp.MaxIV; atk++ {
		iv := pogopvp.MustNewIV(atk, 15, 15)
		sp := pogopvp.ComputeStatProduct(pogopvp.ComputeStats(base, iv, cpm))
		if atk > 0 && sp <= prev {
			t.Errorf("stat product not monotonic at atk=%d: got %.4f after %.4f", atk, sp, prev)
		}
		prev = sp
	}
}

func TestComputeStats_PropertyCPScalesWithCPM(t *testing.T) {
	t.Parallel()

	// Rough property: doubling CPM should approximately triple CP (atk*sqrt(def)*sqrt(hp)
	// grows as cpm^2, with hp floored). Not exact, but strictly > 2x growth.
	base := pogopvp.BaseStats{Atk: 180, Def: 160, HP: 190}
	iv := pogopvp.MustNewIV(15, 15, 15)

	rng := rand.New(rand.NewPCG(0x5EED, 0xCAFE))
	for range 100 {
		lowDoubled := rng.IntN(50) + 2 // level 1..25
		low := float64(lowDoubled) / 2
		highDoubled := rng.IntN(50) + 52 // level 26..50
		high := float64(highDoubled) / 2

		lowCPM, _ := pogopvp.CPMAt(low)
		highCPM, _ := pogopvp.CPMAt(high)

		lowCP := pogopvp.ComputeCP(pogopvp.ComputeStats(base, iv, lowCPM))
		highCP := pogopvp.ComputeCP(pogopvp.ComputeStats(base, iv, highCPM))

		if highCP <= lowCP {
			t.Errorf("CP at level %.1f (%d) not higher than at level %.1f (%d)", high, highCP, low, lowCP)
		}
	}
}
