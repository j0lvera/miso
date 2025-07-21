# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),                                                                                    
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.3.2] - 2025-07-22
### Fixed
- Use `amd64` for x86_64 architecture in `install.sh` to match GoReleaser artifacts.

## [0.3.1] - 2025-07-22

### Fixed

- Use `GITHUB_TOKEN` in `install.sh` to avoid API rate limits in CI.

## [0.3.0] - 2025-07-22

### Added

- Add `fish` shell support to the installation script.

### Fixed

- Adjust install script to use lowercase OS names to match GoReleaser artifacts.
- Adjust install script to remove `v` prefix from version for tarball filename.

## [0.2.1] - 2025-07-22
### Fixed
- Add LICENSE file to fix GoReleaser build.

## [0.2.0] - 2025-07-21
### Added
- Add `install.sh` script for easy installation via curl.
### Fixed
- Grant `pull-requests: write` permission to workflow to allow PR comments.
- Add `context` key to `miso.yml` to support non-diff reviews.
- Make `validate-config` command argument optional to support CI usage.
- Handle initial commit in `push` workflow to prevent diff errors.

## [0.1.0] - 2024-05-22

### Added

- Initial release.
- CLI with `review`, `diff`, `validate-config`, and `test-pattern` commands.
- Configuration using `miso.yml` for file-matching patterns and review guides.
- Ability to review single files and git diffs.
- GitHub Action for automating code reviews on pull requests.
- Self-review capability for the miso codebase using a Go practices guide.

[unreleased]: https://github.com/j0lvera/miso/compare/v0.3.2...HEAD
[0.3.2]: https://github.com/j0lvera/miso/compare/v0.3.1...v0.3.2
[0.3.1]: https://github.com/j0lvera/miso/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/j0lvera/miso/compare/v0.2.1...v0.3.0
[0.2.1]: https://github.com/j0lvera/miso/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/j0lvera/miso/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/j0lvera/miso/releases/tag/v0.1.0  
