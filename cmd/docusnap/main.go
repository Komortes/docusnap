package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

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
	default:
		printHelp()
		os.Exit(1)
	}
}

func runScan(args []string) {
	fs := flag.NewFlagSet("scan", flag.ExitOnError)
	path := fs.String("path", ".", "Path to project")
	pretty := fs.Bool("pretty", true, "Pretty JSON output")
	_ = fs.Parse(args)

	snap, err := scanner.Scan(*path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "scan error:", err)
		os.Exit(2)
	}

	var out []byte
	if *pretty {
		out, _ = json.MarshalIndent(snap, "", "  ")
	} else {
		out, _ = json.Marshal(snap)
	}

	fmt.Println(string(out))
}

func printHelp() {
	fmt.Println(`DocuSnap (MVP)

Usage:
  docusnap scan --path . [--pretty=true]

Example:
  docusnap scan --path .
`)
}