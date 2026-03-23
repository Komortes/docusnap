package ci

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/oleksandrskoruk/docusnap/internal/model"
	"github.com/oleksandrskoruk/docusnap/internal/render"
	"github.com/oleksandrskoruk/docusnap/internal/scanner"
)

type Mode string

const (
	ModeCheck  Mode = "check"
	ModeUpdate Mode = "update"
)

type Options struct {
	ProjectPath  string
	SnapshotPath string
	DocsDir      string
	Pretty       bool
	Format       render.Format
}

type Result struct {
	Generated []string
	Removed   []string
	Outdated  []string
}

func ParseMode(value string) (Mode, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "check":
		return ModeCheck, nil
	case "update":
		return ModeUpdate, nil
	default:
		return "", fmt.Errorf("unsupported ci mode %q (expected check or update)", value)
	}
}

func Run(opts Options, mode Mode) (Result, error) {
	snap, err := scanner.Scan(opts.ProjectPath)
	if err != nil {
		return Result{}, err
	}

	switch mode {
	case ModeCheck:
		return runCheck(opts, snap)
	case ModeUpdate:
		return runUpdate(opts, snap)
	default:
		return Result{}, fmt.Errorf("unsupported ci mode %q", mode)
	}
}

func runCheck(opts Options, snap model.Snapshot) (Result, error) {
	tmpDir, err := os.MkdirTemp("", "docusnap-ci-*")
	if err != nil {
		return Result{}, err
	}
	defer os.RemoveAll(tmpDir)

	tmpSnapshot := filepath.Join(tmpDir, "snapshot.json")
	tmpDocs := filepath.Join(tmpDir, "docs")

	if err := model.WriteSnapshot(tmpSnapshot, snap, opts.Pretty); err != nil {
		return Result{}, err
	}
	if _, err := render.GenerateWithOptions(snap, tmpDocs, render.GenerateOptions{Format: opts.Format}); err != nil {
		return Result{}, err
	}

	outdated := make([]string, 0)
	if different, err := fileDiffers(opts.SnapshotPath, tmpSnapshot); err != nil {
		return Result{}, err
	} else if different {
		outdated = append(outdated, displayPath(opts.ProjectPath, opts.SnapshotPath))
	}

	expectedArtifacts := render.ArtifactNames(opts.Format)
	for _, name := range expectedArtifacts {
		target := filepath.Join(opts.DocsDir, name)
		candidate := filepath.Join(tmpDocs, name)
		if different, err := fileDiffers(target, candidate); err != nil {
			return Result{}, err
		} else if different {
			outdated = append(outdated, displayPath(opts.ProjectPath, target))
		}
	}

	expectedSet := map[string]struct{}{}
	for _, name := range expectedArtifacts {
		expectedSet[name] = struct{}{}
	}
	for _, name := range render.AllArtifactNames() {
		if _, ok := expectedSet[name]; ok {
			continue
		}
		target := filepath.Join(opts.DocsDir, name)
		if _, err := os.Stat(target); err == nil {
			outdated = append(outdated, displayPath(opts.ProjectPath, target))
		}
	}

	sort.Strings(outdated)
	result := Result{Outdated: outdated}
	if len(outdated) == 0 {
		return result, nil
	}

	return result, fmt.Errorf("generated artifacts are out of date:\n- %s", strings.Join(outdated, "\n- "))
}

func runUpdate(opts Options, snap model.Snapshot) (Result, error) {
	if err := model.WriteSnapshot(opts.SnapshotPath, snap, opts.Pretty); err != nil {
		return Result{}, err
	}

	generated, err := render.GenerateWithOptions(snap, opts.DocsDir, render.GenerateOptions{Format: opts.Format})
	if err != nil {
		return Result{}, err
	}

	expected := map[string]struct{}{}
	for _, path := range generated {
		expected[filepath.Base(path)] = struct{}{}
	}

	removed := make([]string, 0)
	for _, name := range render.AllArtifactNames() {
		if _, ok := expected[name]; ok {
			continue
		}
		path := filepath.Join(opts.DocsDir, name)
		if err := os.Remove(path); err == nil {
			removed = append(removed, path)
		} else if err != nil && !os.IsNotExist(err) {
			return Result{}, err
		}
	}

	sort.Strings(generated)
	sort.Strings(removed)
	return Result{Generated: generated, Removed: removed}, nil
}

func fileDiffers(left, right string) (bool, error) {
	leftData, leftErr := os.ReadFile(left)
	rightData, rightErr := os.ReadFile(right)

	switch {
	case leftErr == nil && rightErr == nil:
		return !bytes.Equal(leftData, rightData), nil
	case os.IsNotExist(leftErr) && rightErr == nil:
		return true, nil
	case leftErr == nil && os.IsNotExist(rightErr):
		return true, nil
	case os.IsNotExist(leftErr) && os.IsNotExist(rightErr):
		return false, nil
	case leftErr != nil:
		return false, leftErr
	default:
		return false, rightErr
	}
}

func displayPath(projectPath, path string) string {
	rel, err := filepath.Rel(projectPath, path)
	if err == nil && rel != "" && rel != "." && !strings.HasPrefix(rel, "..") {
		return filepath.ToSlash(rel)
	}
	return path
}
