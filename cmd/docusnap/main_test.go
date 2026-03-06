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
	if !strings.Contains(string(data), "- laravel") {
		t.Fatalf("expected README to use snapshot from --path, got:\n%s", string(data))
	}
}
