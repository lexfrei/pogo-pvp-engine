# pogo-pvp-engine

Pure Go library implementing Pokémon GO PvP battle simulation and ranking
math. Intended as the engine layer for MCP servers, Discord bots, CLIs, or
any other consumer that needs accurate 1v1 PvP outcomes without pulling in a
JavaScript runtime.

**Status**: early development. Currently implemented: foundation types
and math (`IV`, `Pokemon`, `Form`, `BaseStats`, `Stats`, `ComputeStats`,
`ComputeCP`, `ComputeStatProduct`, `CPMAt`), gamemaster parser
(`Gamemaster`, `Species`, `Move`, `ParseGamemaster`), PvP damage
(`TypeEffectiveness`, `StabFactor`, `CalcDamage`), a simplified 1v1
battle simulator (`Simulate`, `Combatant`, `BattleOptions`,
`BattleResult`), and a pure stat-product IV/level optimizer
(`FindOptimalSpread`, `FindSpreadOpts`, `OptimalSpread`).

Battle simulator caveats: no Charge-Move-Priority on simultaneous
throws, no shadow Atk/Def scaling, fast-damage resolves before charged
throws on the shared tick. These gaps will close before the full
ranker lands. A matchup-weighted ranker is still pending. No tagged
release exists yet; the import path is not guaranteed stable until the
first semver tag.

## Disclaimer

This project is not affiliated with, endorsed by, or sponsored by Niantic,
Inc., Nintendo, The Pokémon Company, Game Freak, or Creatures Inc. "Pokémon"
and related names are trademarks of their respective owners.

The library operates on factual game data (stat lines, movesets, CPM values)
sourced from the open-source [PvPoke][pvpoke] project (MIT licensed). No
artwork, sprites, or audio is distributed. Pokémon are identified by string
id only.

## Dependencies

Zero external Go dependencies. Standard library only.

## License

BSD 3-Clause. See [LICENSE](LICENSE).

[pvpoke]: https://github.com/pvpoke/pvpoke
