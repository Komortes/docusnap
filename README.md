# DocuSnap

[![CI](https://github.com/Komortes/docusnap/actions/workflows/docusnap-docs.yml/badge.svg)](https://github.com/Komortes/docusnap/actions/workflows/docusnap-docs.yml)
[![Go](https://img.shields.io/github/go-mod/go-version/Komortes/docusnap?color=00ADD8)](go.mod)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

DocuSnap is a local-first CLI for scanning repositories, generating documentation from real code and config files, and comparing repository snapshots between versions.

Instead of maintaining hand-written docs that drift over time, the tool extracts what the project actually uses and turns it into Markdown, HTML, and machine-readable output.

## Highlights

- scans a repository and generates `snapshot.json`
- detects languages, package managers, frameworks, and infrastructure signals
- extracts dependencies from common ecosystem files
- finds API routes for supported stacks, OpenAPI specs, Next.js API handlers, and ASP.NET apps
- detects Java repositories via Maven and Gradle manifests and extracts Spring routes
- builds project structure, manifest inventory, dependency summaries, and API inventory
- renders Markdown docs, Mermaid graphs, and a ready-to-open HTML documentation page
- includes CI smoke checks for tests, scanning, and documentation generation
- ships packaged release artifacts, checksums, a Homebrew formula, and an installer script
- compares old and new snapshots with a readable diff
- works locally and in CI

## Why This Project

Repository documentation usually gets stale for one simple reason: the code changes faster than the docs.

DocuSnap solves that by treating the repository itself as the source of truth and generating:

- structured snapshot data for tooling
- readable Markdown docs for humans
- change reports for pull requests and release checks

This makes repository state easier to inspect, review, and automate.

## What This Demonstrates

- CLI design with focused commands for scan, analyze, render, run, ci, and diff workflows
- repository scanning across multiple language ecosystems without external runtime dependencies
- structured snapshot modeling with human-readable and machine-readable outputs
- generated documentation using Markdown, HTML, Go templates, and Mermaid diagrams
- automated test coverage around scanners, renderers, diffing, CI behavior, and end-to-end fixtures
- release packaging for Linux, macOS, and Windows with checksums, installer script, and Homebrew formula generation

## Demo

**`docusnap analyze --path .`** — human-readable project summary:

```
Project summary

Path
/home/user/myproject

Languages
- go
- javascript
- python

Frameworks
- express
- fastapi
- gin

Package managers
- go
- npm
- pip

Repository shape
- total files: 64
- source files: 41
- test files: 9
- manifest files: 5
- config files: 4

Dependencies
- go: 4
- npm: 18
- pip: 6
- total: 28

API endpoints
- 12 routes detected
- DELETE: 2
- GET: 6
- POST: 3
- PUT: 1

API groups
- /api: 12 (DELETE, GET, POST, PUT)

Services
- docker
- postgres
- redis
```

**`docusnap diff old.json new.json`** — what changed between two snapshots:

```
Changes detected

Languages
+ python

Dependencies
+ requests (pip)
+ sqlalchemy (pip)

Endpoints
+ POST /api/users
+ GET /api/users/:id
- GET /api/user/:id
```

**`docusnap run --path . --format both`** — scan and generate docs in one step:

```
snapshot written: snapshot.json
generated files:
- docs/README.generated.md
- docs/architecture.md
- docs/dependencies.md
- docs/dependency-graph.md
- docs/endpoints.md
- docs/module-graph.md
- docs/project-structure.md
- docs/index.html
```

## Tech Stack

- `Go`
- standard library CLI
- `JSON` for snapshot output
- `Markdown` and `Go templates` for docs
- `Mermaid` for graphs
- GitHub Actions for CI and release automation

## Run Locally

Build the binary:

```bash
make build
```

Check build metadata:

```bash
./bin/docusnap version
```

Run against the current repository:

```bash
go run ./cmd/docusnap run --path . --format both
```

Run against another local repository:

```bash
go run ./cmd/docusnap generate --path /absolute/path/to/project --format html
```

## Install

Local install:

```bash
make install
```

Direct Go install (version metadata will show as `dev`):

```bash
go install ./cmd/docusnap
```

Installer script from the latest GitHub release:

```bash
curl -fsSL https://github.com/Komortes/docusnap/releases/latest/download/install.sh | bash
```

Version metadata is injected through `ldflags`. If the current commit is tagged, the build uses that exact tag. Otherwise it falls back to `dev-<short-sha>`.

## Commands

- `docusnap version` - show build version, commit, and build time
- `docusnap scan --path . --out snapshot.json` - scan a repository and write a snapshot
- `docusnap analyze --path .` - print a project summary
- `docusnap render --path . --snapshot snapshot.json --out docs --format markdown` - render docs from a snapshot
- `docusnap run --path . --format both` - scan and render markdown plus HTML in one step
- `docusnap generate --path /absolute/path/to/project --format html` - alias for `run`, useful for one-shot documentation generation
- `docusnap ci --path . --mode check --format markdown` - verify generated snapshot and docs are up to date when they are tracked
- `docusnap ci --path . --mode update --format markdown` - rewrite tracked generated snapshot and docs in place
- `docusnap diff old.json new.json` - compare two snapshots

## What It Generates

- `snapshot.json`
- `docs/README.generated.md`
- `docs/project-structure.md`
- `docs/dependencies.md`
- `docs/endpoints.md`
- `docs/architecture.md`
- `docs/module-graph.md`
- `docs/dependency-graph.md`
- `docs/index.html` when `--format html` or `--format both`

## Supported Detection

Languages:

- `Go`
- `C#`
- `Java`
- `PHP`
- `JavaScript / TypeScript`
- `Python`
- `Rust`

Package managers:

- `go`
- `composer`
- `npm`
- `pip`
- `poetry`
- `cargo`
- `nuget`
- `maven`
- `gradle`

Framework signals:

- `Laravel`
- `React`
- `Express`
- `Next.js`
- `FastAPI`
- `Flask`
- `Django`
- `Gin`
- `Echo`
- `ASP.NET`
- `Spring`
- `OpenAPI`

Infrastructure hints:

- `docker-compose`
- `.env`
- Kubernetes manifests
- Terraform

## Example Workflow

Generate a snapshot:

```bash
go run ./cmd/docusnap scan --path /absolute/path/to/project --out /absolute/path/to/project/snapshot.json
```

Render docs:

```bash
go run ./cmd/docusnap render --path /absolute/path/to/project --snapshot /absolute/path/to/project/snapshot.json --out /absolute/path/to/project/docs --format markdown
```

Render an HTML documentation page:

```bash
go run ./cmd/docusnap generate --path /absolute/path/to/project --docs /absolute/path/to/project/docs --format html
```

Render both Markdown and HTML:

```bash
go run ./cmd/docusnap run --path /absolute/path/to/project --docs /absolute/path/to/project/docs --format both
```

Run the same smoke generation used by CI:

```bash
go run ./cmd/docusnap run --path . --snapshot /tmp/docusnap-snapshot.json --docs /tmp/docusnap-docs --format both
```

Refresh generated artifacts in place for a repository that tracks them:

```bash
go run ./cmd/docusnap ci --path /absolute/path/to/project --snapshot /absolute/path/to/project/snapshot.json --docs /absolute/path/to/project/docs --format markdown --mode update
```

Build release archives locally:

```bash
make dist VERSION=v0.1.0
```

Compare two versions:

```bash
go run ./cmd/docusnap diff old.json new.json
```

Write a Markdown diff report:

```bash
go run ./cmd/docusnap diff --markdown-out docs/changes.md old.json new.json
```

## CI

The repository includes a docs workflow at `.github/workflows/docusnap-docs.yml`.

The workflow runs the full Go test suite and smoke-generates a snapshot plus Markdown/HTML documentation into temporary paths. Generated snapshots include machine-specific paths and timestamps, so generated documentation is treated as build output rather than committed repository state.

## Releases

The repository also includes `.github/workflows/docusnap-release.yml`.

Release flow:

```bash
git tag v0.1.0
git push origin v0.1.0
```

The release workflow:

- runs the test suite
- builds a bundled release for Linux, macOS, and Windows
- packages archives with the binary, `README.md`, and installer script
- generates `SHA256SUMS.txt`
- emits `docusnap.rb` for Homebrew distribution
- uploads `install.sh` as a release asset
- publishes assets to GitHub Releases

Release artifacts are generated into `release/` locally or in GitHub Actions and are intentionally ignored by git.

## Limitations

Current analyzers are intentionally pragmatic, not fully AST-based everywhere.

Known tradeoffs:

- some route and import extraction is regex-based
- highly dynamic or multiline route definitions can be missed
- architecture graphs are summarized for readability rather than full fidelity

## Development

Run tests:

```bash
go test ./...
```

Run end-to-end fixture tests:

```bash
go test ./internal/e2e
```
