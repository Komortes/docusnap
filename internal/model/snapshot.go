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

// ProjectStats describes the overall repository footprint.
type ProjectStats struct {
	TotalFiles    int `json:"totalFiles"`
	SourceFiles   int `json:"sourceFiles"`
	TestFiles     int `json:"testFiles"`
	ManifestFiles int `json:"manifestFiles"`
	ConfigFiles   int `json:"configFiles"`
}

// ManifestFile is a notable project manifest or operational config.
type ManifestFile struct {
	Path string `json:"path"`
	Kind string `json:"kind"`
}

// DirectorySummary aggregates the repository structure into readable buckets.
type DirectorySummary struct {
	Path          string   `json:"path"`
	FileCount     int      `json:"fileCount"`
	SourceFiles   int      `json:"sourceFiles"`
	TestFiles     int      `json:"testFiles"`
	ManifestFiles int      `json:"manifestFiles"`
	ConfigFiles   int      `json:"configFiles"`
	Languages     []string `json:"languages"`
	NotableFiles  []string `json:"notableFiles"`
}

// APIGroup summarizes endpoints by top-level route prefix.
type APIGroup struct {
	Prefix     string   `json:"prefix"`
	RouteCount int      `json:"routeCount"`
	Methods    []string `json:"methods"`
}

// Snapshot is the machine-readable description of a scanned repository.
type Snapshot struct {
	ProjectName     string                  `json:"projectName"`
	ProjectPath     string                  `json:"projectPath"`
	ScannedAt       string                  `json:"scannedAt"`
	Languages       []string                `json:"languages"`
	PackageManagers []string                `json:"packageManagers"`
	Frameworks      []string                `json:"frameworks"`
	Dependencies    map[string][]Dependency `json:"dependencies"`
	Routes          []Route                 `json:"routes"`
	APIGroups       []APIGroup              `json:"apiGroups"`
	ConfigFiles     []string                `json:"configFiles"`
	Infrastructure  []string                `json:"infrastructureServices"`
	DetectedFiles   []string                `json:"detectedFiles"`
	ProjectStats    ProjectStats            `json:"projectStats"`
	ManifestFiles   []ManifestFile          `json:"manifestFiles"`
	DirectoryLayout []DirectorySummary      `json:"directoryLayout"`
	EntryPoints     []string                `json:"entryPoints"`
}
