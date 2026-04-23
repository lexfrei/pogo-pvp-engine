# pogo-pvp-engine

Pure Go library implementing Pokémon GO PvP battle simulation and ranking
math. Intended as the engine layer for MCP servers, Discord bots, CLIs, or
any other consumer that needs accurate 1v1 PvP outcomes without pulling in a
JavaScript runtime.

**Status**: early development. Currently implemented: foundation types and math (`IV`, `Pokemon`, `Form`, `BaseStats`, `Stats`, `ComputeStats`, `ComputeCP`, `ComputeStatProduct`, `CPMAt`), gamemaster parser (`Gamemaster`, `Species`, `Move`, `Cup`, `CupFilter`, `ParseGamemaster`) — `Species` carries `LegacyMoves`, `EliteMoves`, `Evolutions`, `PreEvolution`, `Tags`, `Released`, `ThirdMoveCost` (second-move unlock stardust), and `BuddyDistance` in addition to the stat line and move lists; `IsLegacyMove(species, moveID)` and `IsEliteMove(species, moveID)` are nil-safe per-species lookups for the two disjoint pvpoke restricted-move categories (legacy = permanently removed; elite = Elite TM / Community Day). Cups are parsed into a map keyed by cup id with include/exclude filter lists. PvP damage (`TypeEffectiveness`, `StabFactor`, `CalcDamage`), a simplified 1v1 battle simulator (`Simulate`, `Combatant`, `BattleOptions`, `BattleResult`) that applies Shadow ATK×1.2 / DEF÷1.2 when `Combatant.IsShadow=true`, a pure stat-product IV/level optimizer (`FindOptimalSpread`, `FindSpreadOpts`, `OptimalSpread`), and a CP → level inverse (`LevelForCP`, `LevelResult`, `ErrCPTooLow`) for cases where the caller knows the CP and needs the underlying level.

Battle simulator caveats: no Charge-Move-Priority on simultaneous
throws; fast-damage resolves before charged throws on the shared
tick. A matchup-weighted ranker is still pending. No tagged release
exists yet; the import path is not guaranteed stable until the first
semver tag.

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
