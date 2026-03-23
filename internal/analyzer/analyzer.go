package analyzer

import (
	"fmt"
	"sort"
	"strings"

	"github.com/oleksandrskoruk/docusnap/internal/model"
)

// RenderSummary builds a concise project intelligence report.
func RenderSummary(snap model.Snapshot) string {
	var b strings.Builder

	fmt.Fprintf(&b, "Project summary\n\n")
	fmt.Fprintf(&b, "Path\n%s\n\n", snap.ProjectPath)

	fmt.Fprintf(&b, "Languages\n")
	if len(snap.Languages) == 0 {
		fmt.Fprintf(&b, "- n/a\n\n")
	} else {
		for _, lang := range snap.Languages {
			fmt.Fprintf(&b, "- %s\n", lang)
		}
		fmt.Fprintf(&b, "\n")
	}

	fmt.Fprintf(&b, "Frameworks\n")
	if len(snap.Frameworks) == 0 {
		fmt.Fprintf(&b, "- n/a\n\n")
	} else {
		for _, fw := range snap.Frameworks {
			fmt.Fprintf(&b, "- %s\n", fw)
		}
		fmt.Fprintf(&b, "\n")
	}

	fmt.Fprintf(&b, "Package managers\n")
	if len(snap.PackageManagers) == 0 {
		fmt.Fprintf(&b, "- n/a\n\n")
	} else {
		for _, manager := range snap.PackageManagers {
			fmt.Fprintf(&b, "- %s\n", manager)
		}
		fmt.Fprintf(&b, "\n")
	}

	fmt.Fprintf(&b, "Repository shape\n")
	if snap.ProjectStats.TotalFiles == 0 {
		fmt.Fprintf(&b, "- files: n/a\n\n")
	} else {
		fmt.Fprintf(&b, "- total files: %d\n", snap.ProjectStats.TotalFiles)
		fmt.Fprintf(&b, "- source files: %d\n", snap.ProjectStats.SourceFiles)
		fmt.Fprintf(&b, "- test files: %d\n", snap.ProjectStats.TestFiles)
		fmt.Fprintf(&b, "- manifest files: %d\n", snap.ProjectStats.ManifestFiles)
		fmt.Fprintf(&b, "- config files: %d\n\n", snap.ProjectStats.ConfigFiles)
	}

	fmt.Fprintf(&b, "Dependencies\n")
	managers := make([]string, 0, len(snap.Dependencies))
	for manager := range snap.Dependencies {
		managers = append(managers, manager)
	}
	sort.Strings(managers)
	if len(managers) == 0 {
		fmt.Fprintf(&b, "- n/a\n\n")
	} else {
		totalDeps := 0
		for _, manager := range managers {
			count := len(snap.Dependencies[manager])
			totalDeps += count
			fmt.Fprintf(&b, "- %s: %d\n", manager, count)
		}
		fmt.Fprintf(&b, "- total: %d\n", totalDeps)
		fmt.Fprintf(&b, "\n")
	}

	fmt.Fprintf(&b, "API endpoints\n")
	fmt.Fprintf(&b, "- %d routes detected\n", len(snap.Routes))
	methodCount := routeMethodCount(snap.Routes)
	if len(methodCount) > 0 {
		methods := make([]string, 0, len(methodCount))
		for method := range methodCount {
			methods = append(methods, method)
		}
		sort.Strings(methods)
		for _, method := range methods {
			fmt.Fprintf(&b, "- %s: %d\n", method, methodCount[method])
		}
	}
	fmt.Fprintf(&b, "\n")

	fmt.Fprintf(&b, "API groups\n")
	if len(snap.APIGroups) == 0 {
		fmt.Fprintf(&b, "- n/a\n\n")
	} else {
		for _, group := range snap.APIGroups {
			fmt.Fprintf(&b, "- %s: %d (%s)\n", group.Prefix, group.RouteCount, strings.Join(group.Methods, ", "))
		}
		fmt.Fprintf(&b, "\n")
	}

	fmt.Fprintf(&b, "Services\n")
	if len(snap.Infrastructure) == 0 {
		fmt.Fprintf(&b, "- n/a\n")
	} else {
		for _, service := range snap.Infrastructure {
			fmt.Fprintf(&b, "- %s\n", service)
		}
	}

	return b.String()
}

func routeMethodCount(routes []model.Route) map[string]int {
	out := map[string]int{}
	for _, route := range routes {
		method := strings.ToUpper(strings.TrimSpace(route.Method))
		if method == "" {
			method = "UNKNOWN"
		}
		out[method]++
	}
	return out
}
