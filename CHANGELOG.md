# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.0] - 2026-09-27

### Added
- GitHub addon download counts are now read from the catalogue's new `downloads` field
- New `forever` game track for GitHub addons with the `forever` flavour (release.json spec 1-0-3)
  - strongbox needs matching support before it accepts catalogues containing `forever`

## [0.1.0] - 2025-10-05

### Added
- Initial Go translation of [strongbox-catalogue-builder](https://github.com/ogri-la/strongbox-catalogue-builder)
- WoWInterface scraping with API v3/v4 support
- GitHub catalogue integration
- HTTP caching
- Concurrent scraping with worker pools
- Catalogue validation
- Catalogues sorted by source-id

[unreleased]: https://github.com/ogri-la/strongbox-catalogue-builder-go/compare/0.2.0...HEAD
[0.2.0]: https://github.com/ogri-la/strongbox-catalogue-builder-go/compare/0.1.0...0.2.0
[0.1.0]: https://github.com/ogri-la/strongbox-catalogue-builder-go/releases/tag/0.1.0
