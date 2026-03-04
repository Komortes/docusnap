package scanner

import (
	"os"
	"path/filepath"
	"sort"

	"github.com/oleksandrskoruk/docusnap/internal/model"
)

var markers = []string{
	"composer.json",
	"package.json",
	"go.mod",
	"Cargo.toml",
}

func Scan(root string) (model.Snapshot, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return model.Snapshot{}, err
	}

	var foundFiles []string
	var detected []string

	for _, name := range markers {
		p := filepath.Join(rootAbs, name)
		if _, err := os.Stat(p); err == nil {
			foundFiles = append(foundFiles, name)
			switch name {
			case "composer.json":
				detected = append(detected, "php/composer")
			case "package.json":
				detected = append(detected, "node/npm")
			case "go.mod":
				detected = append(detected, "go")
			case "Cargo.toml":
				detected = append(detected, "rust/cargo")
			}
		}
	}

	sort.Strings(foundFiles)
	sort.Strings(detected)

	return model.Snapshot{
		ProjectPath: rootAbs,
		Detected:    detected,
		Files:       foundFiles,
	}, nil
}