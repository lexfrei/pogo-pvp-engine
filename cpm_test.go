package pogopvp_test

import (
	"errors"
	"math"
	"testing"

	pogopvp "github.com/lexfrei/pogo-pvp-engine"
)

// cpmEpsilon is the tolerance used when comparing CPM values. The canonical
// values from pvpoke carry ~15 digits of precision; 1e-12 is tight enough to
// catch off-by-one table indexing without being sensitive to float noise.
const cpmEpsilon = 1e-12

func TestCPMAt_AnchorLevels(t *testing.T) {
	t.Parallel()

	cases := []struct {
		level float64
		want  float64
	}{
		{1.0, 0.0939999967813491},
		{1.5, 0.135137430784308},
		{10.0, 0.422500014305114},
		{20.0, 0.597400009632110},
		{30.0, 0.731700003147125},
		{40.0, 0.790300011634826},
		{50.0, 0.840300023555755},
		{51.0, 0.845300018787384},
	}

	for _, tc := range cases {
		t.Run("", func(t *testing.T) {
			t.Parallel()

			got, err := pogopvp.CPMAt(tc.level)
			if err != nil {
				t.Fatalf("CPMAt(%.1f) returned error: %v", tc.level, err)
			}
			if math.Abs(got-tc.want) > cpmEpsilon {
				t.Errorf("CPMAt(%.1f) = %.15f, want %.15f (|diff| > %g)",
					tc.level, got, tc.want, cpmEpsilon)
			}
		})
	}
}

func TestCPMAt_Monotonic(t *testing.T) {
	t.Parallel()

	var prev float64
	for doubled := 2; doubled <= 102; doubled++ {
		level := float64(doubled) / 2
		got, err := pogopvp.CPMAt(level)
		if err != nil {
			t.Fatalf("CPMAt(%.1f) returned error: %v", level, err)
		}
		if doubled > 2 && got <= prev {
			t.Errorf("CPMAt not monotonic: CPMAt(%.1f) = %.15f, previous = %.15f",
				level, got, prev)
		}
		prev = got
	}
}

func TestCPMAt_Invalid(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		level float64
		want  error
	}{
		{"below min", 0.5, pogopvp.ErrInvalidLevel},
		{"negative", -1.0, pogopvp.ErrInvalidLevel},
		{"above max", 51.5, pogopvp.ErrInvalidLevel},
		{"not on grid", 10.25, pogopvp.ErrInvalidLevel},
		{"NaN", math.NaN(), pogopvp.ErrInvalidLevel},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := pogopvp.CPMAt(tc.level)
			if err == nil {
				t.Fatalf("CPMAt(%.2f) expected error, got nil", tc.level)
			}
			if !errors.Is(err, tc.want) {
				t.Errorf("CPMAt(%.2f) error = %v, want wrapping %v", tc.level, err, tc.want)
			}
		})
	}
}
