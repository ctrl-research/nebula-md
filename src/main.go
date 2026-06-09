package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Config — adjust these to match your vault layout.
// BASALT_INPUT and BASALT_OUTPUT override these defaults.
var (
	SourceDir = getSourceDir()
	OutputDir = getOutputDir()
)

func getSourceDir() string {
	if v := os.Getenv("BASALT_INPUT"); v != "" {
		return v
	}
	return "../vault"
}

func getOutputDir() string {
	if v := os.Getenv("BASALT_OUTPUT"); v != "" {
		return v
	}
	return "../output"
}

// ignoredDirs holds directory names to skip during vault walks
var ignoredDirs []string

func isIgnored(path string) bool {
	parts := strings.Split(filepath.ToSlash(path), "/")
	for _, part := range parts {
		for _, ign := range ignoredDirs {
			if part == ign {
				return true
			}
		}
	}
	return false
}

// GraphMode controls the graph renderer style.
type GraphMode string

const (
	GraphMode2D     GraphMode = "2d"
	GraphModeNebula GraphMode = "nebula"
)

// SiteConfig holds site-level configuration from .env or environment variables.
type SiteConfig struct {
	SiteName              string    // displayed in header
	SiteTheme             string    // "dark" or "light"
	IgnoredDirs           []string  // directory names to skip during vault walk
	GraphNodeSizeByEdges  bool      // size graph nodes by edge count
	ShowLinks            bool      // show links/backlinks sidebar section
	FeatureAutoFolderMOC bool      // auto-add sibling links to folder index pages
	GraphMode             GraphMode // "2d" (default) or "nebula" (3D galaxy)
}

// readConfig reads site configuration from .env and environment variables.
// Environment variables (BASALT_SITE_NAME, BASALT_SITE_THEME) override .env file values.
func readConfig() SiteConfig {
	cfg := SiteConfig{SiteName: "Basalt", SiteTheme: "dark", GraphNodeSizeByEdges: true, GraphMode: GraphMode2D}
	for _, envPath := range []string{".env", "../.env", "../../.env"} {
		if _, err := os.Stat(envPath); err == nil {
			if data, err := os.ReadFile(envPath); err == nil {
				for _, line := range strings.Split(string(data), "\n") {
					line = strings.TrimSpace(line)
					if line == "" || strings.HasPrefix(line, "#") {
						continue
					}
					eq := strings.Index(line, "=")
					if eq <= 0 {
						continue
					}
					key := strings.TrimSpace(line[:eq])
					val := strings.TrimSpace(line[eq+1:])
					if len(val) >= 2 {
						if (val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'') {
							val = val[1 : len(val)-1]
						}
					}
					if key == "BASALT_SITE_NAME" {
						cfg.SiteName = val
					} else if key == "BASALT_SITE_THEME" && (val == "light" || val == "dark") {
						cfg.SiteTheme = val
} else if key == "BASALT_GRAPH_NODE_SIZE_BY_EDGES" {
					cfg.GraphNodeSizeByEdges = val == "true" || val == "1" || val == "yes"
				} else if key == "BASALT_SHOW_LINKS" {
					cfg.ShowLinks = val == "true" || val == "1" || val == "yes"
				} else if key == "BASALT_FEATURE_AUTO_FOLDER_MOC" {
					cfg.FeatureAutoFolderMOC = val == "true" || val == "1" || val == "yes"
				} else if key == "BASALT_GRAPH_MODE" && (val == "2d" || val == "nebula") {
					cfg.GraphMode = GraphMode(val)
				}
				}
			}
			break
		}
	}
	if v := os.Getenv("BASALT_SITE_NAME"); v != "" {
		cfg.SiteName = v
	}
	if v := os.Getenv("BASALT_SITE_THEME"); v == "light" || v == "dark" {
		cfg.SiteTheme = v
	}
	// Parse ignored directories (comma-separated)
	if v := os.Getenv("BASALT_IGNORED_DIRS"); v != "" {
		for _, part := range strings.Split(v, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				cfg.IgnoredDirs = append(cfg.IgnoredDirs, part)
			}
		}
	}
	// Parse boolean flags
	if v := os.Getenv("BASALT_GRAPH_NODE_SIZE_BY_EDGES"); v != "" {
		cfg.GraphNodeSizeByEdges = v == "true" || v == "1" || v == "yes"
	}
	if v := os.Getenv("BASALT_SHOW_LINKS"); v != "" {
		cfg.ShowLinks = v == "true" || v == "1" || v == "yes"
	}
	if v := os.Getenv("BASALT_FEATURE_AUTO_FOLDER_MOC"); v != "" {
		cfg.FeatureAutoFolderMOC = v == "true" || v == "1" || v == "yes"
	}
	if v := os.Getenv("BASALT_GRAPH_MODE"); v == "nebula" {
		cfg.GraphMode = GraphModeNebula
	} else {
		cfg.GraphMode = GraphMode2D
	}
	return cfg
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	if _, err := os.Stat(SourceDir); os.IsNotExist(err) {
		return fmt.Errorf("source directory %q not found", SourceDir)
	}

	if err := os.MkdirAll(OutputDir, 0755); err != nil {
		return fmt.Errorf("creating output dir: %w", err)
	}

	graphDir := filepath.Join(OutputDir, "graph")
	if err := os.MkdirAll(graphDir, 0755); err != nil {
		return fmt.Errorf("creating graph dir: %w", err)
	}

	fmt.Println("Building Basalt Site...")

	siteCfg := readConfig()
	fmt.Printf("Config: site_name=%q theme=%q\n", siteCfg.SiteName, siteCfg.SiteTheme)
	ignoredDirs = siteCfg.IgnoredDirs

	// Build full vault graph (computes all pages, edges, writes backlinks.json)
	graph, _, pageTitles, err := buildGraph(SourceDir)
	if err != nil {
		return fmt.Errorf("building graph: %w", err)
	}

	graphJSON, err := json.MarshalIndent(graph, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling graph: %w", err)
	}
	if err := os.WriteFile(filepath.Join(OutputDir, "graph.json"), graphJSON, 0644); err != nil {
		return fmt.Errorf("writing graph.json: %w", err)
	}
	fmt.Printf("Graph: %d nodes, %d edges\n", len(graph.Nodes), len(graph.Edges))

	// Load the set of all existing pages for stub detection
	existingPages := make(map[string]bool)
	for _, node := range graph.Nodes {
		if !node.Stub {
			existingPages[node.ID] = true
		}
	}

	// Load backlinks map for per-page backlink lookup
	backlinksMap := loadBacklinks()

	// Build navigation tree for left sidebar
	navTree := buildNavTree(SourceDir)
	navTreeJSON, _ := json.Marshal(navTree)

	// Build folder siblings map if feature flag is enabled
	var folderSiblings map[string][]string
	if siteCfg.FeatureAutoFolderMOC {
		folderSiblings = buildFolderSiblings(graph.Nodes)
		// Add sibling edges to global graph
		for indexID, sibs := range folderSiblings {
			for _, sib := range sibs {
				graph.Edges = append(graph.Edges, GraphEdge{Source: indexID, Target: sib})
			}
		}
		// Re-write graph.json with sibling edges included
		graphJSON, _ = json.MarshalIndent(graph, "", "  ")
		os.WriteFile(filepath.Join(OutputDir, "graph.json"), graphJSON, 0644)
	}

	parser := NewMarkdownParser()

	// Walk the vault and generate HTML for each markdown file
	err = filepath.Walk(SourceDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !strings.HasSuffix(path, ".md") || isIgnored(path) {
			return nil
		}

		relPath, _ := filepath.Rel(SourceDir, path)

		// Read raw content for tag extraction before ProcessFile strips frontmatter
		rawContent, _ := os.ReadFile(path)
		tags := extractTags(rawContent)
		date := extractDate(rawContent)

		title, htmlBody, linkTargets, linkHrefs, err := parser.ProcessFile(path, relPath)
		if err != nil {
			fmt.Printf("Error processing %s: %v\n", path, err)
			return nil
		}

		readingTime := computeReadingTime(htmlBody)

		pageID := filepath.Join(filepath.Dir(relPath), toHTMLName(relPath))
		outputSubdir := filepath.Join(OutputDir, filepath.Dir(relPath))
		if merr := os.MkdirAll(outputSubdir, 0755); merr != nil {
			return merr
		}

		// Build per-page graph data
		pageGraph := buildPageGraph(pageID, linkTargets, linkHrefs, backlinksMap, existingPages, pageTitles, folderSiblings, tags)
		pageGraph.CurrentHref = pageID + ".html"
		pageGraph.TableOfContents = extractTOC(htmlBody)
		pageGraph.Date = date
		pageGraph.ReadingTime = readingTime

		// Write HTML page
		outputFile := filepath.Join(OutputDir, pageID+".html")
		html := generateHTMLTemplate(title, string(htmlBody), relPath, pageGraph, string(navTreeJSON), siteCfg)
		if werr := os.WriteFile(outputFile, []byte(html), 0644); werr != nil {
			return werr
		}
		fmt.Printf("Generated: %s\n", outputFile)

		return nil
	})
	if err != nil {
		return fmt.Errorf("walking vault: %w", err)
	}

	// Generate stub pages for dead link targets
	stubCount := 0
	for _, node := range graph.Nodes {
		if node.Stub {
			stubFile := filepath.Join(OutputDir, node.ID+".html")
			if err := os.WriteFile(stubFile, []byte(generateStubHTML(node.ID)), 0644); err != nil {
				return err
			}
			fmt.Printf("Stubbed: %s\n", stubFile)
			stubCount++
		}
	}
	if stubCount > 0 {
		fmt.Printf("Generated %d stub pages\n", stubCount)
	}

	// Build and write search index
	searchIndex := buildSearchIndex(SourceDir)
	searchJSON, _ := json.MarshalIndent(searchIndex, "", "  ")
	if err := os.WriteFile(filepath.Join(OutputDir, "search.json"), searchJSON, 0644); err != nil {
		return fmt.Errorf("writing search.json: %w", err)
	}
	fmt.Printf("Search index: %d pages\n", len(searchIndex))

	writeGraphViewer(graphDir, graphJSON, siteCfg.SiteTheme, siteCfg.SiteName, siteCfg.GraphNodeSizeByEdges, siteCfg.GraphMode)
	fmt.Println("Build complete.")
	return nil
}

// buildFolderSiblings builds a map: folder index pageID -> all sibling pageIDs in that folder (excl. index itself).
// e.g. recipes/index -> [recipes/chicken, recipes/beef, ...]
func buildFolderSiblings(nodes []GraphNode) map[string][]string {
	// First pass: group pages by their folder
	folderPages := make(map[string][]string) // folderDir -> [pageIDs]
	for _, n := range nodes {
		if n.Stub {
			continue
		}
		dir := filepath.Dir(n.ID)
		if dir == "." {
			dir = ""
		}
		folderPages[dir] = append(folderPages[dir], n.ID)
	}

	// Second pass: for each folder with an index, siblings = all non-index pages in that folder
	siblings := make(map[string][]string)
	for _, pages := range folderPages {
		var indexID string
		var nonIndex []string
		for _, p := range pages {
			if toHTMLName(p) == "index" {
				indexID = p
			} else {
				nonIndex = append(nonIndex, p)
			}
		}
		if indexID != "" && len(nonIndex) > 0 {
			siblings[indexID] = nonIndex
		}
	}
	return siblings
}

// buildPageGraph builds the per-page graph data for a given page:
// - Links: pages this page wiki-links to (plus sibling folder pages for folder indexes)
// - Backlinks: pages that link to this page
func buildPageGraph(pageID string, linkTargets []string, linkHrefs []string, backlinksMap map[string][]string, existingPages map[string]bool, pageTitles map[string]string, folderSiblings map[string][]string, tags []string) *PageGraph {
	pg := &PageGraph{Links: []GraphRef{}, Backlinks: []GraphRef{}, Tags: tags}

	// Build Links — use linkHrefs (computed relative hrefs) not bare target paths
	for i, target := range linkTargets {
		href := target + ".html" // fallback
		if i < len(linkHrefs) {
			href = linkHrefs[i] + ".html"
		}
		title := toHTMLName(target)
		if t, ok := pageTitles[target]; ok && t != "" {
			title = t
		}
		if title == "index" {
			parts := strings.Split(target, "/")
			if len(parts) > 1 {
				title = parts[len(parts)-2]
			}
		}
		pg.Links = append(pg.Links, GraphRef{
			Title: title,
			Href:  href,
			Stub:  !existingPages[target],
		})
	}

	// For folder index pages, add links to all sibling pages in the same folder
	if sibs, ok := folderSiblings[pageID]; ok {
		for _, sib := range sibs {
			title := toHTMLName(sib)
			if t, ok := pageTitles[sib]; ok && t != "" {
				title = t
			}
			pg.Links = append(pg.Links, GraphRef{
				Title: title,
				Href:  computeRelHref(pageID, sib),
				Stub:  false,
			})
		}
	}

	// Build Backlinks — compute relative hrefs from this page's directory
	for _, source := range backlinksMap[pageID] {
		title := toHTMLName(source)
		if t, ok := pageTitles[source]; ok && t != "" {
			title = t
		}
		// If source is an index page, use folder name
		if title == "index" {
			parts := strings.Split(source, "/")
			if len(parts) > 1 {
				title = parts[len(parts)-2]
			}
		}
		pg.Backlinks = append(pg.Backlinks, GraphRef{
			Title: title,
			Href:  computeRelHref(pageID, source),
			Stub:  !existingPages[source],
		})
	}

	return pg
}

// loadBacklinks reads the backlinks map from backlinks.json
func loadBacklinks() map[string][]string {
	var m map[string][]string
	data, err := os.ReadFile(filepath.Join(OutputDir, "backlinks.json"))
	if err != nil {
		return make(map[string][]string)
	}
	if err := json.Unmarshal(data, &m); err != nil {
		return make(map[string][]string)
	}
	return m
}
