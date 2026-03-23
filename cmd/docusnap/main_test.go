package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oleksandrskoruk/docusnap/internal/model"
)

func TestRunRenderUsesPathForSnapshotAndOutput(t *testing.T) {
	projectDir := t.TempDir()

	snap := model.Snapshot{
		ProjectName:     "app",
		ProjectPath:     projectDir,
		ScannedAt:       "2026-03-06T00:00:00Z",
		Languages:       []string{"php"},
		Frameworks:      []string{"laravel"},
		PackageManagers: []string{"composer"},
		Dependencies: map[string][]model.Dependency{
			"composer": {
				{Name: "laravel/framework", Version: "^11.0"},
			},
		},
	}

	snapshotPath := filepath.Join(projectDir, "snapshot.json")
	if err := model.WriteSnapshot(snapshotPath, snap, true); err != nil {
		t.Fatalf("write snapshot: %v", err)
	}

	outDirName := "outdocs"
	runRender([]string{"--path", projectDir, "--snapshot", "snapshot.json", "--out", outDirName})

	readmePath := filepath.Join(projectDir, outDirName, "README.generated.md")
	data, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("read generated README: %v", err)
	}
	if !strings.Contains(string(data), "| Frameworks | laravel |") {
		t.Fatalf("expected README to use snapshot from --path, got:\n%s", string(data))
	}
}

func TestRunRenderSupportsHTMLFormat(t *testing.T) {
	projectDir := t.TempDir()

	snap := model.Snapshot{
		ProjectName: "app",
		ProjectPath: projectDir,
		ScannedAt:   "2026-03-06T00:00:00Z",
		Languages:   []string{"go"},
		Routes: []model.Route{
			{Method: "GET", Path: "/health", Controller: "Health"},
		},
	}

	snapshotPath := filepath.Join(projectDir, "snapshot.json")
	if err := model.WriteSnapshot(snapshotPath, snap, true); err != nil {
		t.Fatalf("write snapshot: %v", err)
	}

	runRender([]string{"--path", projectDir, "--snapshot", "snapshot.json", "--out", "site", "--format", "html"})

	htmlPath := filepath.Join(projectDir, "site", "index.html")
	data, err := os.ReadFile(htmlPath)
	if err != nil {
		t.Fatalf("read html documentation: %v", err)
	}
	if !strings.Contains(string(data), "<!DOCTYPE html>") {
		t.Fatalf("expected HTML output, got:\n%s", string(data))
	}
	if !strings.Contains(string(data), "/health") {
		t.Fatalf("expected route inventory in HTML output, got:\n%s", string(data))
	}
}

func TestRunScanUsesPathForRelativeOutput(t *testing.T) {
	projectDir := t.TempDir()

	goFile := filepath.Join(projectDir, "main.go")
	if err := os.WriteFile(goFile, []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}

	runScan([]string{"--path", projectDir, "--out", "snapshot.json"})

	snapshotPath := filepath.Join(projectDir, "snapshot.json")
	if _, err := os.Stat(snapshotPath); err != nil {
		t.Fatalf("expected snapshot in project dir: %v", err)
	}
}

func TestLoadOrScanUsesPathForRelativeSnapshot(t *testing.T) {
	projectDir := t.TempDir()

	snap := model.Snapshot{
		ProjectPath: projectDir,
		ScannedAt:   "2026-03-06T00:00:00Z",
		Languages:   []string{"python"},
	}

	snapshotPath := filepath.Join(projectDir, "snapshot.json")
	if err := model.WriteSnapshot(snapshotPath, snap, true); err != nil {
		t.Fatalf("write snapshot: %v", err)
	}

	got, err := loadOrScan("snapshot.json", projectDir)
	if err != nil {
		t.Fatalf("loadOrScan: %v", err)
	}
	if len(got.Languages) != 1 || got.Languages[0] != "python" {
		t.Fatalf("expected snapshot from project dir, got: %#v", got.Languages)
	}
}

func TestVersionStringIncludesBuildMetadata(t *testing.T) {
	prevVersion := version
	prevCommit := commit
	prevBuildDate := buildDate
	t.Cleanup(func() {
		version = prevVersion
		commit = prevCommit
		buildDate = prevBuildDate
	})

	version = "1.2.3"
	commit = "abc1234"
	buildDate = "2026-03-11T12:00:00Z"

	got := versionString()
	if !strings.Contains(got, "DocuSnap 1.2.3") {
		t.Fatalf("expected version in output, got:\n%s", got)
	}
	if !strings.Contains(got, "commit: abc1234") {
		t.Fatalf("expected commit in output, got:\n%s", got)
	}
	if !strings.Contains(got, "built: 2026-03-11T12:00:00Z") {
		t.Fatalf("expected build date in output, got:\n%s", got)
	}
}
