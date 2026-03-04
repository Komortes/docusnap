package diff

import (
	"testing"

	"github.com/oleksandrskoruk/docusnap/internal/model"
)

func TestCompareDetectsFrameworkDependencyAndRouteChanges(t *testing.T) {
	oldSnap := model.Snapshot{
		Frameworks: []string{"laravel"},
		Dependencies: map[string][]model.Dependency{
			"composer": {
				{Name: "laravel/framework", Version: "^11.0"},
			},
		},
		Routes: []model.Route{{Method: "GET", Path: "/health", Controller: "HealthController@index"}},
	}

	newSnap := model.Snapshot{
		Frameworks: []string{"laravel", "react"},
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
