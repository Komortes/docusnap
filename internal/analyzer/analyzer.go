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

	fmt.Fprintf(&b, "Dependencies\n")
	managers := make([]string, 0, len(snap.Dependencies))
	for manager := range snap.Dependencies {
		managers = append(managers, manager)
	}
	sort.Strings(managers)
	if len(managers) == 0 {
		fmt.Fprintf(&b, "- n/a\n\n")
	} else {
		for _, manager := range managers {
			fmt.Fprintf(&b, "- %s: %d\n", manager, len(snap.Dependencies[manager]))
		}
		fmt.Fprintf(&b, "\n")
	}

	fmt.Fprintf(&b, "API endpoints\n- %d routes detected\n\n", len(snap.Routes))

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
