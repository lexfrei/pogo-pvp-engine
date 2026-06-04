package pogopvp_test

// Shared test fixtures referenced from more than one _test.go file in this
// package. Centralizing them keeps the literals in a single place and avoids
// repeated magic strings across the suite.
const (
	// Synthetic move identifiers used by the battle simulator tests.
	moveTackle = "TACKLE"
	moveGain   = "GAIN"
	moveJolt   = "JOLT"
	moveBoom   = "BOOM"

	// Pokémon species identifiers reused across constructor and stat tests.
	speciesWhiscash  = "whiscash"
	speciesRegisteel = "registeel"
	speciesMachamp   = "machamp"

	// Form names, used both as ParseForm input and String() expectation.
	formRegular  = "regular"
	formShadow   = "shadow"
	formPurified = "purified"

	// Shared table-test case names.
	caseLevelBelowMin = "level below min"
	caseLevelAboveMax = "level above max"
	caseAtkOverMax    = "atk over max"
	caseDefOverMax    = "def over max"
	caseStaOverMax    = "sta over max"
)
