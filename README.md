# DocuSnap

DocuSnap is a local-first CLI for scanning repositories, generating documentation from real code and config files, and comparing repository snapshots between versions.

Instead of maintaining hand-written docs that drift over time, the tool extracts what the project actually uses and turns it into Markdown and machine-readable output.

## Highlights

- scans a repository and generates `snapshot.json`
- detects languages, package managers, frameworks, and infrastructure signals
- extracts dependencies from common ecosystem files
- finds API routes for supported stacks
- renders Markdown docs and Mermaid graphs
- compares old and new snapshots with a readable diff
- works locally and in CI

## Why This Project

Repository documentation usually gets stale for one simple reason: the code changes faster than the docs.

DocuSnap solves that by treating the repository itself as the source of truth and generating:

- structured snapshot data for tooling
- readable Markdown docs for humans
- change reports for pull requests and release checks

This makes repository state easier to inspect, review, and automate.

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
go run ./cmd/docusnap run --path .
```

Run against another local repository:

```bash
go run ./cmd/docusnap run --path /absolute/path/to/project
```

## Install

Local install:

```bash
make install
```

Direct Go install:

```bash
go install ./cmd/docusnap
```

Version metadata is injected through `ldflags`. If the current commit is tagged, the build uses that exact tag. Otherwise it falls back to `dev-<short-sha>`.

## Commands

- `docusnap version` - show build version, commit, and build time
- `docusnap scan --path . --out snapshot.json` - scan a repository and write a snapshot
- `docusnap analyze --path .` - print a project summary
- `docusnap render --path . --snapshot snapshot.json --out docs` - render docs from a snapshot
- `docusnap run --path .` - scan and render in one step
- `docusnap diff old.json new.json` - compare two snapshots

## What It Generates

- `snapshot.json`
- `docs/README.generated.md`
- `docs/dependencies.md`
- `docs/endpoints.md`
- `docs/architecture.md`
- `docs/module-graph.md`
- `docs/dependency-graph.md`

## Supported Detection

Languages:

- `Go`
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

Framework signals:

- `Laravel`
- `React`
- `Express`
- `FastAPI`
- `Flask`
- `Django`
- `Gin`
- `Echo`

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
go run ./cmd/docusnap render --path /absolute/path/to/project --snapshot /absolute/path/to/project/snapshot.json --out /absolute/path/to/project/docs
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

It supports two modes:

- `check` - regenerate docs and fail if tracked artifacts are outdated
- `update` - regenerate docs and commit updated artifacts back to the branch

This is the main CI workflow for keeping generated documentation reproducible.

## Releases

The repository also includes `.github/workflows/docusnap-release.yml`.

Release flow:

```bash
git tag v0.1.0
git push origin v0.1.0
```

The release workflow:

- runs the test suite
- builds binaries for Linux, macOS, and Windows
- packages release archives
- generates checksums
- publishes assets to GitHub Releases

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
