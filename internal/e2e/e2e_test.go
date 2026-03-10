package e2e

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/oleksandrskoruk/docusnap/internal/diff"
	"github.com/oleksandrskoruk/docusnap/internal/model"
	"github.com/oleksandrskoruk/docusnap/internal/render"
	"github.com/oleksandrskoruk/docusnap/internal/scanner"
)

var scannedAtPattern = regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z`)

func TestFixtureScanMatchesGolden(t *testing.T) {
	projectDir := copyFixtureProject(t, "sample_repo")

	snap, err := scanner.Scan(projectDir)
	if err != nil {
		t.Fatalf("scan fixture: %v", err)
	}

	normalized := normalizeSnapshot(snap, projectDir)
	got, err := marshalPrettyJSON(normalized)
	if err != nil {
		t.Fatalf("marshal snapshot: %v", err)
	}

	assertGoldenFile(t, "scan_snapshot.golden.json", string(got))
}

func TestFixtureRenderMatchesGoldens(t *testing.T) {
	projectDir := copyFixtureProject(t, "sample_repo")

	snap, err := scanner.Scan(projectDir)
	if err != nil {
		t.Fatalf("scan fixture: %v", err)
	}

	outDir := filepath.Join(t.TempDir(), "docs")
	if _, err := render.Generate(snap, outDir); err != nil {
		t.Fatalf("render fixture: %v", err)
	}

	files := map[string]string{
		"README.generated.md": "render_README.generated.md.golden",
		"dependencies.md":     "render_dependencies.md.golden",
		"endpoints.md":        "render_endpoints.md.golden",
		"dependency-graph.md": "render_dependency-graph.md.golden",
		"module-graph.md":     "render_module-graph.md.golden",
		"architecture.md":     "render_architecture.md.golden",
	}

	for generated, golden := range files {
		data, err := os.ReadFile(filepath.Join(outDir, generated))
		if err != nil {
			t.Fatalf("read generated file %s: %v", generated, err)
		}
		content := normalizeDocContent(string(data), projectDir)
		assertGoldenFile(t, golden, content)
	}
}

func TestDiffOutputsMatchGoldens(t *testing.T) {
	oldSnap := model.Snapshot{
		ProjectPath:     "<PROJECT_PATH>",
		ScannedAt:       "<SCANNED_AT>",
		Languages:       []string{"go", "javascript", "python"},
		PackageManagers: []string{"go", "npm", "pip"},
		Frameworks:      []string{"express", "fastapi", "gin", "react"},
		Dependencies: map[string][]model.Dependency{
			"go": {
				{Name: "github.com/gin-gonic/gin", Version: "v1.10.0"},
			},
			"npm": {
				{Name: "express", Version: "^5.0.0"},
				{Name: "react", Version: "^18.3.0"},
			},
			"pip": {
				{Name: "fastapi", Version: "==0.115.0"},
			},
		},
		Routes: []model.Route{
			{Method: "GET", Path: "/api/health", Controller: "server.Health"},
			{Method: "GET", Path: "/py/health", Controller: "health"},
			{Method: "GET", Path: "/web/health", Controller: "healthHandler"},
		},
		ConfigFiles:    []string{".env", "docker-compose.yml"},
		Infrastructure: []string{"docker", "env-file", "postgres", "redis"},
		DetectedFiles:  []string{"go.mod", "package.json", "requirements.txt"},
	}

	newSnap := model.Snapshot{
		ProjectPath:     "<PROJECT_PATH>",
		ScannedAt:       "<SCANNED_AT>",
		Languages:       []string{"go", "javascript", "python", "rust"},
		PackageManagers: []string{"cargo", "go", "npm", "pip"},
		Frameworks:      []string{"express", "fastapi", "gin", "next.js"},
		Dependencies: map[string][]model.Dependency{
			"cargo": {
				{Name: "serde", Version: "1.0"},
			},
			"go": {
				{Name: "github.com/gin-gonic/gin", Version: "v1.10.0"},
			},
			"npm": {
				{Name: "express", Version: "^5.0.0"},
				{Name: "next", Version: "^15.0.0"},
			},
			"pip": {
				{Name: "fastapi", Version: "==0.115.0"},
			},
		},
		Routes: []model.Route{
			{Method: "GET", Path: "/api/health", Controller: "server.Health"},
			{Method: "GET", Path: "/py/health", Controller: "health"},
			{Method: "POST", Path: "/api/orders", Controller: "server.CreateOrder"},
		},
		ConfigFiles:    []string{"docker-compose.yml", "k8s/deployment.yaml"},
		Infrastructure: []string{"docker", "kubernetes", "postgres", "redis"},
		DetectedFiles:  []string{"Cargo.toml", "go.mod", "package.json", "requirements.txt"},
	}

	result := diff.Compare(oldSnap, newSnap)

	jsonData, err := marshalPrettyJSON(result)
	if err != nil {
		t.Fatalf("marshal diff json: %v", err)
	}
	assertGoldenFile(t, "diff_result.golden.json", string(jsonData))
	assertGoldenFile(t, "diff_text.golden.txt", result.RenderText()+"\n")
	assertGoldenFile(t, "diff_markdown.golden.md", result.RenderMarkdown())
}

func copyFixtureProject(t *testing.T, fixtureName string) string {
	t.Helper()

	srcRoot := filepath.Join("testdata", fixtureName)
	dstRoot := filepath.Join(t.TempDir(), fixtureName)

	err := filepath.Walk(srcRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(srcRoot, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dstRoot, rel)

		if info.IsDir() {
			return os.MkdirAll(dstPath, 0o755)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dstPath, data, 0o644)
	})
	if err != nil {
		t.Fatalf("copy fixture %s: %v", fixtureName, err)
	}

	return dstRoot
}

func normalizeSnapshot(snap model.Snapshot, projectDir string) model.Snapshot {
	snap.ProjectPath = "<PROJECT_PATH>"
	snap.ScannedAt = "<SCANNED_AT>"
	return snap
}

func normalizeDocContent(content, projectDir string) string {
	content = strings.ReplaceAll(content, projectDir, "<PROJECT_PATH>")
	content = scannedAtPattern.ReplaceAllString(content, "<SCANNED_AT>")
	return content
}

func assertGoldenFile(t *testing.T, name, got string) {
	t.Helper()

	goldenPath := filepath.Join("testdata", "golden", name)
	wantBytes, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden %s: %v", name, err)
	}

	want := string(wantBytes)
	if got != want {
		t.Fatalf("golden mismatch for %s\n--- got ---\n%s\n--- want ---\n%s", name, got, want)
	}
}

func marshalPrettyJSON(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
