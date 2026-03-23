package scanner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/oleksandrskoruk/docusnap/internal/model"
)

var (
	openAPIMethodPattern        = regexp.MustCompile(`^\s{2,}(get|post|put|patch|delete|options|head|trace)\s*:\s*$`)
	openAPIOperationIDPattern   = regexp.MustCompile(`^\s{4,}operationId\s*:\s*['"]?([^'"]+)['"]?\s*$`)
	dotNetMinimalRoutePattern   = regexp.MustCompile(`\b[A-Za-z_][A-Za-z0-9_]*\.Map(Get|Post|Put|Patch|Delete|Options|Head)\(\s*"([^"]+)"(?:\s*,\s*([A-Za-z_][A-Za-z0-9_\.]*))?`)
	dotNetMethodsRoutePattern   = regexp.MustCompile(`\b[A-Za-z_][A-Za-z0-9_]*\.MapMethods\(\s*"([^"]+)"\s*,\s*(?:new\s*\[\]\s*)?\{\s*([^}]*)\}\s*,\s*([A-Za-z_][A-Za-z0-9_\.]*)`)
	dotNetRouteAttrPattern      = regexp.MustCompile(`^\s*\[Route\("([^"]+)"\)\]`)
	dotNetHttpMethodAttrPattern = regexp.MustCompile(`^\s*\[Http(Get|Post|Put|Patch|Delete|Options|Head)(?:\("([^"]*)"\))?\]`)
	dotNetClassPattern          = regexp.MustCompile(`\bclass\s+([A-Za-z_][A-Za-z0-9_]*)\b`)
	dotNetMethodPattern         = regexp.MustCompile(`\b(?:public|protected|internal|private)(?:\s+async)?(?:\s+[A-Za-z0-9_<>\[\],?.]+)+\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(`)
	dotNetPackagePattern        = regexp.MustCompile(`<PackageReference\s+Include="([^"]+)"\s+Version="([^"]+)"\s*/?>`)
)

func isOpenAPIFile(path, base string) bool {
	lower := strings.ToLower(base)
	switch lower {
	case "openapi.yaml", "openapi.yml", "openapi.json", "swagger.yaml", "swagger.yml", "swagger.json":
		return true
	}

	if ext := strings.ToLower(filepath.Ext(path)); ext != ".yaml" && ext != ".yml" && ext != ".json" {
		return false
	}

	lowerPath := strings.ToLower(filepath.ToSlash(path))
	return strings.Contains(lowerPath, "/openapi") || strings.Contains(lowerPath, "/swagger")
}

func parseOpenAPIRoutes(path string) ([]model.Route, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false, err
	}

	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return nil, false, nil
	}

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".json":
		return parseOpenAPIJSONRoutes(data)
	case ".yaml", ".yml":
		return parseOpenAPIYAMLRoutes(data)
	default:
		return nil, false, nil
	}
}

func parseOpenAPIJSONRoutes(data []byte) ([]model.Route, bool, error) {
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, false, err
	}

	if _, ok := doc["openapi"]; !ok {
		if _, ok := doc["swagger"]; !ok {
			return nil, false, nil
		}
	}

	rawPaths, ok := doc["paths"].(map[string]any)
	if !ok || len(rawPaths) == 0 {
		return nil, true, nil
	}

	routes := make([]model.Route, 0)
	for rawPath, entry := range rawPaths {
		methods, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		for rawMethod, methodBody := range methods {
			method := strings.ToUpper(strings.TrimSpace(rawMethod))
			if !isHTTPMethod(method) {
				continue
			}
			controller := "openapi"
			if body, ok := methodBody.(map[string]any); ok {
				if operationID, ok := body["operationId"].(string); ok && strings.TrimSpace(operationID) != "" {
					controller = strings.TrimSpace(operationID)
				}
			}
			routes = append(routes, model.Route{
				Method:     method,
				Path:       normalizeOpenAPIPath(rawPath),
				Controller: controller,
			})
		}
	}

	return deduplicateRoutes(routes), true, nil
}

func parseOpenAPIYAMLRoutes(data []byte) ([]model.Route, bool, error) {
	lines := strings.Split(string(data), "\n")
	usedOpenAPI := false
	inPaths := false
	currentPath := ""
	currentMethod := ""
	routes := make([]model.Route, 0)

	for _, rawLine := range lines {
		line := strings.TrimRight(rawLine, "\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(trimmed, "openapi:") || strings.HasPrefix(trimmed, "swagger:") {
			usedOpenAPI = true
		}
		if trimmed == "paths:" {
			inPaths = true
			currentPath = ""
			currentMethod = ""
			continue
		}
		if !inPaths {
			continue
		}

		indent := len(line) - len(strings.TrimLeft(line, " "))
		if indent == 0 && !strings.HasPrefix(trimmed, "/") {
			inPaths = false
			currentPath = ""
			currentMethod = ""
			continue
		}
		if indent == 2 && strings.HasSuffix(trimmed, ":") && strings.HasPrefix(trimmed, "/") {
			currentPath = strings.TrimSuffix(trimmed, ":")
			currentMethod = ""
			continue
		}
		if currentPath == "" {
			continue
		}
		if m := openAPIMethodPattern.FindStringSubmatch(line); len(m) == 2 {
			currentMethod = strings.ToUpper(m[1])
			routes = append(routes, model.Route{
				Method:     currentMethod,
				Path:       normalizeOpenAPIPath(currentPath),
				Controller: "openapi",
			})
			continue
		}
		if currentMethod != "" {
			if m := openAPIOperationIDPattern.FindStringSubmatch(line); len(m) == 2 {
				routes[len(routes)-1].Controller = strings.TrimSpace(m[1])
			}
		}
	}

	return deduplicateRoutes(routes), usedOpenAPI, nil
}

func normalizeOpenAPIPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "{", ":")
	path = strings.ReplaceAll(path, "}", "")
	return normalizeRoutePath(path)
}

func isNextJSAPIFile(path string) bool {
	lowerPath := strings.ToLower(filepath.ToSlash(path))
	switch {
	case strings.Contains(lowerPath, "/pages/api/"):
		return hasCodeExtension(lowerPath)
	case strings.Contains(lowerPath, "/app/api/") && strings.HasSuffix(lowerPath, "/route.js"),
		strings.Contains(lowerPath, "/app/api/") && strings.HasSuffix(lowerPath, "/route.jsx"),
		strings.Contains(lowerPath, "/app/api/") && strings.HasSuffix(lowerPath, "/route.ts"),
		strings.Contains(lowerPath, "/app/api/") && strings.HasSuffix(lowerPath, "/route.tsx"):
		return true
	default:
		return false
	}
}

func parseNextJSAPIRoutes(path string) ([]model.Route, bool, error) {
	if !isNextJSAPIFile(path) {
		return nil, false, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false, err
	}

	routePath := nextJSRoutePath(path)
	if routePath == "" {
		return nil, false, nil
	}

	lowerPath := strings.ToLower(filepath.ToSlash(path))
	controller := filepath.Base(path)
	if strings.Contains(lowerPath, "/app/api/") {
		methods := make([]string, 0)
		for _, method := range []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD"} {
			if strings.Contains(string(data), "export async function "+method) || strings.Contains(string(data), "export function "+method) {
				methods = append(methods, method)
			}
		}
		if len(methods) == 0 {
			methods = []string{"ANY"}
		}

		routes := make([]model.Route, 0, len(methods))
		for _, method := range methods {
			routes = append(routes, model.Route{
				Method:     method,
				Path:       routePath,
				Controller: controller,
			})
		}
		return routes, true, nil
	}

	return []model.Route{{
		Method:     "ANY",
		Path:       routePath,
		Controller: controller,
	}}, true, nil
}

func nextJSRoutePath(path string) string {
	slashPath := filepath.ToSlash(path)
	if idx := strings.Index(strings.ToLower(slashPath), "/pages/api/"); idx >= 0 {
		route := slashPath[idx+len("/pages/api"):]
		route = strings.TrimSuffix(route, filepath.Ext(route))
		route = strings.TrimSuffix(route, "/index")
		return normalizeNextDynamicSegments("/api" + route)
	}
	if idx := strings.Index(strings.ToLower(slashPath), "/app/api/"); idx >= 0 {
		route := slashPath[idx+len("/app"):]
		route = strings.TrimSuffix(route, filepath.Ext(route))
		route = strings.TrimSuffix(route, "/route")
		return normalizeNextDynamicSegments(route)
	}
	return ""
}

func normalizeNextDynamicSegments(path string) string {
	parts := strings.Split(filepath.ToSlash(path), "/")
	for i, part := range parts {
		if strings.HasPrefix(part, "[...") && strings.HasSuffix(part, "]") {
			parts[i] = ":" + strings.TrimSuffix(strings.TrimPrefix(part, "[..."), "]") + "*"
			continue
		}
		if strings.HasPrefix(part, "[[...") && strings.HasSuffix(part, "]]") {
			parts[i] = ":" + strings.TrimSuffix(strings.TrimPrefix(part, "[[..."), "]]") + "*"
			continue
		}
		if strings.HasPrefix(part, "[") && strings.HasSuffix(part, "]") {
			parts[i] = ":" + strings.TrimSuffix(strings.TrimPrefix(part, "["), "]")
		}
	}
	return normalizeRoutePath(strings.Join(parts, "/"))
}

func hasCodeExtension(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".js", ".jsx", ".ts", ".tsx", ".mjs", ".cjs":
		return true
	default:
		return false
	}
}

func isDotNetProjectFile(base string) bool {
	return strings.HasSuffix(strings.ToLower(base), ".csproj")
}

func parseCsproj(path string) ([]model.Dependency, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	deps := make([]model.Dependency, 0)
	for _, match := range dotNetPackagePattern.FindAllStringSubmatch(string(data), -1) {
		if len(match) != 3 {
			continue
		}
		deps = append(deps, model.Dependency{
			Name:    strings.TrimSpace(match[1]),
			Version: strings.TrimSpace(match[2]),
		})
	}
	return uniqueDependencies(deps), nil
}

func isDotNetRoutesFile(path string) bool {
	if strings.ToLower(filepath.Ext(path)) != ".cs" {
		return false
	}
	base := strings.ToLower(filepath.Base(path))
	return !strings.HasSuffix(base, ".designer.cs") && !strings.Contains(base, "test")
}

func parseDotNetRoutes(path string) ([]model.Route, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false, err
	}

	lines := strings.Split(string(data), "\n")
	routes := make([]model.Route, 0)
	usedAspNet := false
	classPrefix := ""
	currentClass := ""
	pendingMethods := make([]string, 0)
	pendingSuffix := ""

	for _, rawLine := range lines {
		line := strings.TrimSpace(strings.TrimRight(rawLine, "\r"))
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		if m := dotNetMinimalRoutePattern.FindStringSubmatch(line); len(m) >= 3 {
			controller := "handler"
			if len(m) >= 4 && strings.TrimSpace(m[3]) != "" {
				controller = strings.TrimSpace(m[3])
			}
			routes = append(routes, model.Route{
				Method:     strings.ToUpper(m[1]),
				Path:       normalizeRoutePath(m[2]),
				Controller: controller,
			})
			usedAspNet = true
			continue
		}
		if m := dotNetMethodsRoutePattern.FindStringSubmatch(line); len(m) == 4 {
			methodTokens := quotedStringPattern.FindAllStringSubmatch(m[2], -1)
			for _, token := range methodTokens {
				method := token[1]
				if method == "" {
					method = token[2]
				}
				method = strings.ToUpper(strings.TrimSpace(method))
				if method == "" {
					continue
				}
				routes = append(routes, model.Route{
					Method:     method,
					Path:       normalizeRoutePath(m[1]),
					Controller: strings.TrimSpace(m[3]),
				})
			}
			usedAspNet = true
			continue
		}

		if m := dotNetRouteAttrPattern.FindStringSubmatch(line); len(m) == 2 {
			classPrefix = strings.TrimSpace(m[1])
			usedAspNet = true
			continue
		}
		if m := dotNetClassPattern.FindStringSubmatch(line); len(m) == 2 {
			currentClass = strings.TrimSpace(m[1])
			if strings.Contains(classPrefix, "[controller]") {
				controllerName := strings.TrimSuffix(currentClass, "Controller")
				classPrefix = strings.ReplaceAll(classPrefix, "[controller]", controllerName)
			}
			continue
		}
		if m := dotNetHttpMethodAttrPattern.FindStringSubmatch(line); len(m) >= 2 {
			pendingMethods = []string{strings.ToUpper(strings.TrimSpace(m[1]))}
			if len(m) >= 3 {
				pendingSuffix = strings.TrimSpace(m[2])
			} else {
				pendingSuffix = ""
			}
			usedAspNet = true
			continue
		}
		if len(pendingMethods) > 0 {
			if m := dotNetMethodPattern.FindStringSubmatch(line); len(m) == 2 {
				controller := strings.TrimSpace(m[1])
				fullPath := joinAttributeRoutePaths(classPrefix, pendingSuffix)
				for _, method := range pendingMethods {
					routes = append(routes, model.Route{
						Method:     method,
						Path:       fullPath,
						Controller: controller,
					})
				}
				pendingMethods = nil
				pendingSuffix = ""
				continue
			}
			if strings.HasPrefix(line, "[") {
				pendingMethods = nil
				pendingSuffix = ""
			}
		}
	}

	return deduplicateRoutes(routes), usedAspNet, nil
}

func isHTTPMethod(method string) bool {
	switch strings.ToUpper(strings.TrimSpace(method)) {
	case "GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD", "TRACE":
		return true
	default:
		return false
	}
}
