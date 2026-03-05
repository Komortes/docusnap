package analyzer

import (
	"strings"
	"testing"

	"github.com/oleksandrskoruk/docusnap/internal/model"
)

func TestRenderSummaryIncludesDependencyAndRouteBreakdown(t *testing.T) {
	snap := model.Snapshot{
		ProjectPath:     "/repo/app",
		Languages:       []string{"go", "javascript"},
		Frameworks:      []string{"gin", "react"},
		PackageManagers: []string{"go", "npm"},
		Infrastructure:  []string{"docker", "redis"},
		Dependencies: map[string][]model.Dependency{
			"go": {
				{Name: "github.com/gin-gonic/gin", Version: "v1.10.0"},
			},
			"npm": {
				{Name: "react", Version: "^18.3.0"},
				{Name: "vite", Version: "^6.0.0"},
			},
		},
		Routes: []model.Route{
			{Method: "GET", Path: "/health", Controller: "healthHandler"},
			{Method: "GET", Path: "/api/orders", Controller: "listOrders"},
			{Method: "POST", Path: "/api/orders", Controller: "createOrder"},
		},
	}

	out := RenderSummary(snap)

	checks := []string{
		"Package managers",
		"- go",
		"- npm",
		"Dependencies",
		"- go: 1",
		"- npm: 2",
		"- total: 3",
		"API endpoints",
		"- 3 routes detected",
		"- GET: 2",
		"- POST: 1",
	}

	for _, check := range checks {
		if !strings.Contains(out, check) {
			t.Fatalf("expected output to contain %q, got:\n%s", check, out)
		}
	}
}
