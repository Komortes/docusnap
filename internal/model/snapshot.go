package model

// Dependency describes a package entry extracted from a manifest file.
type Dependency struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Route is a generic API endpoint descriptor.
type Route struct {
	Method     string `json:"method"`
	Path       string `json:"path"`
	Controller string `json:"controller"`
}

// Snapshot is the machine-readable description of a scanned repository.
type Snapshot struct {
	ProjectPath     string                  `json:"projectPath"`
	ScannedAt       string                  `json:"scannedAt"`
	Languages       []string                `json:"languages"`
	PackageManagers []string                `json:"packageManagers"`
	Frameworks      []string                `json:"frameworks"`
	Dependencies    map[string][]Dependency `json:"dependencies"`
	Routes          []Route                 `json:"routes"`
	ConfigFiles     []string                `json:"configFiles"`
	Infrastructure  []string                `json:"infrastructureServices"`
	DetectedFiles   []string                `json:"detectedFiles"`
}
