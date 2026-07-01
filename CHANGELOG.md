# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com),
and this project adheres to [Semantic Versioning](https://semver.org).

## [Unreleased]

## [v0.4.0] - 2026-07-01

### Added

- `gh rdm doctor` to diagnose local server and SSH/Codespaces tunnel connectivity.
- `gh rdm tunnel [codespace]` to start the local server when needed and run the recommended Codespaces SSH tunnel.
- GitHub Actions CI workflow and repository license.

### Changed

- Prefer `localhost:7391` forwarding with `ExitOnForwardFailure=yes` in setup and README examples.
- Release workflow now uses the Go version from `go.mod`.

## [v0.3.3] - 2026-03-06

### Changed

- Documented screenshot auto-copy behavior and the `--copy` flag.

## [v0.3.2] - 2026-03-06

### Added

- Auto-copy screenshot and clipboard image `@` references after saving.

## [v0.3.1] - 2026-03-06

### Fixed

- Corrected screenshot `@` reference formatting in README and command output.

## [v0.3.0] - 2026-03-06

### Added

- `gh rdm screenshot` to fetch the latest local screenshot through the tunnel.
- `gh rdm clipboard-image` to fetch an image from the local clipboard.

### Changed

- `make` now defaults to the help target.
- Removed the experimental notice from the README.

## [v0.2.2] - 2026-02-23

### Added

- Interactive setup wizard (`gh rdm setup`).
- Makefile with standard dev commands.
- Changelog.

### Fixed

- Release artifacts use static linking and binary archives suitable for `gh extension install`.

## [v0.1.0] - 2026-02-22

### Added

- Initial implementation of gh-rdm CLI extension (remote dev manager with server/client).

[Unreleased]: https://github.com/maxbeizer/gh-rdm/compare/v0.4.0...HEAD
[v0.4.0]: https://github.com/maxbeizer/gh-rdm/compare/v0.3.3...v0.4.0
[v0.3.3]: https://github.com/maxbeizer/gh-rdm/compare/v0.3.2...v0.3.3
[v0.3.2]: https://github.com/maxbeizer/gh-rdm/compare/v0.3.1...v0.3.2
[v0.3.1]: https://github.com/maxbeizer/gh-rdm/compare/v0.3.0...v0.3.1
[v0.3.0]: https://github.com/maxbeizer/gh-rdm/compare/v0.2.2...v0.3.0
[v0.2.2]: https://github.com/maxbeizer/gh-rdm/compare/v0.1.0...v0.2.2
[v0.1.0]: https://github.com/maxbeizer/gh-rdm/releases/tag/v0.1.0
