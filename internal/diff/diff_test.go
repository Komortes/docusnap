package diff

import (
	"strings"
	"testing"

	"github.com/oleksandrskoruk/docusnap/internal/model"
)

func TestCompareDetectsFrameworkDependencyAndRouteChanges(t *testing.T) {
	oldSnap := model.Snapshot{
		Languages:       []string{"php"},
		PackageManagers: []string{"composer"},
		Infrastructure:  []string{"mysql"},
		Frameworks:      []string{"laravel"},
		Dependencies: map[string][]model.Dependency{
			"composer": {
				{Name: "laravel/framework", Version: "^11.0"},
			},
		},
		Routes: []model.Route{{Method: "GET", Path: "/health", Controller: "HealthController@index"}},
	}

	newSnap := model.Snapshot{
		Languages:       []string{"php", "typescript"},
		PackageManagers: []string{"composer", "npm"},
		Infrastructure:  []string{"mysql", "redis"},
		Frameworks:      []string{"laravel", "react"},
		Dependencies: map[string][]model.Dependency{
			"composer": {
				{Name: "laravel/framework", Version: "^11.0"},
				{Name: "stripe/stripe-php", Version: "^15.0"},
			},
		},
		Routes: []model.Route{{Method: "POST", Path: "/api/payment", Controller: "PaymentController@store"}},
	}

	result := Compare(oldSnap, newSnap)
	if !result.HasChanges() {
		t.Fatalf("expected diff to contain changes")
	}

	if len(result.AddedFrameworks) != 1 || result.AddedFrameworks[0] != "react" {
		t.Fatalf("unexpected added frameworks: %#v", result.AddedFrameworks)
	}
	if len(result.AddedLanguages) != 1 || result.AddedLanguages[0] != "typescript" {
		t.Fatalf("unexpected added languages: %#v", result.AddedLanguages)
	}
	if len(result.AddedPackageManagers) != 1 || result.AddedPackageManagers[0] != "npm" {
		t.Fatalf("unexpected added package managers: %#v", result.AddedPackageManagers)
	}
	if len(result.AddedInfrastructure) != 1 || result.AddedInfrastructure[0] != "redis" {
		t.Fatalf("unexpected added infrastructure: %#v", result.AddedInfrastructure)
	}

	addedComposer := result.AddedDependencies["composer"]
	if len(addedComposer) != 1 || addedComposer[0].Name != "stripe/stripe-php" {
		t.Fatalf("unexpected added dependencies: %#v", result.AddedDependencies)
	}

	if len(result.RemovedRoutes) != 1 || result.RemovedRoutes[0].Path != "/health" {
		t.Fatalf("unexpected removed routes: %#v", result.RemovedRoutes)
	}
	if len(result.AddedRoutes) != 1 || result.AddedRoutes[0].Path != "/api/payment" {
		t.Fatalf("unexpected added routes: %#v", result.AddedRoutes)
	}
}

func TestRenderTextIncludesNewSections(t *testing.T) {
	result := Result{
		AddedLanguages:       []string{"go"},
		AddedPackageManagers: []string{"go"},
		AddedInfrastructure:  []string{"postgres"},
	}

	out := result.RenderText()
	if !strings.Contains(out, "Languages") || !strings.Contains(out, "+ go") {
		t.Fatalf("expected languages section in output, got:\n%s", out)
	}
	if !strings.Contains(out, "Package managers") {
		t.Fatalf("expected package managers section in output, got:\n%s", out)
	}
	if !strings.Contains(out, "Infrastructure services") || !strings.Contains(out, "+ postgres") {
		t.Fatalf("expected infrastructure section in output, got:\n%s", out)
	}
}

func TestRenderMarkdownIncludesSections(t *testing.T) {
	result := Result{
		AddedLanguages:      []string{"go"},
		AddedInfrastructure: []string{"redis"},
		AddedRoutes: []model.Route{
			{Method: "GET", Path: "/health", Controller: "healthHandler"},
		},
	}

	out := result.RenderMarkdown()
	if !strings.Contains(out, "# Snapshot Changes") {
		t.Fatalf("expected markdown title, got:\n%s", out)
	}
	if !strings.Contains(out, "## Languages") || !strings.Contains(out, "`go`") {
		t.Fatalf("expected languages section, got:\n%s", out)
	}
	if !strings.Contains(out, "## Endpoints") || !strings.Contains(out, "GET /health") {
		t.Fatalf("expected endpoints section, got:\n%s", out)
	}
	if !strings.Contains(out, "## Infrastructure services") || !strings.Contains(out, "`redis`") {
		t.Fatalf("expected infrastructure section, got:\n%s", out)
	}
}
