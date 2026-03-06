package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oleksandrskoruk/docusnap/internal/model"
)

func TestGenerateIncludesDependencyGraph(t *testing.T) {
	projectDir := t.TempDir()
	outDir := filepath.Join(projectDir, "docs")

	snap := model.Snapshot{
		ProjectPath: projectDir,
		Dependencies: map[string][]model.Dependency{
			"npm": {
				{Name: "react", Version: "^18.3.0"},
				{Name: "express", Version: "^5.0.0"},
			},
			"go": {
				{Name: "github.com/gin-gonic/gin", Version: "v1.10.0"},
			},
		},
	}

	files, err := Generate(snap, outDir)
	if err != nil {
		t.Fatalf("generate docs: %v", err)
	}
	if len(files) == 0 {
		t.Fatalf("expected generated files")
	}

	graphPath := filepath.Join(outDir, "dependency-graph.md")
	data, err := os.ReadFile(graphPath)
	if err != nil {
		t.Fatalf("read dependency graph file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "graph LR") {
		t.Fatalf("expected mermaid graph, got:\n%s", content)
	}
	if !strings.Contains(content, "react") || !strings.Contains(content, "github.com/gin-gonic/gin") {
		t.Fatalf("expected dependency nodes in graph, got:\n%s", content)
	}
}

func TestGenerateIncludesModuleGraph(t *testing.T) {
	projectDir := t.TempDir()
	outDir := filepath.Join(projectDir, "docs")

	if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte("module example.com/proj\n\ngo 1.25\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectDir, "internal", "a"), 0o755); err != nil {
		t.Fatalf("mkdir go src: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectDir, "internal", "b"), 0o755); err != nil {
		t.Fatalf("mkdir go src: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "internal", "a", "a.go"), []byte("package a\nimport _ \"example.com/proj/internal/b\"\n"), 0o644); err != nil {
		t.Fatalf("write a.go: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "internal", "b", "b.go"), []byte("package b\n"), 0o644); err != nil {
		t.Fatalf("write b.go: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(projectDir, "src"), 0o755); err != nil {
		t.Fatalf("mkdir js src: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "src", "index.js"), []byte("import './util.js'\n"), 0o644); err != nil {
		t.Fatalf("write index.js: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "src", "util.js"), []byte("export const v = 1\n"), 0o644); err != nil {
		t.Fatalf("write util.js: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(projectDir, "app"), 0o755); err != nil {
		t.Fatalf("mkdir py src: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "app", "api.py"), []byte("from .service import fn\n"), 0o644); err != nil {
		t.Fatalf("write api.py: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "app", "service.py"), []byte("def fn():\n    return 1\n"), 0o644); err != nil {
		t.Fatalf("write service.py: %v", err)
	}

	snap := model.Snapshot{ProjectPath: projectDir}
	if _, err := Generate(snap, outDir); err != nil {
		t.Fatalf("generate docs: %v", err)
	}

	graphPath := filepath.Join(outDir, "module-graph.md")
	data, err := os.ReadFile(graphPath)
	if err != nil {
		t.Fatalf("read module graph file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "graph LR") {
		t.Fatalf("expected mermaid graph, got:\n%s", content)
	}
	if !strings.Contains(content, "internal/a/a.go") || !strings.Contains(content, "internal/b") {
		t.Fatalf("expected go module edge in graph, got:\n%s", content)
	}
	if !strings.Contains(content, "src/index.js") || !strings.Contains(content, "src/util.js") {
		t.Fatalf("expected js module edge in graph, got:\n%s", content)
	}
	if !strings.Contains(content, "app/api.py") || !strings.Contains(content, "app/service") {
		t.Fatalf("expected python module edge in graph, got:\n%s", content)
	}
}
