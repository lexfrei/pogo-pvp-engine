package pogopvp_test

import (
	"math"
	"testing"

	pogopvp "github.com/lexfrei/pogo-pvp-engine"
)

// Values carry the 1.60000002384185791015625 tail from the JS source, so a
// product of two multipliers drifts around 7e-8 from the clean decimal.
// The epsilon below is tight enough to catch a wrong-bucket lookup while
// tolerating the expected float32 noise.
const effectivenessEpsilon = 1e-6

// Reference values are pulled directly from pvpoke's DamageCalculator
// (src/js/battle/DamageCalculator.js): weaknesses give SUPER_EFFECTIVE
// (1.6), resistances give RESISTED (0.625), immunities give DOUBLE_RESISTED
// (0.390625). Against dual-type defenders the multipliers compound.
func TestTypeEffectiveness_Single(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		moveType   string
		defTypes   []string
		wantResult float64
	}{
		{"fire vs grass (super-effective)", "fire", []string{"grass"}, 1.6},
		{"grass vs fire (resisted)", "grass", []string{"fire"}, 0.625},
		{"normal vs ghost (immune)", "normal", []string{"ghost"}, 0.390625},
		{"ghost vs normal (immune)", "ghost", []string{"normal"}, 0.390625},
		{"ground vs electric (immune one-way)", "ground", []string{"electric"}, 1.6},
		{"electric vs ground (immune)", "electric", []string{"ground"}, 0.390625},
		{"water vs water (resisted)", "water", []string{"water"}, 0.625},
		{"normal vs normal (neutral)", "normal", []string{"normal"}, 1.0},
		{"dragon vs fairy (immune)", "dragon", []string{"fairy"}, 0.390625},
		{"poison vs steel (immune)", "poison", []string{"steel"}, 0.390625},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := pogopvp.TypeEffectiveness(tc.moveType, tc.defTypes)
			if math.Abs(got-tc.wantResult) > effectivenessEpsilon {
				t.Errorf("TypeEffectiveness(%q, %v) = %.6f, want %.6f",
					tc.moveType, tc.defTypes, got, tc.wantResult)
			}
		})
	}
}

func TestTypeEffectiveness_Dual(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		moveType   string
		defTypes   []string
		wantResult float64
	}{
		{"grass vs water/ground (double super-effective)", "grass", []string{"water", "ground"}, 2.56},
		{"ice vs flying/dragon (double super-effective)", "ice", []string{"flying", "dragon"}, 2.56},
		{"fire vs water/rock (both resisted)", "fire", []string{"water", "rock"}, 0.390625},
		{"fire vs water/grass (super then resist = neutral)", "fire", []string{"water", "grass"}, 1.0},
		{"bug vs fire/flying (resist then resist)", "bug", []string{"fire", "flying"}, 0.390625},
		{"fighting vs ghost/psychic (immune then resist)", "fighting", []string{"ghost", "psychic"}, 0.244140625},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := pogopvp.TypeEffectiveness(tc.moveType, tc.defTypes)
			if math.Abs(got-tc.wantResult) > effectivenessEpsilon {
				t.Errorf("TypeEffectiveness(%q, %v) = %.6f, want %.6f",
					tc.moveType, tc.defTypes, got, tc.wantResult)
			}
		})
	}
}

func TestTypeEffectiveness_CaseInsensitive(t *testing.T) {
	t.Parallel()

	if got := pogopvp.TypeEffectiveness("FIRE", []string{"Grass"}); math.Abs(got-1.6) > effectivenessEpsilon {
		t.Errorf("mixed-case lookup = %.6f, want 1.6", got)
	}
}

func TestTypeEffectiveness_UnknownType(t *testing.T) {
	t.Parallel()

	// Unknown attacker type: no entries match, defender is taken as a
	// neutral passer-through. Returns 1.0 without panicking.
	if got := pogopvp.TypeEffectiveness("nonexistent", []string{"grass"}); got != 1.0 {
		t.Errorf("unknown attacker type = %.6f, want 1.0 neutral", got)
	}

	if got := pogopvp.TypeEffectiveness("fire", []string{"nonexistent"}); got != 1.0 {
		t.Errorf("unknown defender type = %.6f, want 1.0 neutral", got)
	}
}

func TestTypeEffectiveness_EmptyDefenderTypes(t *testing.T) {
	t.Parallel()

	if got := pogopvp.TypeEffectiveness("fire", nil); got != 1.0 {
		t.Errorf("nil defender types = %.6f, want 1.0", got)
	}

	if got := pogopvp.TypeEffectiveness("fire", []string{}); got != 1.0 {
		t.Errorf("empty defender types = %.6f, want 1.0", got)
	}
}
