package diff

import (
	"fmt"
	"sort"
	"strings"

	"github.com/oleksandrskoruk/docusnap/internal/model"
)

type Result struct {
	AddedFrameworks     []string                      `json:"addedFrameworks"`
	RemovedFrameworks   []string                      `json:"removedFrameworks"`
	AddedDependencies   map[string][]model.Dependency `json:"addedDependencies"`
	RemovedDependencies map[string][]model.Dependency `json:"removedDependencies"`
	AddedRoutes         []model.Route                 `json:"addedRoutes"`
	RemovedRoutes       []model.Route                 `json:"removedRoutes"`
}

func Compare(oldSnap, newSnap model.Snapshot) Result {
	return Result{
		AddedFrameworks:     addedStrings(oldSnap.Frameworks, newSnap.Frameworks),
		RemovedFrameworks:   addedStrings(newSnap.Frameworks, oldSnap.Frameworks),
		AddedDependencies:   compareDependencies(oldSnap.Dependencies, newSnap.Dependencies),
		RemovedDependencies: compareDependencies(newSnap.Dependencies, oldSnap.Dependencies),
		AddedRoutes:         compareRoutes(oldSnap.Routes, newSnap.Routes),
		RemovedRoutes:       compareRoutes(newSnap.Routes, oldSnap.Routes),
	}
}

func (r Result) HasChanges() bool {
	if len(r.AddedFrameworks) > 0 || len(r.RemovedFrameworks) > 0 {
		return true
	}
	if len(r.AddedRoutes) > 0 || len(r.RemovedRoutes) > 0 {
		return true
	}
	for _, deps := range r.AddedDependencies {
		if len(deps) > 0 {
			return true
		}
	}
	for _, deps := range r.RemovedDependencies {
		if len(deps) > 0 {
			return true
		}
	}
	return false
}

func (r Result) RenderText() string {
	if !r.HasChanges() {
		return "No changes detected"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Changes detected\n\n")

	if hasDependencyChanges(r.AddedDependencies, r.RemovedDependencies) {
		fmt.Fprintf(&b, "Dependencies\n")
		for _, line := range dependencyLines("+", r.AddedDependencies) {
			fmt.Fprintf(&b, "%s\n", line)
		}
		for _, line := range dependencyLines("-", r.RemovedDependencies) {
			fmt.Fprintf(&b, "%s\n", line)
		}
		fmt.Fprintf(&b, "\n")
	}

	if len(r.AddedRoutes) > 0 || len(r.RemovedRoutes) > 0 {
		fmt.Fprintf(&b, "Endpoints\n")
		for _, route := range r.AddedRoutes {
			fmt.Fprintf(&b, "+ %s %s\n", route.Method, route.Path)
		}
		for _, route := range r.RemovedRoutes {
			fmt.Fprintf(&b, "- %s %s\n", route.Method, route.Path)
		}
		fmt.Fprintf(&b, "\n")
	}

	if len(r.AddedFrameworks) > 0 || len(r.RemovedFrameworks) > 0 {
		fmt.Fprintf(&b, "Frameworks\n")
		for _, fw := range r.AddedFrameworks {
			fmt.Fprintf(&b, "+ %s\n", fw)
		}
		for _, fw := range r.RemovedFrameworks {
			fmt.Fprintf(&b, "- %s\n", fw)
		}
	}

	return strings.TrimSpace(b.String())
}

func addedStrings(oldItems, newItems []string) []string {
	oldSet := map[string]struct{}{}
	for _, item := range oldItems {
		oldSet[item] = struct{}{}
	}

	added := make([]string, 0)
	for _, item := range newItems {
		if _, ok := oldSet[item]; !ok {
			added = append(added, item)
		}
	}

	sort.Strings(added)
	return added
}

func compareDependencies(oldDeps, newDeps map[string][]model.Dependency) map[string][]model.Dependency {
	out := map[string][]model.Dependency{}
	managers := map[string]struct{}{}
	for manager := range oldDeps {
		managers[manager] = struct{}{}
	}
	for manager := range newDeps {
		managers[manager] = struct{}{}
	}

	for manager := range managers {
		oldSet := dependencySet(oldDeps[manager])
		added := make([]model.Dependency, 0)
		for _, dep := range newDeps[manager] {
			if _, ok := oldSet[dependencyKey(dep)]; !ok {
				added = append(added, dep)
			}
		}
		if len(added) > 0 {
			sortDependencies(added)
			out[manager] = added
		}
	}

	return out
}

func compareRoutes(oldRoutes, newRoutes []model.Route) []model.Route {
	oldSet := map[string]struct{}{}
	for _, route := range oldRoutes {
		oldSet[routeKey(route)] = struct{}{}
	}

	added := make([]model.Route, 0)
	for _, route := range newRoutes {
		if _, ok := oldSet[routeKey(route)]; !ok {
			added = append(added, route)
		}
	}

	sort.Slice(added, func(i, j int) bool {
		if added[i].Method == added[j].Method {
			return added[i].Path < added[j].Path
		}
		return added[i].Method < added[j].Method
	})
	return added
}

func dependencySet(items []model.Dependency) map[string]struct{} {
	set := make(map[string]struct{}, len(items))
	for _, item := range items {
		set[dependencyKey(item)] = struct{}{}
	}
	return set
}

func dependencyKey(dep model.Dependency) string {
	return dep.Name + "@" + dep.Version
}

func routeKey(route model.Route) string {
	return route.Method + " " + route.Path + " " + route.Controller
}

func hasDependencyChanges(added, removed map[string][]model.Dependency) bool {
	for _, items := range added {
		if len(items) > 0 {
			return true
		}
	}
	for _, items := range removed {
		if len(items) > 0 {
			return true
		}
	}
	return false
}

func dependencyLines(prefix string, groups map[string][]model.Dependency) []string {
	managers := make([]string, 0, len(groups))
	for manager := range groups {
		managers = append(managers, manager)
	}
	sort.Strings(managers)

	lines := make([]string, 0)
	for _, manager := range managers {
		for _, dep := range groups[manager] {
			lines = append(lines, fmt.Sprintf("%s %s (%s)", prefix, dep.Name, manager))
		}
	}
	return lines
}

func sortDependencies(items []model.Dependency) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].Name == items[j].Name {
			return items[i].Version < items[j].Version
		}
		return items[i].Name < items[j].Name
	})
}
