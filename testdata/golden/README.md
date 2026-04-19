# Golden-snapshot corpus

This directory holds canonical [Simulate] outputs for a set of
fixed combatant pairs, used as a regression harness by
`golden_test.go`. Each `*.json` is one snapshot covering inputs
(attacker spec + defender spec + simulation options) and the
expected [BattleResult] fields — the test rebuilds the combatants,
runs `Simulate`, and asserts every field matches exactly.

## Coverage

The current snapshots are seeded from the engine's own `Simulate`
output — they protect against accidental regressions of the
engine's mechanics but do not yet cross-check with pvpoke.com.
Future passes will replace or extend these with snapshots captured
via the pvpoke.com simulator so the engine's behaviour is pinned
to upstream, not to itself.

## Snapshot schema

```json
{
  "name": "short human description",
  "attacker": {
    "species_id": "medicham",
    "iv": [0, 15, 15],
    "level": 50,
    "fast_move": "COUNTER",
    "charged_moves": ["ICE_PUNCH", "PSYCHIC"],
    "shields": 1
  },
  "defender": { "... same shape ..." },
  "options": { "max_turns": 0 },
  "expected": {
    "winner": 0,
    "turns": 123,
    "hp_remaining": [45, 0],
    "energy_at_end": [30, 70],
    "shields_used": [1, 1],
    "charged_fired": [2, 3]
  }
}
```

`winner` uses the engine's constants: `0` = attacker, `1` = defender,
`BattleTie = 2`, `BattleTimeout = 3`.

## Updating snapshots

Use `GOLDEN_UPDATE=1` to overwrite snapshots with the current engine
output:

```bash
GOLDEN_UPDATE=1 go test ./... -run TestGolden
```

Review the diff (`git diff testdata/golden/`) carefully — an
accidental regression in mechanics will silently get "blessed"
otherwise. Commit the diff in the same change that updates the
engine code.

## Adding a new snapshot

Create a new `*.json` file with a filename that matches the
`name` field (lowercase + dashes). Set `expected` to any values
(they will be overwritten). Run the update command above to fill
them in. Commit.
