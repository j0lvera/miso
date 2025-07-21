# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),                                                                                    
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[unreleased]: https://github.com/j0lvera/miso/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/j0lvera/miso/releases/tag/v0.1.0  
