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
		ProjectName: "proj",
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
	if err := os.WriteFile(filepath.Join(projectDir, "src", "util.test.js"), []byte("import './util.js'\n"), 0o644); err != nil {
		t.Fatalf("write util.test.js: %v", err)
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
	if err := os.WriteFile(filepath.Join(projectDir, "app", "test_api.py"), []byte("from .service import fn\n"), 0o644); err != nil {
		t.Fatalf("write test_api.py: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "internal", "a", "a_test.go"), []byte("package a\nimport _ \"example.com/proj/internal/b\"\n"), 0o644); err != nil {
		t.Fatalf("write a_test.go: %v", err)
	}

	snap := model.Snapshot{ProjectName: "proj", ProjectPath: projectDir}
	if _, err := Generate(snap, outDir); err != nil {
		t.Fatalf("generate docs: %v", err)
	}

	graphPath := filepath.Join(outDir, "module-graph.md")
	data, err := os.ReadFile(graphPath)
	if err != nil {
		t.Fatalf("read module graph file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "graph TD") {
		t.Fatalf("expected mermaid graph, got:\n%s", content)
	}
	if !strings.Contains(content, `subgraph G_internal["internal"]`) {
		t.Fatalf("expected grouped internal subgraph, got:\n%s", content)
	}
	if !strings.Contains(content, "internal/a") || !strings.Contains(content, "internal/b") {
		t.Fatalf("expected go module edge in graph, got:\n%s", content)
	}
	if !strings.Contains(content, `subgraph G_src["src"]`) {
		t.Fatalf("expected grouped src subgraph, got:\n%s", content)
	}
	if !strings.Contains(content, "src") || !strings.Contains(content, "src/util") {
		t.Fatalf("expected js module edge in graph, got:\n%s", content)
	}
	if !strings.Contains(content, `subgraph G_app["app"]`) {
		t.Fatalf("expected grouped app subgraph, got:\n%s", content)
	}
	if !strings.Contains(content, "app/api") || !strings.Contains(content, "app/service") {
		t.Fatalf("expected python module edge in graph, got:\n%s", content)
	}
	if strings.Contains(content, "util.test.js") || strings.Contains(content, "test_api.py") || strings.Contains(content, "a_test.go") {
		t.Fatalf("expected test files to be excluded from module graph, got:\n%s", content)
	}
}

func TestGenerateIncludesPHPModuleGraph(t *testing.T) {
	projectDir := t.TempDir()
	outDir := filepath.Join(projectDir, "docs")

	composerJSON := `{
  "autoload": {
    "psr-4": {
      "App\\": "app/"
    }
  },
  "autoload-dev": {
    "psr-4": {
      "Tests\\": "tests/"
    }
  }
}`
	if err := os.WriteFile(filepath.Join(projectDir, "composer.json"), []byte(composerJSON), 0o644); err != nil {
		t.Fatalf("write composer.json: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(projectDir, "app", "Http", "Controllers"), 0o755); err != nil {
		t.Fatalf("mkdir controllers: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectDir, "app", "Services"), 0o755); err != nil {
		t.Fatalf("mkdir services: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectDir, "tests", "Feature"), 0o755); err != nil {
		t.Fatalf("mkdir tests: %v", err)
	}

	controllerPHP := `<?php
namespace App\Http\Controllers;

use App\Services\OrderService;

class OrderController {}
`
	if err := os.WriteFile(filepath.Join(projectDir, "app", "Http", "Controllers", "OrderController.php"), []byte(controllerPHP), 0o644); err != nil {
		t.Fatalf("write controller: %v", err)
	}

	servicePHP := `<?php
namespace App\Services;

use App\Models\Order;

class OrderService {}
`
	if err := os.WriteFile(filepath.Join(projectDir, "app", "Services", "OrderService.php"), []byte(servicePHP), 0o644); err != nil {
		t.Fatalf("write service: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(projectDir, "app", "Models"), 0o755); err != nil {
		t.Fatalf("mkdir models: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "app", "Models", "Order.php"), []byte("<?php\nnamespace App\\Models;\nclass Order {}\n"), 0o644); err != nil {
		t.Fatalf("write model: %v", err)
	}

	testPHP := `<?php
namespace Tests\Feature;

use App\Services\OrderService;

class OrderControllerTest {}
`
	if err := os.WriteFile(filepath.Join(projectDir, "tests", "Feature", "OrderControllerTest.php"), []byte(testPHP), 0o644); err != nil {
		t.Fatalf("write php test: %v", err)
	}

	snap := model.Snapshot{ProjectName: "proj", ProjectPath: projectDir}
	if _, err := Generate(snap, outDir); err != nil {
		t.Fatalf("generate docs: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(outDir, "module-graph.md"))
	if err != nil {
		t.Fatalf("read module graph file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, `subgraph G_app["app"]`) {
		t.Fatalf("expected app subgraph, got:\n%s", content)
	}
	if !strings.Contains(content, "app/Http") || !strings.Contains(content, "app/Services") || !strings.Contains(content, "app/Models") {
		t.Fatalf("expected php module nodes, got:\n%s", content)
	}
	if strings.Contains(content, "tests/Feature") {
		t.Fatalf("expected php tests to be excluded, got:\n%s", content)
	}
}

func TestGenerateIncludesLaravelArchitectureGraph(t *testing.T) {
	projectDir := t.TempDir()
	outDir := filepath.Join(projectDir, "docs")

	composerJSON := `{
  "autoload": {
    "psr-4": {
      "App\\": "app/"
    }
  }
}`
	if err := os.WriteFile(filepath.Join(projectDir, "composer.json"), []byte(composerJSON), 0o644); err != nil {
		t.Fatalf("write composer.json: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(projectDir, "app", "Http", "Controllers"), 0o755); err != nil {
		t.Fatalf("mkdir controllers: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectDir, "app", "Actions"), 0o755); err != nil {
		t.Fatalf("mkdir actions: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectDir, "app", "Services"), 0o755); err != nil {
		t.Fatalf("mkdir services: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectDir, "app", "Models"), 0o755); err != nil {
		t.Fatalf("mkdir models: %v", err)
	}

	controllerPHP := `<?php
namespace App\Http\Controllers;

use App\Actions\SyncOrderAction;

class OrderController {}
`
	if err := os.WriteFile(filepath.Join(projectDir, "app", "Http", "Controllers", "OrderController.php"), []byte(controllerPHP), 0o644); err != nil {
		t.Fatalf("write controller: %v", err)
	}

	actionPHP := `<?php
namespace App\Actions;

use App\Services\OrderService;

class SyncOrderAction {}
`
	if err := os.WriteFile(filepath.Join(projectDir, "app", "Actions", "SyncOrderAction.php"), []byte(actionPHP), 0o644); err != nil {
		t.Fatalf("write action: %v", err)
	}

	servicePHP := `<?php
namespace App\Services;

use App\Models\Order;

class OrderService {}
`
	if err := os.WriteFile(filepath.Join(projectDir, "app", "Services", "OrderService.php"), []byte(servicePHP), 0o644); err != nil {
		t.Fatalf("write service: %v", err)
	}

	modelPHP := `<?php
namespace App\Models;

class Order {}
`
	if err := os.WriteFile(filepath.Join(projectDir, "app", "Models", "Order.php"), []byte(modelPHP), 0o644); err != nil {
		t.Fatalf("write model: %v", err)
	}

	snap := model.Snapshot{
		ProjectName: "proj",
		ProjectPath: projectDir,
		Frameworks:  []string{"laravel"},
	}
	if _, err := Generate(snap, outDir); err != nil {
		t.Fatalf("generate docs: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(outDir, "architecture.md"))
	if err != nil {
		t.Fatalf("read architecture file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "## Laravel Layer Graph") {
		t.Fatalf("expected Laravel graph section, got:\n%s", content)
	}
	if !strings.Contains(content, `L_Http["Http"]`) || !strings.Contains(content, `L_Actions["Actions"]`) ||
		!strings.Contains(content, `L_Services["Services"]`) || !strings.Contains(content, `L_Models["Models"]`) {
		t.Fatalf("expected Laravel layer nodes, got:\n%s", content)
	}
	if !strings.Contains(content, "L_Http --> L_Actions") ||
		!strings.Contains(content, "L_Actions --> L_Services") ||
		!strings.Contains(content, "L_Services --> L_Models") {
		t.Fatalf("expected Laravel layer edges, got:\n%s", content)
	}
}

func TestGenerateIncludesProjectStructureDocument(t *testing.T) {
	projectDir := t.TempDir()
	outDir := filepath.Join(projectDir, "docs")

	snap := model.Snapshot{
		ProjectName: "proj",
		ProjectPath: projectDir,
		ProjectStats: model.ProjectStats{
			TotalFiles:    12,
			SourceFiles:   6,
			TestFiles:     2,
			ManifestFiles: 2,
			ConfigFiles:   1,
		},
		ManifestFiles: []model.ManifestFile{
			{Kind: "dependency", Path: "package.json"},
			{Kind: "runtime", Path: ".env"},
		},
		DirectoryLayout: []model.DirectorySummary{
			{
				Path:          "root",
				FileCount:     3,
				SourceFiles:   1,
				ManifestFiles: 2,
				NotableFiles:  []string{"package.json", ".env"},
			},
			{
				Path:         "src",
				FileCount:    5,
				SourceFiles:  5,
				Languages:    []string{"javascript"},
				NotableFiles: []string{"src/index.js"},
			},
		},
		EntryPoints: []string{"src/index.js"},
	}

	if _, err := Generate(snap, outDir); err != nil {
		t.Fatalf("generate docs: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(outDir, "project-structure.md"))
	if err != nil {
		t.Fatalf("read project structure file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "# Project Structure") {
		t.Fatalf("expected project structure heading, got:\n%s", content)
	}
	if !strings.Contains(content, "package.json") || !strings.Contains(content, "src/index.js") {
		t.Fatalf("expected manifest and entry point details, got:\n%s", content)
	}
}

func TestGenerateSupportsHTMLOutput(t *testing.T) {
	projectDir := t.TempDir()
	outDir := filepath.Join(projectDir, "docs")

	snap := model.Snapshot{
		ProjectName: "proj",
		ProjectPath: projectDir,
		ProjectStats: model.ProjectStats{
			TotalFiles:    4,
			SourceFiles:   2,
			ManifestFiles: 1,
		},
		DirectoryLayout: []model.DirectorySummary{
			{
				Path:         "root",
				FileCount:    2,
				SourceFiles:  1,
				NotableFiles: []string{"main.go"},
			},
		},
		Routes: []model.Route{
			{Method: "GET", Path: "/health", Controller: "Health"},
		},
	}

	files, err := GenerateWithOptions(snap, outDir, GenerateOptions{Format: FormatHTML})
	if err != nil {
		t.Fatalf("generate html docs: %v", err)
	}
	if len(files) != 1 || filepath.Base(files[0]) != "index.html" {
		t.Fatalf("expected only index.html, got: %#v", files)
	}

	data, err := os.ReadFile(filepath.Join(outDir, "index.html"))
	if err != nil {
		t.Fatalf("read html output: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "<!DOCTYPE html>") {
		t.Fatalf("expected HTML document, got:\n%s", content)
	}
	if !strings.Contains(content, "Project Structure") || !strings.Contains(content, "/health") {
		t.Fatalf("expected rendered sections in html output, got:\n%s", content)
	}
}
