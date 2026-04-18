package pogopvp_test

import (
	"errors"
	"testing"

	pogopvp "github.com/lexfrei/pogo-pvp-engine"
)

func TestNewIV_Valid(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name          string
		atk, def, sta int
	}{
		{"all zeros", 0, 0, 0},
		{"all fifteens", 15, 15, 15},
		{"mixed mid", 10, 11, 12},
		{"boundary low", 0, 15, 0},
		{"boundary high", 15, 0, 15},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			iv, err := pogopvp.NewIV(tc.atk, tc.def, tc.sta)
			if err != nil {
				t.Fatalf("NewIV(%d, %d, %d) returned unexpected error: %v", tc.atk, tc.def, tc.sta, err)
			}

			if got, want := int(iv.Atk), tc.atk; got != want {
				t.Errorf("iv.Atk = %d, want %d", got, want)
			}
			if got, want := int(iv.Def), tc.def; got != want {
				t.Errorf("iv.Def = %d, want %d", got, want)
			}
			if got, want := int(iv.Sta), tc.sta; got != want {
				t.Errorf("iv.Sta = %d, want %d", got, want)
			}
		})
	}
}

func TestNewIV_OutOfRange(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name          string
		atk, def, sta int
	}{
		{"atk above max", 16, 0, 0},
		{"def above max", 0, 16, 0},
		{"sta above max", 0, 0, 16},
		{"atk negative", -1, 0, 0},
		{"def negative", 0, -1, 0},
		{"sta negative", 0, 0, -1},
		{"all out of range high", 20, 20, 20},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := pogopvp.NewIV(tc.atk, tc.def, tc.sta)
			if err == nil {
				t.Fatalf("NewIV(%d, %d, %d) expected error, got nil", tc.atk, tc.def, tc.sta)
			}

			if !errors.Is(err, pogopvp.ErrIVOutOfRange) {
				t.Errorf("NewIV(%d, %d, %d) error = %v, want wrapping ErrIVOutOfRange",
					tc.atk, tc.def, tc.sta, err)
			}
		})
	}
}

func TestMustNewIV_PanicsOnInvalid(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("MustNewIV(16, 0, 0) did not panic")
		}
	}()

	_ = pogopvp.MustNewIV(16, 0, 0)
}

func TestMustNewIV_NoPanicOnValid(t *testing.T) {
	t.Parallel()

	iv := pogopvp.MustNewIV(15, 15, 15)
	if iv.Atk != 15 || iv.Def != 15 || iv.Sta != 15 {
		t.Errorf("MustNewIV(15, 15, 15) = %+v, want {15, 15, 15}", iv)
	}
}
