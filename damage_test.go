package pogopvp_test

import (
	"testing"

	pogopvp "github.com/lexfrei/pogo-pvp-engine"
)

func TestStabFactor(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name          string
		moveType      string
		attackerTypes []string
		want          float64
	}{
		{"stab match first", "fire", []string{"fire", "poison"}, pogopvp.StabMultiplier},
		{"stab match second", "psychic", []string{"fighting", "psychic"}, pogopvp.StabMultiplier},
		{"no stab", "fire", []string{"water"}, pogopvp.NeutralMatchup},
		{"stab case insensitive", "Fire", []string{"FIRE"}, pogopvp.StabMultiplier},
		{"empty types", "fire", nil, pogopvp.NeutralMatchup},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := pogopvp.StabFactor(tc.moveType, tc.attackerTypes)
			if got != tc.want {
				t.Errorf("StabFactor(%q, %v) = %v, want %v", tc.moveType, tc.attackerTypes, got, tc.want)
			}
		})
	}
}

// Anchor values verified in Python using the pvpoke formula:
//
//	damage = floor(power * stab * (atk/def) * effectiveness * 0.5 * BONUS) + 1
func TestCalcDamage_Anchors(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name          string
		power         int
		stab          float64
		effectiveness float64
		attack        float64
		defense       float64
		want          int
	}{
		{
			name:          "medicham counter vs machamp (fighting resists fighting)",
			power:         8,
			stab:          pogopvp.StabMultiplier,
			effectiveness: pogopvp.Resisted,
			attack:        107.4808015823363,
			defense:       123.65730053186412,
			want:          4,
		},
		{
			name:          "neutral fast move",
			power:         10,
			stab:          pogopvp.NeutralMatchup,
			effectiveness: pogopvp.NeutralMatchup,
			attack:        150.0,
			defense:       100.0,
			want:          10, // floor(10 * 1 * 1.5 * 1 * 0.5 * 1.3) + 1 = 9 + 1 = 10
		},
		{
			name:          "super effective with stab",
			power:         50,
			stab:          pogopvp.StabMultiplier,
			effectiveness: pogopvp.SuperEffective,
			attack:        120.0,
			defense:       110.0,
			want:          69, // floor(50 * 1.2 * (120/110) * 1.6 * 0.5 * 1.3) + 1 = 68 + 1
		},
		{
			name:          "minimum damage floor",
			power:         1,
			stab:          pogopvp.NeutralMatchup,
			effectiveness: pogopvp.DoubleResisted,
			attack:        100.0,
			defense:       300.0,
			want:          1, // very low raw damage → +1 floor
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := pogopvp.CalcDamage(tc.power, tc.stab, tc.effectiveness, tc.attack, tc.defense)
			if got != tc.want {
				t.Errorf("CalcDamage = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestCalcDamage_MinimumIsOne(t *testing.T) {
	t.Parallel()

	// Even with zero power / zero stats, damage floor is 1 (the trailing +1).
	if got := pogopvp.CalcDamage(0, 1, 1, 0, 1); got != 1 {
		t.Errorf("CalcDamage(zero) = %d, want 1", got)
	}
}
