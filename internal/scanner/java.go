package scanner

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/oleksandrskoruk/docusnap/internal/model"
)

type mavenProject struct {
	Dependencies []mavenDependency `xml:"dependencies>dependency"`
}

type mavenDependency struct {
	GroupID    string `xml:"groupId"`
	ArtifactID string `xml:"artifactId"`
	Version    string `xml:"version"`
}

var (
	gradleDependencyPattern      = regexp.MustCompile(`^\s*(?:api|implementation|compileOnly|runtimeOnly|testImplementation|testRuntimeOnly|annotationProcessor|kapt)\s*(?:\(\s*)?["']([^:"']+):([^:"']+)(?::([^"')]+))?["']`)
	gradleNamedDependencyPattern = regexp.MustCompile(`^\s*(?:api|implementation|compileOnly|runtimeOnly|testImplementation|testRuntimeOnly|annotationProcessor|kapt)\s+group:\s*["']([^"']+)["']\s*,\s*name:\s*["']([^"']+)["'](?:\s*,\s*version:\s*["']([^"']+)["'])?`)
	springMappingPattern         = regexp.MustCompile(`^\s*@(?:(Get|Post|Put|Patch|Delete|Options|Head)Mapping|RequestMapping)(?:\((.*)\))?\s*$`)
	springPathAttrPattern        = regexp.MustCompile(`(?:value|path)\s*=\s*"([^"]*)"`)
	springMethodEnumPattern      = regexp.MustCompile(`RequestMethod\.(GET|POST|PUT|PATCH|DELETE|OPTIONS|HEAD)`)
	springQuotedStringPattern    = regexp.MustCompile(`"([^"]*)"`)
	javaClassPattern             = regexp.MustCompile(`\bclass\s+([A-Za-z_][A-Za-z0-9_]*)\b`)
	javaMethodPattern            = regexp.MustCompile(`\b(?:public|protected|private)(?:\s+static)?(?:\s+final)?(?:\s+[A-Za-z0-9_<>\[\],?.]+)+\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(`)
)

func isMavenProjectFile(base string) bool {
	return strings.EqualFold(base, "pom.xml")
}

func isGradleProjectFile(base string) bool {
	lower := strings.ToLower(base)
	return lower == "build.gradle" || lower == "build.gradle.kts"
}

func parsePomXML(path string) ([]model.Dependency, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var project mavenProject
	if err := xml.Unmarshal(data, &project); err != nil {
		return nil, err
	}

	deps := make([]model.Dependency, 0, len(project.Dependencies))
	for _, dep := range project.Dependencies {
		groupID := strings.TrimSpace(dep.GroupID)
		artifactID := strings.TrimSpace(dep.ArtifactID)
		if groupID == "" || artifactID == "" {
			continue
		}
		deps = append(deps, model.Dependency{
			Name:    groupID + ":" + artifactID,
			Version: strings.TrimSpace(dep.Version),
		})
	}
	return uniqueDependencies(deps), nil
}

func parseGradleFile(path string) ([]model.Dependency, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	deps := make([]model.Dependency, 0)
	lines := strings.Split(string(data), "\n")
	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		if m := gradleDependencyPattern.FindStringSubmatch(line); len(m) >= 3 {
			version := ""
			if len(m) >= 4 {
				version = strings.TrimSpace(m[3])
			}
			deps = append(deps, model.Dependency{
				Name:    strings.TrimSpace(m[1]) + ":" + strings.TrimSpace(m[2]),
				Version: version,
			})
			continue
		}

		if m := gradleNamedDependencyPattern.FindStringSubmatch(line); len(m) >= 3 {
			version := ""
			if len(m) >= 4 {
				version = strings.TrimSpace(m[3])
			}
			deps = append(deps, model.Dependency{
				Name:    strings.TrimSpace(m[1]) + ":" + strings.TrimSpace(m[2]),
				Version: version,
			})
		}
	}

	return uniqueDependencies(deps), nil
}

func isJavaRoutesFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".java" {
		return false
	}
	slashPath := strings.ToLower(filepath.ToSlash(path))
	base := filepath.Base(slashPath)
	return !strings.Contains(slashPath, "/src/test/") &&
		!strings.HasSuffix(base, "test.java") &&
		!strings.HasSuffix(base, "tests.java")
}

func parseSpringRoutes(path string) ([]model.Route, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false, err
	}

	lines := strings.Split(string(data), "\n")
	routes := make([]model.Route, 0)
	currentClass := ""
	classPrefix := ""
	pendingPath := ""
	pendingMethods := []string(nil)
	usedSpring := false

	for _, rawLine := range lines {
		line := strings.TrimSpace(strings.TrimRight(rawLine, "\r"))
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		if m := springMappingPattern.FindStringSubmatch(line); len(m) >= 2 {
			pathValue, methods := parseSpringMapping(m[1], m[2])
			pendingPath = pathValue
			pendingMethods = methods
			usedSpring = true
			continue
		}

		if m := javaClassPattern.FindStringSubmatch(line); len(m) == 2 {
			currentClass = strings.TrimSpace(m[1])
			classPrefix = pendingPath
			pendingPath = ""
			pendingMethods = nil
			continue
		}

		if len(pendingMethods) > 0 || pendingPath != "" {
			if m := javaMethodPattern.FindStringSubmatch(line); len(m) == 2 {
				methodName := strings.TrimSpace(m[1])
				methods := pendingMethods
				if len(methods) == 0 {
					methods = []string{"ANY"}
				}
				for _, method := range methods {
					routes = append(routes, model.Route{
						Method:     method,
						Path:       joinAttributeRoutePaths(classPrefix, pendingPath),
						Controller: springControllerName(currentClass, methodName),
					})
				}
				pendingPath = ""
				pendingMethods = nil
				continue
			}

			if strings.HasPrefix(line, "@") {
				continue
			}
		}
	}

	return deduplicateRoutes(routes), usedSpring, nil
}

func parseSpringMapping(kind, args string) (string, []string) {
	args = strings.TrimSpace(args)
	pathValue := "/"

	if args != "" {
		if m := springPathAttrPattern.FindStringSubmatch(args); len(m) == 2 {
			pathValue = strings.TrimSpace(m[1])
		} else if m := springQuotedStringPattern.FindStringSubmatch(args); len(m) == 2 {
			pathValue = strings.TrimSpace(m[1])
		}
	}

	switch strings.ToUpper(strings.TrimSpace(kind)) {
	case "GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD":
		return pathValue, []string{strings.ToUpper(strings.TrimSpace(kind))}
	default:
		methodSet := map[string]struct{}{}
		for _, match := range springMethodEnumPattern.FindAllStringSubmatch(args, -1) {
			if len(match) < 2 {
				continue
			}
			methodSet[strings.ToUpper(strings.TrimSpace(match[1]))] = struct{}{}
		}
		methods := make([]string, 0, len(methodSet))
		for method := range methodSet {
			methods = append(methods, method)
		}
		sort.Strings(methods)
		return pathValue, methods
	}
}

func springControllerName(className, methodName string) string {
	className = strings.TrimSpace(className)
	methodName = strings.TrimSpace(methodName)
	switch {
	case className == "" && methodName == "":
		return "handler"
	case className == "":
		return methodName
	case methodName == "":
		return className
	default:
		return className + "#" + methodName
	}
}

func joinAttributeRoutePaths(prefix, suffix string) string {
	prefix = strings.TrimSpace(prefix)
	suffix = strings.TrimSpace(suffix)
	switch {
	case prefix == "" && suffix == "":
		return "/"
	case prefix == "":
		return normalizeRoutePath(suffix)
	case suffix == "":
		return normalizeRoutePath(prefix)
	default:
		return joinRoutePaths(prefix, suffix)
	}
}
