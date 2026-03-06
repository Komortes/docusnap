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
async def health():
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
-e .
--editable git+https://github.com/pallets/flask.git#egg=flask
-c constraints.txt
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

func TestScanPyprojectWithoutPoetryDoesNotSetPoetryManager(t *testing.T) {
	tmp := t.TempDir()

	pyproject := `[project]
name = "demo"
dependencies = [
  "requests>=2.0.0",
]
`
	if err := os.WriteFile(filepath.Join(tmp, "pyproject.toml"), []byte(pyproject), 0o644); err != nil {
		t.Fatalf("write pyproject.toml: %v", err)
	}

	snap, err := Scan(tmp)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if containsString(snap.PackageManagers, "poetry") {
		t.Fatalf("did not expect poetry manager, got %#v", snap.PackageManagers)
	}
	if !containsString(snap.PackageManagers, "pip") {
		t.Fatalf("expected pip manager, got %#v", snap.PackageManagers)
	}
}

func TestParseFlaskRoutes(t *testing.T) {
	tmp := t.TempDir()
	pyPath := filepath.Join(tmp, "app.py")
	content := `from flask import Flask, Blueprint

app = Flask(__name__)
bp = Blueprint("api", __name__, url_prefix="/api")

@app.get("/health")
def health():
    return "ok"

@bp.route("/orders", methods=["POST"])
def create_order():
    return "ok"
`
	if err := os.WriteFile(pyPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write app.py: %v", err)
	}

	routes, usedFlask, err := parseFlaskRoutes(pyPath)
	if err != nil {
		t.Fatalf("parse flask routes: %v", err)
	}
	if !usedFlask {
		t.Fatalf("expected flask usage")
	}
	if len(routes) != 2 {
		t.Fatalf("expected 2 routes, got %d (%#v)", len(routes), routes)
	}
	if routes[0].Method != "GET" || routes[0].Path != "/health" {
		t.Fatalf("unexpected first route: %#v", routes[0])
	}
	if routes[1].Method != "POST" || routes[1].Path != "/api/orders" {
		t.Fatalf("unexpected second route: %#v", routes[1])
	}
}

func TestParseDjangoRoutes(t *testing.T) {
	tmp := t.TempDir()
	pyPath := filepath.Join(tmp, "urls.py")
	content := `from django.urls import path
from . import views

urlpatterns = [
    path("health/", views.health),
    path("orders/", views.OrderView.as_view()),
]
`
	if err := os.WriteFile(pyPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write urls.py: %v", err)
	}

	routes, usedDjango, err := parseDjangoRoutes(pyPath)
	if err != nil {
		t.Fatalf("parse django routes: %v", err)
	}
	if !usedDjango {
		t.Fatalf("expected django usage")
	}
	if len(routes) != 2 {
		t.Fatalf("expected 2 routes, got %d (%#v)", len(routes), routes)
	}
	if routes[0].Method != "ANY" || routes[0].Path != "/health/" || routes[0].Controller != "views.health" {
		t.Fatalf("unexpected first route: %#v", routes[0])
	}
	if routes[1].Method != "ANY" || routes[1].Controller != "views.OrderView@as_view" {
		t.Fatalf("unexpected second route: %#v", routes[1])
	}
}

func TestScanDetectsFlaskAndDjangoRoutesAsFrameworks(t *testing.T) {
	tmp := t.TempDir()

	flaskFile := `from flask import Flask
app = Flask(__name__)
@app.get("/health")
def health():
    return "ok"
`
	if err := os.WriteFile(filepath.Join(tmp, "app.py"), []byte(flaskFile), 0o644); err != nil {
		t.Fatalf("write app.py: %v", err)
	}

	djangoFile := `from django.urls import path
from . import views
urlpatterns = [
    path("orders/", views.orders),
]
`
	if err := os.WriteFile(filepath.Join(tmp, "urls.py"), []byte(djangoFile), 0o644); err != nil {
		t.Fatalf("write urls.py: %v", err)
	}

	snap, err := Scan(tmp)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if !containsString(snap.Frameworks, "flask") || !containsString(snap.Frameworks, "django") {
		t.Fatalf("expected flask+django frameworks, got %#v", snap.Frameworks)
	}
}

func TestScanDetectsKubernetesEnvTerraformInfrastructure(t *testing.T) {
	tmp := t.TempDir()

	if err := os.MkdirAll(filepath.Join(tmp, "k8s"), 0o755); err != nil {
		t.Fatalf("mkdir k8s: %v", err)
	}
	k8sManifest := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: api
spec:
  template:
    spec:
      containers:
      - name: db
        image: postgres:16
`
	if err := os.WriteFile(filepath.Join(tmp, "k8s", "deployment.yaml"), []byte(k8sManifest), 0o644); err != nil {
		t.Fatalf("write k8s deployment: %v", err)
	}

	envContent := `DATABASE_URL=postgres://localhost:5432/app
REDIS_URL=redis://localhost:6379/0
`
	if err := os.WriteFile(filepath.Join(tmp, ".env"), []byte(envContent), 0o644); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tmp, "infra"), 0o755); err != nil {
		t.Fatalf("mkdir infra: %v", err)
	}
	tfContent := `terraform {
  required_providers {
    aws = { source = "hashicorp/aws" }
  }
}

resource "aws_eks_cluster" "main" {}
resource "aws_elasticache_cluster" "cache" {}
`
	if err := os.WriteFile(filepath.Join(tmp, "infra", "main.tf"), []byte(tfContent), 0o644); err != nil {
		t.Fatalf("write main.tf: %v", err)
	}

	snap, err := Scan(tmp)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	requiredInfra := []string{"kubernetes", "postgres", "redis", "terraform"}
	for _, infra := range requiredInfra {
		if !containsString(snap.Infrastructure, infra) {
			t.Fatalf("expected infrastructure %q, got %#v", infra, snap.Infrastructure)
		}
	}

	requiredConfigs := []string{".env", "infra/main.tf", "k8s/deployment.yaml"}
	for _, cfg := range requiredConfigs {
		if !containsString(snap.ConfigFiles, cfg) {
			t.Fatalf("expected config file %q, got %#v", cfg, snap.ConfigFiles)
		}
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
