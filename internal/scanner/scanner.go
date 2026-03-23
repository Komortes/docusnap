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
	"requirements.txt",
	"pyproject.toml",
	"pom.xml",
	"build.gradle",
	"build.gradle.kts",
	"openapi.yaml",
	"openapi.yml",
	"openapi.json",
	"swagger.yaml",
	"swagger.yml",
	"swagger.json",
	"next.config.js",
	"next.config.mjs",
	"next.config.ts",
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

	structure := newStructureCollector(rootAbs)

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
	foundGinRoutes := false
	foundEchoRoutes := false
	foundFastAPIRoutes := false
	foundFlaskRoutes := false
	foundDjangoRoutes := false
	foundNextJSRoutes := false
	foundOpenAPIRoutes := false
	foundAspNetRoutes := false
	foundSpringRoutes := false

	err = filepath.WalkDir(rootAbs, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if d.IsDir() {
			base := d.Name()
			if base == ".git" || base == "node_modules" || base == "vendor" || base == "target" || base == "build" || base == "dist" || base == "out" || strings.HasPrefix(base, ".") {
				if path != rootAbs {
					return filepath.SkipDir
				}
			}
			return nil
		}

		base := filepath.Base(path)
		rel, relErr := filepath.Rel(rootAbs, path)
		if relErr != nil {
			rel = base
		}
		rel = filepath.ToSlash(rel)

		manifestKind := ""
		isConfigFile := false
		if _, ok := techMarkerLookup[base]; ok || isDotNetProjectFile(base) {
			foundTechFiles[rel] = struct{}{}
			manifestKind = manifestKindForFile(path, base)

			switch base {
			case "package.json":
				deps, err := parsePackageJSON(path)
				if err == nil && len(deps) > 0 {
					dependencies["npm"] = append(dependencies["npm"], deps...)
				}
			case "pom.xml":
				deps, err := parsePomXML(path)
				if err == nil && len(deps) > 0 {
					dependencies["maven"] = append(dependencies["maven"], deps...)
				}
			case "build.gradle", "build.gradle.kts":
				deps, err := parseGradleFile(path)
				if err == nil && len(deps) > 0 {
					dependencies["gradle"] = append(dependencies["gradle"], deps...)
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
			case "requirements.txt":
				deps, err := parseRequirementsTxt(path)
				if err == nil && len(deps) > 0 {
					dependencies["pip"] = append(dependencies["pip"], deps...)
				}
			case "pyproject.toml":
				projectDeps, poetryDeps, err := parsePyProjectToml(path)
				if err == nil {
					if len(projectDeps) > 0 {
						dependencies["pip"] = append(dependencies["pip"], projectDeps...)
					}
					if len(poetryDeps) > 0 {
						dependencies["poetry"] = append(dependencies["poetry"], poetryDeps...)
					}
				}
			}
			if isDotNetProjectFile(base) {
				deps, err := parseCsproj(path)
				if err == nil && len(deps) > 0 {
					dependencies["nuget"] = append(dependencies["nuget"], deps...)
				}
			}
		}

		if _, ok := configMarkerLookup[base]; ok {
			foundConfigFiles[rel] = struct{}{}
			isConfigFile = true
			if manifestKind == "" {
				manifestKind = manifestKindForFile(path, base)
			}
		}
		if isEnvConfigFile(base) || isTerraformConfigFile(base) || isKubernetesConfigCandidate(path, base) {
			foundConfigFiles[rel] = struct{}{}
			isConfigFile = true
			if manifestKind == "" {
				manifestKind = manifestKindForFile(path, base)
			}
		}

		structure.recordFile(rootAbs, path, manifestKind, isConfigFile)

		if isLaravelRoutesFile(path, base) {
			parsedRoutes, err := parseLaravelRoutes(path)
			if err == nil && len(parsedRoutes) > 0 {
				routes = append(routes, parsedRoutes...)
				foundLaravelRoutes = true
			}
		}
		if isOpenAPIFile(path, base) {
			parsedRoutes, usedOpenAPI, err := parseOpenAPIRoutes(path)
			if err == nil {
				if len(parsedRoutes) > 0 {
					routes = append(routes, parsedRoutes...)
				}
				foundOpenAPIRoutes = foundOpenAPIRoutes || usedOpenAPI
			}
		}
		if isExpressRoutesFile(path) {
			parsedRoutes, err := parseExpressRoutes(path)
			if err == nil && len(parsedRoutes) > 0 {
				routes = append(routes, parsedRoutes...)
				foundExpressRoutes = true
			}
		}
		if isNextJSAPIFile(path) {
			parsedRoutes, usedNextJS, err := parseNextJSAPIRoutes(path)
			if err == nil {
				if len(parsedRoutes) > 0 {
					routes = append(routes, parsedRoutes...)
				}
				foundNextJSRoutes = foundNextJSRoutes || usedNextJS
			}
		}
		if isGoRoutesFile(path) {
			parsedRoutes, usedGin, usedEcho, err := parseGoRoutes(path)
			if err == nil && len(parsedRoutes) > 0 {
				routes = append(routes, parsedRoutes...)
				foundGinRoutes = foundGinRoutes || usedGin
				foundEchoRoutes = foundEchoRoutes || usedEcho
			}
		}
		if isPythonRoutesFile(path) {
			parsedRoutes, usedFastAPI, err := parseFastAPIRoutes(path)
			if err == nil && len(parsedRoutes) > 0 {
				routes = append(routes, parsedRoutes...)
				foundFastAPIRoutes = foundFastAPIRoutes || usedFastAPI
			}
			flaskRoutes, usedFlask, err := parseFlaskRoutes(path)
			if err == nil && len(flaskRoutes) > 0 {
				routes = append(routes, flaskRoutes...)
				foundFlaskRoutes = foundFlaskRoutes || usedFlask
			}
			if isDjangoURLsFile(path) {
				djangoRoutes, usedDjango, err := parseDjangoRoutes(path)
				if err == nil && len(djangoRoutes) > 0 {
					routes = append(routes, djangoRoutes...)
					foundDjangoRoutes = foundDjangoRoutes || usedDjango
				}
			}
		}
		if isDotNetRoutesFile(path) {
			parsedRoutes, usedAspNet, err := parseDotNetRoutes(path)
			if err == nil {
				if len(parsedRoutes) > 0 {
					routes = append(routes, parsedRoutes...)
				}
				foundAspNetRoutes = foundAspNetRoutes || usedAspNet
			}
		}
		if isJavaRoutesFile(path) {
			parsedRoutes, usedSpring, err := parseSpringRoutes(path)
			if err == nil {
				if len(parsedRoutes) > 0 {
					routes = append(routes, parsedRoutes...)
				}
				foundSpringRoutes = foundSpringRoutes || usedSpring
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
	managers = mergeSortedUnique(managers, dependencyManagerKeys(dependencies))
	frameworks := sortedSet(frameworkSet)
	infrastructure := detectInfrastructureServices(rootAbs, configs)
	routes = deduplicateRoutes(routes)
	apiGroups := buildAPIGroups(routes)
	if foundLaravelRoutes {
		frameworkSet["laravel"] = struct{}{}
	}
	if foundExpressRoutes {
		frameworkSet["express"] = struct{}{}
	}
	if foundGinRoutes {
		frameworkSet["gin"] = struct{}{}
	}
	if foundEchoRoutes {
		frameworkSet["echo"] = struct{}{}
	}
	if foundFastAPIRoutes {
		frameworkSet["fastapi"] = struct{}{}
	}
	if foundFlaskRoutes {
		frameworkSet["flask"] = struct{}{}
	}
	if foundDjangoRoutes {
		frameworkSet["django"] = struct{}{}
	}
	if foundNextJSRoutes {
		frameworkSet["next.js"] = struct{}{}
	}
	if foundOpenAPIRoutes {
		frameworkSet["openapi"] = struct{}{}
	}
	if foundAspNetRoutes {
		frameworkSet["asp.net"] = struct{}{}
	}
	if foundSpringRoutes {
		frameworkSet["spring"] = struct{}{}
	}
	if foundLaravelRoutes || foundExpressRoutes || foundGinRoutes || foundEchoRoutes || foundFastAPIRoutes || foundFlaskRoutes || foundDjangoRoutes || foundNextJSRoutes || foundOpenAPIRoutes || foundAspNetRoutes || foundSpringRoutes {
		frameworks = sortedSet(frameworkSet)
	}

	return model.Snapshot{
		ProjectName:     structure.projectName,
		ProjectPath:     rootAbs,
		ScannedAt:       time.Now().UTC().Format(time.RFC3339),
		Languages:       languages,
		PackageManagers: managers,
		Frameworks:      frameworks,
		Dependencies:    dependencies,
		Routes:          routes,
		APIGroups:       apiGroups,
		ConfigFiles:     configs,
		Infrastructure:  infrastructure,
		DetectedFiles:   detectedFiles,
		ProjectStats:    structure.projectStats(),
		ManifestFiles:   structure.manifestFiles(),
		DirectoryLayout: structure.directoryLayout(),
		EntryPoints:     structure.entryPointList(),
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
		if strings.HasSuffix(strings.ToLower(file), "pom.xml") {
			languageSet["java"] = struct{}{}
			managerSet["maven"] = struct{}{}
		}
		if strings.HasSuffix(strings.ToLower(file), "build.gradle") || strings.HasSuffix(strings.ToLower(file), "build.gradle.kts") {
			languageSet["java"] = struct{}{}
			managerSet["gradle"] = struct{}{}
		}
		if strings.HasSuffix(strings.ToLower(file), ".csproj") {
			languageSet["csharp"] = struct{}{}
			managerSet["nuget"] = struct{}{}
		}
		if strings.HasSuffix(file, "Cargo.toml") {
			languageSet["rust"] = struct{}{}
			managerSet["cargo"] = struct{}{}
		}
		if strings.HasSuffix(strings.ToLower(file), "requirements.txt") {
			languageSet["python"] = struct{}{}
			managerSet["pip"] = struct{}{}
		}
		if strings.HasSuffix(strings.ToLower(file), "pyproject.toml") {
			languageSet["python"] = struct{}{}
			managerSet["pip"] = struct{}{}
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

func dependencyManagerKeys(dependencies map[string][]model.Dependency) []string {
	keys := make([]string, 0, len(dependencies))
	for key := range dependencies {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func mergeSortedUnique(a, b []string) []string {
	set := map[string]struct{}{}
	for _, item := range a {
		set[item] = struct{}{}
	}
	for _, item := range b {
		set[item] = struct{}{}
	}
	return sortedSet(set)
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

func parseRequirementsTxt(path string) ([]model.Dependency, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	deps := make([]model.Dependency, 0)
	s := bufio.NewScanner(file)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "-") {
			continue
		}

		if strings.Contains(line, "://") {
			lower := strings.ToLower(line)
			eggIndex := strings.Index(lower, "#egg=")
			if eggIndex > 0 && eggIndex+5 < len(line) {
				name := strings.TrimSpace(line[eggIndex+5:])
				if name != "" {
					deps = append(deps, model.Dependency{Name: strings.ToLower(name), Version: ""})
				}
			}
			continue
		}

		line = stripInlineComment(line)
		if line == "" {
			continue
		}

		name, version := splitPythonRequirement(line)
		if name == "" {
			continue
		}
		deps = append(deps, model.Dependency{Name: strings.ToLower(name), Version: version})
	}
	if err := s.Err(); err != nil {
		return nil, err
	}

	return uniqueDependencies(deps), nil
}

func parsePyProjectToml(path string) ([]model.Dependency, []model.Dependency, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	projectDeps := make([]model.Dependency, 0)
	poetryDeps := make([]model.Dependency, 0)

	currentSection := ""
	inProjectDepsArray := false

	s := bufio.NewScanner(file)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = stripInlineComment(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentSection = strings.Trim(line, "[]")
			inProjectDepsArray = false
			continue
		}

		if currentSection == "project" {
			if strings.HasPrefix(line, "dependencies") {
				inProjectDepsArray = strings.Contains(line, "[") && !strings.Contains(line, "]")
				for _, dep := range parseQuotedDependencies(line) {
					name, version := splitPythonRequirement(dep)
					if name != "" {
						projectDeps = append(projectDeps, model.Dependency{Name: strings.ToLower(name), Version: version})
					}
				}
				continue
			}
			if inProjectDepsArray {
				for _, dep := range parseQuotedDependencies(line) {
					name, version := splitPythonRequirement(dep)
					if name != "" {
						projectDeps = append(projectDeps, model.Dependency{Name: strings.ToLower(name), Version: version})
					}
				}
				if strings.Contains(line, "]") {
					inProjectDepsArray = false
				}
				continue
			}
		}

		if currentSection == "tool.poetry.dependencies" {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}
			name := strings.TrimSpace(parts[0])
			if strings.EqualFold(name, "python") || name == "" {
				continue
			}
			version := strings.TrimSpace(parts[1])
			version = strings.Trim(version, `"'`)
			poetryDeps = append(poetryDeps, model.Dependency{Name: strings.ToLower(name), Version: version})
		}
	}
	if err := s.Err(); err != nil {
		return nil, nil, err
	}

	return uniqueDependencies(projectDeps), uniqueDependencies(poetryDeps), nil
}

func splitPythonRequirement(line string) (string, string) {
	i := strings.IndexAny(line, "<>!=~[ ")
	if i == -1 {
		return strings.TrimSpace(line), ""
	}
	name := strings.TrimSpace(line[:i])
	version := strings.TrimSpace(line[i:])
	return name, version
}

func stripInlineComment(line string) string {
	inSingleQuoted := false
	inDoubleQuoted := false
	escaped := false

	for i, ch := range line {
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' {
			escaped = true
			continue
		}
		if ch == '\'' && !inDoubleQuoted {
			inSingleQuoted = !inSingleQuoted
			continue
		}
		if ch == '"' && !inSingleQuoted {
			inDoubleQuoted = !inDoubleQuoted
			continue
		}
		if ch == '#' && !inSingleQuoted && !inDoubleQuoted {
			return strings.TrimSpace(line[:i])
		}
	}
	return strings.TrimSpace(line)
}

func parseQuotedDependencies(line string) []string {
	out := make([]string, 0)
	for _, match := range quotedStringPattern.FindAllStringSubmatch(line, -1) {
		if len(match) < 3 {
			continue
		}
		if match[1] != "" {
			out = append(out, match[1])
		} else if match[2] != "" {
			out = append(out, match[2])
		}
	}
	return out
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
		case (manager == "maven" || manager == "gradle") && (strings.HasSuffix(n, ":spring-boot-starter-web") || strings.HasSuffix(n, ":spring-webmvc") || strings.HasSuffix(n, ":spring-web") || strings.HasSuffix(n, ":spring-boot-starter-webflux")):
			set["spring"] = struct{}{}
		case manager == "go" && n == "github.com/gin-gonic/gin":
			set["gin"] = struct{}{}
		case manager == "go" && n == "github.com/labstack/echo/v4":
			set["echo"] = struct{}{}
		case manager == "nuget" && strings.Contains(n, "aspnetcore"):
			set["asp.net"] = struct{}{}
		case (manager == "pip" || manager == "poetry") && n == "django":
			set["django"] = struct{}{}
		case (manager == "pip" || manager == "poetry") && n == "fastapi":
			set["fastapi"] = struct{}{}
		case (manager == "pip" || manager == "poetry") && n == "flask":
			set["flask"] = struct{}{}
		}
	}
}

func detectInfrastructureServices(rootAbs string, configs []string) []string {
	set := map[string]struct{}{}
	for _, cfg := range configs {
		lower := strings.ToLower(cfg)
		configPath := filepath.Join(rootAbs, cfg)

		if strings.Contains(lower, "docker-compose") || strings.Contains(lower, "dockerfile") {
			set["docker"] = struct{}{}
		}
		if strings.Contains(lower, "docker-compose") {
			services, err := parseDockerComposeServices(configPath)
			if err == nil {
				for _, service := range services {
					set[service] = struct{}{}
				}
			}
		}
		if isEnvConfigFile(filepath.Base(cfg)) {
			set["env-file"] = struct{}{}
			services, err := parseEnvServices(configPath)
			if err == nil {
				for _, service := range services {
					set[service] = struct{}{}
				}
			}
		}

		if isTerraformConfigFile(filepath.Base(cfg)) {
			services, err := parseTerraformServices(configPath)
			if err == nil {
				for _, service := range services {
					set[service] = struct{}{}
				}
			}
		}

		if isKubernetesConfigCandidate(configPath, filepath.Base(cfg)) {
			services, err := parseKubernetesManifestServices(configPath)
			if err == nil {
				for _, service := range services {
					set[service] = struct{}{}
				}
			}
		}
	}
	return sortedSet(set)
}

func isEnvConfigFile(base string) bool {
	lower := strings.ToLower(base)
	return lower == ".env" || strings.HasPrefix(lower, ".env.")
}

func isTerraformConfigFile(base string) bool {
	return strings.HasSuffix(strings.ToLower(base), ".tf")
}

func isKubernetesConfigCandidate(path, base string) bool {
	ext := strings.ToLower(filepath.Ext(base))
	if ext != ".yaml" && ext != ".yml" {
		return false
	}

	lowerPath := strings.ToLower(filepath.ToSlash(path))
	lowerBase := strings.ToLower(base)
	if strings.Contains(lowerPath, "/k8s/") ||
		strings.Contains(lowerPath, "/kubernetes/") ||
		strings.Contains(lowerPath, "/helm/") ||
		strings.Contains(lowerBase, "deployment") ||
		strings.Contains(lowerBase, "statefulset") ||
		strings.Contains(lowerBase, "daemonset") ||
		strings.Contains(lowerBase, "service") ||
		strings.Contains(lowerBase, "ingress") ||
		strings.Contains(lowerBase, "kustomization") ||
		strings.Contains(lowerBase, "values") ||
		strings.Contains(lowerBase, "chart") {
		return true
	}
	return false
}

func parseDockerComposeServices(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	found := map[string]struct{}{}
	inServices := false
	servicesIndent := -1
	currentServiceIndent := -1

	s := bufio.NewScanner(file)
	for s.Scan() {
		raw := s.Text()
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		indent := leadingIndent(raw)
		if !inServices {
			if trimmed == "services:" {
				inServices = true
				servicesIndent = indent
			}
			continue
		}

		if indent <= servicesIndent && !strings.HasPrefix(trimmed, "-") {
			inServices = false
			continue
		}

		if isYAMLKey(trimmed) && indent == servicesIndent+2 {
			serviceName := strings.TrimSuffix(trimmed, ":")
			currentServiceIndent = indent
			if svc := detectInfraService(serviceName); svc != "" {
				found[svc] = struct{}{}
			}
			continue
		}

		if currentServiceIndent != -1 && indent <= currentServiceIndent {
			currentServiceIndent = -1
		}
		if currentServiceIndent == -1 {
			continue
		}

		if strings.HasPrefix(trimmed, "image:") {
			image := strings.TrimSpace(strings.TrimPrefix(trimmed, "image:"))
			image = strings.Trim(image, `"'`)
			if svc := detectInfraService(image); svc != "" {
				found[svc] = struct{}{}
			}
		}
	}
	if err := s.Err(); err != nil {
		return nil, err
	}

	return sortedSet(found), nil
}

func parseEnvServices(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	found := map[string]struct{}{}
	s := bufio.NewScanner(file)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		value = strings.Trim(value, `"'`)
		if svc := detectInfraService(key); svc != "" {
			found[svc] = struct{}{}
		}
		if svc := detectInfraService(value); svc != "" {
			found[svc] = struct{}{}
		}
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	return sortedSet(found), nil
}

func parseTerraformServices(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	found := map[string]struct{}{}
	s := bufio.NewScanner(file)
	for s.Scan() {
		line := strings.ToLower(strings.TrimSpace(s.Text()))
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}

		if svc := detectInfraService(line); svc != "" {
			found[svc] = struct{}{}
		}
		if strings.Contains(line, "terraform {") || strings.Contains(line, `provider "`) {
			found["terraform"] = struct{}{}
		}
		if strings.Contains(line, "aws_eks_cluster") ||
			strings.Contains(line, "google_container_cluster") ||
			strings.Contains(line, "azurerm_kubernetes_cluster") ||
			strings.Contains(line, "kubernetes_") ||
			strings.Contains(line, "helm_release") {
			found["kubernetes"] = struct{}{}
		}
		if strings.Contains(line, "aws_elasticache") {
			found["redis"] = struct{}{}
		}
		if strings.Contains(line, "aws_mq_broker") {
			found["rabbitmq"] = struct{}{}
		}
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	return sortedSet(found), nil
}

func parseKubernetesManifestServices(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	found := map[string]struct{}{}
	hasAPIVersion := false
	hasKind := false

	s := bufio.NewScanner(file)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		lower := strings.ToLower(line)
		if strings.HasPrefix(lower, "apiversion:") {
			hasAPIVersion = true
		}
		if strings.HasPrefix(lower, "kind:") {
			hasKind = true
		}
		if strings.HasPrefix(lower, "image:") {
			image := strings.TrimSpace(strings.TrimPrefix(line, "image:"))
			image = strings.Trim(image, `"'`)
			if svc := detectInfraService(image); svc != "" {
				found[svc] = struct{}{}
			}
		}
		if strings.HasPrefix(lower, "name:") {
			name := strings.TrimSpace(strings.TrimPrefix(lower, "name:"))
			if svc := detectInfraService(name); svc != "" {
				found[svc] = struct{}{}
			}
		}
	}
	if err := s.Err(); err != nil {
		return nil, err
	}

	if hasAPIVersion && hasKind {
		found["kubernetes"] = struct{}{}
	}
	return sortedSet(found), nil
}

func leadingIndent(line string) int {
	n := 0
	for _, ch := range line {
		if ch == ' ' {
			n++
			continue
		}
		if ch == '\t' {
			n += 2
			continue
		}
		break
	}
	return n
}

func isYAMLKey(trimmed string) bool {
	return strings.HasSuffix(trimmed, ":") && !strings.Contains(trimmed, " ")
}

func detectInfraService(value string) string {
	v := strings.ToLower(value)
	switch {
	case strings.Contains(v, "postgres"):
		return "postgres"
	case strings.Contains(v, "mysql"):
		return "mysql"
	case strings.Contains(v, "mariadb"):
		return "mariadb"
	case strings.Contains(v, "redis"):
		return "redis"
	case strings.Contains(v, "mongodb") || strings.Contains(v, "mongo"):
		return "mongodb"
	case strings.Contains(v, "rabbitmq"):
		return "rabbitmq"
	case strings.Contains(v, "kafka"):
		return "kafka"
	case strings.Contains(v, "zookeeper"):
		return "zookeeper"
	case strings.Contains(v, "elasticsearch"):
		return "elasticsearch"
	case strings.Contains(v, "opensearch"):
		return "opensearch"
	case strings.Contains(v, "clickhouse"):
		return "clickhouse"
	case strings.Contains(v, "nginx"):
		return "nginx"
	case strings.Contains(v, "minio"):
		return "minio"
	case strings.Contains(v, "memcached"):
		return "memcached"
	case strings.Contains(v, "sqlserver") || strings.Contains(v, "mssql"):
		return "sqlserver"
	case strings.Contains(v, "kubernetes") || strings.Contains(v, "k8s"):
		return "kubernetes"
	case strings.Contains(v, "terraform"):
		return "terraform"
	default:
		return ""
	}
}

func isLaravelRoutesFile(path, base string) bool {
	if base != "web.php" && base != "api.php" {
		return false
	}
	normalized := filepath.ToSlash(path)
	return strings.Contains(normalized, "/routes/")
}

var (
	quotedStringPattern         = regexp.MustCompile(`["]([^"]+)["]|[']([^']+)[']`)
	laravelClassHandlerPattern  = regexp.MustCompile(`Route::(?i)(get|post|put|patch|delete|options|any)\s*\(\s*['"]([^'"]+)['"]\s*,\s*\[\s*([A-Za-z0-9_\\]+)::class\s*,\s*['"]([A-Za-z0-9_]+)['"]\s*\]`)
	laravelStringHandlerPattern = regexp.MustCompile(`Route::(?i)(get|post|put|patch|delete|options|any)\s*\(\s*['"]([^'"]+)['"]\s*,\s*['"]([^'"]+)['"]`)
	expressRoutePattern         = regexp.MustCompile(`\b(?:app|router)\.(get|post|put|patch|delete|options|head|all)\s*\(\s*['"\x60]([^'"\x60]+)['"\x60](?:\s*,\s*([A-Za-z0-9_$.]+))?`)
	expressRouteChainPattern    = regexp.MustCompile(`\b(?:app|router)\.route\s*\(\s*['"\x60]([^'"\x60]+)['"\x60]\s*\)\.(get|post|put|patch|delete|options|head|all)\s*\(`)
	fastAPIRouterDeclPattern    = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)\s*=\s*APIRouter\((.*)\)`)
	fastAPIPrefixPattern        = regexp.MustCompile(`prefix\s*=\s*["']([^"']+)["']`)
	fastAPIDecoratorPattern     = regexp.MustCompile(`^@([A-Za-z_][A-Za-z0-9_]*)\.(get|post|put|patch|delete|options|head)\(\s*["']([^"']+)["']`)
	pythonDefPattern            = regexp.MustCompile(`^(?:async\s+)?def\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(`)
	flaskAppDeclPattern         = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)\s*=\s*Flask\(`)
	flaskBlueprintDeclPattern   = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)\s*=\s*Blueprint\((.*)\)`)
	flaskURLPrefixPattern       = regexp.MustCompile(`url_prefix\s*=\s*["']([^"']+)["']`)
	flaskDecoratorRoutePattern  = regexp.MustCompile(`^@([A-Za-z_][A-Za-z0-9_]*)\.route\(\s*["']([^"']+)["'](?:\s*,\s*methods\s*=\s*\[([^\]]+)\])?`)
	flaskDecoratorMethodPattern = regexp.MustCompile(`^@([A-Za-z_][A-Za-z0-9_]*)\.(get|post|put|patch|delete|options|head)\(\s*["']([^"']+)["']`)
	methodTokenPattern          = regexp.MustCompile(`["']([A-Za-z]+)["']`)
	djangoPathPattern           = regexp.MustCompile(`(?:path|re_path)\(\s*["']([^"']+)["']\s*,\s*([A-Za-z_][A-Za-z0-9_\.]*)(?:\.as_view\(\))?`)
	goGinRootPattern            = regexp.MustCompile(`([A-Za-z_][A-Za-z0-9_]*)\s*:=\s*gin\.(Default|New)\s*\(`)
	goEchoRootPattern           = regexp.MustCompile(`([A-Za-z_][A-Za-z0-9_]*)\s*:=\s*echo\.New\s*\(`)
	goGroupPattern              = regexp.MustCompile(`([A-Za-z_][A-Za-z0-9_]*)\s*:=\s*([A-Za-z_][A-Za-z0-9_]*)\.Group\(\s*["\x60]([^"\x60]*)["\x60]`)
	goRoutePattern              = regexp.MustCompile(`([A-Za-z_][A-Za-z0-9_]*)\.(GET|POST|PUT|PATCH|DELETE|OPTIONS|HEAD|Any|Match)\(\s*["\x60]([^"\x60]+)["\x60](?:\s*,\s*([A-Za-z_][A-Za-z0-9_./]*))?`)
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

func isPythonRoutesFile(path string) bool {
	if strings.ToLower(filepath.Ext(path)) != ".py" {
		return false
	}
	base := strings.ToLower(filepath.Base(path))
	if strings.HasSuffix(base, "_test.py") || strings.HasPrefix(base, "test_") {
		return false
	}
	return true
}

func parseFastAPIRoutes(path string) ([]model.Route, bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, false, err
	}
	defer file.Close()

	routerPrefixByVar := map[string]string{}
	routes := make([]model.Route, 0)
	usedFastAPI := false

	pendingReceiver := ""
	pendingMethod := ""
	pendingPath := ""

	s := bufio.NewScanner(file)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if m := fastAPIRouterDeclPattern.FindStringSubmatch(line); len(m) == 3 {
			prefix := ""
			if pm := fastAPIPrefixPattern.FindStringSubmatch(m[2]); len(pm) == 2 {
				prefix = normalizeRoutePath(pm[1])
			}
			routerPrefixByVar[m[1]] = prefix
			usedFastAPI = true
		}

		if m := fastAPIDecoratorPattern.FindStringSubmatch(line); len(m) == 4 {
			pendingReceiver = m[1]
			pendingMethod = strings.ToUpper(m[2])
			pendingPath = normalizeRoutePath(m[3])
			usedFastAPI = true
			continue
		}

		if pendingMethod != "" {
			if m := pythonDefPattern.FindStringSubmatch(line); len(m) == 2 {
				fullPath := pendingPath
				if prefix, ok := routerPrefixByVar[pendingReceiver]; ok && prefix != "" {
					fullPath = joinRoutePaths(prefix, pendingPath)
				}
				routes = append(routes, model.Route{
					Method:     pendingMethod,
					Path:       fullPath,
					Controller: m[1],
				})
				pendingReceiver = ""
				pendingMethod = ""
				pendingPath = ""
				continue
			}

			if strings.HasPrefix(line, "@") {
				pendingReceiver = ""
				pendingMethod = ""
				pendingPath = ""
			}
		}
	}
	if err := s.Err(); err != nil {
		return nil, false, err
	}

	return routes, usedFastAPI, nil
}

func parseFlaskRoutes(path string) ([]model.Route, bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, false, err
	}
	defer file.Close()

	flaskAppVars := map[string]struct{}{}
	blueprintPrefixByVar := map[string]string{}
	routes := make([]model.Route, 0)
	usedFlask := false

	pendingReceiver := ""
	pendingPath := ""
	pendingMethods := make([]string, 0)

	s := bufio.NewScanner(file)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if m := flaskAppDeclPattern.FindStringSubmatch(line); len(m) == 2 {
			flaskAppVars[m[1]] = struct{}{}
			usedFlask = true
		}

		if m := flaskBlueprintDeclPattern.FindStringSubmatch(line); len(m) == 3 {
			prefix := ""
			if pm := flaskURLPrefixPattern.FindStringSubmatch(m[2]); len(pm) == 2 {
				prefix = normalizeRoutePath(pm[1])
			}
			blueprintPrefixByVar[m[1]] = prefix
			usedFlask = true
		}

		if m := flaskDecoratorRoutePattern.FindStringSubmatch(line); len(m) >= 3 {
			if !isKnownFlaskReceiver(m[1], flaskAppVars, blueprintPrefixByVar) {
				continue
			}
			pendingReceiver = m[1]
			pendingPath = normalizeRoutePath(m[2])
			pendingMethods = parseFlaskMethods(m[3])
			if len(pendingMethods) == 0 {
				pendingMethods = []string{"ANY"}
			}
			usedFlask = true
			continue
		}

		if m := flaskDecoratorMethodPattern.FindStringSubmatch(line); len(m) == 4 {
			if !isKnownFlaskReceiver(m[1], flaskAppVars, blueprintPrefixByVar) {
				continue
			}
			pendingReceiver = m[1]
			pendingPath = normalizeRoutePath(m[3])
			pendingMethods = []string{strings.ToUpper(m[2])}
			usedFlask = true
			continue
		}

		if len(pendingMethods) > 0 {
			if m := pythonDefPattern.FindStringSubmatch(line); len(m) == 2 {
				fullPath := pendingPath
				if prefix, ok := blueprintPrefixByVar[pendingReceiver]; ok && prefix != "" {
					fullPath = joinRoutePaths(prefix, pendingPath)
				}
				for _, method := range pendingMethods {
					routes = append(routes, model.Route{
						Method:     method,
						Path:       fullPath,
						Controller: m[1],
					})
				}
				pendingReceiver = ""
				pendingPath = ""
				pendingMethods = nil
				continue
			}

			if strings.HasPrefix(line, "@") {
				pendingReceiver = ""
				pendingPath = ""
				pendingMethods = nil
			}
		}
	}
	if err := s.Err(); err != nil {
		return nil, false, err
	}

	return routes, usedFlask, nil
}

func isKnownFlaskReceiver(receiver string, flaskAppVars map[string]struct{}, blueprintPrefixByVar map[string]string) bool {
	if _, ok := flaskAppVars[receiver]; ok {
		return true
	}
	_, ok := blueprintPrefixByVar[receiver]
	return ok
}

func parseFlaskMethods(value string) []string {
	out := make([]string, 0)
	for _, m := range methodTokenPattern.FindAllStringSubmatch(value, -1) {
		if len(m) < 2 || m[1] == "" {
			continue
		}
		out = append(out, strings.ToUpper(strings.TrimSpace(m[1])))
	}
	return out
}

func isDjangoURLsFile(path string) bool {
	return strings.EqualFold(filepath.Base(path), "urls.py")
}

func parseDjangoRoutes(path string) ([]model.Route, bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, false, err
	}
	defer file.Close()

	routes := make([]model.Route, 0)
	usedDjango := false

	s := bufio.NewScanner(file)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		m := djangoPathPattern.FindStringSubmatch(line)
		if len(m) != 3 {
			continue
		}

		pathValue := normalizeRoutePath(m[1])
		controller := strings.TrimSuffix(m[2], ".as_view")
		if controller == "include" {
			continue
		}
		if strings.Contains(line, ".as_view(") {
			controller += "@as_view"
		}
		routes = append(routes, model.Route{
			Method:     "ANY",
			Path:       pathValue,
			Controller: controller,
		})
		usedDjango = true
	}
	if err := s.Err(); err != nil {
		return nil, false, err
	}

	return routes, usedDjango, nil
}

func isGoRoutesFile(path string) bool {
	if strings.ToLower(filepath.Ext(path)) != ".go" {
		return false
	}
	base := strings.ToLower(filepath.Base(path))
	return !strings.HasSuffix(base, "_test.go")
}

func parseGoRoutes(path string) ([]model.Route, bool, bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, false, false, err
	}
	defer file.Close()

	frameworkByVar := map[string]string{}
	groupPrefixByVar := map[string]string{}
	routes := make([]model.Route, 0)
	usedGin := false
	usedEcho := false

	s := bufio.NewScanner(file)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		if m := goGinRootPattern.FindStringSubmatch(line); len(m) == 3 {
			frameworkByVar[m[1]] = "gin"
			usedGin = true
		}
		if m := goEchoRootPattern.FindStringSubmatch(line); len(m) == 2 {
			frameworkByVar[m[1]] = "echo"
			usedEcho = true
		}
		if m := goGroupPattern.FindStringSubmatch(line); len(m) == 4 {
			groupVar := m[1]
			parentVar := m[2]
			prefix := normalizeRoutePath(m[3])
			parentPrefix := groupPrefixByVar[parentVar]
			groupPrefixByVar[groupVar] = joinRoutePaths(parentPrefix, prefix)
			if fw, ok := frameworkByVar[parentVar]; ok {
				frameworkByVar[groupVar] = fw
				if fw == "gin" {
					usedGin = true
				}
				if fw == "echo" {
					usedEcho = true
				}
			}
		}

		m := goRoutePattern.FindStringSubmatch(line)
		if len(m) < 4 {
			continue
		}
		receiver := m[1]
		method := strings.ToUpper(m[2])
		pathValue := normalizeRoutePath(m[3])
		controller := "handler"
		if len(m) >= 5 && m[4] != "" {
			controller = m[4]
		}
		fullPath := joinRoutePaths(groupPrefixByVar[receiver], pathValue)
		if fullPath == "" {
			fullPath = pathValue
		}

		if fw, ok := frameworkByVar[receiver]; ok {
			if fw == "gin" {
				usedGin = true
			}
			if fw == "echo" {
				usedEcho = true
			}
			routes = append(routes, model.Route{
				Method:     method,
				Path:       fullPath,
				Controller: controller,
			})
			continue
		}

		// Fall back for common root variable names when assignment is in another file.
		if receiver == "r" || receiver == "router" || receiver == "engine" {
			routes = append(routes, model.Route{
				Method:     method,
				Path:       fullPath,
				Controller: controller,
			})
		}
	}
	if err := s.Err(); err != nil {
		return nil, false, false, err
	}

	return routes, usedGin, usedEcho, nil
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

func joinRoutePaths(prefix, route string) string {
	if prefix == "" {
		return normalizeRoutePath(route)
	}
	if route == "" || route == "/" {
		return normalizeRoutePath(prefix)
	}

	p := strings.TrimSuffix(prefix, "/")
	r := strings.TrimPrefix(route, "/")
	if p == "" {
		return normalizeRoutePath(r)
	}
	return normalizeRoutePath(p + "/" + r)
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
