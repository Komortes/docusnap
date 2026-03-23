package render

import (
	"fmt"
	htmltemplate "html/template"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/oleksandrskoruk/docusnap/internal/model"
)

type Format string

const (
	FormatMarkdown Format = "markdown"
	FormatHTML     Format = "html"
	FormatBoth     Format = "both"
)

type GenerateOptions struct {
	Format Format
}

type dependencyManagerSummary struct {
	Name  string
	Count int
}

type routeMethodSummary struct {
	Method string
	Count  int
}

type manifestView struct {
	Kind string
	Path string
}

type directorySummaryView struct {
	Path             string
	FileCount        int
	SourceFiles      int
	TestFiles        int
	ManifestFiles    int
	ConfigFiles      int
	LanguagesText    string
	NotableFilesText string
}

type apiGroupView struct {
	Prefix      string
	RouteCount  int
	MethodsText string
}

var markdownArtifactNames = []string{
	"README.generated.md",
	"project-structure.md",
	"dependencies.md",
	"endpoints.md",
	"dependency-graph.md",
	"module-graph.md",
	"architecture.md",
}

var htmlTemplateSource = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{ .ProjectDisplayName }} Documentation</title>
  <style>
    :root {
      --bg: #f5f1e8;
      --surface: rgba(255, 255, 255, 0.92);
      --surface-strong: #ffffff;
      --ink: #1f2933;
      --muted: #5f6b76;
      --line: rgba(31, 41, 51, 0.12);
      --accent: #0f766e;
      --accent-soft: rgba(15, 118, 110, 0.12);
      --shadow: 0 22px 60px rgba(31, 41, 51, 0.12);
    }

    * {
      box-sizing: border-box;
    }

    body {
      margin: 0;
      font-family: "Iowan Old Style", "Palatino Linotype", "Book Antiqua", Georgia, serif;
      color: var(--ink);
      background:
        radial-gradient(circle at top left, rgba(15, 118, 110, 0.16), transparent 28%),
        radial-gradient(circle at top right, rgba(180, 83, 9, 0.12), transparent 24%),
        linear-gradient(180deg, #f9f7f0 0%, var(--bg) 100%);
      line-height: 1.6;
    }

    a {
      color: var(--accent);
    }

    .page {
      max-width: 1200px;
      margin: 0 auto;
      padding: 40px 20px 72px;
    }

    .hero {
      background: linear-gradient(135deg, rgba(255, 255, 255, 0.96), rgba(255, 248, 235, 0.9));
      border: 1px solid var(--line);
      border-radius: 28px;
      padding: 32px;
      box-shadow: var(--shadow);
      margin-bottom: 28px;
    }

    .eyebrow {
      display: inline-block;
      margin-bottom: 12px;
      padding: 6px 10px;
      border-radius: 999px;
      background: var(--accent-soft);
      color: var(--accent);
      font-size: 0.8rem;
      letter-spacing: 0.08em;
      text-transform: uppercase;
      font-family: "Avenir Next", "Segoe UI", sans-serif;
      font-weight: 700;
    }

    h1, h2, h3 {
      margin: 0 0 12px;
      line-height: 1.15;
    }

    h1 {
      font-size: clamp(2.4rem, 5vw, 4.2rem);
      letter-spacing: -0.04em;
    }

    h2 {
      font-size: 1.5rem;
      margin-top: 0;
    }

    .hero p,
    .section p {
      margin: 0;
      color: var(--muted);
    }

    .meta {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
      gap: 12px;
      margin-top: 24px;
    }

    .metric-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
      gap: 12px;
      margin-bottom: 28px;
    }

    .metric {
      background: var(--surface);
      border: 1px solid var(--line);
      border-radius: 20px;
      padding: 18px;
      box-shadow: var(--shadow);
    }

    .metric strong {
      display: block;
      font-size: 1.9rem;
      line-height: 1;
      margin-bottom: 6px;
    }

    .metric span,
    .meta span {
      display: block;
      color: var(--muted);
      font-size: 0.92rem;
      font-family: "Avenir Next", "Segoe UI", sans-serif;
    }

    .meta strong {
      display: block;
      margin-bottom: 4px;
      font-family: "Avenir Next", "Segoe UI", sans-serif;
      font-size: 0.82rem;
      text-transform: uppercase;
      letter-spacing: 0.08em;
      color: var(--muted);
    }

    .nav {
      display: flex;
      flex-wrap: wrap;
      gap: 10px;
      margin-bottom: 28px;
    }

    .nav a {
      text-decoration: none;
      padding: 10px 14px;
      border-radius: 999px;
      background: rgba(255, 255, 255, 0.72);
      border: 1px solid var(--line);
      font-family: "Avenir Next", "Segoe UI", sans-serif;
      font-size: 0.92rem;
    }

    .section {
      background: var(--surface);
      border: 1px solid var(--line);
      border-radius: 24px;
      padding: 24px;
      margin-bottom: 20px;
      box-shadow: var(--shadow);
    }

    .section + .section {
      margin-top: 20px;
    }

    .list {
      margin: 16px 0 0;
      padding-left: 20px;
    }

    .list li + li {
      margin-top: 6px;
    }

    table {
      width: 100%;
      border-collapse: collapse;
      margin-top: 16px;
      font-size: 0.95rem;
      overflow: hidden;
    }

    th,
    td {
      padding: 12px 14px;
      border-bottom: 1px solid var(--line);
      text-align: left;
      vertical-align: top;
    }

    th {
      font-family: "Avenir Next", "Segoe UI", sans-serif;
      font-size: 0.8rem;
      letter-spacing: 0.06em;
      text-transform: uppercase;
      color: var(--muted);
    }

    code {
      font-family: "SFMono-Regular", "Consolas", "Liberation Mono", monospace;
      font-size: 0.92em;
      background: rgba(15, 118, 110, 0.08);
      padding: 0.12em 0.35em;
      border-radius: 0.35em;
    }

    .diagram {
      margin-top: 16px;
      padding: 16px;
      border-radius: 16px;
      background: rgba(15, 118, 110, 0.05);
      border: 1px solid rgba(15, 118, 110, 0.14);
      overflow: auto;
    }

    .diagram .mermaid {
      min-width: 320px;
    }

    @media (max-width: 720px) {
      .page {
        padding: 24px 14px 48px;
      }

      .hero,
      .section {
        padding: 20px;
        border-radius: 20px;
      }

      table,
      thead,
      tbody,
      th,
      td,
      tr {
        display: block;
      }

      thead {
        display: none;
      }

      tr {
        padding: 12px 0;
        border-bottom: 1px solid var(--line);
      }

      td {
        padding: 4px 0;
        border: 0;
      }
    }
  </style>
  <script type="module">
    import mermaid from "https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.esm.min.mjs";
    mermaid.initialize({ startOnLoad: true, theme: "neutral" });
  </script>
</head>
<body>
  <main class="page">
    <section class="hero">
      <span class="eyebrow">Generated Documentation</span>
      <h1>{{ .ProjectDisplayName }}</h1>
      <p>Static documentation assembled from repository structure, manifests, dependencies, and detected API endpoints.</p>
      <div class="meta">
        <div><strong>Path</strong><span>{{ .ProjectPath }}</span></div>
        <div><strong>Scanned</strong><span>{{ .ScannedAt }}</span></div>
        <div><strong>Languages</strong><span>{{ .LanguagesText }}</span></div>
        <div><strong>Frameworks</strong><span>{{ .FrameworksText }}</span></div>
      </div>
    </section>

    <section class="metric-grid">
      <article class="metric"><strong>{{ .ProjectStats.TotalFiles }}</strong><span>Total files</span></article>
      <article class="metric"><strong>{{ .ProjectStats.SourceFiles }}</strong><span>Source files</span></article>
      <article class="metric"><strong>{{ .TotalDependencies }}</strong><span>Dependencies</span></article>
      <article class="metric"><strong>{{ len .Routes }}</strong><span>API routes</span></article>
      <article class="metric"><strong>{{ .ProjectStats.ManifestFiles }}</strong><span>Manifests</span></article>
      <article class="metric"><strong>{{ .ProjectStats.ConfigFiles }}</strong><span>Config files</span></article>
    </section>

    <nav class="nav">
      <a href="#overview">Overview</a>
      <a href="#structure">Structure</a>
      <a href="#dependencies">Dependencies</a>
      <a href="#api">API</a>
      <a href="#architecture">Architecture</a>
      <a href="#graphs">Graphs</a>
    </nav>

    <section class="section" id="overview">
      <h2>Overview</h2>
      <table>
        <tbody>
          <tr><th>Package managers</th><td>{{ .PackageManagersText }}</td></tr>
          <tr><th>Infrastructure</th><td>{{ .InfrastructureText }}</td></tr>
        </tbody>
      </table>
      <h3>Entry points</h3>
      {{- if .EntryPoints }}
      <ul class="list">
        {{- range .EntryPoints }}
        <li><code>{{ . }}</code></li>
        {{- end }}
      </ul>
      {{- else }}
      <p>No explicit entry points were detected.</p>
      {{- end }}
    </section>

    <section class="section" id="structure">
      <h2>Project Structure</h2>
      {{- if .ManifestViews }}
      <h3>Manifest inventory</h3>
      <table>
        <thead>
          <tr><th>Kind</th><th>Path</th></tr>
        </thead>
        <tbody>
          {{- range .ManifestViews }}
          <tr><td>{{ .Kind }}</td><td><code>{{ .Path }}</code></td></tr>
          {{- end }}
        </tbody>
      </table>
      {{- end }}
      <h3>Directory layout</h3>
      {{- if .DirectoryViews }}
      <table>
        <thead>
          <tr><th>Path</th><th>Files</th><th>Source</th><th>Tests</th><th>Manifests</th><th>Config</th><th>Languages</th><th>Notable files</th></tr>
        </thead>
        <tbody>
          {{- range .DirectoryViews }}
          <tr>
            <td><code>{{ .Path }}</code></td>
            <td>{{ .FileCount }}</td>
            <td>{{ .SourceFiles }}</td>
            <td>{{ .TestFiles }}</td>
            <td>{{ .ManifestFiles }}</td>
            <td>{{ .ConfigFiles }}</td>
            <td>{{ .LanguagesText }}</td>
            <td>{{ .NotableFilesText }}</td>
          </tr>
          {{- end }}
        </tbody>
      </table>
      {{- else }}
      <p>No directory layout was detected.</p>
      {{- end }}
    </section>

    <section class="section" id="dependencies">
      <h2>Dependencies</h2>
      {{- if .DependencySummaries }}
      <table>
        <thead>
          <tr><th>Manager</th><th>Count</th></tr>
        </thead>
        <tbody>
          {{- range .DependencySummaries }}
          <tr><td>{{ .Name }}</td><td>{{ .Count }}</td></tr>
          {{- end }}
        </tbody>
      </table>
      {{- else }}
      <p>No dependencies were detected.</p>
      {{- end }}

      {{- if .DependencyManagers }}
      {{- range .DependencyManagers }}
      <h3>{{ . }}</h3>
      {{- $deps := index $.Dependencies . }}
      {{- if $deps }}
      <table>
        <thead>
          <tr><th>Name</th><th>Version</th></tr>
        </thead>
        <tbody>
          {{- range $deps }}
          <tr><td>{{ .Name }}</td><td>{{ .Version }}</td></tr>
          {{- end }}
        </tbody>
      </table>
      {{- else }}
      <p>No dependencies were detected for this manager.</p>
      {{- end }}
      {{- end }}
      {{- end }}
    </section>

    <section class="section" id="api">
      <h2>API Inventory</h2>
      {{- if .RouteMethodSummaries }}
      <h3>Method summary</h3>
      <table>
        <thead>
          <tr><th>Method</th><th>Count</th></tr>
        </thead>
        <tbody>
          {{- range .RouteMethodSummaries }}
          <tr><td>{{ .Method }}</td><td>{{ .Count }}</td></tr>
          {{- end }}
        </tbody>
      </table>
      {{- end }}

      {{- if .APIGroupViews }}
      <h3>Route groups</h3>
      <table>
        <thead>
          <tr><th>Prefix</th><th>Routes</th><th>Methods</th></tr>
        </thead>
        <tbody>
          {{- range .APIGroupViews }}
          <tr><td><code>{{ .Prefix }}</code></td><td>{{ .RouteCount }}</td><td>{{ .MethodsText }}</td></tr>
          {{- end }}
        </tbody>
      </table>
      {{- end }}

      <h3>Route inventory</h3>
      {{- if .Routes }}
      <table>
        <thead>
          <tr><th>Method</th><th>Path</th><th>Controller</th></tr>
        </thead>
        <tbody>
          {{- range .Routes }}
          <tr><td>{{ .Method }}</td><td><code>{{ .Path }}</code></td><td>{{ .Controller }}</td></tr>
          {{- end }}
        </tbody>
      </table>
      {{- else }}
      <p>No API endpoints were detected.</p>
      {{- end }}
    </section>

    <section class="section" id="architecture">
      <h2>Architecture Signals</h2>
      <table>
        <tbody>
          <tr><th>Languages</th><td>{{ .LanguagesText }}</td></tr>
          <tr><th>Frameworks</th><td>{{ .FrameworksText }}</td></tr>
          <tr><th>Infrastructure</th><td>{{ .InfrastructureText }}</td></tr>
        </tbody>
      </table>
    </section>

    <section class="section" id="graphs">
      <h2>Graphs</h2>
      {{- if .DependencyGraphCode }}
      <h3>Dependency graph</h3>
      <div class="diagram"><div class="mermaid">{{ .DependencyGraphCode }}</div></div>
      {{- end }}
      {{- if .ModuleGraphCode }}
      <h3>Module graph</h3>
      <div class="diagram"><div class="mermaid">{{ .ModuleGraphCode }}</div></div>
      {{- end }}
      {{- if .LaravelArchitectureCode }}
      <h3>Laravel layer graph</h3>
      <div class="diagram"><div class="mermaid">{{ .LaravelArchitectureCode }}</div></div>
      {{- end }}
      {{- if and (eq .DependencyGraphCode "") (eq .ModuleGraphCode "") (eq .LaravelArchitectureCode "") }}
      <p>No graphable relationships were detected.</p>
      {{- end }}
    </section>
  </main>
</body>
</html>
`

func ParseFormat(value string) (Format, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "markdown", "md":
		return FormatMarkdown, nil
	case "html":
		return FormatHTML, nil
	case "both":
		return FormatBoth, nil
	default:
		return "", fmt.Errorf("unsupported format %q (expected markdown, html, or both)", value)
	}
}

func ArtifactNames(format Format) []string {
	switch format {
	case FormatMarkdown:
		return append([]string(nil), markdownArtifactNames...)
	case FormatHTML:
		return []string{"index.html"}
	case FormatBoth:
		out := append([]string(nil), markdownArtifactNames...)
		out = append(out, "index.html")
		return out
	default:
		return nil
	}
}

func AllArtifactNames() []string {
	return ArtifactNames(FormatBoth)
}

func GenerateWithOptions(snap model.Snapshot, outDir string, opts GenerateOptions) ([]string, error) {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, err
	}

	format, err := ParseFormat(string(opts.Format))
	if err != nil {
		return nil, err
	}

	data := buildTemplateData(snap)
	templateDir := filepath.Join(snap.ProjectPath, "templates")
	generated := make([]string, 0)

	if format == FormatMarkdown || format == FormatBoth {
		files, err := renderMarkdownDocs(outDir, templateDir, data)
		if err != nil {
			return nil, err
		}
		generated = append(generated, files...)
	}

	if format == FormatHTML || format == FormatBoth {
		file, err := renderHTMLSite(outDir, templateDir, data)
		if err != nil {
			return nil, err
		}
		generated = append(generated, file)
	}

	sort.Strings(generated)
	return generated, nil
}

func buildTemplateData(snap model.Snapshot) templateData {
	managers := make([]string, 0, len(snap.Dependencies))
	dependencySummaries := make([]dependencyManagerSummary, 0, len(snap.Dependencies))
	totalDependencies := 0
	for manager, deps := range snap.Dependencies {
		managers = append(managers, manager)
		dependencySummaries = append(dependencySummaries, dependencyManagerSummary{
			Name:  manager,
			Count: len(deps),
		})
		totalDependencies += len(deps)
	}
	sort.Strings(managers)
	sort.Slice(dependencySummaries, func(i, j int) bool {
		return dependencySummaries[i].Name < dependencySummaries[j].Name
	})

	directoryViews := make([]directorySummaryView, 0, len(snap.DirectoryLayout))
	for _, summary := range snap.DirectoryLayout {
		directoryViews = append(directoryViews, directorySummaryView{
			Path:             summary.Path,
			FileCount:        summary.FileCount,
			SourceFiles:      summary.SourceFiles,
			TestFiles:        summary.TestFiles,
			ManifestFiles:    summary.ManifestFiles,
			ConfigFiles:      summary.ConfigFiles,
			LanguagesText:    listText(summary.Languages),
			NotableFilesText: listText(summary.NotableFiles),
		})
	}

	manifestViews := make([]manifestView, 0, len(snap.ManifestFiles))
	for _, manifest := range snap.ManifestFiles {
		manifestViews = append(manifestViews, manifestView{
			Kind: manifest.Kind,
			Path: manifest.Path,
		})
	}

	apiGroups := make([]apiGroupView, 0, len(snap.APIGroups))
	for _, group := range snap.APIGroups {
		apiGroups = append(apiGroups, apiGroupView{
			Prefix:      group.Prefix,
			RouteCount:  group.RouteCount,
			MethodsText: listText(group.Methods),
		})
	}

	dependencyGraph := buildDependencyGraphMermaid(snap.Dependencies)
	moduleGraph := buildModuleGraphMermaid(snap.ProjectPath)
	laravelGraph := buildLaravelArchitectureMermaid(snap)

	return templateData{
		Snapshot:                   snap,
		ProjectDisplayName:         displayProjectName(snap),
		LanguagesText:              listText(snap.Languages),
		FrameworksText:             listText(snap.Frameworks),
		PackageManagersText:        listText(snap.PackageManagers),
		InfrastructureText:         listText(snap.Infrastructure),
		DependencyManagers:         managers,
		DependencySummaries:        dependencySummaries,
		TotalDependencies:          totalDependencies,
		RouteMethodSummaries:       buildRouteMethodSummaries(snap.Routes),
		ManifestViews:              manifestViews,
		DirectoryViews:             directoryViews,
		APIGroupViews:              apiGroups,
		DependencyGraphMermaid:     dependencyGraph,
		DependencyGraphCode:        stripMermaidFence(dependencyGraph),
		ModuleGraphMermaid:         moduleGraph,
		ModuleGraphCode:            stripMermaidFence(moduleGraph),
		LaravelArchitectureMermaid: laravelGraph,
		LaravelArchitectureCode:    stripMermaidFence(laravelGraph),
	}
}

func renderMarkdownDocs(outDir, templateDir string, data templateData) ([]string, error) {
	generated := make([]string, 0, len(markdownTemplateSources))
	for name, src := range markdownTemplateSources {
		path := filepath.Join(outDir, name)
		templateBody, err := resolveTemplateSource(templateDir, name, src)
		if err != nil {
			return nil, err
		}
		if err := renderFile(path, templateBody, data); err != nil {
			return nil, fmt.Errorf("render %s: %w", name, err)
		}
		generated = append(generated, path)
	}
	return generated, nil
}

func renderHTMLSite(outDir, templateDir string, data templateData) (string, error) {
	path := filepath.Join(outDir, "index.html")
	templateBody, err := resolveTemplateSource(templateDir, "index.html", htmlTemplateSource)
	if err != nil {
		return "", err
	}
	if err := renderHTMLFile(path, templateBody, data); err != nil {
		return "", fmt.Errorf("render index.html: %w", err)
	}
	return path, nil
}

func renderHTMLFile(path, src string, data templateData) error {
	tmpl, err := htmltemplate.New(filepath.Base(path)).Parse(src)
	if err != nil {
		return err
	}

	var b strings.Builder
	if err := tmpl.Execute(&b, data); err != nil {
		return err
	}

	return os.WriteFile(path, []byte(strings.TrimSpace(b.String())+"\n"), 0o644)
}

func displayProjectName(snap model.Snapshot) string {
	if strings.TrimSpace(snap.ProjectName) != "" {
		return snap.ProjectName
	}
	base := filepath.Base(strings.TrimSpace(snap.ProjectPath))
	if base == "" || base == "." || base == "/" {
		return "Project"
	}
	return base
}

func listText(items []string) string {
	if len(items) == 0 {
		return "n/a"
	}
	return strings.Join(items, ", ")
}

func buildRouteMethodSummaries(routes []model.Route) []routeMethodSummary {
	if len(routes) == 0 {
		return nil
	}

	counts := map[string]int{}
	for _, route := range routes {
		method := strings.ToUpper(strings.TrimSpace(route.Method))
		if method == "" {
			method = "UNKNOWN"
		}
		counts[method]++
	}

	methods := make([]string, 0, len(counts))
	for method := range counts {
		methods = append(methods, method)
	}
	sort.Strings(methods)

	out := make([]routeMethodSummary, 0, len(methods))
	for _, method := range methods {
		out = append(out, routeMethodSummary{
			Method: method,
			Count:  counts[method],
		})
	}
	return out
}

func stripMermaidFence(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	value = strings.TrimPrefix(value, "~~~mermaid")
	value = strings.TrimPrefix(value, "```mermaid")
	value = strings.TrimSuffix(value, "~~~")
	value = strings.TrimSuffix(value, "```")
	return strings.TrimSpace(value)
}
