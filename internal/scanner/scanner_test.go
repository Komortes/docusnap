package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanDetectsCoreSignals(t *testing.T) {
	tmp := t.TempDir()

	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module example.com/app\n\nrequire github.com/gin-gonic/gin v1.10.0\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "Dockerfile"), []byte("FROM golang:1.25\n"), 0o644); err != nil {
		t.Fatalf("write Dockerfile: %v", err)
	}

	snap, err := Scan(tmp)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	if len(snap.Languages) != 1 || snap.Languages[0] != "go" {
		t.Fatalf("unexpected languages: %#v", snap.Languages)
	}
	if len(snap.PackageManagers) != 1 || snap.PackageManagers[0] != "go" {
		t.Fatalf("unexpected managers: %#v", snap.PackageManagers)
	}
	if len(snap.Dependencies["go"]) != 1 || snap.Dependencies["go"][0].Name != "github.com/gin-gonic/gin" {
		t.Fatalf("unexpected go deps: %#v", snap.Dependencies["go"])
	}
	if len(snap.Frameworks) != 1 || snap.Frameworks[0] != "gin" {
		t.Fatalf("unexpected frameworks: %#v", snap.Frameworks)
	}
	if len(snap.Infrastructure) != 1 || snap.Infrastructure[0] != "docker" {
		t.Fatalf("unexpected infrastructure: %#v", snap.Infrastructure)
	}
}

func TestParseLaravelRoutes(t *testing.T) {
	tmp := t.TempDir()
	routesFile := filepath.Join(tmp, "api.php")
	content := `<?php
Route::get('/api/orders', [OrderController::class, 'index']);
Route::post('/api/orders', 'OrderController@store');
`
	if err := os.WriteFile(routesFile, []byte(content), 0o644); err != nil {
		t.Fatalf("write routes: %v", err)
	}

	routes, err := parseLaravelRoutes(routesFile)
	if err != nil {
		t.Fatalf("parse routes: %v", err)
	}
	if len(routes) != 2 {
		t.Fatalf("expected 2 routes, got %d", len(routes))
	}
	if routes[0].Method != "GET" || routes[0].Path != "/api/orders" || routes[0].Controller != "OrderController@index" {
		t.Fatalf("unexpected first route: %#v", routes[0])
	}
	if routes[1].Method != "POST" || routes[1].Controller != "OrderController@store" {
		t.Fatalf("unexpected second route: %#v", routes[1])
	}
}

func TestParseExpressRoutes(t *testing.T) {
	tmp := t.TempDir()
	routesFile := filepath.Join(tmp, "server.js")
	content := `const express = require('express');
const app = express();
app.get('/health', healthHandler);
router.post('/api/orders', ordersController.create);
router.route('/api/users').delete(removeUser);
`
	if err := os.WriteFile(routesFile, []byte(content), 0o644); err != nil {
		t.Fatalf("write routes: %v", err)
	}

	routes, err := parseExpressRoutes(routesFile)
	if err != nil {
		t.Fatalf("parse routes: %v", err)
	}
	if len(routes) != 3 {
		t.Fatalf("expected 3 routes, got %d", len(routes))
	}
	if routes[0].Method != "GET" || routes[0].Path != "/health" || routes[0].Controller != "healthHandler" {
		t.Fatalf("unexpected first route: %#v", routes[0])
	}
	if routes[1].Method != "POST" || routes[1].Path != "/api/orders" || routes[1].Controller != "ordersController.create" {
		t.Fatalf("unexpected second route: %#v", routes[1])
	}
	if routes[2].Method != "DELETE" || routes[2].Path != "/api/users" {
		t.Fatalf("unexpected third route: %#v", routes[2])
	}
}

func TestScanDetectsExpressRoutesAsFramework(t *testing.T) {
	tmp := t.TempDir()

	serverPath := filepath.Join(tmp, "server.js")
	content := `const express = require('express');
const app = express();
app.get('/health', healthHandler);
`
	if err := os.WriteFile(serverPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write server.js: %v", err)
	}

	snap, err := Scan(tmp)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	if len(snap.Routes) != 1 || snap.Routes[0].Path != "/health" || snap.Routes[0].Method != "GET" {
		t.Fatalf("unexpected routes: %#v", snap.Routes)
	}
	foundExpress := false
	for _, framework := range snap.Frameworks {
		if framework == "express" {
			foundExpress = true
			break
		}
	}
	if !foundExpress {
		t.Fatalf("expected express framework, got %#v", snap.Frameworks)
	}
}
