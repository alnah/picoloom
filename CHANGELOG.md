# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [2.2.0] - 2026-09-29

### New Features

- Admonitions in Markdown, with both blockquote alerts (`> [!NOTE] Title`) and `::: TYPE Title` fences, nesting, custom titles, and passthrough for unknown types. See [Admonitions](README.md#admonitions).
- Five admonition types (`note`, `tip`, `warning`, `important`, `caution`) plus the aliases `info`, `success`, and `danger`, each with an accent color overridable in custom CSS.
- Admonition styling for all eight built-in styles, with the pipeline and overlay documented in [Layout](docs/LAYOUT.md) and [Architecture](docs/ARCHITECTURE.md).

### Fixed

- Upgraded `github.com/yuin/goldmark` from v1.7.13 to v1.7.17 to fix GO-2026-5320, an XSS issue in link and image rendering.
