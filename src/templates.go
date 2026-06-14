package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// generateHTMLTemplate produces the full HTML page for a rendered markdown file.
// navTreeJSON is the hierarchical navigation tree as JSON.
func generateHTMLTemplate(title string, htmlContent string, sourcePath string, pageGraph *PageGraph, navTreeJSON string, siteCfg SiteConfig) string {
	pageGraphJSON, _ := json.Marshal(pageGraph)
	backlinksHTML := buildBacklinksHTML(pageGraph, siteCfg.ShowLinks)
	tagsHTML := buildTagsHTML(pageGraph)
	tocHTML := buildTocHTML(pageGraph)
	siteNameJS, _ := json.Marshal(siteCfg.SiteName)
	graphModeJS, _ := json.Marshal(siteCfg.GraphMode)

	css := `
	/* Google Fonts: Libre Baskerville for headings and site name, Lilex for body */
	@import url('https://fonts.googleapis.com/css2?family=Lilex:wght@300;400;500;600&family=Libre+Baskerville:ital,wght@0,400;0,700;1,400&display=swap');

	/* Dark mode — deep black with subtle blue undertone */
	:root, [data-theme="dark"] { --bg: #161a22; --text: #e0e0e0; --link: #6bb3d9; --sidebar-bg: #1a1f2a; --border: #2a3040; --heading: #ffffff; --muted: #888888; --card-bg: #1e2432; --graph-node: #999; }
	/* Light mode */
	[data-theme="light"] { --bg: #f8f8f8; --text: #333; --link: #2980b9; --sidebar-bg: #f0f0f0; --border: #e1e4e8; --heading: #1a1a1a; --muted: #888888; --card-bg: #ffffff; --graph-node: #ccc; }
	* { box-sizing: border-box; }
	body { font-family: 'Lilex', -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; line-height: 1.6; margin: 0; background: var(--bg); color: var(--text); font-weight: 400; }
	h1, h2, h3, h4, h5, h6 { font-family: 'Libre Baskerville', Georgia, 'Times New Roman', serif; font-weight: 700; }
	.layout { display: grid; grid-template-columns: 1fr 2fr 1fr; width: 100%; max-width: 100vw; align-items: start; }
	/* Mobile nav toggle */
	.mobile-nav-toggle { display: none; }
	.mobile-header { display: none; }
	@media (max-width: 768px) {
		.mobile-nav-toggle { display: block; background: var(--sidebar-bg); border: 1px solid var(--border); color: var(--text); border-radius: 6px; padding: 8px 12px; font-size: 1.2em; cursor: pointer; }
		.mobile-header { position: fixed; top: 0; left: 0; right: 0; z-index: 998; display: flex; align-items: center; gap: 8px; padding: 8px 12px; background: var(--sidebar-bg); border-bottom: 1px solid var(--border); height: 52px; box-sizing: border-box; }
		.mobile-header .mobile-site-name { flex: 1; font-size: 1em; font-weight: 600; color: var(--heading); margin: 0; padding: 0; border: none; }
		.layout { grid-template-columns: 1fr; padding-top: 52px; }
		.sidebar-nav {
			position: fixed; top: 0; left: 0; height: 100vh; width: 100vw; z-index: 1000;
			transform: translateX(-100%); transition: transform 0.25s ease;
			box-shadow: 2px 0 8px rgba(0,0,0,0.3);
			display: none;
		}
		.sidebar-nav.open { transform: translateX(0); display: block; }
		.sidebar-nav.closed { transform: translateX(-100%); display: none; }
		.content-col { padding: 16px 20px; align-self: start; }
		.page-footer { text-align: center; }
		.sidebar-right { display: block; border-left: none; border-top: 1px solid var(--border); position: static; margin-top: 0; }
		.sidebar-right .sidebar-section { margin-bottom: 8px; }
		.graph-header button { font-size: 1.8em; }
	}
	/* Left sidebar — nav */
	.sidebar-nav { background: var(--sidebar-bg); border-right: 1px solid var(--border); padding: 20px 16px; position: sticky; top: 0; height: 100vh; overflow-y: auto; }
	.sidebar-nav h2 { margin: 0 0 12px; font-size: 0.8em; text-transform: uppercase; letter-spacing: 0.05em; color: var(--muted); }
	.nav-tree { font-size: 0.9em; }
	.nav-folder { margin: 4px 0; }
	.nav-folder-header { cursor: pointer; padding: 4px 6px; border-radius: 4px; display: flex; align-items: center; gap: 4px; color: var(--text); user-select: none; }
	.nav-folder-header:hover { background: rgba(255,255,255,0.05); }
	[data-theme="light"] .nav-folder-header:hover { background: rgba(0,0,0,0.05); }
	.nav-folder-header .icon { font-size: 0.7em; transition: transform 0.15s; display: inline-block; }
	.nav-folder-header .icon.open { transform: rotate(90deg); }
	.nav-folder-children { padding-left: 16px; display: none; }
	.nav-folder-children.open { display: block; }
	.nav-page { padding: 4px 6px; border-radius: 4px; }
	.nav-page a, .nav-folder-header a { color: var(--link); text-decoration: none; font-weight: 400; }
	.nav-page a:visited, .nav-folder-header a:visited { color: var(--link); }
	.nav-page a:hover, .nav-folder-header a:hover { text-decoration: underline; }
	.nav-page.active a { font-weight: 700; text-decoration: underline; color: var(--link); }
	/* Center content */
	.content-col { padding: 20px; min-width: 0; }
	.content-col h1 { border-bottom: 1px solid var(--border); padding-bottom: 10px; margin: 0 0 6px; font-size: 1.5em; color: var(--heading); }
	.page-meta { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; font-size: 0.8em; color: var(--muted); }
	.page-meta-left { font-style: italic; }
	.page-meta-right { font-style: normal; }
	.markdown-body { background: var(--card-bg); padding: 24px; border-radius: 6px; box-shadow: 0 1px 3px rgba(0,0,0,0.08); }
	.markdown-body p, .markdown-body li { font-size: 16px; }
	.markdown-body h2 { margin-top: 28px; color: var(--heading); }
	.markdown-body h3 { color: var(--heading); }
	.markdown-body a { color: var(--link); text-decoration: none; font-weight: 500; }
	.markdown-body a:hover { text-decoration: underline; }
	.markdown-body img { max-width: 100%; height: auto; display: block; margin: 0 auto; }
	/* Markdown tables */
	.markdown-body table { border-collapse: collapse; width: 100%; margin: 16px 0; font-size: 0.9em; }
	.markdown-body table th { background: var(--sidebar-bg); color: var(--heading); font-weight: 600; text-align: left; padding: 8px 12px; border: 1px solid var(--border); }
	.markdown-body table td { padding: 8px 12px; border: 1px solid var(--border); }
	.markdown-body table tr:nth-child(even) { background: rgba(255,255,255,0.03); }
	[data-theme="light"] .markdown-body table tr:nth-child(even) { background: rgba(0,0,0,0.03); }
	.markdown-body table tr:hover { background: rgba(107,179,217,0.08); }
	[data-theme="light"] .markdown-body table tr:hover { background: rgba(41,128,185,0.08); }
	.markdown-body figure.image-caption { margin: 16px 0; text-align: center; }
	.markdown-body figure.image-caption img { margin: 0 auto 8px; }
	.markdown-body figure.image-caption figcaption { font-style: italic; color: var(--muted); font-size: 0.9em; margin-top: 4px; }
	/* Page footer */
	.page-footer { margin-top: 24px; padding: 12px 0; border-top: 1px solid var(--border); font-size: 0.8em; color: var(--muted); text-align: center; }
	.page-footer a { color: var(--link); text-decoration: none; }
	.page-footer a:hover { text-decoration: underline; }
	/* Right sidebar */
	.sidebar-right { background: var(--sidebar-bg); border-left: 1px solid var(--border); padding: 20px 16px; position: sticky; top: 0; height: 100vh; overflow-y: auto; }
	.sidebar-right h2 { margin: 0; font-size: 0.8em; text-transform: uppercase; letter-spacing: 0.05em; color: var(--muted); }
	.graph-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
	.graph-header button { background: none; border: none; color: var(--muted); cursor: pointer; font-size: 1.35em; padding: 0; line-height: 1; }
	.graph-header button:hover { color: var(--text); }
	.full-graph-overlay { position: fixed; top: 0; left: 0; right: 0; bottom: 0; background: rgba(0,0,0,0.7); z-index: 1000; display: flex; align-items: center; justify-content: center; }
	.full-graph-modal { background: var(--sidebar-bg); border: 1px solid var(--border); border-radius: 8px; width: 90vw; height: 85vh; display: flex; flex-direction: column; overflow: hidden; }
	.full-graph-header { display: flex; justify-content: space-between; align-items: center; padding: 12px 16px; border-bottom: 1px solid var(--border); }
	.full-graph-header h2 { margin: 0; font-size: 0.9em; color: var(--heading); text-transform: uppercase; letter-spacing: 0.05em; }
	.full-graph-header button { background: none; border: none; color: var(--muted); cursor: pointer; font-size: 1.2em; padding: 0; line-height: 1; }
	.full-graph-header button:hover { color: var(--text); }
	#full-graph-container { flex: 1; overflow: hidden; }
	#full-graph-container iframe { width: 100%; height: 100%; border: none; background: var(--bg); }
	.full-graph-iframe { width: 100%; height: 100%; border: none; }
	#local-graph { width: 100%; height: 180px; background: var(--card-bg); border: 1px solid var(--border); border-radius: 6px; margin-bottom: 16px; }
	#local-graph circle { fill: #ccc !important; }
	#local-graph .node.hovered circle, #local-graph .node.neighbor circle { fill: var(--link) !important; }
	#local-graph .node.dimmed circle { opacity: 0.15; }
	#local-graph .node.dimmed circle { opacity: 0.15; }
	.sidebar-section { margin-bottom: 16px; }
	.sidebar-section h3 { margin: 0 0 8px; font-size: 0.75em; text-transform: uppercase; letter-spacing: 0.05em; color: var(--muted); }
	.sidebar-links { background: var(--card-bg); border: 1px solid var(--border); border-radius: 6px; padding: 12px; font-size: 0.85em; }
	.sidebar-links ul { margin: 0; padding-left: 16px; }
	.sidebar-links li { margin: 4px 0; }
	.sidebar-links a { color: var(--link); text-decoration: none; font-weight: 500; }
	.sidebar-links a:hover { text-decoration: underline; }
	a.stub-link { color: #e67e22; font-style: italic; }
	.tags { margin-top: 16px; display: flex; flex-wrap: wrap; gap: 6px; align-items: center; }
	.tags-label { font-size: 0.8em; color: var(--muted); margin-right: 4px; }
	.tag { display: inline-block; padding: 2px 8px; background: var(--link); color: var(--bg); border-radius: 12px; font-size: 0.8em; font-weight: 500; opacity: 0.85; }
	/* Callouts — Obsidian-style */
	/* Callout base styles — matches Obsidian */
	.callout { border: 1px solid var(--callout-border, var(--border)); border-left: 3px solid var(--callout-color, var(--link)); border-radius: 6px; margin: 16px 0; background: var(--callout-bg, var(--card-bg)); }
	.callout-title { display: flex; align-items: center; gap: 8px; padding: 8px 12px; background: var(--callout-title-bg, var(--sidebar-bg)); color: var(--callout-color, var(--link)); border-bottom: 1px solid var(--callout-border, var(--border)); font-weight: 600; font-size: 0.875em; }
	.callout-icon { display: inline-flex; align-items: center; flex-shrink: 0; }
	.callout-icon svg { width: 16px; height: 16px; }
	.callout-content { padding: 12px; }
	.callout-content > p:last-child { margin-bottom: 0; }
	.callout-content > *:last-child { margin-bottom: 0; }
	/* Callout type colors — light mode */
	:root, [data-theme="light"] {
		--callout-note-bg: #e7f2f8; --callout-note-border: #6bb3d9; --callout-note-color: #1e6a99; --callout-note-title-bg: #d4eaf4;
		--callout-tip-bg: #e8f8ed; --callout-tip-border: #2ecc71; --callout-tip-color: #1e7a44; --callout-tip-title-bg: #d4f2de;
		--callout-warning-bg: #fef9e7; --callout-warning-border: #f39c12; --callout-warning-color: #8a6d0a; --callout-warning-title-bg: #fdf0c4;
		--callout-danger-bg: #fdeaea; --callout-danger-border: #e74c3c; --callout-danger-color: #a52a1e; --callout-danger-title-bg: #fdddd9;
		--callout-example-bg: #f5e6ff; --callout-example-border: #9b59b6; --callout-example-color: #6a3286; --callout-example-title-bg: #eddbfe;
		--callout-info-bg: #e7f2f8; --callout-info-border: #6bb3d9; --callout-info-color: #1e6a99; --callout-info-title-bg: #d4eaf4;
		--callout-success-bg: #e8f8ed; --callout-success-border: #2ecc71; --callout-success-color: #1e7a44; --callout-success-title-bg: #d4f2de;
		--callout-question-bg: #e7f2f8; --callout-question-border: #3498db; --callout-question-color: #1e5a89; --callout-question-title-bg: #d4e4f4;
		--callout-default-bg: var(--card-bg); --callout-default-border: var(--link); --callout-default-color: var(--link); --callout-default-title-bg: var(--sidebar-bg);
	}
	/* Callout type colors — dark mode */
	[data-theme="dark"] {
		--callout-note-bg: #1a3a4a; --callout-note-border: #4a9ac4; --callout-note-color: #9dd5f5; --callout-note-title-bg: #1e4a5a;
		--callout-tip-bg: #1a3d2e; --callout-tip-border: #27ae60; --callout-tip-color: #8ee5a0; --callout-tip-title-bg: #1e4d3a;
		--callout-warning-bg: #3d3018; --callout-warning-border: #d68910; --callout-warning-color: #f9e0a0; --callout-warning-title-bg: #4a3a1e;
		--callout-danger-bg: #3d1a1a; --callout-danger-border: #c0392b; --callout-danger-color: #f5a0a0; --callout-danger-title-bg: #4a1e1e;
		--callout-example-bg: #3a1a3d; --callout-example-border: #8e44ad; --callout-example-color: #d5a0f5; --callout-example-title-bg: #4a1e4a;
		--callout-info-bg: #1a3a4a; --callout-info-border: #4a9ac4; --callout-info-color: #9dd5f5; --callout-info-title-bg: #1e4a5a;
		--callout-success-bg: #1a3d2e; --callout-success-border: #27ae60; --callout-success-color: #8ee5a0; --callout-success-title-bg: #1e4d3a;
		--callout-question-bg: #1a3a4a; --callout-question-border: #2980b9; --callout-question-color: #9dd5f5; --callout-question-title-bg: #1e4a5a;
		--callout-default-bg: var(--card-bg); --callout-default-border: var(--link); --callout-default-color: var(--link); --callout-default-title-bg: var(--sidebar-bg);
	}
	/* Callout type data-attribute selectors */
	.callout[data-callout="note"] { --callout-bg: var(--callout-note-bg); --callout-border: var(--callout-note-border); --callout-color: var(--callout-note-color); --callout-title-bg: var(--callout-note-title-bg); }
	.callout[data-callout="tip"], .callout[data-callout="hint"], .callout[data-callout="important"] { --callout-bg: var(--callout-tip-bg); --callout-border: var(--callout-tip-border); --callout-color: var(--callout-tip-color); --callout-title-bg: var(--callout-tip-title-bg); }
	.callout[data-callout="warning"], .callout[data-callout="caution"], .callout[data-callout="attention"] { --callout-bg: var(--callout-warning-bg); --callout-border: var(--callout-warning-border); --callout-color: var(--callout-warning-color); --callout-title-bg: var(--callout-warning-title-bg); }
	.callout[data-callout="danger"], .callout[data-callout="error"] { --callout-bg: var(--callout-danger-bg); --callout-border: var(--callout-danger-border); --callout-color: var(--callout-danger-color); --callout-title-bg: var(--callout-danger-title-bg); }
	.callout[data-callout="example"] { --callout-bg: var(--callout-example-bg); --callout-border: var(--callout-example-border); --callout-color: var(--callout-example-color); --callout-title-bg: var(--callout-example-title-bg); }
	.callout[data-callout="info"] { --callout-bg: var(--callout-info-bg); --callout-border: var(--callout-info-border); --callout-color: var(--callout-info-color); --callout-title-bg: var(--callout-info-title-bg); }
	.callout[data-callout="success"], .callout[data-callout="check"], .callout[data-callout="done"] { --callout-bg: var(--callout-success-bg); --callout-border: var(--callout-success-border); --callout-color: var(--callout-success-color); --callout-title-bg: var(--callout-success-title-bg); }
	.callout[data-callout="question"], .callout[data-callout="help"], .callout[data-callout="faq"] { --callout-bg: var(--callout-question-bg); --callout-border: var(--callout-question-border); --callout-color: var(--callout-question-color); --callout-title-bg: var(--callout-question-title-bg); }
	/* Foldable callout (details/summary — matches Obsidian) */
	.callout details { border: none; background: transparent; }
	.callout summary { display: flex; align-items: center; gap: 8px; padding: 8px 12px; background: var(--callout-title-bg, var(--sidebar-bg)); color: var(--callout-color, var(--link)); cursor: pointer; font-weight: 600; font-size: 0.875em; list-style: none; user-select: none; }
	.callout summary::-webkit-details-marker { display: none; }
	.callout summary::before { content: "▶"; font-size: 0.7em; transition: transform 0.15s; }
	.callout[open] summary::before { transform: rotate(90deg); }
	.callout summary .callout-icon { display: inline-flex; align-items: center; flex-shrink: 0; }
	.callout summary .callout-icon svg { width: 16px; height: 16px; }
	.callout summary .callout-title { padding: 0; background: transparent; border: none; }
	.callout summary.callout-icon { display: flex; align-items: center; margin-right: 8px; }
	.callout summary.callout-icon svg { flex-shrink: 0; }
	/* Collapsible sections */
	details.collapsible { border: 1px solid var(--border); border-radius: 4px; margin: 16px 0; overflow: hidden; }
	details.collapsible summary { padding: 10px 14px; background: var(--card-bg); cursor: pointer; font-weight: 600; font-size: 0.95em; user-select: none; list-style: none; }
	details.collapsible summary::-webkit-details-marker { display: none; }
	details.collapsible summary::before { content: "▶ "; font-size: 0.8em; transition: transform 0.15s; display: inline-block; }
	details.collapsible[open] summary::before { transform: rotate(90deg); }
	details.collapsible .collapsible-content { padding: 12px 14px; border-top: 1px solid var(--border); }
	details.collapsible .collapsible-content p:last-child { margin-bottom: 0; }
	/* Table of contents */
	.toc { margin-top: 16px; font-size: 0.85em; }
	.toc h3 { margin: 0 0 8px; font-size: 0.75em; text-transform: uppercase; letter-spacing: 0.05em; color: var(--muted); }
	.toc-list { list-style: none; margin: 0; padding: 0; }
	.toc-item { margin: 2px 0; }
	.toc-item a { color: var(--link); text-decoration: none; }
	.toc-item a:hover { text-decoration: underline; }
	.toc-item.level-1 { padding-left: 0; }
	.toc-item.level-2 { padding-left: 12px; }
	.toc-item.level-3 { padding-left: 24px; }
	.toc-item.level-4 { padding-left: 36px; }
	.toc-item.level-5 { padding-left: 48px; }
	.toc-item.level-6 { padding-left: 60px; }
	/* Theme toggle */
	.site-name { border-bottom: 1px solid var(--border); padding-bottom: 10px; margin: 0 0 12px; font-size: 1.5em; font-weight: 700; color: var(--heading); padding-left: 6px; font-family: 'Libre Baskerville', Georgia, 'Times New Roman', serif; }
	.sidebar-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
	.search-bar { width: 100%; background: var(--card-bg); border: 1px solid var(--border); color: var(--muted); cursor: pointer; padding: 6px 10px; border-radius: 4px; font-size: 0.85em; text-align: left; margin-bottom: 12px; display: flex; align-items: center; justify-content: space-between; }
	.search-bar .icon { font-size: 2em; }
	.search-bar:hover { border-color: var(--link); color: var(--text); }
	.search-overlay { position: fixed; top: 0; left: 0; right: 0; bottom: 0; background: rgba(0,0,0,0.7); z-index: 1000; display: flex; align-items: flex-start; justify-content: center; padding-top: 10vh; }
	.search-modal { background: var(--sidebar-bg); border: 1px solid var(--border); border-radius: 8px; width: 90vw; max-width: 600px; max-height: 80vh; display: flex; flex-direction: column; overflow: hidden; }
	.search-header { display: flex; align-items: center; border-bottom: 1px solid var(--border); padding: 12px 16px; gap: 12px; }
	#search-input { flex: 1; background: none; border: none; color: var(--text); font-size: 1em; outline: none; }
	#search-input::placeholder { color: var(--muted); }
	#close-search { background: none; border: none; color: var(--muted); cursor: pointer; font-size: 1.2em; padding: 0; line-height: 1; }
	#close-search:hover { color: var(--text); }
	#search-results { overflow-y: auto; padding: 8px; }
	.search-result { display: block; padding: 10px 12px; border-radius: 4px; text-decoration: none; color: var(--text); }
	.search-result:hover { background: var(--card-bg); }
	.search-result-title { font-weight: 600; margin-bottom: 4px; color: var(--heading); }
	.search-result-tags { margin-bottom: 4px; display: flex; flex-wrap: wrap; gap: 6px; }
	.search-result-snippet { font-size: 0.8em; color: var(--muted); line-height: 1.4; }
	.search-result-snippet mark { background: rgba(255,220,50,0.3); color: inherit; border-radius: 2px; }
	.search-empty { padding: 20px; text-align: center; color: var(--muted); font-size: 0.9em; }
	.sidebar-header h2 { margin: 0; }
	.theme-toggle { background: none; border: none; color: var(--muted); cursor: pointer; padding: 0; font-size: 1.2em; line-height: 1; display: flex; align-items: center; justify-content: center; width: 24px; height: 24px; }
	.theme-toggle:hover { color: var(--text); }
	.theme-toggle svg { width: 1em; height: 1em; }
	/* Embedded graph (via %% graph %% / <!-- graph --> directive in a page) */
	.graph-embed { width: 100%; height: 480px; margin: 1.5em 0; border: 1px solid var(--border); border-radius: 8px; overflow: hidden; background: #06060f; }
	.graph-embed iframe { display: block; width: 100%; height: 100%; border: 0; }
	`

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en" data-theme="%[13]s">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <link rel="icon" href="data:image/svg+xml,<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 32 32'><circle cx='16' cy='16' r='14' fill='%%23161a22' stroke='%%236bb3d9' stroke-width='2'/><circle cx='10' cy='12' r='2.5' fill='%%236bb3d9'/><circle cx='22' cy='12' r='2.5' fill='%%236bb3d9'/><circle cx='16' cy='22' r='2.5' fill='%%236bb3d9'/><line x1='10' y1='12' x2='16' y2='22' stroke='%%236bb3d9' stroke-width='1.5'/><line x1='22' y1='12' x2='16' y2='22' stroke='%%236bb3d9' stroke-width='1.5'/><line x1='10' y1='12' x2='22' y2='12' stroke='%%236bb3d9' stroke-width='1.5'/></svg>" type="image/svg+xml">
    <title>%[1]s - %[12]s</title>
    <style>%[2]s</style>
</head>
<body>
    <div class="mobile-header">
        <button id="mobile-nav-toggle" class="mobile-nav-toggle" aria-label="Toggle navigation">☰</button>
        <span class="mobile-site-name">%[12]s</span>
    </div>
<div class="layout">
    <aside class="sidebar-nav closed">
        <div class="site-name">%[12]s</div>
        <div class="sidebar-header">
            <h2>Browse</h2>
            <button class="theme-toggle" id="theme-toggle" title="Toggle dark/light mode">&#9788;</button>
        </div>
        <button id="open-search" class="search-bar" type="button">Search <span class="icon">&#8981;</span></button>
        <nav class="nav-tree" id="nav-tree"></nav>
    </aside>
    <main class="content-col">
        <h1>%[1]s</h1>
        <div class="page-meta">
            <span class="page-meta-left">%[4]s</span>
            <span class="page-meta-right">%[5]s</span>
        </div>
        <div class="markdown-body">
            %[6]s
        </div>
        <footer class="page-footer">
            Created by <a href="https://basalt.j6n.dev" target="_blank" rel="noopener">Basalt</a>
        </footer>
    </main>
    <aside class="sidebar-right">
        <div class="graph-header">
            <h2>Graph</h2>
            <button id="open-full-graph" title="Full vault graph" aria-label="Open full vault graph">⤢</button>
        </div>
        <div id="local-graph"></div>
        %[7]s
        %[8]s
        %[9]s
    </aside>
</div>
<script>
window.siteName = %[14]s;
    // Mobile nav toggle
    var navToggle = document.getElementById('mobile-nav-toggle');
    var sidebarNav = document.querySelector('.sidebar-nav');
    if (navToggle) {
        navToggle.addEventListener('click', function() { 
            if (sidebarNav.classList.contains('open')) {
                sidebarNav.classList.remove('open');
                sidebarNav.classList.add('closed');
            } else {
                sidebarNav.classList.remove('closed');
                sidebarNav.classList.add('open');
            }
        });
    }
window.siteTheme = "%[13]s";
window.graphMode = %[15]s;
window.pageGraphData = %[10]s;
window.navTree = %[11]s;
</script>
<script>
// ---- Nav: render immediately ----
(function() {
    function escHtml(s) { return s.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;'); }
    function toggleNav(el) {
        var children = el.nextElementSibling;
        var icon = el.querySelector('.icon');
        children.classList.toggle('open');
        icon.classList.toggle('open');
    }
    window.toggleNavFolder = function(el) {
        var children = el.nextElementSibling;
        var icon = el.querySelector('.icon');
        var fid = children.id;
        children.classList.toggle('open');
        icon.classList.toggle('open');
        var expanded = getExpandedFolders();
        if (children.classList.contains('open')) {
            if (expanded.indexOf(fid) < 0) expanded.push(fid);
        } else {
            expanded = expanded.filter(function(f) { return f !== fid; });
        }
        saveExpandedFolders(expanded);
    };
    function buildNavHTML(nodes, parentPath) {
        var html = '';
        var depth = (window.pageGraphData && window.pageGraphData.currentHref) ? window.pageGraphData.currentHref.split('/').length - 1 : 0;
        var baseDepth = depth;
        var prefix = depth > 0 ? '../'.repeat(depth) : '';
        var expandedFolders = getExpandedFolders();
        for (var i = 0; i < nodes.length; i++) {
            var node = nodes[i];
            if (node.children) {
                var folderId = 'navf-' + (parentPath ? parentPath + '-' : '') + node.name;
                var isOpen = expandedFolders.indexOf(folderId) >= 0;
                var folderLabel = escHtml(node.name);
                var iconClass = isOpen ? 'icon open' : 'icon';
                html += '<div class="nav-folder">';
                if (node.indexHref) {
                    var folderLink = '<a href="' + prefix + node.indexHref + '" onclick="event.stopPropagation()">' + folderLabel + '</a>';
                    html += '<div class="nav-folder-header" onclick="toggleNavFolder(this)">';
                    html += '<span class="' + iconClass + '">&#9654;</span> ' + folderLink;
                    html += '</div>';
                } else {
                    html += '<div class="nav-folder-header" onclick="toggleNavFolder(this)">';
                    html += '<span class="' + iconClass + '">&#9654;</span> ' + folderLabel;
                    html += '</div>';
                }
                html += '<div class="nav-folder-children' + (isOpen ? ' open' : '') + '" id="' + folderId + '">';
                html += buildNavHTML(node.children, folderId);
                html += '</div></div>';
            } else {
                var href = prefix + node.href;
                var isActive = window.pageGraphData && window.pageGraphData.currentHref && window.pageGraphData.currentHref === node.href;
                var cls = isActive ? 'nav-page active' : 'nav-page';
                html += '<div class="' + cls + '"><a href="' + href + '">' + escHtml(node.name) + '</a></div>';
            }
        }
        return html;
    }
    function getExpandedFolders() {
        try { return JSON.parse(sessionStorage.getItem('basalt-nav-open') || []); } catch(e) { return []; }
    }
    function saveExpandedFolders(folders) {
        try { sessionStorage.setItem('basalt-nav-open', JSON.stringify(folders)); } catch(e) {}
    }
    var navEl = document.getElementById('nav-tree');
    if (navEl) navEl.innerHTML = buildNavHTML(window.navTree || [], '');
})();
</script>
<script>
// ---- Theme toggle ----
(function() {
    var html = document.documentElement;
    var toggle = document.getElementById('theme-toggle');
    // Apply saved preference or default to dark
    var saved = localStorage.getItem('basalt-theme');
    if (saved) { html.setAttribute('data-theme', saved); }
    else { html.setAttribute('data-theme', 'dark'); }
    updateIcon();
    toggle.addEventListener('click', function() {
        var current = html.getAttribute('data-theme');
        var next = current === 'dark' ? 'light' : 'dark';
        html.setAttribute('data-theme', next);
        localStorage.setItem('basalt-theme', next);
        updateIcon();
    });
    function updateIcon() {
        var isDark = html.getAttribute('data-theme') === 'dark';
        // Inline SVG icons that use currentColor (matches text color)
        if (isDark) {
            toggle.innerHTML = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/></svg>';
            toggle.title = 'Switch to light mode';
        } else {
            toggle.innerHTML = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="5"/><line x1="12" y1="1" x2="12" y2="3"/><line x1="12" y1="21" x2="12" y2="23"/><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/><line x1="1" y1="12" x2="3" y2="12"/><line x1="21" y1="12" x2="23" y2="12"/><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/></svg>';
            toggle.title = 'Switch to dark mode';
        }
    }
})();
</script>
<script>
// ---- D3 graph: load async and draw ----
(function() {
    var _cur = window.pageGraphData && window.pageGraphData.currentHref ? window.pageGraphData.currentHref : "";
    var _dp = _cur.split('/').length - 1;
    var _d3p = _dp > 0 ? '../'.repeat(_dp) + 'graph/d3.min.js' : 'graph/d3.min.js';
    function drawGraph() {
        try {
        console.log('graph: drawGraph called');
        var container = document.getElementById('local-graph');
        if (!container) { console.log('graph: no container'); return; }
        if (typeof window.d3 === 'undefined') { console.log('graph: d3 not loaded'); return; }
        var _d3 = window.d3;
        console.log('graph: d3 version=' + (_d3.version || 'unknown'));
        console.log('graph: d3 forceSimulation=' + typeof _d3.forceSimulation);
        var data = window.pageGraphData;
        var pageId = _cur.replace('.html', '');
        var nodes = [{ id: pageId, title: document.title.replace(' - ' + window.siteName, ''), href: _cur, current: true }];
        var nodeIds = {};
        nodeIds[pageId] = true;
        data.links.forEach(function(l) { var id = l.href.replace('.html',''); if (!nodeIds[id]) { nodes.push({ id: id, title: l.title, href: l.href, stub: l.stub }); nodeIds[id] = true; } });
        data.backlinks.forEach(function(bl) { var id = bl.href.replace('.html',''); if (!nodeIds[id]) { nodes.push({ id: id, title: bl.title, href: bl.href }); nodeIds[id] = true; } });
        var edges = [];
        data.links.forEach(function(l) { edges.push({ source: pageId, target: l.href.replace('.html','') }); });
        data.backlinks.forEach(function(bl) { edges.push({ source: bl.href.replace('.html',''), target: pageId }); });
        var w = container.clientWidth || 180;
        var h = 180;
        var svg = _d3.select(container).append('svg').attr('width', w).attr('height', h);
        // Create SVG groups BEFORE simulation starts so tick can update them
        console.log('graph: svg created, w=' + w + ', h=' + h);
        console.log('graph: nodes count=' + nodes.length + ', edges count=' + edges.length);
        var linkG = svg.append('g');
        var nodeG = svg.append('g');
        console.log('graph: groups created, linkG type=' + typeof linkG + ', nodeG type=' + typeof nodeG);
        var sim = _d3.forceSimulation(nodes)
            .force('link', _d3.forceLink(edges).id(function(d) { return d.id; }).distance(40))
            .force('charge', _d3.forceManyBody().strength(-80))
            .force('center', _d3.forceCenter(w / 2, h / 2))
            .force('collision', _d3.forceCollide().radius(15));
        // Render nodes/links immediately (before sim ticks)
        var link = linkG.selectAll('line').data(edges).enter().append('line').style('stroke', '#ccc').style('stroke-width', 1.5);
        var node = nodeG.selectAll('g').data(nodes).enter().append('g')
            .attr('class', function(d) { return 'node' + (d.stub ? ' stub' : '') + (d.current ? ' current' : ''); })
            .style('cursor', function(d) { return d.stub || d.current ? 'default' : 'pointer'; });
        var draggingNodeId = null;
        node.call(_d3.drag()
            .on('start', function(e) { 
                if (!e.active) sim.alphaTarget(0.3).restart(); 
                e.subject.fx = e.subject.x; e.subject.fy = e.subject.y;
                draggingNodeId = e.subject.id;
                svg.classed('dragging', true);
            })
            .on('drag', function(e) { e.subject.fx = e.x; e.subject.fy = e.y; })
            .on('end', function(e) { 
                if (!e.active) sim.alphaTarget(0); 
                e.subject.fx = null; e.subject.fy = null; 
                draggingNodeId = null;
                svg.classed('dragging', false);
            }));
        node.on('click', function(e, d) { if (!d.stub && !d.current) window.location.href = d.href; });
        node.on('mouseover', function(e, d) {
            var nid = d.id;
            var connected = new Set([pageId]);
            node.classed('hovered', function(n) { return n.id === nid; });
            node.classed('neighbor', function(n) { return n.id !== nid && (n.id === pageId || d.id === pageId); });
            node.classed('dimmed', function(n) { return n.id !== nid && n.id !== pageId && d.id !== pageId; });
            node.selectAll('circle').style('fill', function(n) { return n.id === nid || (n.id === pageId || d.id === pageId) ? 'var(--link)' : '#ccc'; });
            node.selectAll('circle').style('opacity', function(n) { return n.id !== nid && n.id !== pageId && d.id !== pageId ? '0.15' : '1'; });
            link.style('stroke', function(l) { return (l.source.id === nid || l.target.id === nid || l.source.id === pageId || l.target.id === pageId) ? 'var(--link)' : '#ccc'; });
            link.style('stroke-opacity', function(l) { return (l.source.id === nid || l.target.id === nid || l.source.id === pageId || l.target.id === pageId) ? 1 : 0.15; });
        });
        node.on('mouseout', function(e, d) {
            if (draggingNodeId !== null) return;
            node.classed('hovered', false).classed('neighbor', false).classed('dimmed', false);
            node.selectAll('circle').style('fill', '#ccc').style('opacity', '1');
            link.style('stroke', '#ccc').style('stroke-opacity', 1);
        });
        node.append('circle').attr('r', function(d) { return d.current ? 7 : 4 });
        node.append('text').attr('dx', 0).attr('dy', function(d) { var r = d.current ? 7 : 4; return r + 10; }).attr('text-anchor', 'middle').style('font-size', '9px').style('fill', 'currentColor').style('opacity', '0.8').text(function(d) { return d.title; });
        console.log('graph: sim created, node count=' + nodes.length);
        console.log('graph: link selection=' + (typeof link) + ', node selection=' + (typeof node));
        console.log('graph: calling tick...');
        // Update positions on every tick
        sim.on('tick', function() {
            try {
            link.attr('x1', function(d) { return d.source.x; }).attr('y1', function(d) { return d.source.y; })
              .attr('x2', function(d) { return d.target.x; }).attr('y2', function(d) { return d.target.y; });
            node.attr('transform', function(d) { return 'translate(' + d.x + ',' + d.y + ')'; });
            } catch(e) { console.log('graph: tick error=' + e); }
        });
        console.log('graph: tick registered, simulation should be running');
        } catch(e) { console.log('graph: drawGraph error: ' + e); }
    }
    var s = document.createElement("script");
    s.src = _d3p;
    s.onload = function() { console.log('graph: script loaded'); drawGraph(); };
    s.onerror = function() { console.log('graph: script failed to load: ' + _d3p); };
    document.head.appendChild(s);
})();
</script>
<div id="full-graph-overlay" class="full-graph-overlay" style="display:none;">
    <div class="full-graph-modal">
        <div class="full-graph-header">
            <h2>Full Vault Graph</h2>
            <button id="close-full-graph" aria-label="Close">&times;</button>
        </div>
        <div id="full-graph-container"><iframe id="graph-iframe" class="full-graph-iframe" src="" style="display:none;"></iframe></div>
    </div>
</div>
<script>
// ---- Full vault graph modal ----
(function() {
    var overlay = document.getElementById('full-graph-overlay');
    var container = document.getElementById('full-graph-container');
    var iframe = document.getElementById('graph-iframe');
    var openBtn = document.getElementById('open-full-graph');
    var closeBtn = document.getElementById('close-full-graph');

    openBtn.addEventListener('click', function() {
        // Compute path to graph/index.html from current page
        var segs = window.pageGraphData.currentHref.split('/').filter(Boolean);
        var depth = Math.max(0, segs.length - 1);
        var base = depth > 0 ? '../'.repeat(depth) : '';
        var graphPath = base + 'graph/' + (window.graphMode === 'nebula' ? 'nebula.html' : 'index.html');
        var iframe = document.getElementById('graph-iframe');
        iframe.src = graphPath;
        iframe.style.display = 'block';
        overlay.style.display = 'flex';
    });

    closeBtn.addEventListener('click', function() {
        overlay.style.display = 'none';
        iframe.style.display = 'none';
        iframe.src = '';
    });

    overlay.addEventListener('click', function(e) {
        if (e.target === overlay) {
            overlay.style.display = 'none';
            iframe.style.display = 'none';
            iframe.src = '';
        }
    });
})();
</script>
<div id="search-overlay" class="search-overlay" style="display:none;">
    <div class="search-modal">
        <div class="search-header">
            <input id="search-input" type="text" placeholder="Search pages..." autocomplete="off" />
            <button id="close-search" aria-label="Close">&times;</button>
        </div>
        <div id="search-results"></div>
    </div>
</div>
<script>
// ---- Search modal ----
(function() {
    var overlay = document.getElementById('search-overlay');
    var input = document.getElementById('search-input');
    var results = document.getElementById('search-results');
    var openBtn = document.getElementById('open-search');
    var closeBtn = document.getElementById('close-search');
    var searchIndex = null;

    function escHtml(s) { return s.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;'); }

    function highlight(text, term) {
        if (!term) return escHtml(text);
        var idx = text.toLowerCase().indexOf(term.toLowerCase());
        if (idx < 0) return escHtml(text.slice(0, 200));
        var start = Math.max(0, idx - 60);
        var end = Math.min(text.length, idx + term.length + 120);
        var snippet = (start > 0 ? '...' : '') + text.slice(start, end) + (end < text.length ? '...' : '');
        var re = new RegExp(escHtml(term).replace(/[-\\^$*+?.()|[\]{}]/g, '\\$&'), 'gi');
        return escHtml(snippet).replace(re, function(m) { return '<mark>' + m + '</mark>'; });
    }

    function doSearch(term) {
        if (!searchIndex) return;
        var q = term.toLowerCase();
        var matches = [];
        for (var i = 0; i < searchIndex.length; i++) {
            var e = searchIndex[i];
            var score = 0;
            if (e.title.toLowerCase().indexOf(q) >= 0) score += 10;
            if (e.content.toLowerCase().indexOf(q) >= 0) score += 1;
            if (e.tags) { for (var t = 0; t < e.tags.length; t++) { if (e.tags[t].toLowerCase().indexOf(q) >= 0) score += 5; } }
            if (score > 0) matches.push({ entry: e, score: score });
        }
        matches.sort(function(a, b) { return b.score - a.score; });
        if (matches.length === 0 || term.length === 0) {
            results.innerHTML = '<div class="search-empty">Start typing to search...</div>';
            return;
        }
        var html = '';
        for (var j = 0; j < Math.min(matches.length, 20); j++) {
            var m = matches[j];
            var e = m.entry;
            // Compute depth for relative path
            var depth = (window.pageGraphData && window.pageGraphData.currentHref) ? window.pageGraphData.currentHref.split('/').length - 1 : 0;
            var prefix = depth > 0 ? '../'.repeat(depth) : '';
            html += '<a class="search-result" href="' + prefix + e.path + '">';
            html += '<div class="search-result-title">' + escHtml(e.title) + '</div>';
            if (e.tags && e.tags.length > 0) { html += '<div class="search-result-tags">' + e.tags.map(function(t) { return '<span class="tag">' + escHtml(t) + '</span>'; }).join('') + '</div>'; }
            html += '<div class="search-result-snippet">' + highlight(e.content, term) + '</div>';
            html += '</a>';
        }
        results.innerHTML = html;
    }

    openBtn.addEventListener('click', function() {
        overlay.style.display = 'flex';
        input.focus();
        if (!searchIndex) {
            var depth = (window.pageGraphData && window.pageGraphData.currentHref) ? window.pageGraphData.currentHref.split('/').length - 1 : 0;
            var prefix = depth > 0 ? '../'.repeat(depth) : '';
            fetch(prefix + 'search.json').then(function(r) { return r.json(); }).then(function(data) {
                searchIndex = data;
                doSearch(input.value);
            }).catch(function() { searchIndex = []; });
        }
    });

    input.addEventListener('input', function() { doSearch(input.value); });

    closeBtn.addEventListener('click', function() {
        overlay.style.display = 'none';
        input.value = '';
        results.innerHTML = '';
    });

    overlay.addEventListener('click', function(e) {
        if (e.target === overlay) {
            overlay.style.display = 'none';
            input.value = '';
            results.innerHTML = '';
        }
    });

    document.addEventListener('keydown', function(e) {
        if (e.key === 'Escape' && overlay.style.display === 'flex') {
            overlay.style.display = 'none';
            input.value = '';
            results.innerHTML = '';
        }
    });
})();
</script>
</body>
</html>`,
	title, css, title,
		pageGraph.ReadingTime,
		pageGraph.Date,
		htmlContent,
		backlinksHTML,
		tagsHTML,
		tocHTML,
		string(pageGraphJSON), navTreeJSON,
		siteCfg.SiteName, siteCfg.SiteTheme, string(siteNameJS), string(graphModeJS))
}

// buildBacklinksHTML renders Links and Backlinks for the sidebar
func buildBacklinksHTML(pg *PageGraph, showLinks bool) string {
	if !showLinks {
		return ""
	}
	if pg == nil || (len(pg.Links) == 0 && len(pg.Backlinks) == 0) {
		return ""
	}
	s := "<div class=\"sidebar-section\"><div class=\"sidebar-links\">"
	if len(pg.Links) > 0 {
		s += "<h3>Links</h3><ul>"
		for _, l := range pg.Links {
		 cls := map[bool]string{true: " class=\"stub-link\""}[l.Stub]
		 s += fmt.Sprintf("<li><a href=\"%s\"%s>%s</a>%s</li>", l.Href, cls, l.Title, map[bool]string{true: " *(stub)"}[l.Stub])
		}
		s += "</ul>"
	}
	if len(pg.Backlinks) > 0 {
		s += "<h3>Backlinks</h3><ul>"
		for _, bl := range pg.Backlinks {
		 s += fmt.Sprintf("<li><a href=\"%s\">%s</a></li>", bl.Href, bl.Title)
		}
		s += "</ul>"
	}
	s += "</div></div>"
	return s
}

// buildTagsHTML renders the tags section for a page
func buildTagsHTML(pg *PageGraph) string {
	if pg == nil || len(pg.Tags) == 0 {
		return ""
	}
	s := "<div class=\"tags\"><span class=\"tags-label\">Tags:</span>"
	for _, tag := range pg.Tags {
		s += fmt.Sprintf("<span class=\"tag\">%s</span>", tag)
	}
	s += "</div>"
	return s
}

// buildTocHTML renders the table of contents for the sidebar
func buildTocHTML(pg *PageGraph) string {
	if pg == nil || len(pg.TableOfContents) == 0 {
		return ""
	}
	s := "<div class=\"toc\"><h3>On this page</h3><ul class=\"toc-list\">"
	for _, entry := range pg.TableOfContents {
		s += fmt.Sprintf("<li class=\"toc-item level-%d\"><a href=\"#%s\">%s</a></li>", entry.Level, entry.ID, entry.Text)
	}
	s += "</ul></div>"
	return s
}

// generateStubHTML creates a placeholder page for a dead link target
func generateStubHTML(pageID string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <link rel="icon" href="data:image/svg+xml,<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 32 32'><circle cx='16' cy='16' r='14' fill='%%23161a22' stroke='%%236bb3d9' stroke-width='2'/><circle cx='10' cy='12' r='2.5' fill='%%236bb3d9'/><circle cx='22' cy='12' r='2.5' fill='%%236bb3d9'/><circle cx='16' cy='22' r='2.5' fill='%%236bb3d9'/><line x1='10' y1='12' x2='16' y2='22' stroke='%%236bb3d9' stroke-width='1.5'/><line x1='22' y1='12' x2='16' y2='22' stroke='%%236bb3d9' stroke-width='1.5'/><line x1='10' y1='12' x2='22' y2='12' stroke='%%236bb3d9' stroke-width='1.5'/></svg>" type="image/svg+xml">
    <title>%s — Create Page</title>
    <style>
        :root { --bg: #f8f8f8; --text: #333; --link: #2980b9; }
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; line-height: 1.6; margin: 0; background: var(--bg); color: var(--text); display: flex; min-height: 100vh; }
        .layout { display: grid; grid-template-columns: 1fr 2fr 1fr; width: 100%%; }
        .content-col { padding: 20px; }
        .stub { background: #fff3cd; border: 1px solid #ffc107; padding: 20px; border-radius: 6px; }
        .stub h2 { margin-top: 0; color: #856404; }
        code { background: #f8f8f8; padding: 2px 6px; border-radius: 3px; }
    </style>
</head>
<body>
<button id="mobile-nav-toggle" class="mobile-nav-toggle" aria-label="Toggle navigation">☰</button>
<div class="layout">
    <main class="content-col">
        <h1>%[1]s</h1>
        <div class="stub">
            <h2>📄 Page Not Found</h2>
            <p>This page doesn't exist yet. To create it, add a file named <code>%s.md</code> to your vault.</p>
        </div>
    </main>
</div>
</body>
</html>`, pageID, pageID, pageID)
}

// writeGraphViewer writes the full vault graph viewers.
// Both renderers are always generated so the nebula's 2D button can link to the
// classic D3 view; graphMode only controls which one the site nav opens.
func writeGraphViewer(graphDir string, graphJSON []byte, siteTheme string, siteName string, nodeSizeByEdges bool, graphMode GraphMode) {
	downloadD3(graphDir)
	writeFullGraphViewer(graphDir, graphJSON, siteTheme, siteName, nodeSizeByEdges)
	writeFullGraphViewerNebula(graphDir, graphJSON, siteTheme, siteName, nodeSizeByEdges)
}

func writeFullGraphViewer(graphDir string, graphJSON []byte, siteTheme string, siteName string, nodeSizeByEdges bool) {
	html := `<!DOCTYPE html>
<html lang="en" data-theme="%s">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <link rel="icon" href="data:image/svg+xml,<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 32 32'><circle cx='16' cy='16' r='14' fill='%%23161a22' stroke='%%236bb3d9' stroke-width='2'/><circle cx='10' cy='12' r='2.5' fill='%%236bb3d9'/><circle cx='22' cy='12' r='2.5' fill='%%236bb3d9'/><circle cx='16' cy='22' r='2.5' fill='%%236bb3d9'/><line x1='10' y1='12' x2='16' y2='22' stroke='%%236bb3d9' stroke-width='1.5'/><line x1='22' y1='12' x2='16' y2='22' stroke='%%236bb3d9' stroke-width='1.5'/><line x1='10' y1='12' x2='22' y2='12' stroke='%%236bb3d9' stroke-width='1.5'/></svg>" type="image/svg+xml">
    <title>Graph View — %s</title>
    <style>
        :root, [data-theme="dark"] { --bg: #1e1e1e; --text: #e0e0e0; --border: #3a3a3a; --heading: #ffffff; --card-bg: #2a2a2a; --link: #6bb3d9; --graph-node: #999; }
        [data-theme="light"] { --bg: #f8f8f8; --text: #333; --border: #e1e4e8; --heading: #1a1a1a; --card-bg: #ffffff; --link: #2980b9; --graph-node: #ccc; }
        html, body { overflow: hidden; height: 100%%; margin: 0; }
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; background: var(--bg); color: var(--text); }
        #graph { width: 100vw; height: 100vh; overflow: hidden; }
        .node { cursor: pointer; }
        .node circle { fill: var(--graph-node); stroke: none; }
        .node.stub circle { fill: #e67e22; }
        .node text { font-size: 12px; fill: currentColor; opacity: 0.85; pointer-events: none; transition: opacity 0.2s; }
        .link { stroke: #ccc; stroke-width: 1px; transition: stroke-opacity 0.2s; }
        .node.dimmed circle { opacity: 0.15; }
        .node.dimmed text { opacity: 0.3; }
        .link.dimmed { stroke-opacity: 0.15; }
        .node.hovered circle { fill: var(--link); }
        .node.neighbor circle { fill: var(--link); }
        .link.connected { stroke: var(--link); stroke-opacity: 1; }
        #legend { position: absolute; top: 20px; right: 20px; background: var(--card-bg); padding: 15px; border-radius: 6px; box-shadow: 0 1px 3px rgba(0,0,0,0.2); font-size: 0.85em; border: 1px solid var(--border); }
        #legend h3 { margin: 0 0 10px; color: var(--heading); }
        #legend span { display: inline-block; width: 12px; height: 12px; border-radius: 50%%; margin-right: 6px; vertical-align: middle; }
        .legend-page { background: var(--link); }
        .legend-stub { background: #e67e22; }
        #mode-toggle { position: absolute; bottom: 20px; right: 20px; display: flex; gap: 4px; }
        #mode-toggle a, #mode-toggle span.current {
            background: var(--card-bg); border: 1px solid var(--border); border-radius: 6px;
            padding: 6px 12px; font-size: 0.8em; text-decoration: none; color: var(--text);
        }
        #mode-toggle span.current { background: var(--link); color: var(--bg); border-color: var(--link); }
        #mode-toggle a:hover { border-color: var(--link); }
    </style>
</head>
<body>
    <div id="legend">
        <h3>Legend</h3>
        <div><span class="legend-page"></span>Page</div>
        <div><span class="legend-stub"></span>Stub (dead link)</div>
    </div>
    <div id="mode-toggle">
        <a href="nebula.html">3D</a>
        <span class="current">2D</span>
    </div>
    <div id="graph"></div>
    <script src="d3.min.js"></script>
    <script>
    var graph = %s;
    var w = document.getElementById("graph").clientWidth;
    var h = document.getElementById("graph").clientHeight;
    var svg = d3.select("#graph").append("svg").attr("width", w).attr("height", h);
    // Zoom/pan via scroll wheel and drag on SVG background
    var zoomG = svg.append("g");
    svg.call(d3.zoom().scaleExtent([0.1, 4]).on("zoom", function(e) { zoomG.attr("transform", e.transform); }));
    var sim = d3.forceSimulation(graph.nodes)
        .force("link", d3.forceLink(graph.edges).id(function(d) { return d.id; }).distance(180))
        .force("charge", d3.forceManyBody().strength(-2))
        .force("center", d3.forceCenter(w / 2, h / 2))
        .force("collision", d3.forceCollide().radius(20))
        .alpha(0.3);
    var link = zoomG.selectAll("line").data(graph.edges).enter().append("line").attr("class", "link");
    // Build neighbor set for hover highlighting (edges are still strings here)
    var neighborOf = {};
    graph.nodes.forEach(function(n) { neighborOf[n.id] = new Set(); });
    graph.edges.forEach(function(e) {
        var sid = typeof e.source === 'object' ? e.source.id : e.source;
        var tid = typeof e.target === 'object' ? e.target.id : e.target;
        neighborOf[sid].add(tid);
        neighborOf[tid].add(sid);
    });
    // Compute edge count per node for optional size-by-edges feature
    var edgeCount = {};
    graph.nodes.forEach(function(n) { edgeCount[n.id] = 0; });
    graph.edges.forEach(function(e) {
        var sid = typeof e.source === 'object' ? e.source.id : e.source;
        var tid = typeof e.target === 'object' ? e.target.id : e.target;
        edgeCount[sid] = (edgeCount[sid] || 0) + 1;
        edgeCount[tid] = (edgeCount[tid] || 0) + 1;
    });
    // BFS to find all nodes reachable from startId (entire connected component)
    function reachableNodes(startId) {
        var visited = new Set();
        var queue = [startId];
        visited.add(startId);
        while (queue.length > 0) {
            var curr = queue.shift();
            var neighbors = neighborOf[curr] || new Set();
            neighbors.forEach(function(nid) {
                if (!visited.has(nid)) {
                    visited.add(nid);
                    queue.push(nid);
                }
            });
        }
        return visited;
    }
    var draggingNodeId = null;
    var node = zoomG.selectAll("g").data(graph.nodes).enter().append("g").attr("class", function(d) { return "node" + (d.stub ? " stub" : ""); })
        .call(d3.drag()
            .on("start", function(e) { if (!e.active) sim.alphaTarget(0.3).restart(); e.subject.fx = e.subject.x; e.subject.fy = e.subject.y; draggingNodeId = e.subject.id; })
            .on("drag", function(e) { e.subject.fx = e.x; e.subject.fy = e.y; })
            .on("end", function(e) { if (!e.active) sim.alphaTarget(0); e.subject.fx = null; e.subject.fy = null; draggingNodeId = null; }))
        .on("mouseover", function(event, d) {
            var nid = d.id;
            // Find all nodes reachable from hovered node (entire connected component)
            var connected = reachableNodes(nid);
            node.classed("hovered", function(n) { return n.id === nid; });
            node.classed("neighbor", function(n) { return n.id !== nid && connected.has(n.id); });
            node.classed("dimmed", function(n) { return !connected.has(n.id); });
            link.classed("dimmed", function(l) {
                var sid = l.source.id || l.source;
                var tid = l.target.id || l.target;
                return !connected.has(sid) || !connected.has(tid);
            });
            link.classed("connected", function(l) {
                var sid = l.source.id || l.source;
                var tid = l.target.id || l.target;
                return connected.has(sid) && connected.has(tid);
            });
        })
        .on("mouseout", function(e, d) {
            if (draggingNodeId !== null) return;
            node.classed("hovered", false).classed("neighbor", false).classed("dimmed", false);
            link.classed("dimmed", false);
            link.classed("connected", false);
        })
        .on("click", function(event, d) { if (!d.stub) { sim.stop(); graph.nodes.forEach(function(n) { n.fx = n.x; n.fy = n.y; }); var _t = new URL("../" + d.path, window.location.href).href; if (window.top !== window.self) { window.top.location.href = _t; } else { window.location.href = _t; } } });
    var nodeRadius = %t;
    node.append("circle").attr("r", function(d) {
        if (nodeRadius) {
            var count = edgeCount[d.id] || 0;
            return 8 + count * 0.75;
        }
        return 8;
    });
    node.append("text").attr("dx", 0).attr("dy", function(d) { return (nodeRadius ? (8 + (edgeCount[d.id] || 0) * 0.75) : 8) + 10; }).attr("text-anchor", "middle").text(function(d) { return d.title; });
    sim.on("tick", function() {
        link.attr("x1", function(d) { return d.source.x; }).attr("y1", function(d) { return d.source.y; })
          .attr("x2", function(d) { return d.target.x; }).attr("y2", function(d) { return d.target.y; });
        node.attr("transform", function(d) { return "translate(" + d.x + "," + d.y + ")"; });
    });
    </script>
</body>
</html>`
	data := fmt.Sprintf(html, siteTheme, siteName, graphJSON, nodeSizeByEdges)
	err := os.WriteFile(filepath.Join(graphDir, "index.html"), []byte(data), 0644)
	if err != nil {
		fmt.Printf("Error writing graph index.html: %v\n", err)
	}
}

// writeFullGraphViewerNebula renders the full vault graph as a 3D galaxy of glowing stars.
func writeFullGraphViewerNebula(graphDir string, graphJSON []byte, siteTheme string, siteName string, nodeSizeByEdges bool) {
	// The nebula viewer ships its own three.js — no CDN dependency at runtime.
	// We bundle it inline via a CDN fetch during build (downloadThreeJS) or inline it.
	// For simplicity we load OrbitControls from a CDN too.
	if graphJSON == nil {
		fmt.Printf("Error: graphJSON is nil — buildGraph likely failed. Skipping nebula.html\n")
		return
	}
	nebulaHTML := `<!DOCTYPE html>
<html lang="en" data-theme="dark">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=no">
    <link rel="icon" href="data:image/svg+xml,<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 32 32'><circle cx='16' cy='16' r='14' fill='%23161a22' stroke='%236bb3d9' stroke-width='2'/><circle cx='10' cy='12' r='2.5' fill='%236bb3d9'/><circle cx='22' cy='12' r='2.5' fill='%236bb3d9'/><circle cx='16' cy='22' r='2.5' fill='%236bb3d9'/><line x1='10' y1='12' x2='16' y2='22' stroke='%236bb3d9' stroke-width='1.5'/><line x1='22' y1='12' x2='16' y2='22' stroke='%236bb3d9' stroke-width='1.5'/><line x1='10' y1='12' x2='22' y2='12' stroke='%236bb3d9' stroke-width='1.5'/></svg>" type="image/svg+xml">
    <title>Graph — Nebula</title>
    <style>
        :root, [data-theme="dark"] {
            --bg: #06060f;
            --text: #e0e0e0;
            --border: #1a1a2e;
            --heading: #ffffff;
            --card-bg: #0e0e1f;
            --link: #6bb3d9;
            --muted: #556;
        }
        [data-theme="light"] {
            --bg: #f0f0f8;
            --text: #1a1a2e;
            --border: #ccc;
            --heading: #1a1a2e;
            --card-bg: #fff;
            --link: #2980b9;
            --muted: #888;
        }
        * { box-sizing: border-box; margin: 0; padding: 0; }
        html, body { width: 100%; height: 100%; overflow: hidden; margin: 0; background: var(--bg); overscroll-behavior: none; }
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; color: var(--text); }
        #canvas-container { position: fixed; inset: 0; z-index: 0; touch-action: none; }
        canvas { display: block; width: 100% !important; height: 100% !important; touch-action: none; -webkit-user-select: none; user-select: none; }

        /* HUD overlay */
        #hud {
            position: fixed; top: 0; left: 0; right: 0; bottom: 0;
            pointer-events: none;
            z-index: 10;
        }
        #legend {
            position: absolute; top: 20px; right: 20px;
            background: rgba(14,14,31,0.85);
            backdrop-filter: blur(8px);
            border: 1px solid var(--border);
            border-radius: 8px;
            padding: 14px 18px;
            font-size: 12px;
            color: var(--text);
            min-width: 140px;
            pointer-events: auto;
        }
        #legend h3 { margin: 0 0 10px; font-size: 11px; text-transform: uppercase; letter-spacing: 0.08em; color: var(--muted); }
        .legend-row { display: flex; align-items: center; gap: 8px; margin: 5px 0; }
        .legend-dot {
            width: 10px; height: 10px; border-radius: 50%;
            box-shadow: 0 0 6px currentColor;
        }
        .legend-line {
            width: 20px; height: 1px;
            background: currentColor; opacity: 0.5;
        }
        .legend-page .legend-dot { background: #fff; color: #fff; box-shadow: 0 0 6px #fff, 0 0 12px rgba(107,179,217,0.6); }
        .legend-stub .legend-dot { background: none; border: 1.5px dashed #e67e22; box-shadow: none; }
        .legend-edge .legend-line { background: rgba(107,179,217,0.4); }

        /* Search bar */
        #search-container {
            position: absolute; top: 20px; left: 50%; transform: translateX(-50%);
            pointer-events: auto;
        }
        #search-input {
            background: rgba(14,14,31,0.85);
            backdrop-filter: blur(8px);
            border: 1px solid var(--border);
            border-radius: 24px;
            color: var(--text);
            padding: 8px 18px;
            font-size: 13px;
            width: 280px;
            outline: none;
            transition: border-color 0.2s, box-shadow 0.2s;
        }
        #search-input:focus { border-color: var(--link); box-shadow: 0 0 0 2px rgba(107,179,217,0.2); }
        #search-input::placeholder { color: var(--muted); }

        /* Tooltip */
        #tooltip {
            position: absolute;
            display: none;
            background: rgba(14,14,31,0.92);
            backdrop-filter: blur(10px);
            border: 1px solid var(--border);
            border-radius: 8px;
            padding: 10px 14px;
            font-size: 12px;
            color: var(--text);
            pointer-events: none;
            z-index: 20;
            max-width: 220px;
            box-shadow: 0 4px 20px rgba(0,0,0,0.5);
        }
        #tooltip .tt-title { font-weight: 600; color: var(--heading); margin-bottom: 4px; font-size: 13px; }
        #tooltip .tt-meta { color: var(--muted); font-size: 11px; margin-top: 5px; }
        #tooltip .tt-tags { display: flex; flex-wrap: wrap; gap: 4px; margin-top: 5px; }
        #tooltip .tt-tag {
            display: inline-block; padding: 1px 7px;
            background: var(--link); color: var(--bg);
            border-radius: 10px; font-size: 10px; font-weight: 500;
        }

        /* Mode toggle */
        #mode-toggle {
            position: absolute; bottom: 20px; right: 20px;
            pointer-events: auto;
            display: flex; gap: 4px;
        }
        #mode-toggle button {
            background: rgba(14,14,31,0.85);
            border: 1px solid var(--border);
            color: var(--muted);
            border-radius: 6px;
            padding: 6px 12px;
            font-size: 11px;
            cursor: pointer;
            transition: all 0.2s;
        }
        #mode-toggle button:hover { border-color: var(--link); color: var(--text); }
        #mode-toggle button.active { background: var(--link); color: var(--bg); border-color: var(--link); }

        /* Controls hint */
        #controls-hint {
            position: absolute; bottom: 20px; left: 50%; transform: translateX(-50%);
            color: var(--muted); font-size: 11px; text-align: center;
            pointer-events: none;
            opacity: 0.7;
        }

        /* Node count footer inside the legend */
        #node-count {
            margin-top: 10px;
            padding-top: 8px;
            border-top: 1px solid var(--border);
            font-size: 11px;
            color: var(--muted);
        }
        #node-count strong { color: var(--text); }

        /* ---- Mobile / touch layout ---- */
        /* Desktop stacks search (top-center) over a top-right legend and a
           bottom-right toggle. On a phone those overlap, so re-stack into a
           clean vertical flow: search at the very top, hint under it, then
           legend and the mode toggle pinned to the bottom. */
        @media (max-width: 768px) {
            #search-container {
                top: 12px; left: 12px; right: 12px;
                transform: none;
            }
            #search-input {
                width: 100%;
                padding: 11px 16px;
                font-size: 16px; /* >=16px stops iOS from zooming on focus */
            }
            #controls-hint {
                top: 60px; bottom: auto;
                left: 12px; right: 12px;
                transform: none;
                font-size: 10px;
                line-height: 1.4;
            }
            /* Legend becomes a compact horizontal strip above the (wrapping) toggle. */
            #legend {
                top: auto; bottom: 96px; left: 12px; right: 12px;
                min-width: 0;
                padding: 8px 12px;
                display: flex; flex-wrap: wrap;
                align-items: center; justify-content: center;
                gap: 6px 14px;
            }
            #legend h3 { display: none; }
            .legend-row { margin: 0; }
            #node-count {
                flex-basis: 100%;
                margin: 0; padding: 0; border: none;
                text-align: center;
            }
            #mode-toggle {
                bottom: 12px; left: 12px; right: 12px;
                justify-content: center;
                flex-wrap: wrap;
            }
            #mode-toggle button { padding: 9px 13px; font-size: 12px; }
            #tooltip { max-width: 70vw; }
        }

        /* Embed mode (loaded in an iframe via the in-page graph directive): hide all
           HUD chrome so only the 3D scene shows in the box. Interaction still works. */
        body.embed #search-container,
        body.embed #legend,
        body.embed #mode-toggle,
        body.embed #controls-hint { display: none !important; }
    </style>
</head>
<body>
    <div id="canvas-container"></div>
    <div id="hud">
        <div id="search-container">
            <input id="search-input" type="text" placeholder="Search nodes..." autocomplete="off" />
        </div>
        <div id="legend">
            <h3>Legend</h3>
            <div class="legend-row legend-page"><span class="legend-dot"></span>Page</div>
            <div class="legend-row legend-stub"><span class="legend-dot"></span>Stub (dead link)</div>
            <div class="legend-row legend-edge"><span class="legend-line"></span>Wikilink</div>
            <div id="node-count">Nodes: <strong id="node-count-num">0</strong> · Edges: <strong id="edge-count-num">0</strong></div>
        </div>
        <div id="tooltip">
            <div class="tt-title" id="tt-title"></div>
            <div class="tt-meta" id="tt-meta"></div>
            <div class="tt-tags" id="tt-tags"></div>
        </div>
        <div id="controls-hint">Drag to rotate · Scroll to zoom · Click to navigate</div>
        <div id="mode-toggle">
            <button id="btn-labels" onclick="toggleLabels()">Labels: Off</button>
            <button id="btn-lines" onclick="toggleLines()">Lines: Off</button>
            <button class="active" id="btn-curve" onclick="toggleCurve()">Edges: Curved</button>
            <button class="active" id="btn-haze" onclick="toggleHaze()">Haze: On</button>
            <button class="active" id="btn-spin" onclick="toggleSpin()">Spin: On</button>
            <button class="active" id="btn-3d" onclick="setCameraMode('3d')">3D</button>
            <button id="btn-2d" onclick="setCameraMode('2d')">2D</button>
        </div>
    </div>

    <script type="importmap">
    {
        "imports": {
            "three": "https://cdn.jsdelivr.net/npm/three@0.158.0/build/three.module.js",
            "three/addons/": "https://cdn.jsdelivr.net/npm/three@0.158.0/examples/jsm/"
        }
    }
    </script>
    <script type="module">
    import * as THREE from 'three';
    import { OrbitControls } from 'three/addons/controls/OrbitControls.js';
    const THREEOrbitControls = OrbitControls; // alias for legacy THREE.OrbitControls calls
    (function() {
        'use strict';
        // Embed mode (?embed=1): rendered inside a page via the graph directive — strip
        // the HUD chrome so the iframe shows only the scene.
        if (new URLSearchParams(window.location.search).has('embed')) {
            document.body.classList.add('embed');
        }
        var graph = {{GRAPH_JSON}};
        // Unlike the 2D graph, nodes are uniform size in 3D — edge-based sizing is omitted here.
        var NODE_SIZE = 1.8;

        // ---- Helpers ----
        function hashToHue(str) {
            var h = 0;
            for (var i = 0; i < str.length; i++) {
                h = (Math.imul(31, h) + str.charCodeAt(i)) | 0;
            }
            return Math.abs(h) % 360;
        }
        function hslColor(h, s, l) {
            return 'hsl(' + h + ',' + s + '%,' + l + '%)';
        }

        // ---- Build edge count map ----
        var edgeCount = {};
        graph.nodes.forEach(function(n) { edgeCount[n.id] = 0; });
        graph.edges.forEach(function(e) {
            var sid = typeof e.source === 'object' ? e.source.id : e.source;
            var tid = typeof e.target === 'object' ? e.target.id : e.target;
            edgeCount[sid] = (edgeCount[sid] || 0) + 1;
            edgeCount[tid] = (edgeCount[tid] || 0) + 1;
        });

        // ---- Build neighbor map ----
        var neighbors = {};
        graph.nodes.forEach(function(n) { neighbors[n.id] = new Set(); });
        graph.edges.forEach(function(e) {
            var sid = typeof e.source === 'object' ? e.source.id : e.source;
            var tid = typeof e.target === 'object' ? e.target.id : e.target;
            if (neighbors[sid]) neighbors[sid].add(tid);
            if (neighbors[tid]) neighbors[tid].add(sid);
        });

        // ---- Three.js setup ----
        var container = document.getElementById('canvas-container');
        var renderer = new THREE.WebGLRenderer({ antialias: true, alpha: true });
        renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));
        renderer.setSize(window.innerWidth, window.innerHeight);
        renderer.setClearColor(0x06060f, 1);
        // Filmic tone mapping compresses bright additive overlaps (stacked glows, clouds,
        // edge haze) toward a soft white rolloff instead of blowing out linearly. It
        // self-regulates as a vault grows denser. Exposure is the master brightness dial.
        renderer.toneMapping = THREE.ACESFilmicToneMapping;
        renderer.toneMappingExposure = 0.9;
        container.appendChild(renderer.domElement);

        var scene = new THREE.Scene();
        // Exponential depth fog — distant nodes/edges/stars melt into the deep-space
        // background instead of sitting on one flat plane. Matches the clear color.
        scene.fog = new THREE.FogExp2(0x06060f, 0.0008);
        var camera = new THREE.PerspectiveCamera(60, window.innerWidth / window.innerHeight, 0.1, 2000);
        camera.position.set(0, 0, 120);

        var controls = new THREEOrbitControls(camera, renderer.domElement);
        controls.enableDamping = true;
        controls.dampingFactor = 0.08;
        controls.rotateSpeed = 0.5;
        controls.zoomSpeed = 1.2;
        controls.minDistance = 10;
        controls.maxDistance = 1200;
        controls.autoRotate = true;
        controls.autoRotateSpeed = 0.3;

        // ---- Initial zoom: fit every node in view ----
        // The force layout expands over the first seconds, so keep re-fitting the
        // camera distance each frame until the user takes over (drag/zoom).
        var autoFit = true;
        controls.addEventListener('start', function() { autoFit = false; });
        function fitDistance() {
            var maxR = 0;
            for (var i = 0; i < nodeMeshes.length; i++) {
                var len = nodeMeshes[i].position.length();
                if (len > maxR) maxR = len;
            }
            var vFov = camera.fov * Math.PI / 180;
            // In portrait the horizontal fov is the limiting one.
            var hFov = 2 * Math.atan(Math.tan(vFov / 2) * camera.aspect);
            var fov = Math.min(vFov, hFov);
            var dist = (maxR + NODE_SIZE * 2) / Math.tan(fov / 2) * 1.1;
            return Math.min(Math.max(dist, controls.minDistance), controls.maxDistance);
        }

        // ---- Idle spin toggle ----
        var spinEnabled = true;
        window.toggleSpin = function() {
            spinEnabled = !spinEnabled;
            controls.autoRotate = spinEnabled;
            var btn = document.getElementById('btn-spin');
            btn.classList.toggle('active', spinEnabled);
            btn.textContent = spinEnabled ? 'Spin: On' : 'Spin: Off';
        };

        // ---- Edge style toggles (consumed by updateEdgePoints below) ----
        // Render style (flowing stars vs. plain lines), cloud haze, and curved vs. straight.
        var linesMode = false; // false → flowing stars, true → plain lines
        var hazeEnabled = true;
        // Applies the current style flags to every edge's objects (called by the toggles).
        function applyEdgeVisibility() {
            edgeObjects.forEach(function(eo) {
                eo.line.visible = !linesMode;                       // flowing star stream
                if (eo.plainLine) eo.plainLine.visible = linesMode; // plain line
                if (eo.haze) eo.haze.forEach(function(p) { p.visible = hazeEnabled && !linesMode; });
            });
            // Populate line geometry immediately so it doesn't flash at the origin for
            // one frame before the animate loop fills it in.
            if (linesMode) edgeObjects.forEach(function(eo) { updateEdgePoints(eo, 0); });
        }
        window.toggleLines = function() {
            linesMode = !linesMode;
            var btn = document.getElementById('btn-lines');
            btn.classList.toggle('active', linesMode);
            btn.textContent = linesMode ? 'Lines: On' : 'Lines: Off';
            applyEdgeVisibility();
        };
        window.toggleHaze = function() {
            hazeEnabled = !hazeEnabled;
            var btn = document.getElementById('btn-haze');
            btn.classList.toggle('active', hazeEnabled);
            btn.textContent = hazeEnabled ? 'Haze: On' : 'Haze: Off';
            applyEdgeVisibility();
        };
        var curveAmount = 0.18; // 0 → straight edges
        window.toggleCurve = function() {
            curveAmount = curveAmount > 0 ? 0 : 0.18;
            var curved = curveAmount > 0;
            var btn = document.getElementById('btn-curve');
            btn.classList.toggle('active', curved);
            btn.textContent = curved ? 'Edges: Curved' : 'Edges: Straight';
        };

        // ---- View mode (3D nebula vs classic 2D graph) ----
        window.setCameraMode = function(mode) {
            if (mode === '2d') window.location.href = 'index.html';
        };

        // ---- Touch detection (coarse pointer = phones/tablets) ----
        var isTouch = window.matchMedia('(pointer: coarse)').matches;
        if (isTouch) {
            document.getElementById('controls-hint').textContent =
                'Drag to rotate · Pinch to zoom · Tap to select, tap again to open';
        }

        // ---- Node name labels (all nodes, toggled via Labels button) ----
        var labelsEnabled = false;
        var labelSprites = []; // { sprite, mesh }
        function makeLabelSprite(text) {
            var canvas = document.createElement('canvas');
            var ctx = canvas.getContext('2d');
            var fontSize = 28;
            var font = '500 ' + fontSize + 'px -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif';
            ctx.font = font;
            canvas.width = Math.ceil(ctx.measureText(text).width) + 16;
            canvas.height = fontSize + 14;
            ctx.font = font; // canvas resize resets context state
            ctx.textBaseline = 'middle';
            ctx.shadowColor = 'rgba(0,0,0,0.9)';
            ctx.shadowBlur = 6;
            ctx.fillStyle = 'rgba(224,230,240,0.95)';
            ctx.fillText(text, 8, canvas.height / 2);
            var tex = new THREE.CanvasTexture(canvas);
            tex.minFilter = THREE.LinearFilter;
            var mat = new THREE.SpriteMaterial({ map: tex, transparent: true, depthWrite: false, fog: false });
            var sprite = new THREE.Sprite(mat);
            var scale = 0.08; // world units per canvas px (~legend font size on screen)
            sprite.scale.set(canvas.width * scale, canvas.height * scale, 1);
            sprite.center.set(0.5, 0); // anchor at bottom so the label floats above the node
            sprite.raycast = function() {};
            return sprite;
        }
        window.toggleLabels = function() {
            labelsEnabled = !labelsEnabled;
            var btn = document.getElementById('btn-labels');
            btn.classList.toggle('active', labelsEnabled);
            btn.textContent = labelsEnabled ? 'Labels: On' : 'Labels: Off';
            if (labelsEnabled && labelSprites.length === 0) {
                nodeMeshes.forEach(function(m) {
                    var sprite = makeLabelSprite(m.userData.title || m.userData.id);
                    scene.add(sprite);
                    labelSprites.push({ sprite: sprite, mesh: m });
                });
            }
            labelSprites.forEach(function(l) { l.sprite.visible = labelsEnabled; });
        };

        // ---- Drifting nebula clouds (fake volumetric gas) ----
        // Soft additive radial-gradient billboards scattered in the background give
        // the scene its "nebula" haze. They slowly spin and bob in the animate loop.
        function makeCloudTexture(r, g, b) {
            var size = 256;
            var c = document.createElement('canvas');
            c.width = c.height = size;
            var ctx = c.getContext('2d');
            var grad = ctx.createRadialGradient(size / 2, size / 2, 0, size / 2, size / 2, size / 2);
            grad.addColorStop(0.0, 'rgba(' + r + ',' + g + ',' + b + ',0.55)');
            grad.addColorStop(0.4, 'rgba(' + r + ',' + g + ',' + b + ',0.16)');
            grad.addColorStop(1.0, 'rgba(' + r + ',' + g + ',' + b + ',0)');
            ctx.fillStyle = grad;
            ctx.fillRect(0, 0, size, size);
            var tex = new THREE.CanvasTexture(c);
            tex.minFilter = THREE.LinearFilter;
            return tex;
        }
        // Nebula palette: deep blue, teal, violet, magenta, indigo.
        var cloudPalette = [[42, 74, 138], [26, 106, 122], [106, 42, 138], [138, 42, 90], [40, 60, 120]];
        var cloudTextures = cloudPalette.map(function(c) { return makeCloudTexture(c[0], c[1], c[2]); });
        var clouds = []; // { sprite, spin, drift, phase, basePos }
        for (var ci = 0; ci < 6; ci++) {
            var cloudMat = new THREE.SpriteMaterial({
                map: cloudTextures[ci % cloudTextures.length],
                transparent: true, opacity: 0.1,
                blending: THREE.AdditiveBlending, depthWrite: false
            });
            var cloudSprite = new THREE.Sprite(cloudMat);
            cloudSprite.raycast = function() {};
            var cTheta = Math.random() * Math.PI * 2;
            var cPhi = Math.acos(2 * Math.random() - 1);
            var cRad = 350 + Math.random() * 350;
            var cBase = new THREE.Vector3(
                cRad * Math.sin(cPhi) * Math.cos(cTheta),
                cRad * Math.sin(cPhi) * Math.sin(cTheta) * 0.6, // flatten vertically for a disk-ish feel
                cRad * Math.cos(cPhi)
            );
            cloudSprite.position.copy(cBase);
            var cScale = 300 + Math.random() * 350;
            cloudSprite.scale.set(cScale, cScale, 1);
            cloudMat.rotation = Math.random() * Math.PI * 2;
            scene.add(cloudSprite);
            clouds.push({
                sprite: cloudSprite,
                spin: (Math.random() - 0.5) * 0.03,
                drift: 0.1 + Math.random() * 0.2,
                phase: Math.random() * Math.PI * 2,
                basePos: cBase
            });
        }

        // ---- Layered starfield (depth + size variety + twinkle) ----
        function makeStarLayer(count, spread, size, opacity, tint) {
            var geo = new THREE.BufferGeometry();
            var pos = new Float32Array(count * 3);
            for (var i = 0; i < count * 3; i++) pos[i] = (Math.random() - 0.5) * spread;
            geo.setAttribute('position', new THREE.BufferAttribute(pos, 3));
            var mat = new THREE.PointsMaterial({
                color: tint, size: size, opacity: opacity,
                transparent: true, sizeAttenuation: true,
                blending: THREE.AdditiveBlending, depthWrite: false
            });
            var pts = new THREE.Points(geo, mat);
            scene.add(pts);
            return mat;
        }
        // Three depth layers, each twinkling at its own rate/phase: distant blue dust,
        // a white mid-field, and a few large warm foreground stars.
        var starLayers = [
            { mat: makeStarLayer(1200, 2600, 1.2, 0.35, 0x99aaff), base: 0.35, speed: 0.7, amp: 0.10, phase: 0.0 },
            { mat: makeStarLayer(700, 1900, 2.2, 0.55, 0xffffff), base: 0.55, speed: 1.1, amp: 0.15, phase: 1.7 },
            { mat: makeStarLayer(250, 1500, 3.6, 0.75, 0xfff2d0), base: 0.75, speed: 1.7, amp: 0.20, phase: 3.1 }
        ];

        // ---- Node meshes ----
        var nodeMap = {}; // id -> { mesh, glow, data }
        var nodeMeshes = [];

        var nodeGeo = new THREE.SphereGeometry(1, 16, 16);
        var glowGeo = new THREE.SphereGeometry(1, 16, 16);

        graph.nodes.forEach(function(n) {
            var hue = hashToHue(n.id);
            var size = NODE_SIZE;
            var baseColor = new THREE.Color().setHSL(hue / 360, 0.6, 0.85);

            // Core star mesh (transparent so search/hover dimming can fade it)
            var mat = new THREE.MeshBasicMaterial({ color: baseColor, transparent: true, opacity: 1.0 });
            var mesh = new THREE.Mesh(nodeGeo, mat);
            mesh.scale.setScalar(size);

            // Outer glow (slightly larger, transparent). Additive blending makes
            // overlapping auras sum into bright bloom — the nebula/star-cluster look.
            var glowMat = new THREE.MeshBasicMaterial({
                color: baseColor,
                transparent: true,
                opacity: 0.28,
                side: THREE.BackSide,
                blending: THREE.AdditiveBlending,
                depthWrite: false
            });
            // Child of the node mesh, so scale here is relative to the core star.
            var glow = new THREE.Mesh(glowGeo, glowMat);
            glow.scale.setScalar(2.0);
            mesh.add(glow);

            // Even larger dim glow for halo
            var haloMat = new THREE.MeshBasicMaterial({
                color: baseColor,
                transparent: true,
                opacity: 0.12,
                side: THREE.BackSide,
                blending: THREE.AdditiveBlending,
                depthWrite: false
            });
            var halo = new THREE.Mesh(glowGeo, haloMat);
            halo.scale.setScalar(3.4);
            mesh.add(halo);

            // Glow/halo are purely visual — exclude them from raycasting so hover
            // and click only respond to the core star, not its larger aura shells.
            glow.raycast = function() {};
            halo.raycast = function() {};

            // Stub nodes: dashed ring instead of filled star
            if (n.stub) {
                mat = new THREE.MeshBasicMaterial({ color: 0xe67e22, transparent: true, opacity: 0.7, wireframe: true });
                mesh.material = mat;
                glow.material = new THREE.MeshBasicMaterial({ color: 0xe67e22, transparent: true, opacity: 0.12, side: THREE.BackSide, blending: THREE.AdditiveBlending, depthWrite: false });
                halo.material = new THREE.MeshBasicMaterial({ color: 0xe67e22, transparent: true, opacity: 0.06, side: THREE.BackSide, blending: THREE.AdditiveBlending, depthWrite: false });
            }

            // Random position in sphere
            var theta = Math.random() * Math.PI * 2;
            var phi = Math.acos(2 * Math.random() - 1);
            var r = 30 + Math.random() * 70;
            mesh.position.set(
                r * Math.sin(phi) * Math.cos(theta),
                r * Math.sin(phi) * Math.sin(theta),
                r * Math.cos(phi)
            );
            mesh.userData = { id: n.id, title: n.title, stub: !!n.stub, path: n.path, tags: n.tags || [] };
            scene.add(mesh);
            nodeMeshes.push(mesh);
            nodeMap[n.id] = mesh;
        });

        document.getElementById('node-count-num').textContent = graph.nodes.length;

        // ---- Edges as flowing streams of stars ----
        // Each edge is a dense run of tiny soft motes that stream along a gently curved,
        // wandering path from source to target — so a connection reads as a hazy galactic
        // current rather than a hard vector. Per-edge material.opacity still drives the
        // hover/search dimming.

        // Soft round mote texture: a radial gradient so each point is a fuzzy glow ("fog")
        // instead of a hard square. Shared across all edges.
        var dotTexture = (function() {
            var sz = 64;
            var c = document.createElement('canvas');
            c.width = c.height = sz;
            var ctx = c.getContext('2d');
            var g = ctx.createRadialGradient(sz / 2, sz / 2, 0, sz / 2, sz / 2, sz / 2);
            g.addColorStop(0.0, 'rgba(255,255,255,1)');
            g.addColorStop(0.4, 'rgba(255,255,255,0.35)');
            g.addColorStop(1.0, 'rgba(255,255,255,0)');
            ctx.fillStyle = g;
            ctx.fillRect(0, 0, sz, sz);
            var tex = new THREE.CanvasTexture(c);
            tex.minFilter = THREE.LinearFilter;
            return tex;
        })();

        // Softer, wider gradient for the path haze — a gentle cloud, not a crisp dot.
        var hazeTexture = (function() {
            var sz = 128;
            var c = document.createElement('canvas');
            c.width = c.height = sz;
            var ctx = c.getContext('2d');
            var g = ctx.createRadialGradient(sz / 2, sz / 2, 0, sz / 2, sz / 2, sz / 2);
            g.addColorStop(0.0, 'rgba(255,255,255,0.6)');
            g.addColorStop(0.35, 'rgba(255,255,255,0.22)');
            g.addColorStop(1.0, 'rgba(255,255,255,0)');
            ctx.fillStyle = g;
            ctx.fillRect(0, 0, sz, sz);
            var tex = new THREE.CanvasTexture(c);
            tex.minFilter = THREE.LinearFilter;
            return tex;
        })();

        var EDGE_SAMPLES = graph.edges.length > 1500 ? 72 : 132;
        // How many soft haze puffs ride each curve (0 disables haze on huge graphs to
        // keep the sprite/draw-call count sane).
        var HAZE_PUFFS = graph.edges.length > 1500 ? 0 : (graph.edges.length > 600 ? 4 : 7);
        // Re-samples an edge's point buffer every frame. Stars flow along a quadratic
        // Bézier arc (curved, not straight), jitter perpendicular to it (wandering, not a
        // clean path), and wrap seamlessly by fading in at the source / out at the target.
        function updateEdgePoints(eo, time) {
            var sMesh = nodeMap[eo.sourceId];
            var tMesh = nodeMap[eo.targetId];
            if (!sMesh || !tMesh) return;
            var s = sMesh.position, t = tMesh.position;
            var pos = eo.line.geometry.attributes.position;
            var col = eo.line.geometry.attributes.color;
            var base = eo.color, vary = eo.vary, n = eo.samples;

            // Edge direction + length.
            var dx = t.x - s.x, dy = t.y - s.y, dz = t.z - s.z;
            var len = Math.sqrt(dx * dx + dy * dy + dz * dz) + 1e-4;
            var ux = dx / len, uy = dy / len, uz = dz / len;

            // First perpendicular axis: the per-edge bend vector projected off the
            // direction. (Falls back to a world axis if it happens to be parallel.)
            var bnd = eo.bend;
            var d = bnd.x * ux + bnd.y * uy + bnd.z * uz;
            var px = bnd.x - ux * d, py = bnd.y - uy * d, pz = bnd.z - uz * d;
            var pl = Math.sqrt(px * px + py * py + pz * pz);
            if (pl < 1e-3) { px = uy; py = -ux; pz = 0; pl = Math.sqrt(px * px + py * py) + 1e-4; }
            px /= pl; py /= pl; pz /= pl;
            // Second perpendicular axis = dir × p (both unit, so result is unit).
            var qx = uy * pz - uz * py, qy = uz * px - ux * pz, qz = ux * py - uy * px;

            // Bézier control point: midpoint pushed out along the bend axis → a curved arc.
            // curveAmount of 0 collapses the control point onto the midpoint = straight.
            var bend = len * curveAmount;
            var cx = (s.x + t.x) * 0.5 + px * bend;
            var cy = (s.y + t.y) * 0.5 + py * bend;
            var cz = (s.z + t.z) * 0.5 + pz * bend;

            // Plain-line mode: sample the smooth Bézier into the line geometry (no flow,
            // no wander) and skip the star/haze work entirely.
            if (linesMode) {
                var lpos = eo.plainLine.geometry.attributes.position;
                var segs = eo.lineSegs;
                for (var li = 0; li <= segs; li++) {
                    var lf = li / segs;
                    var lo = 1 - lf;
                    lpos.setXYZ(li,
                        lo * lo * s.x + 2 * lo * lf * cx + lf * lf * t.x,
                        lo * lo * s.y + 2 * lo * lf * cy + lf * lf * t.y,
                        lo * lo * s.z + 2 * lo * lf * cz + lf * lf * t.z);
                }
                lpos.needsUpdate = true;
                eo.plainMat.opacity = Math.min(0.7, eo.mat.opacity); // tracks hover/search dimming
                return;
            }

            var phase = time * eo.speed + eo.flowOffset;
            var amp = Math.min(len * 0.01, 1.2); // wander amplitude — tight so the stream stays a thin filament

            for (var i = 0; i < n; i++) {
                // Evenly spaced in [0,1) and advancing with time → a moving stream,
                // clamped clear of the node cores.
                var frac = ((i / n) + phase) % 1;
                var f = frac; // full arc, node center to node center
                var omf = 1 - f;
                // Quadratic Bézier point along the arc.
                var bx = omf * omf * s.x + 2 * omf * f * cx + f * f * t.x;
                var by = omf * omf * s.y + 2 * omf * f * cy + f * f * t.y;
                var bz = omf * omf * s.z + 2 * omf * f * cz + f * f * t.z;
                // Per-star perpendicular wander so the path shimmers rather than tracks
                // a clean line. Two out-of-phase sines on the perpendicular basis.
                var w1 = Math.sin(frac * 12.566 + time * 1.3 + i * 1.7);
                var w2 = Math.cos(frac * 9.425 + time * 1.1 + i * 2.3);
                var wa = amp * (0.5 + 0.5 * vary[i]);
                bx += (px * w1 + qx * w2) * wa;
                by += (py * w1 + qy * w2) * wa;
                bz += (pz * w1 + qz * w2) * wa;
                pos.setXYZ(i, bx, by, bz);
                // Smoothstep fade in over the first 15%, out over the last 8% — the shorter
                // tail keeps the stream bright closer to the target so it doesn't read as
                // disconnecting early (stars move toward the target, so the gap is noticed there).
                var a = frac / 0.15; if (a > 1) a = 1; a = a * a * (3 - 2 * a);
                var bb = (1 - frac) / 0.08; if (bb > 1) bb = 1; if (bb < 0) bb = 0; bb = bb * bb * (3 - 2 * bb);
                var env = a * bb * vary[i];
                col.setXYZ(i, base.r * env, base.g * env, base.b * env);
            }
            pos.needsUpdate = true;
            col.needsUpdate = true;

            // Cloud haze: a chain of soft billboards riding the smooth curve, so the
            // edge reads as a glowing filament with the stars streaming through it.
            var haze = eo.haze;
            if (haze && hazeEnabled) {
                var hn = haze.length;
                var hscale = (len / hn) * 1.4; // overlap neighbours into a thin continuous band
                for (var k = 0; k < hn; k++) {
                    var hf = (k + 0.5) / hn; // match the stars' full-arc reach
                    var ho = 1 - hf;
                    haze[k].position.set(
                        ho * ho * s.x + 2 * ho * hf * cx + hf * hf * t.x,
                        ho * ho * s.y + 2 * ho * hf * cy + hf * hf * t.y,
                        ho * ho * s.z + 2 * ho * hf * cz + hf * hf * t.z
                    );
                    haze[k].scale.set(hscale, hscale, 1);
                }
                // Haze tracks the stream's brightness so hover/search dimming carries over.
                eo.hazeMat.opacity = eo.mat.opacity * 0.12;
            }
        }

        var edgeObjects = []; // { line, sourceId, targetId, mat, samples, color, vary, bend, speed, flowOffset }
        graph.edges.forEach(function(e) {
            var sid = typeof e.source === 'object' ? e.source.id : e.source;
            var tid = typeof e.target === 'object' ? e.target.id : e.target;
            var sMesh = nodeMap[sid];
            var tMesh = nodeMap[tid];
            if (!sMesh || !tMesh) return;

            var hue = hashToHue(sid);
            var col = new THREE.Color().setHSL(hue / 360, 0.6, 0.82);
            var n = EDGE_SAMPLES;
            // Per-star luminosity variance so the stream looks like distinct stars,
            // not a uniform dotted line. Baked once; the fade envelope is per-frame.
            var vary = new Float32Array(n);
            for (var i = 0; i < n; i++) vary[i] = 0.7 + 0.3 * Math.random();
            // Per-edge bend direction → each arc curves a different way.
            var bend = new THREE.Vector3(Math.random() - 0.5, Math.random() - 0.5, Math.random() - 0.5);
            if (bend.lengthSq() < 1e-6) bend.set(0, 1, 0);
            bend.normalize();
            var geo = new THREE.BufferGeometry();
            geo.setAttribute('position', new THREE.BufferAttribute(new Float32Array(n * 3), 3));
            geo.setAttribute('color', new THREE.BufferAttribute(new Float32Array(n * 3), 3));
            // Screen-space sizing (sizeAttenuation:false): a small fixed pixel size so
            // motes stay visible at any zoom — world-space sizing this small renders
            // sub-pixel and vanishes.
            var mat = new THREE.PointsMaterial({
                size: 2.2, map: dotTexture, vertexColors: true, transparent: true, opacity: 0.8,
                sizeAttenuation: false, blending: THREE.AdditiveBlending, depthWrite: false
            });
            var line = new THREE.Points(geo, mat);
            // Positions are rewritten every frame without recomputing the bounding
            // sphere, so disable frustum culling to keep the stream from being dropped.
            line.frustumCulled = false;
            scene.add(line);

            // Soft cloud haze riding the curve (a glowing filament around the stream).
            var haze = null, hazeMat = null;
            if (HAZE_PUFFS > 0) {
                // Normal (not additive) blending so overlapping puffs composite toward
                // the haze colour and can never sum past it → no white blow-out at hubs.
                hazeMat = new THREE.SpriteMaterial({
                    map: hazeTexture, color: col, transparent: true, opacity: 0.2,
                    blending: THREE.NormalBlending, depthWrite: false
                });
                haze = [];
                for (var h = 0; h < HAZE_PUFFS; h++) {
                    var puff = new THREE.Sprite(hazeMat);
                    puff.raycast = function() {};
                    puff.frustumCulled = false;
                    scene.add(puff);
                    haze.push(puff);
                }
            }

            // Plain-line alternative (hidden by default; shown via the Lines toggle).
            // Multi-segment so it follows the same curve/straight setting as the stream.
            var LINE_SEG = 12;
            var lgeo = new THREE.BufferGeometry();
            lgeo.setAttribute('position', new THREE.BufferAttribute(new Float32Array((LINE_SEG + 1) * 3), 3));
            var plainMat = new THREE.LineBasicMaterial({ color: col, transparent: true, opacity: 0.5, depthWrite: false });
            var plainLine = new THREE.Line(lgeo, plainMat);
            plainLine.frustumCulled = false;
            plainLine.visible = false;
            scene.add(plainLine);

            var eo = {
                line: line, sourceId: sid, targetId: tid, mat: mat, samples: n,
                color: col, vary: vary, bend: bend, haze: haze, hazeMat: hazeMat,
                plainLine: plainLine, plainMat: plainMat, lineSegs: LINE_SEG,
                speed: 0.05 + Math.random() * 0.05, // fraction of the link per second
                flowOffset: Math.random()
            };
            updateEdgePoints(eo, 0);
            edgeObjects.push(eo);
        });
        document.getElementById('edge-count-num').textContent = edgeObjects.length;

        // Labels default to on (toggleLabels flips from the initial Off state).
        window.toggleLabels();

        // ---- Raycasting for hover/click ----
        var raycaster = new THREE.Raycaster();
        raycaster.params.Line = { threshold: 2 };
        var mouse = new THREE.Vector2();
        var hoveredNode = null;
        var tooltipEl = document.getElementById('tooltip');

        function updateTooltip(node, x, y) {
            var titleEl = document.getElementById('tt-title');
            var metaEl = document.getElementById('tt-meta');
            var tagsEl = document.getElementById('tt-tags');
            titleEl.textContent = node.userData.title || node.userData.id;
            var ec = edgeCount[node.userData.id] || 0;
            metaEl.textContent = ec + ' connection' + (ec !== 1 ? 's' : '');
            tagsEl.innerHTML = '';
            if (node.userData.stub) {
                var stub = document.createElement('span');
                stub.className = 'tt-tag';
                stub.style.background = '#e67e22';
                stub.textContent = 'stub';
                tagsEl.appendChild(stub);
            }
            (node.userData.tags || []).forEach(function(t) {
                var tag = document.createElement('span');
                tag.className = 'tt-tag';
                tag.textContent = t;
                tagsEl.appendChild(tag);
            });
            tooltipEl.style.display = 'block';
            // Clamp against the tooltip's real size so it stays on-screen at any
            // viewport width (mobile widens it to 70vw via the media query).
            var tw = tooltipEl.offsetWidth || 240;
            var th = tooltipEl.offsetHeight || 80;
            var tx = Math.max(12, Math.min(x + 16, window.innerWidth - tw - 12));
            var ty = Math.max(12, Math.min(y + 16, window.innerHeight - th - 12));
            tooltipEl.style.left = tx + 'px';
            tooltipEl.style.top = ty + 'px';
        }

        function clearTooltip() {
            tooltipEl.style.display = 'none';
            hoveredNode = null;
        }

        function dimAllExcept(keepId) {
            var connected = new Set([keepId]);
            if (neighbors[keepId]) neighbors[keepId].forEach(function(id) { connected.add(id); });

            nodeMeshes.forEach(function(m) {
                var id = m.userData.id;
                if (id === keepId) {
                    m.scale.setScalar(2.2);
                    m.material.opacity = 1.0;
                    m.userData.dimmed = false;
                } else if (connected.has(id)) {
                    m.scale.setScalar(1.5);
                    m.material.opacity = m.userData.stub ? 0.7 : 1.0;
                    m.userData.dimmed = false;
                } else {
                    m.scale.setScalar(0.8);
                    m.material.opacity = 0.15;
                    m.userData.dimmed = true;
                }
            });
            edgeObjects.forEach(function(eo) {
                var s = eo.sourceId, t = eo.targetId;
                if (s === keepId || t === keepId) {
                    eo.mat.opacity = 0.9;
                } else {
                    eo.mat.opacity = 0.18; // dimmed, not hidden — keeps surrounding context
                }
            });
        }

        // Highlight a set of node ids (search results); dim everything else.
        function highlightSet(ids) {
            nodeMeshes.forEach(function(m) {
                if (ids.has(m.userData.id)) {
                    m.scale.setScalar(2.2);
                    m.material.opacity = m.userData.stub ? 0.9 : 1.0;
                    m.userData.dimmed = false;
                } else {
                    m.scale.setScalar(0.8);
                    m.material.opacity = 0.15;
                    m.userData.dimmed = true;
                }
            });
            edgeObjects.forEach(function(eo) {
                // For a search, only edges between two matches stay bright — an
                // edge to a dimmed node would otherwise read as a false match.
                eo.mat.opacity = (ids.has(eo.sourceId) && ids.has(eo.targetId)) ? 0.9 : 0.18;
            });
        }

        function resetHighlights() {
            nodeMeshes.forEach(function(m) {
                m.scale.setScalar(NODE_SIZE);
                m.material.opacity = m.userData.stub ? 0.7 : 1.0;
                m.userData.dimmed = false;
            });
            edgeObjects.forEach(function(eo) { eo.mat.opacity = 0.8; });
        }

        // Active search matches persist as the baseline highlight when not hovering.
        var searchMatches = null;
        function restoreBaseline() {
            if (searchMatches) highlightSet(searchMatches);
            else resetHighlights();
        }

        window.addEventListener('mousemove', function(e) {
            mouse.x = (e.clientX / window.innerWidth) * 2 - 1;
            mouse.y = -(e.clientY / window.innerHeight) * 2 + 1;
            raycaster.setFromCamera(mouse, camera);
            // Non-recursive: only test core star meshes, never their glow/halo children.
            var hits = raycaster.intersectObjects(nodeMeshes, false);
            if (hits.length > 0) {
                var hit = hits[0].object;
                if (hoveredNode !== hit) {
                    hoveredNode = hit;
                    dimAllExcept(hit.userData.id);
                }
                updateTooltip(hit, e.clientX, e.clientY);
                controls.autoRotate = false;
            } else {
                if (hoveredNode !== null) {
                    restoreBaseline();
                    clearTooltip();
                    controls.autoRotate = spinEnabled;
                }
            }
        });

        // Track pointer-down so an orbit drag that ends over a node doesn't navigate.
        var pointerDownPos = null;
        window.addEventListener('pointerdown', function(e) {
            pointerDownPos = { x: e.clientX, y: e.clientY };
        });

        // Last node selected by a tap on a touch device — first tap selects
        // (tooltip + highlight), a second tap on the same node navigates.
        var tappedId = null;

        window.addEventListener('click', function(e) {
            if (e.target !== renderer.domElement) return;
            if (pointerDownPos && (Math.abs(e.clientX - pointerDownPos.x) > 8 || Math.abs(e.clientY - pointerDownPos.y) > 8)) return;

            // Raycast at the click point — on touch devices mousemove never fires,
            // so hoveredNode can't be relied on here.
            mouse.x = (e.clientX / window.innerWidth) * 2 - 1;
            mouse.y = -(e.clientY / window.innerHeight) * 2 + 1;
            raycaster.setFromCamera(mouse, camera);
            var hits = raycaster.intersectObjects(nodeMeshes, false);

            if (hits.length === 0) {
                if (isTouch) {
                    tappedId = null;
                    restoreBaseline();
                    clearTooltip();
                    controls.autoRotate = spinEnabled;
                }
                return;
            }

            var hit = hits[0].object;
            var d = hit.userData;

            if (isTouch && tappedId !== d.id) {
                // First tap: select and show the tooltip instead of navigating.
                tappedId = d.id;
                hoveredNode = hit;
                dimAllExcept(d.id);
                updateTooltip(hit, e.clientX, e.clientY);
                controls.autoRotate = false;
                return;
            }

            if (d.stub || !d.path) return;
            var _t = new URL('../' + d.path, window.location.href).href;
            if (window.top !== window.self) { window.top.location.href = _t; } else { window.location.href = _t; }
        });

        // ---- Search ----
        var searchInput = document.getElementById('search-input');
        searchInput.addEventListener('input', function() {
            var q = this.value.toLowerCase().trim();
            if (!q) {
                searchMatches = null;
                resetHighlights();
                return;
            }
            var ids = new Set();
            nodeMeshes.forEach(function(m) {
                var d = m.userData;
                var title = (d.title || '').toLowerCase();
                var id = (d.id || '').toLowerCase();
                var tagHit = (d.tags || []).some(function(t) { return t.toLowerCase().includes(q); });
                if (title.includes(q) || id.includes(q) || tagHit) ids.add(d.id);
            });
            searchMatches = ids;
            highlightSet(ids);
        });

        searchInput.addEventListener('keydown', function(e) {
            if (e.key === 'Escape') { this.value = ''; searchMatches = null; resetHighlights(); clearTooltip(); }
        });

        // ---- Resize ----
        window.addEventListener('resize', function() {
            camera.aspect = window.innerWidth / window.innerHeight;
            camera.updateProjectionMatrix();
            renderer.setSize(window.innerWidth, window.innerHeight);
        });

        // ---- Simulated physics (gentle repulsion + centering) ----
        // We use a simple Euler integration to spread nodes
        var positions = nodeMeshes.map(function(m) { return m.position.clone(); });
        var velocities = nodeMeshes.map(function() { return new THREE.Vector3(); });

        function simulate(dt) {
            var center = new THREE.Vector3(0, 0, 0);
            var count = positions.length;

            for (var i = 0; i < count; i++) {
                var p = positions[i];
                var v = velocities[i];

                // Repulsion between all pairs
                for (var j = i + 1; j < count; j++) {
                    var q = positions[j];
                    var dx = p.x - q.x, dy = p.y - q.y, dz = p.z - q.z;
                    var dist = Math.sqrt(dx * dx + dy * dy + dz * dz) + 0.001;
                    var force = 800 / (dist * dist);
                    var fx = dx / dist * force, fy = dy / dist * force, fz = dz / dist * force;
                    v.x += fx * dt; v.y += fy * dt; v.z += fz * dt;
                    velocities[j].x -= fx * dt; velocities[j].y -= fy * dt; velocities[j].z -= fz * dt;
                }

                // Center gravity
                var cx = -p.x * 0.005, cy = -p.y * 0.005, cz = -p.z * 0.005;
                v.x += cx * dt; v.y += cy * dt; v.z += cz * dt;

                // Damping
                v.multiplyScalar(0.92);
            }

            // Apply velocities
            for (var k = 0; k < count; k++) {
                positions[k].add(velocities[k].clone().multiplyScalar(dt));
                nodeMeshes[k].position.copy(positions[k]);
            }

        }

        // ---- Animation loop ----
        var lastTime = 0;
        function animate(time) {
            requestAnimationFrame(animate);
            var dt = Math.min((time - lastTime) / 1000, 0.05);
            lastTime = time;

            // Hovering a node freezes the layout so it's easy to read/click.
            if (!hoveredNode) simulate(dt);

            var t = time * 0.001; // seconds, for ambient animation

            // Twinkle: each star layer breathes opacity at its own rate/phase.
            starLayers.forEach(function(L) {
                L.mat.opacity = Math.max(0, L.base + L.amp * Math.sin(t * L.speed + L.phase));
            });

            // Drifting nebula clouds: slow spin plus a gentle vertical bob.
            clouds.forEach(function(c) {
                c.sprite.material.rotation += c.spin * dt;
                c.sprite.position.y = c.basePos.y + Math.sin(t * c.drift + c.phase) * 12;
            });

            // Stream the stars along every edge (continues even when the layout is
            // frozen on hover, so connections always feel alive).
            edgeObjects.forEach(function(eo) { updateEdgePoints(eo, t); });

            if (labelsEnabled) {
                labelSprites.forEach(function(l) {
                    var p = l.mesh.position;
                    l.sprite.position.set(p.x, p.y + NODE_SIZE * 1.4, p.z);
                    // Labels follow the highlight state: only nodes that are part
                    // of the current hover/search set keep their name; the rest
                    // fade out so the lit set reads clearly.
                    l.sprite.material.opacity = l.mesh.userData.dimmed ? 0.22 : 0.95;
                });
            }

            // Ease the camera out to keep the whole graph in frame until the
            // user takes over with a drag or zoom.
            if (autoFit) {
                var target = fitDistance();
                var cur = camera.position.length();
                camera.position.setLength(cur + (target - cur) * 0.08);
            }

            controls.update();
            renderer.render(scene, camera);
        }
        animate(0);
    })();
    </script>
</body>
</html>`
		data := strings.Replace(nebulaHTML, "{{GRAPH_JSON}}", string(graphJSON), 1)
		err := os.WriteFile(filepath.Join(graphDir, "nebula.html"), []byte(data), 0644)
		if err != nil {
			fmt.Printf("Error writing graph nebula.html: %v\n", err)
		}
	}
