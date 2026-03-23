package scanner

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/oleksandrskoruk/docusnap/internal/model"
)

type directoryAccumulator struct {
	FileCount     int
	SourceFiles   int
	TestFiles     int
	ManifestFiles int
	ConfigFiles   int
	languages     map[string]struct{}
	notableFiles  map[string]struct{}
}

type structureCollector struct {
	projectName string
	stats       model.ProjectStats
	manifests   map[string]string
	directories map[string]*directoryAccumulator
	entryPoints map[string]struct{}
}

func newStructureCollector(rootAbs string) *structureCollector {
	return &structureCollector{
		projectName: filepath.Base(rootAbs),
		manifests:   map[string]string{},
		directories: map[string]*directoryAccumulator{},
		entryPoints: map[string]struct{}{},
	}
}

func (c *structureCollector) recordFile(rootAbs, path, manifestKind string, isConfig bool) {
	rel, err := filepath.Rel(rootAbs, path)
	if err != nil {
		return
	}
	rel = filepath.ToSlash(rel)
	if rel == "." || rel == "" {
		return
	}

	c.stats.TotalFiles++

	if manifestKind != "" {
		c.manifests[rel] = manifestKind
	}
	if isConfig {
		c.stats.ConfigFiles++
	}

	sourceLanguage := detectSourceLanguage(rel)
	isTest := isTestSourceFile(rel)
	switch {
	case isTest:
		c.stats.TestFiles++
	case sourceLanguage != "":
		c.stats.SourceFiles++
	}

	if isLikelyEntryPoint(rel) {
		c.entryPoints[rel] = struct{}{}
	}

	key := directorySummaryKey(rel)
	acc := c.directories[key]
	if acc == nil {
		acc = &directoryAccumulator{
			languages:    map[string]struct{}{},
			notableFiles: map[string]struct{}{},
		}
		c.directories[key] = acc
	}

	acc.FileCount++
	if manifestKind != "" {
		acc.ManifestFiles++
	}
	if isConfig {
		acc.ConfigFiles++
	}
	switch {
	case isTest:
		acc.TestFiles++
	case sourceLanguage != "":
		acc.SourceFiles++
		acc.languages[sourceLanguage] = struct{}{}
	}
	if shouldHighlightFile(rel, manifestKind) {
		acc.notableFiles[rel] = struct{}{}
	}
}

func (c *structureCollector) projectStats() model.ProjectStats {
	stats := c.stats
	stats.ManifestFiles = len(c.manifests)
	return stats
}

func (c *structureCollector) manifestFiles() []model.ManifestFile {
	files := make([]model.ManifestFile, 0, len(c.manifests))
	for path, kind := range c.manifests {
		files = append(files, model.ManifestFile{Path: path, Kind: kind})
	}
	sort.Slice(files, func(i, j int) bool {
		if files[i].Kind == files[j].Kind {
			return files[i].Path < files[j].Path
		}
		return files[i].Kind < files[j].Kind
	})
	return files
}

func (c *structureCollector) directoryLayout() []model.DirectorySummary {
	keys := make([]string, 0, len(c.directories))
	for key := range c.directories {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i] == "root" {
			return true
		}
		if keys[j] == "root" {
			return false
		}
		return keys[i] < keys[j]
	})

	layout := make([]model.DirectorySummary, 0, len(keys))
	for _, key := range keys {
		acc := c.directories[key]
		layout = append(layout, model.DirectorySummary{
			Path:          key,
			FileCount:     acc.FileCount,
			SourceFiles:   acc.SourceFiles,
			TestFiles:     acc.TestFiles,
			ManifestFiles: acc.ManifestFiles,
			ConfigFiles:   acc.ConfigFiles,
			Languages:     sortedSet(acc.languages),
			NotableFiles:  cappedSortedKeys(acc.notableFiles, 5),
		})
	}
	return layout
}

func (c *structureCollector) entryPointList() []string {
	return cappedSortedKeys(c.entryPoints, 12)
}

func detectSourceLanguage(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go":
		return "go"
	case ".js", ".jsx", ".mjs", ".cjs":
		return "javascript"
	case ".ts", ".tsx":
		return "typescript"
	case ".py":
		return "python"
	case ".php":
		return "php"
	case ".java":
		return "java"
	case ".cs":
		return "csharp"
	case ".rs":
		return "rust"
	default:
		return ""
	}
}

func isTestSourceFile(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	switch {
	case strings.HasSuffix(base, "_test.go"):
		return true
	case strings.HasSuffix(base, ".test.js"),
		strings.HasSuffix(base, ".spec.js"),
		strings.HasSuffix(base, ".test.mjs"),
		strings.HasSuffix(base, ".spec.mjs"),
		strings.HasSuffix(base, ".test.cjs"),
		strings.HasSuffix(base, ".spec.cjs"),
		strings.HasSuffix(base, ".test.ts"),
		strings.HasSuffix(base, ".spec.ts"),
		strings.HasSuffix(base, ".test.tsx"),
		strings.HasSuffix(base, ".spec.tsx"):
		return true
	case strings.HasPrefix(base, "test_") && strings.HasSuffix(base, ".py"):
		return true
	case strings.HasSuffix(base, "_test.py"):
		return true
	case strings.HasSuffix(base, "test.php"):
		return true
	case strings.HasSuffix(base, "test.java"), strings.HasSuffix(base, "tests.java"):
		return true
	case strings.Contains(filepath.ToSlash(path), "/src/test/"):
		return true
	case strings.Contains(filepath.ToSlash(path), "/tests/"):
		return true
	default:
		return false
	}
}

func isLikelyEntryPoint(path string) bool {
	slashPath := filepath.ToSlash(path)
	base := filepath.Base(slashPath)

	switch base {
	case "main.go", "main.js", "main.ts", "index.js", "index.ts", "server.js", "server.ts", "app.js", "app.ts", "app.py", "server.py", "manage.py", "artisan", "Dockerfile", "Main.java":
		return true
	case "api.php", "web.php":
		return strings.HasPrefix(slashPath, "routes/")
	}

	if strings.HasSuffix(base, "Application.java") {
		return true
	}
	if strings.HasPrefix(slashPath, "cmd/") && strings.HasSuffix(slashPath, "/main.go") {
		return true
	}
	if strings.HasSuffix(slashPath, "/main.go") {
		return true
	}
	return false
}

func shouldHighlightFile(path, manifestKind string) bool {
	if manifestKind != "" || isLikelyEntryPoint(path) {
		return true
	}

	base := filepath.Base(path)
	switch base {
	case "README.md", "Makefile":
		return true
	default:
		return false
	}
}

func directorySummaryKey(path string) string {
	dir := filepath.ToSlash(filepath.Dir(path))
	if dir == "." || dir == "" {
		return "root"
	}

	parts := strings.Split(dir, "/")
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" || part == "." {
			continue
		}
		filtered = append(filtered, part)
	}
	if len(filtered) == 0 {
		return "root"
	}
	if len(filtered) > 2 {
		filtered = filtered[:2]
	}
	return strings.Join(filtered, "/")
}

func cappedSortedKeys(set map[string]struct{}, limit int) []string {
	out := make([]string, 0, len(set))
	for item := range set {
		out = append(out, item)
	}
	sort.Strings(out)
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

func buildAPIGroups(routes []model.Route) []model.APIGroup {
	if len(routes) == 0 {
		return nil
	}

	type accumulator struct {
		count   int
		methods map[string]struct{}
	}

	groups := map[string]*accumulator{}
	for _, route := range routes {
		prefix := routeGroupPrefix(route.Path)
		group := groups[prefix]
		if group == nil {
			group = &accumulator{methods: map[string]struct{}{}}
			groups[prefix] = group
		}
		group.count++

		method := strings.ToUpper(strings.TrimSpace(route.Method))
		if method == "" {
			method = "UNKNOWN"
		}
		group.methods[method] = struct{}{}
	}

	out := make([]model.APIGroup, 0, len(groups))
	for prefix, group := range groups {
		out = append(out, model.APIGroup{
			Prefix:     prefix,
			RouteCount: group.count,
			Methods:    sortedSet(group.methods),
		})
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].RouteCount == out[j].RouteCount {
			return out[i].Prefix < out[j].Prefix
		}
		return out[i].RouteCount > out[j].RouteCount
	})
	return out
}

func routeGroupPrefix(path string) string {
	normalized := normalizeRoutePath(path)
	if normalized == "/" {
		return normalized
	}

	trimmed := strings.Trim(normalized, "/")
	if trimmed == "" {
		return "/"
	}

	parts := strings.Split(trimmed, "/")
	return "/" + parts[0]
}

func manifestKindForFile(path, base string) string {
	switch {
	case isDotNetProjectFile(base):
		return "dependency"
	case containsExactString(techMarkers, base):
		if strings.HasPrefix(base, "openapi.") || strings.HasPrefix(base, "swagger.") {
			return "api"
		}
		if strings.HasPrefix(base, "next.config.") {
			return "configuration"
		}
		return "dependency"
	case base == "docker-compose.yml", base == "docker-compose.yaml":
		return "infrastructure"
	case base == "Dockerfile":
		return "runtime"
	case base == ".env" || isEnvConfigFile(base):
		return "runtime"
	case isTerraformConfigFile(base) || isKubernetesConfigCandidate(path, base):
		return "infrastructure"
	case containsExactString(configMarkers, base):
		return "configuration"
	default:
		return ""
	}
}

func containsExactString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
