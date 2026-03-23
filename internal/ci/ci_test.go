package ci

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oleksandrskoruk/docusnap/internal/render"
)

func TestRunUpdateWritesSnapshotAndDocs(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectDir, "main.go"), []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}

	snapshotPath := filepath.Join(projectDir, "snapshot.json")
	docsDir := filepath.Join(projectDir, "docs")

	result, err := Run(Options{
		ProjectPath:  projectDir,
		SnapshotPath: snapshotPath,
		DocsDir:      docsDir,
		Pretty:       true,
		Format:       render.FormatBoth,
	}, ModeUpdate)
	if err != nil {
		t.Fatalf("run update: %v", err)
	}

	if len(result.Generated) == 0 {
		t.Fatalf("expected generated files")
	}
	if _, err := os.Stat(snapshotPath); err != nil {
		t.Fatalf("expected snapshot output: %v", err)
	}
	if _, err := os.Stat(filepath.Join(docsDir, "README.generated.md")); err != nil {
		t.Fatalf("expected markdown docs: %v", err)
	}
	if _, err := os.Stat(filepath.Join(docsDir, "index.html")); err != nil {
		t.Fatalf("expected html docs: %v", err)
	}
}

func TestRunCheckFailsWhenGeneratedDocsAreOutdated(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectDir, "main.go"), []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}

	snapshotPath := filepath.Join(projectDir, "snapshot.json")
	docsDir := filepath.Join(projectDir, "docs")

	if _, err := Run(Options{
		ProjectPath:  projectDir,
		SnapshotPath: snapshotPath,
		DocsDir:      docsDir,
		Pretty:       true,
		Format:       render.FormatMarkdown,
	}, ModeUpdate); err != nil {
		t.Fatalf("seed generated docs: %v", err)
	}

	if err := os.WriteFile(filepath.Join(projectDir, "server.js"), []byte("const express = require('express');\nconst app = express();\napp.get('/health', health);\n"), 0o644); err != nil {
		t.Fatalf("write server.js: %v", err)
	}

	_, err := Run(Options{
		ProjectPath:  projectDir,
		SnapshotPath: snapshotPath,
		DocsDir:      docsDir,
		Pretty:       true,
		Format:       render.FormatMarkdown,
	}, ModeCheck)
	if err == nil {
		t.Fatalf("expected outdated docs error")
	}
	if !strings.Contains(err.Error(), "generated artifacts are out of date") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(err.Error(), "snapshot.json") {
		t.Fatalf("expected snapshot mismatch in error: %v", err)
	}
}
