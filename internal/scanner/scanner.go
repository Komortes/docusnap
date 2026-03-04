package scanner

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/oleksandrskoruk/docusnap/internal/model"
)

var techMarkers = []string{
	"composer.json",
	"package.json",
	"go.mod",
	"Cargo.toml",
}

var configMarkers = []string{
	"docker-compose.yml",
	"docker-compose.yaml",
	"Dockerfile",
	".env",
}

type packageJSON struct {
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

type composerJSON struct {
	Require    map[string]string `json:"require"`
	RequireDev map[string]string `json:"require-dev"`
}

func Scan(root string) (model.Snapshot, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return model.Snapshot{}, err
	}

	techMarkerLookup := make(map[string]struct{}, len(techMarkers))
	for _, marker := range techMarkers {
		techMarkerLookup[marker] = struct{}{}
	}
	configMarkerLookup := make(map[string]struct{}, len(configMarkers))
	for _, marker := range configMarkers {
		configMarkerLookup[marker] = struct{}{}
	}
	foundTechFiles := map[string]struct{}{}
	foundConfigFiles := map[string]struct{}{}

	dependencies := map[string][]model.Dependency{}
	frameworkSet := map[string]struct{}{}
	routes := make([]model.Route, 0)
	foundLaravelRoutes := false
	foundExpressRoutes := false

	err = filepath.WalkDir(rootAbs, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if d.IsDir() {
			base := d.Name()
			if base == ".git" || base == "node_modules" || base == "vendor" || strings.HasPrefix(base, ".") {
				if path != rootAbs {
					return filepath.SkipDir
				}
			}
			return nil
		}

		base := filepath.Base(path)
		if _, ok := techMarkerLookup[base]; ok {
			rel, relErr := filepath.Rel(rootAbs, path)
			if relErr != nil {
				rel = base
			}
			foundTechFiles[rel] = struct{}{}

			switch base {
			case "package.json":
				deps, err := parsePackageJSON(path)
				if err == nil && len(deps) > 0 {
					dependencies["npm"] = append(dependencies["npm"], deps...)
				}
			case "composer.json":
				deps, err := parseComposerJSON(path)
				if err == nil && len(deps) > 0 {
					dependencies["composer"] = append(dependencies["composer"], deps...)
				}
			case "go.mod":
				deps, err := parseGoMod(path)
				if err == nil && len(deps) > 0 {
					dependencies["go"] = append(dependencies["go"], deps...)
				}
			case "Cargo.toml":
				deps, err := parseCargoToml(path)
				if err == nil && len(deps) > 0 {
					dependencies["cargo"] = append(dependencies["cargo"], deps...)
				}
			}
		}

		if _, ok := configMarkerLookup[base]; ok {
			rel, relErr := filepath.Rel(rootAbs, path)
			if relErr != nil {
				rel = base
			}
			foundConfigFiles[rel] = struct{}{}
		}

		if isLaravelRoutesFile(path, base) {
			parsedRoutes, err := parseLaravelRoutes(path)
			if err == nil && len(parsedRoutes) > 0 {
				routes = append(routes, parsedRoutes...)
				foundLaravelRoutes = true
			}
		}
		if isExpressRoutesFile(path) {
			parsedRoutes, err := parseExpressRoutes(path)
			if err == nil && len(parsedRoutes) > 0 {
				routes = append(routes, parsedRoutes...)
				foundExpressRoutes = true
			}
		}

		return nil
	})
	if err != nil {
		return model.Snapshot{}, err
	}

	for manager := range dependencies {
		sortDependencies(dependencies[manager])
		detectFrameworksFromDependencies(manager, dependencies[manager], frameworkSet)
	}

	detectedFiles := collectDetectedFiles(foundTechFiles)
	configs := collectDetectedFiles(foundConfigFiles)
	languages, managers := detectLanguagesAndManagers(detectedFiles)
	frameworks := sortedSet(frameworkSet)
	infrastructure := detectInfrastructureServices(configs)
	routes = deduplicateRoutes(routes)
	if foundLaravelRoutes {
		frameworkSet["laravel"] = struct{}{}
	}
	if foundExpressRoutes {
		frameworkSet["express"] = struct{}{}
	}
	if foundLaravelRoutes || foundExpressRoutes {
		frameworks = sortedSet(frameworkSet)
	}

	return model.Snapshot{
		ProjectPath:     rootAbs,
		ScannedAt:       time.Now().UTC().Format(time.RFC3339),
		Languages:       languages,
		PackageManagers: managers,
		Frameworks:      frameworks,
		Dependencies:    dependencies,
		Routes:          routes,
		ConfigFiles:     configs,
		Infrastructure:  infrastructure,
		DetectedFiles:   detectedFiles,
	}, nil
}

func detectLanguagesAndManagers(found []string) ([]string, []string) {
	languageSet := map[string]struct{}{}
	managerSet := map[string]struct{}{}

	for _, file := range found {
		if strings.HasSuffix(file, "composer.json") {
			languageSet["php"] = struct{}{}
			managerSet["composer"] = struct{}{}
		}
		if strings.HasSuffix(file, "package.json") {
			languageSet["javascript"] = struct{}{}
			managerSet["npm"] = struct{}{}
		}
		if strings.HasSuffix(file, "go.mod") {
			languageSet["go"] = struct{}{}
			managerSet["go"] = struct{}{}
		}
		if strings.HasSuffix(file, "Cargo.toml") {
			languageSet["rust"] = struct{}{}
			managerSet["cargo"] = struct{}{}
		}
	}

	return sortedSet(languageSet), sortedSet(managerSet)
}

func collectDetectedFiles(found map[string]struct{}) []string {
	files := make([]string, 0, len(found))
	for file := range found {
		files = append(files, file)
	}
	sort.Strings(files)
	return files
}

func sortedSet(items map[string]struct{}) []string {
	out := make([]string, 0, len(items))
	for item := range items {
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}

func parsePackageJSON(path string) ([]model.Dependency, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var doc packageJSON
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}

	deps := make([]model.Dependency, 0, len(doc.Dependencies)+len(doc.DevDependencies))
	for name, version := range doc.Dependencies {
		deps = append(deps, model.Dependency{Name: name, Version: version})
	}
	for name, version := range doc.DevDependencies {
		deps = append(deps, model.Dependency{Name: name, Version: version})
	}

	return uniqueDependencies(deps), nil
}

func parseComposerJSON(path string) ([]model.Dependency, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var doc composerJSON
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}

	deps := make([]model.Dependency, 0, len(doc.Require)+len(doc.RequireDev))
	for name, version := range doc.Require {
		deps = append(deps, model.Dependency{Name: name, Version: version})
	}
	for name, version := range doc.RequireDev {
		deps = append(deps, model.Dependency{Name: name, Version: version})
	}

	return uniqueDependencies(deps), nil
}

func parseGoMod(path string) ([]model.Dependency, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	inRequireBlock := false
	deps := make([]model.Dependency, 0)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		if strings.HasPrefix(line, "require (") {
			inRequireBlock = true
			continue
		}

		if inRequireBlock && line == ")" {
			inRequireBlock = false
			continue
		}

		if strings.HasPrefix(line, "require ") {
			parts := strings.Fields(strings.TrimPrefix(line, "require "))
			if len(parts) >= 2 {
				deps = append(deps, model.Dependency{Name: parts[0], Version: parts[1]})
			}
			continue
		}

		if inRequireBlock {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				deps = append(deps, model.Dependency{Name: parts[0], Version: parts[1]})
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return uniqueDependencies(deps), nil
}

func parseCargoToml(path string) ([]model.Dependency, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	inDependencies := false
	deps := make([]model.Dependency, 0)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			inDependencies = line == "[dependencies]"
			continue
		}

		if !inDependencies {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		name := strings.TrimSpace(parts[0])
		version := strings.TrimSpace(parts[1])
		version = strings.Trim(version, "\"")
		if name != "" {
			deps = append(deps, model.Dependency{Name: name, Version: version})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return uniqueDependencies(deps), nil
}

func uniqueDependencies(items []model.Dependency) []model.Dependency {
	seen := make(map[string]string, len(items))
	for _, item := range items {
		if item.Name == "" {
			continue
		}
		seen[item.Name] = item.Version
	}

	out := make([]model.Dependency, 0, len(seen))
	for name, version := range seen {
		out = append(out, model.Dependency{Name: name, Version: version})
	}
	sortDependencies(out)
	return out
}

func sortDependencies(items []model.Dependency) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].Name == items[j].Name {
			return items[i].Version < items[j].Version
		}
		return items[i].Name < items[j].Name
	})
}

func detectFrameworksFromDependencies(manager string, deps []model.Dependency, set map[string]struct{}) {
	for _, dep := range deps {
		n := strings.ToLower(dep.Name)
		switch {
		case manager == "composer" && n == "laravel/framework":
			set["laravel"] = struct{}{}
		case manager == "npm" && n == "react":
			set["react"] = struct{}{}
		case manager == "npm" && n == "express":
			set["express"] = struct{}{}
		case manager == "npm" && n == "next":
			set["next.js"] = struct{}{}
		case manager == "npm" && (n == "@angular/core" || n == "angular"):
			set["angular"] = struct{}{}
		case manager == "go" && n == "github.com/gin-gonic/gin":
			set["gin"] = struct{}{}
		case manager == "go" && n == "github.com/labstack/echo/v4":
			set["echo"] = struct{}{}
		}
	}
}

func detectInfrastructureServices(configs []string) []string {
	set := map[string]struct{}{}
	for _, cfg := range configs {
		lower := strings.ToLower(cfg)
		if strings.Contains(lower, "docker-compose") || strings.Contains(lower, "dockerfile") {
			set["docker"] = struct{}{}
		}
		if strings.Contains(lower, ".env") {
			set["env-file"] = struct{}{}
		}
	}
	return sortedSet(set)
}

func isLaravelRoutesFile(path, base string) bool {
	if base != "web.php" && base != "api.php" {
		return false
	}
	normalized := filepath.ToSlash(path)
	return strings.Contains(normalized, "/routes/")
}

var (
	laravelClassHandlerPattern  = regexp.MustCompile(`Route::(?i)(get|post|put|patch|delete|options|any)\s*\(\s*['"]([^'"]+)['"]\s*,\s*\[\s*([A-Za-z0-9_\\]+)::class\s*,\s*['"]([A-Za-z0-9_]+)['"]\s*\]`)
	laravelStringHandlerPattern = regexp.MustCompile(`Route::(?i)(get|post|put|patch|delete|options|any)\s*\(\s*['"]([^'"]+)['"]\s*,\s*['"]([^'"]+)['"]`)
	expressRoutePattern         = regexp.MustCompile(`\b(?:app|router)\.(get|post|put|patch|delete|options|head|all)\s*\(\s*['"\x60]([^'"\x60]+)['"\x60](?:\s*,\s*([A-Za-z0-9_$.]+))?`)
	expressRouteChainPattern    = regexp.MustCompile(`\b(?:app|router)\.route\s*\(\s*['"\x60]([^'"\x60]+)['"\x60]\s*\)\.(get|post|put|patch|delete|options|head|all)\s*\(`)
)

func parseLaravelRoutes(path string) ([]model.Route, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	routes := make([]model.Route, 0)
	s := bufio.NewScanner(file)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "#") {
			continue
		}
		if !strings.Contains(line, "Route::") {
			continue
		}

		if m := laravelClassHandlerPattern.FindStringSubmatch(line); len(m) == 5 {
			routes = append(routes, model.Route{
				Method:     strings.ToUpper(m[1]),
				Path:       normalizeRoutePath(m[2]),
				Controller: shortLaravelController(m[3]) + "@" + m[4],
			})
			continue
		}
		if m := laravelStringHandlerPattern.FindStringSubmatch(line); len(m) == 4 {
			routes = append(routes, model.Route{
				Method:     strings.ToUpper(m[1]),
				Path:       normalizeRoutePath(m[2]),
				Controller: m[3],
			})
		}
	}
	if err := s.Err(); err != nil {
		return nil, err
	}

	return routes, nil
}

func isExpressRoutesFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".js", ".mjs", ".cjs", ".ts":
	default:
		return false
	}

	base := strings.ToLower(filepath.Base(path))
	if strings.Contains(base, ".min.") {
		return false
	}
	return true
}

func parseExpressRoutes(path string) ([]model.Route, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	routes := make([]model.Route, 0)
	s := bufio.NewScanner(file)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		if m := expressRoutePattern.FindStringSubmatch(line); len(m) >= 3 {
			controller := "handler"
			if len(m) >= 4 && m[3] != "" {
				controller = m[3]
			}
			routes = append(routes, model.Route{
				Method:     strings.ToUpper(m[1]),
				Path:       normalizeRoutePath(m[2]),
				Controller: controller,
			})
		}
		if m := expressRouteChainPattern.FindStringSubmatch(line); len(m) == 3 {
			routes = append(routes, model.Route{
				Method:     strings.ToUpper(m[2]),
				Path:       normalizeRoutePath(m[1]),
				Controller: "handler",
			})
		}
	}
	if err := s.Err(); err != nil {
		return nil, err
	}

	return routes, nil
}

func normalizeRoutePath(path string) string {
	if path == "" {
		return "/"
	}
	if strings.HasPrefix(path, "/") {
		return path
	}
	return "/" + path
}

func shortLaravelController(v string) string {
	parts := strings.Split(v, `\`)
	return parts[len(parts)-1]
}

func deduplicateRoutes(items []model.Route) []model.Route {
	set := map[string]model.Route{}
	for _, item := range items {
		key := item.Method + "|" + item.Path + "|" + item.Controller
		set[key] = item
	}

	out := make([]model.Route, 0, len(set))
	for _, route := range set {
		out = append(out, route)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Method == out[j].Method {
			return out[i].Path < out[j].Path
		}
		return out[i].Method < out[j].Method
	})
	return out
}
