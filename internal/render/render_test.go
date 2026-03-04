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
