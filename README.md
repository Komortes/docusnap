# DocuSnap

DocuSnap is a local-first CLI that scans a repository, builds a machine-readable snapshot of its current state, and renders repository documentation from the code that actually exists.

The product goal for `v1` is narrow on purpose:

- scan a repository locally
- extract technologies, dependencies, routes, and infrastructure signals
- render stable Markdown docs
- compare two snapshots and show what changed
- run the same workflow in CI

## Why

Hand-written repository documentation drifts.

Code review diffs show line changes, but they do not clearly show higher-level repository changes such as:

- new dependencies
- removed endpoints
- framework changes
- infrastructure changes

DocuSnap makes repository state observable with:

- `snapshot.json` for tooling and automation
- generated Markdown docs for people
- snapshot diff reports for change intelligence

## Current Product Scope

`DocuSnap v1` is a local CLI for repository snapshot, documentation rendering, and snapshot diff.

Core commands:

- `scan`: inspect a project and produce `snapshot.json`
- `analyze`: print a concise project summary
- `render`: generate docs from a snapshot
- `run`: scan + write snapshot + render docs
- `diff`: compare two snapshots

Supported analysis today:

- languages: `Go`, `PHP`, `JavaScript/TypeScript`, `Python`, `Rust`
- package managers: `go`, `composer`, `npm`, `pip`, `poetry`, `cargo`
- framework detection: `Laravel`, `React`, `Express`, `FastAPI`, `Flask`, `Django`, `Gin`, `Echo`
- dependency extraction
- route extraction for supported frameworks
- infrastructure hints from `docker-compose`, `.env`, Kubernetes manifests, and Terraform
- Markdown output with dependency and architecture graphs

## Quick Start

Run against the current repository:

```bash
go run ./cmd/docusnap run --path .
```

Run against another local repository:

```bash
go run ./cmd/docusnap run --path /absolute/path/to/project
```

Outputs:

- `snapshot.json`
- `docs/README.generated.md`
- `docs/dependencies.md`
- `docs/endpoints.md`
- `docs/architecture.md`
- `docs/module-graph.md`
- `docs/dependency-graph.md`

## Command Examples

Scan only:

```bash
go run ./cmd/docusnap scan --path /absolute/path/to/project --out /absolute/path/to/project/snapshot.json
```

Analyze:

```bash
go run ./cmd/docusnap analyze --path /absolute/path/to/project
```

Render docs from an existing snapshot:

```bash
go run ./cmd/docusnap render \
  --path /absolute/path/to/project \
  --snapshot /absolute/path/to/project/snapshot.json \
  --out /absolute/path/to/project/docs
```

Compare two snapshots:

```bash
go run ./cmd/docusnap diff old.json new.json
```

Write a Markdown diff report:

```bash
go run ./cmd/docusnap diff \
  --markdown-out /absolute/path/to/project/docs/changes.md \
  old.json \
  new.json
```

## Example Snapshot

```json
{
  "projectPath": "/repo/project",
  "languages": ["php", "javascript"],
  "frameworks": ["laravel", "react"],
  "packageManagers": ["composer", "npm"],
  "dependencies": {
    "composer": [
      { "name": "laravel/framework", "version": "^11.0" }
    ],
    "npm": [
      { "name": "react", "version": "^18.0.0" }
    ]
  },
  "routes": [
    {
      "method": "GET",
      "path": "/api/orders",
      "controller": "OrderController@index"
    }
  ]
}
```

## Example Diff

Text output:

```text
Changes detected

Dependencies
+ stripe/stripe-php
- laravel/sanctum

Endpoints
+ POST /api/payment
```

## CI Workflow

This repository includes a GitHub Actions workflow at `.github/workflows/docusnap-docs.yml`.

It supports two product modes:

- `check`: run tests, regenerate `snapshot.json` and `docs/`, and fail if generated artifacts are outdated
- `update`: regenerate artifacts and commit them back to `main` when they changed

This is the intended CI story for `DocuSnap v1`: generated repository docs should be reproducible and reviewable.

## Limitations

Current route and import extraction is heuristic in several places.

Known limitations:

- some parsers are regex-based rather than full AST-based
- multiline or highly dynamic route definitions can be missed
- architecture graphs are best-effort and intentionally summarized for readability

## Development

Run the full test suite:

```bash
go test ./...
```

Run fixture-based end-to-end tests:

```bash
go test ./internal/e2e
```

## Roadmap

`Done in v1`

- repository scan
- dependency extraction
- framework detection
- route extraction for supported frameworks
- snapshot generation
- Markdown rendering
- snapshot diff
- CI drift check/update workflow

`Pinned for later`

- richer architecture views beyond current summary graphs
- deeper AST-based parsers for route and import extraction
- more framework-specific analyzers
- cloud/service detection expansion
- release packaging and distribution
- plugin system

See [docs/ROADMAP.md](docs/ROADMAP.md) for the scoped roadmap.
