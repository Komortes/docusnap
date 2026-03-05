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

func TestParseFastAPIRoutes(t *testing.T) {
	tmp := t.TempDir()
	pyPath := filepath.Join(tmp, "main.py")
	content := `from fastapi import FastAPI, APIRouter

app = FastAPI()
router = APIRouter(prefix="/api")

@app.get("/health")
def health():
    return {"ok": True}

@router.post("/orders")
def create_order():
    return {"id": 1}
`
	if err := os.WriteFile(pyPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write main.py: %v", err)
	}

	routes, usedFastAPI, err := parseFastAPIRoutes(pyPath)
	if err != nil {
		t.Fatalf("parse fastapi routes: %v", err)
	}
	if !usedFastAPI {
		t.Fatalf("expected fastapi usage detected")
	}
	if len(routes) != 2 {
		t.Fatalf("expected 2 routes, got %d (%#v)", len(routes), routes)
	}
	if routes[0].Method != "GET" || routes[0].Path != "/health" || routes[0].Controller != "health" {
		t.Fatalf("unexpected first route: %#v", routes[0])
	}
	if routes[1].Method != "POST" || routes[1].Path != "/api/orders" || routes[1].Controller != "create_order" {
		t.Fatalf("unexpected second route: %#v", routes[1])
	}
}

func TestScanDetectsFastAPIRoutesAsFramework(t *testing.T) {
	tmp := t.TempDir()
	pyPath := filepath.Join(tmp, "api.py")
	content := `from fastapi import FastAPI

app = FastAPI()

@app.get("/health")
def health():
    return {"ok": True}
`
	if err := os.WriteFile(pyPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write api.py: %v", err)
	}

	snap, err := Scan(tmp)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if len(snap.Routes) != 1 || snap.Routes[0].Method != "GET" || snap.Routes[0].Path != "/health" {
		t.Fatalf("unexpected routes: %#v", snap.Routes)
	}
	if !containsString(snap.Frameworks, "fastapi") {
		t.Fatalf("expected fastapi framework, got %#v", snap.Frameworks)
	}
}

func TestParseGoRoutes(t *testing.T) {
	tmp := t.TempDir()
	routesFile := filepath.Join(tmp, "main.go")
	content := `package main

import (
	"github.com/gin-gonic/gin"
	"github.com/labstack/echo/v4"
)

func main() {
	r := gin.Default()
	api := r.Group("/api")
	api.GET("/orders", listOrders)

	e := echo.New()
	v1 := e.Group("/v1")
	v1.POST("/payments", createPayment)
}
`
	if err := os.WriteFile(routesFile, []byte(content), 0o644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}

	routes, usedGin, usedEcho, err := parseGoRoutes(routesFile)
	if err != nil {
		t.Fatalf("parse go routes: %v", err)
	}
	if len(routes) != 2 {
		t.Fatalf("expected 2 routes, got %d", len(routes))
	}
	if !usedGin || !usedEcho {
		t.Fatalf("expected both gin and echo detected, got usedGin=%v usedEcho=%v", usedGin, usedEcho)
	}
	if routes[0].Method != "GET" || routes[0].Path != "/api/orders" || routes[0].Controller != "listOrders" {
		t.Fatalf("unexpected first route: %#v", routes[0])
	}
	if routes[1].Method != "POST" || routes[1].Path != "/v1/payments" || routes[1].Controller != "createPayment" {
		t.Fatalf("unexpected second route: %#v", routes[1])
	}
}

func TestScanDetectsGinRoutesAsFramework(t *testing.T) {
	tmp := t.TempDir()

	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module example.com/app\n\ngo 1.25\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	mainGo := `package main

import "github.com/gin-gonic/gin"

func main() {
	r := gin.Default()
	r.GET("/health", healthHandler)
}
`
	if err := os.WriteFile(filepath.Join(tmp, "main.go"), []byte(mainGo), 0o644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}

	snap, err := Scan(tmp)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if len(snap.Routes) != 1 || snap.Routes[0].Path != "/health" || snap.Routes[0].Method != "GET" {
		t.Fatalf("unexpected routes: %#v", snap.Routes)
	}

	foundGin := false
	for _, framework := range snap.Frameworks {
		if framework == "gin" {
			foundGin = true
			break
		}
	}
	if !foundGin {
		t.Fatalf("expected gin framework, got %#v", snap.Frameworks)
	}
}

func TestParseDockerComposeServices(t *testing.T) {
	tmp := t.TempDir()
	composePath := filepath.Join(tmp, "docker-compose.yml")
	content := `version: "3.9"
services:
  app:
    build: .
  db:
    image: postgres:16
  cache:
    image: redis:7
  broker:
    image: rabbitmq:3-management
`
	if err := os.WriteFile(composePath, []byte(content), 0o644); err != nil {
		t.Fatalf("write docker-compose.yml: %v", err)
	}

	services, err := parseDockerComposeServices(composePath)
	if err != nil {
		t.Fatalf("parse docker-compose services: %v", err)
	}
	if !containsString(services, "postgres") || !containsString(services, "redis") || !containsString(services, "rabbitmq") {
		t.Fatalf("unexpected services: %#v", services)
	}
}

func TestScanDetectsComposeInfrastructureServices(t *testing.T) {
	tmp := t.TempDir()
	composePath := filepath.Join(tmp, "docker-compose.yml")
	content := `services:
  db:
    image: mysql:8
  cache:
    image: redis:7
`
	if err := os.WriteFile(composePath, []byte(content), 0o644); err != nil {
		t.Fatalf("write docker-compose.yml: %v", err)
	}

	snap, err := Scan(tmp)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	if !containsString(snap.Infrastructure, "docker") || !containsString(snap.Infrastructure, "mysql") || !containsString(snap.Infrastructure, "redis") {
		t.Fatalf("unexpected infrastructure: %#v", snap.Infrastructure)
	}
}

func TestParseRequirementsTxt(t *testing.T) {
	tmp := t.TempDir()
	reqPath := filepath.Join(tmp, "requirements.txt")
	content := `# app deps
fastapi>=0.115.0
uvicorn[standard]==0.30.0
django==5.1.1 # inline comment
-r requirements-dev.txt
`
	if err := os.WriteFile(reqPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write requirements.txt: %v", err)
	}

	deps, err := parseRequirementsTxt(reqPath)
	if err != nil {
		t.Fatalf("parse requirements: %v", err)
	}
	if len(deps) != 3 {
		t.Fatalf("expected 3 dependencies, got %d (%#v)", len(deps), deps)
	}
	if deps[0].Name != "django" || deps[1].Name != "fastapi" || deps[2].Name != "uvicorn" {
		t.Fatalf("unexpected dependencies: %#v", deps)
	}
}

func TestScanDetectsPythonSignals(t *testing.T) {
	tmp := t.TempDir()

	requirements := `fastapi>=0.115.0
`
	if err := os.WriteFile(filepath.Join(tmp, "requirements.txt"), []byte(requirements), 0o644); err != nil {
		t.Fatalf("write requirements.txt: %v", err)
	}

	pyproject := `[project]
name = "demo"
dependencies = [
  "django>=5.1.0",
]

[tool.poetry.dependencies]
python = "^3.12"
flask = "^3.0.0"
`
	if err := os.WriteFile(filepath.Join(tmp, "pyproject.toml"), []byte(pyproject), 0o644); err != nil {
		t.Fatalf("write pyproject.toml: %v", err)
	}

	snap, err := Scan(tmp)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	if !containsString(snap.Languages, "python") {
		t.Fatalf("expected python language, got %#v", snap.Languages)
	}
	if !containsString(snap.PackageManagers, "pip") || !containsString(snap.PackageManagers, "poetry") {
		t.Fatalf("expected pip+poetry managers, got %#v", snap.PackageManagers)
	}
	if !containsString(snap.Frameworks, "fastapi") || !containsString(snap.Frameworks, "django") || !containsString(snap.Frameworks, "flask") {
		t.Fatalf("expected python frameworks, got %#v", snap.Frameworks)
	}
	if len(snap.Dependencies["pip"]) == 0 || len(snap.Dependencies["poetry"]) == 0 {
		t.Fatalf("expected pip and poetry dependencies, got %#v", snap.Dependencies)
	}
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
