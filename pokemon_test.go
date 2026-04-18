package pogopvp_test

import (
	"errors"
	"testing"

	pogopvp "github.com/lexfrei/pogo-pvp-engine"
)

func TestParseForm_Valid(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in   string
		want pogopvp.Form
	}{
		{"regular", pogopvp.FormRegular},
		{"shadow", pogopvp.FormShadow},
		{"purified", pogopvp.FormPurified},
	}

	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()

			got, err := pogopvp.ParseForm(tc.in)
			if err != nil {
				t.Fatalf("ParseForm(%q) returned error: %v", tc.in, err)
			}
			if got != tc.want {
				t.Errorf("ParseForm(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestParseForm_Invalid(t *testing.T) {
	t.Parallel()

	cases := []string{"", "Regular", "SHADOW", "mega", "unknown"}
	for _, in := range cases {
		t.Run(in, func(t *testing.T) {
			t.Parallel()

			_, err := pogopvp.ParseForm(in)
			if err == nil {
				t.Fatalf("ParseForm(%q) expected error, got nil", in)
			}
			if !errors.Is(err, pogopvp.ErrInvalidForm) {
				t.Errorf("ParseForm(%q) error = %v, want wrapping ErrInvalidForm", in, err)
			}
		})
	}
}

func TestForm_String(t *testing.T) {
	t.Parallel()

	cases := []struct {
		form pogopvp.Form
		want string
	}{
		{pogopvp.FormRegular, "regular"},
		{pogopvp.FormShadow, "shadow"},
		{pogopvp.FormPurified, "purified"},
	}

	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			t.Parallel()

			if got := tc.form.String(); got != tc.want {
				t.Errorf("Form(%d).String() = %q, want %q", tc.form, got, tc.want)
			}
		})
	}
}

func TestNewPokemon_Valid(t *testing.T) {
	t.Parallel()

	iv := pogopvp.MustNewIV(0, 15, 15)
	cases := []struct {
		name    string
		species string
		form    pogopvp.Form
		level   float64
		xl      bool
	}{
		{"min level", "whiscash", pogopvp.FormRegular, 1.0, false},
		{"half level", "whiscash", pogopvp.FormRegular, 21.5, false},
		{"cap without xl", "medicham", pogopvp.FormRegular, 40.0, false},
		{"xl over cap", "registeel", pogopvp.FormRegular, 50.0, true},
		{"max level xl", "registeel", pogopvp.FormRegular, 51.0, true},
		{"shadow", "machamp", pogopvp.FormShadow, 40.0, false},
		{"purified", "machamp", pogopvp.FormPurified, 40.0, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			poke, err := pogopvp.NewPokemon(tc.species, tc.form, iv, tc.level, tc.xl)
			if err != nil {
				t.Fatalf("NewPokemon returned error: %v", err)
			}
			if poke.SpeciesID != tc.species {
				t.Errorf("SpeciesID = %q, want %q", poke.SpeciesID, tc.species)
			}
			if poke.Form != tc.form {
				t.Errorf("Form = %v, want %v", poke.Form, tc.form)
			}
			if poke.Level != tc.level {
				t.Errorf("Level = %v, want %v", poke.Level, tc.level)
			}
			if poke.XL != tc.xl {
				t.Errorf("XL = %v, want %v", poke.XL, tc.xl)
			}
			if poke.IV != iv {
				t.Errorf("IV = %+v, want %+v", poke.IV, iv)
			}
		})
	}
}

func TestNewPokemon_Invalid(t *testing.T) {
	t.Parallel()

	iv := pogopvp.MustNewIV(0, 15, 15)
	cases := []struct {
		name    string
		species string
		form    pogopvp.Form
		level   float64
		xl      bool
		target  error
	}{
		{"empty species", "", pogopvp.FormRegular, 40.0, false, pogopvp.ErrEmptySpeciesID},
		{"level below min", "whiscash", pogopvp.FormRegular, 0.5, false, pogopvp.ErrInvalidLevel},
		{"level above max", "whiscash", pogopvp.FormRegular, 51.5, true, pogopvp.ErrInvalidLevel},
		{"non-half level", "whiscash", pogopvp.FormRegular, 40.25, false, pogopvp.ErrInvalidLevel},
		{"non-xl above 40 without xl true", "whiscash", pogopvp.FormRegular, 41.5, false, pogopvp.ErrInvalidLevel},
		{"unknown form value far", "whiscash", pogopvp.Form(99), 40.0, false, pogopvp.ErrInvalidForm},
		{"unknown form value boundary", "whiscash", pogopvp.Form(3), 40.0, false, pogopvp.ErrInvalidForm},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := pogopvp.NewPokemon(tc.species, tc.form, iv, tc.level, tc.xl)
			if err == nil {
				t.Fatalf("NewPokemon expected error, got nil")
			}
			if !errors.Is(err, tc.target) {
				t.Errorf("NewPokemon error = %v, want wrapping %v", err, tc.target)
			}
		})
	}
}
