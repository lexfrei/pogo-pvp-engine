# pogo-pvp-engine

Pure Go library implementing Pokémon GO PvP battle simulation and ranking
math. Intended as the engine layer for MCP servers, Discord bots, CLIs, or
any other consumer that needs accurate 1v1 PvP outcomes without pulling in a
JavaScript runtime.

**Status**: early development. Phase 1 is complete — foundation types and
math are available (`IV`, `Pokemon`, `Form`, `BaseStats`, `Stats`,
`ComputeStats`, `ComputeCP`, `ComputeStatProduct`, `CPMAt`, `Gamemaster`,
`Species`, `Move`, `ParseGamemaster`). Battle simulation, AI, ranker, and
IV-finder are still pending. No tagged release yet, and the repository is
not yet published to GitHub, so `go get github.com/lexfrei/pogo-pvp-engine`
does not resolve.

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
