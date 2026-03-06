package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/oleksandrskoruk/docusnap/internal/analyzer"
	"github.com/oleksandrskoruk/docusnap/internal/diff"
	"github.com/oleksandrskoruk/docusnap/internal/model"
	"github.com/oleksandrskoruk/docusnap/internal/render"
	"github.com/oleksandrskoruk/docusnap/internal/scanner"
)

func main() {
	if len(os.Args) < 2 {
		printHelp()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "scan":
		runScan(os.Args[2:])
	case "analyze":
		runAnalyze(os.Args[2:])
	case "diff":
		runDiff(os.Args[2:])
	case "render":
		runRender(os.Args[2:])
	case "run":
		runFullRun(os.Args[2:])
	default:
		printHelp()
		os.Exit(1)
	}
}

func runScan(args []string) {
	fs := flag.NewFlagSet("scan", flag.ExitOnError)
	path := fs.String("path", ".", "Path to project")
	out := fs.String("out", "", "Write snapshot JSON to file")
	pretty := fs.Bool("pretty", true, "Pretty JSON output")
	_ = fs.Parse(args)

	snap, err := scanner.Scan(*path)
	if err != nil {
		exitErr("scan error", err)
	}

	if *out != "" {
		if err := model.WriteSnapshot(*out, snap, *pretty); err != nil {
			exitErr("write snapshot", err)
		}
		fmt.Printf("snapshot written: %s\n", *out)
		return
	}

	printJSON(snap, *pretty)
}

func runAnalyze(args []string) {
	fs := flag.NewFlagSet("analyze", flag.ExitOnError)
	path := fs.String("path", ".", "Path to project")
	snapshotPath := fs.String("snapshot", "", "Analyze existing snapshot file")
	_ = fs.Parse(args)

	snap, err := loadOrScan(*snapshotPath, *path)
	if err != nil {
		exitErr("analyze", err)
	}

	fmt.Println(analyzer.RenderSummary(snap))
}

func runDiff(args []string) {
	fs := flag.NewFlagSet("diff", flag.ExitOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	markdownOut := fs.String("markdown-out", "", "Write markdown report to file")
	pretty := fs.Bool("pretty", true, "Pretty JSON output")
	_ = fs.Parse(args)

	rest := fs.Args()
	if len(rest) != 2 {
		fmt.Fprintln(os.Stderr, "usage: docusnap diff [--json] [--markdown-out changes.md] old.json new.json")
		os.Exit(1)
	}

	oldSnap, err := model.ReadSnapshot(rest[0])
	if err != nil {
		exitErr("read old snapshot", err)
	}
	newSnap, err := model.ReadSnapshot(rest[1])
	if err != nil {
		exitErr("read new snapshot", err)
	}

	result := diff.Compare(oldSnap, newSnap)
	if *markdownOut != "" {
		if err := writeTextFile(*markdownOut, result.RenderMarkdown()); err != nil {
			exitErr("write markdown report", err)
		}
		fmt.Printf("markdown report written: %s\n", *markdownOut)
	}
	if *jsonOut {
		printJSON(result, *pretty)
		return
	}

	fmt.Println(result.RenderText())
}

func runRender(args []string) {
	fs := flag.NewFlagSet("render", flag.ExitOnError)
	path := fs.String("path", ".", "Path to project")
	snapshotPath := fs.String("snapshot", "snapshot.json", "Snapshot file path")
	outDir := fs.String("out", "docs", "Output docs directory")
	pretty := fs.Bool("pretty", true, "Pretty snapshot JSON when auto-generated")
	_ = fs.Parse(args)

	snap, err := loadSnapshotOrGenerate(*snapshotPath, *path, *pretty)
	if err != nil {
		exitErr("render", err)
	}

	resolvedOutDir := resolveOutputPath(*path, *outDir)
	generated, err := render.Generate(snap, resolvedOutDir)
	if err != nil {
		exitErr("render", err)
	}

	fmt.Println("generated files:")
	for _, file := range generated {
		fmt.Printf("- %s\n", file)
	}
}

func runFullRun(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	path := fs.String("path", ".", "Path to project")
	snapshotPath := fs.String("snapshot", "snapshot.json", "Output snapshot file")
	docsDir := fs.String("docs", "docs", "Output docs directory")
	pretty := fs.Bool("pretty", true, "Pretty snapshot JSON")
	_ = fs.Parse(args)

	snap, err := scanner.Scan(*path)
	if err != nil {
		exitErr("scan", err)
	}

	resolvedSnapshotPath := resolveOutputPath(*path, *snapshotPath)
	if err := model.WriteSnapshot(resolvedSnapshotPath, snap, *pretty); err != nil {
		exitErr("write snapshot", err)
	}

	resolvedDocsDir := resolveOutputPath(*path, *docsDir)
	generated, err := render.Generate(snap, resolvedDocsDir)
	if err != nil {
		exitErr("render", err)
	}

	fmt.Printf("snapshot written: %s\n", resolvedSnapshotPath)
	fmt.Println("generated files:")
	for _, file := range generated {
		fmt.Printf("- %s\n", file)
	}
}

func loadOrScan(snapshotPath, path string) (model.Snapshot, error) {
	if snapshotPath != "" {
		return model.ReadSnapshot(snapshotPath)
	}
	return scanner.Scan(path)
}

func loadSnapshotOrGenerate(snapshotPath, projectPath string, pretty bool) (model.Snapshot, error) {
	resolvedSnapshotPath := resolveOutputPath(projectPath, snapshotPath)
	if _, err := os.Stat(resolvedSnapshotPath); err == nil {
		return model.ReadSnapshot(resolvedSnapshotPath)
	}

	snap, err := scanner.Scan(projectPath)
	if err != nil {
		return model.Snapshot{}, err
	}

	if err := model.WriteSnapshot(resolvedSnapshotPath, snap, pretty); err != nil {
		return model.Snapshot{}, err
	}

	return snap, nil
}

func resolveOutputPath(projectPath, value string) string {
	if filepath.IsAbs(value) {
		return value
	}
	return filepath.Join(projectPath, value)
}

func printJSON(v any, pretty bool) {
	var out []byte
	var err error

	if pretty {
		out, err = json.MarshalIndent(v, "", "  ")
	} else {
		out, err = json.Marshal(v)
	}
	if err != nil {
		exitErr("json marshal", err)
	}

	fmt.Println(string(out))
}

func exitErr(label string, err error) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", label, err)
	os.Exit(2)
}

func printHelp() {
	fmt.Print(`DocuSnap

Usage:
  docusnap scan --path . [--out snapshot.json]
  docusnap analyze --path .
  docusnap diff [--json] [--markdown-out changes.md] old.json new.json
  docusnap render --snapshot snapshot.json --out docs
  docusnap run --path .

Examples:
  docusnap scan --path . --out snapshot.json
  docusnap analyze --path .
  docusnap diff --markdown-out docs/changes.md old.json new.json
  docusnap render --snapshot snapshot.json --out docs
  docusnap run --path .
`)
}

func writeTextFile(path string, content string) error {
	dir := filepath.Dir(path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(path, []byte(content), 0o644)
}
