package main

import (
	"encoding/json"
	"fmt"
	"html"
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
	breadcrumbsHTML := buildBreadcrumbsHTML(sourcePath)
	siteNameJS, _ := json.Marshal(siteCfg.SiteName)
	graphModeJS, _ := json.Marshal(siteCfg.GraphMode)
	// Tables need a scroll container of their own so a wide table never forces
	// the whole page to scroll sideways.
	htmlContent = wrapTables(htmlContent)

	css := `
	/* Inter for UI + prose, Lilex for code */
	@import url('https://fonts.googleapis.com/css2?family=Inter:ital,opsz,wght@0,14..32,400..700;1,14..32,400..600&family=Lilex:wght@400;500&display=swap');

	/* ---------- Design tokens ---------- */
	:root, [data-theme="dark"] {
		color-scheme: dark;
		--bg: #0b0d11;
		--bg-elev: #101319;
		--sidebar-bg: #0d1016;
		--card-bg: #14181f;
		--hover: rgba(255,255,255,0.05);
		--border: rgba(255,255,255,0.07);
		--border-strong: rgba(255,255,255,0.14);
		--text: #c7d0dc;
		--heading: #f1f5f9;
		--muted: #7d8797;
		--link: #5eb1ef;
		--accent: #5eb1ef;
		--accent-hover: #8bc9f7;
		--accent-soft: rgba(94,177,239,0.13);
		--stub: #e8a34d;
		--graph-node: #4e586d;
		--graph-edge: rgba(255,255,255,0.14);
		--scrim: rgba(4,6,10,0.66);
		--shadow-lg: 0 24px 70px -16px rgba(0,0,0,0.75);
		--callout-tint: 13%;
		--callout-line: 26%;
	}
	[data-theme="light"] {
		color-scheme: light;
		--bg: #ffffff;
		--bg-elev: #f6f8fa;
		--sidebar-bg: #fbfcfd;
		--card-bg: #ffffff;
		--hover: rgba(15,23,42,0.045);
		--border: rgba(15,23,42,0.09);
		--border-strong: rgba(15,23,42,0.17);
		--text: #3f4855;
		--heading: #0d1420;
		--muted: #6b7480;
		--link: #1a6fd4;
		--accent: #1a6fd4;
		--accent-hover: #12539f;
		--accent-soft: rgba(26,111,212,0.09);
		--stub: #b8722a;
		--graph-node: #c2cad6;
		--graph-edge: rgba(15,23,42,0.16);
		--scrim: rgba(15,23,42,0.28);
		--shadow-lg: 0 24px 60px -18px rgba(15,23,42,0.25);
		--callout-tint: 7%;
		--callout-line: 22%;
	}

	/* ---------- Base ---------- */
	* { box-sizing: border-box; }
	html { scroll-behavior: smooth; }
	body {
		margin: 0; background: var(--bg); color: var(--text);
		font-family: 'Inter', -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
		font-size: 16px; line-height: 1.7; font-weight: 400; letter-spacing: -0.006em;
		-webkit-font-smoothing: antialiased; -moz-osx-font-smoothing: grayscale;
		font-feature-settings: 'liga' 1, 'calt' 1;
	}
	h1, h2, h3, h4, h5, h6 { color: var(--heading); font-weight: 600; line-height: 1.28; letter-spacing: -0.022em; }
	code, pre, kbd, samp { font-family: 'Lilex', ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace; }
	::selection { background: color-mix(in srgb, var(--accent) 26%, transparent); }
	:focus-visible { outline: 2px solid var(--accent); outline-offset: 2px; border-radius: 6px; }
	button { font: inherit; }
	/* Thin, unobtrusive scrollbars */
	* { scrollbar-width: thin; scrollbar-color: var(--border-strong) transparent; }
	::-webkit-scrollbar { width: 11px; height: 11px; }
	::-webkit-scrollbar-track { background: transparent; }
	::-webkit-scrollbar-thumb { background: var(--border-strong); border-radius: 99px; border: 3px solid transparent; background-clip: content-box; }
	::-webkit-scrollbar-thumb:hover { background: var(--muted); background-clip: content-box; }

	.layout { display: grid; grid-template-columns: 276px minmax(0, 1fr) 292px; width: 100%; align-items: start; }

	/* ---------- Left sidebar ---------- */
	.sidebar-nav { background: var(--sidebar-bg); border-right: 1px solid var(--border); padding: 22px 14px 40px; position: sticky; top: 0; height: 100vh; overflow-y: auto; overscroll-behavior: contain; }
	.site-name { display: flex; align-items: center; gap: 9px; margin: 0 0 18px; padding: 0 8px; font-size: 15px; font-weight: 600; letter-spacing: -0.02em; color: var(--heading); }
	.site-name .logo { width: 19px; height: 19px; flex: none; color: var(--accent); }
	.sidebar-header { display: flex; justify-content: space-between; align-items: center; margin: 0 8px 10px; }
	.sidebar-nav h2, .sidebar-right h2, .sidebar-section h3, .toc h3 {
		margin: 0; font-size: 11px; font-weight: 600; text-transform: uppercase; letter-spacing: 0.085em; color: var(--muted);
	}
	.icon-btn {
		display: inline-flex; align-items: center; justify-content: center;
		width: 28px; height: 28px; padding: 0; border: 1px solid transparent; border-radius: 8px;
		background: none; color: var(--muted); cursor: pointer; transition: background 0.14s, color 0.14s, border-color 0.14s;
	}
	.icon-btn:hover { background: var(--hover); color: var(--heading); border-color: var(--border); }
	.icon-btn svg { width: 15px; height: 15px; }
	.theme-toggle svg { width: 15px; height: 15px; }
	/* Search trigger */
	.search-bar {
		display: flex; align-items: center; gap: 8px; width: 100%; margin-bottom: 14px;
		background: var(--card-bg); border: 1px solid var(--border); border-radius: 10px;
		color: var(--muted); cursor: pointer; padding: 8px 10px; font-size: 13px; text-align: left;
		transition: border-color 0.15s, color 0.15s, background 0.15s;
	}
	.search-bar:hover { border-color: var(--border-strong); color: var(--text); }
	.search-bar svg { width: 14px; height: 14px; flex: none; opacity: 0.8; }
	.search-bar kbd {
		margin-left: auto; font-size: 10.5px; line-height: 1; padding: 4px 6px; border-radius: 5px;
		border: 1px solid var(--border); background: var(--bg-elev); color: var(--muted);
	}
	/* Nav tree */
	.nav-tree { font-size: 13.5px; }
	.nav-folder { margin: 1px 0; }
	.nav-folder-header {
		display: flex; align-items: center; gap: 6px; padding: 6px 8px; border-radius: 8px;
		cursor: pointer; color: var(--text); font-weight: 500; user-select: none;
		transition: background 0.14s, color 0.14s;
	}
	.nav-folder-header:hover { background: var(--hover); color: var(--heading); }
	.nav-folder-header .icon { display: inline-flex; align-items: center; justify-content: center; width: 13px; height: 13px; flex: none; color: var(--muted); transition: transform 0.18s ease; }
	.nav-folder-header .icon svg { width: 13px; height: 13px; }
	.nav-folder-header .icon.open { transform: rotate(90deg); }
	.nav-folder-header a { color: inherit; text-decoration: none; }
	.nav-folder-children { position: relative; margin: 1px 0 2px 13px; padding-left: 12px; display: none; }
	.nav-folder-children::before { content: ''; position: absolute; left: 0; top: 3px; bottom: 3px; width: 1px; background: var(--border); }
	.nav-folder-children.open { display: block; }
	.nav-page a {
		display: block; padding: 6px 9px; border-radius: 8px; color: var(--muted);
		text-decoration: none; overflow-wrap: anywhere; transition: background 0.14s, color 0.14s;
	}
	.nav-page a:visited { color: var(--muted); }
	.nav-page a:hover { background: var(--hover); color: var(--heading); }
	.nav-page.active a, .nav-page.active a:visited { background: var(--accent-soft); color: var(--accent); font-weight: 600; }

	/* ---------- Center content ---------- */
	/* width:100% is required alongside the auto margins — on a grid item, auto
	   margins alone switch the box from stretch to shrink-to-fit. */
	.content-col { min-width: 0; width: 100%; max-width: 840px; margin: 0 auto; padding: 60px clamp(20px, 4vw, 56px) 100px; }
	.breadcrumbs { display: flex; align-items: center; gap: 6px; flex-wrap: wrap; font-size: 12.5px; color: var(--muted); margin-bottom: 12px; }
	.breadcrumbs svg { width: 11px; height: 11px; opacity: 0.5; flex: none; }
	.content-col h1 { margin: 0 0 12px; font-size: clamp(1.85rem, 1.35rem + 1.3vw, 2.3rem); font-weight: 650; letter-spacing: -0.032em; }
	.page-meta { display: flex; align-items: center; flex-wrap: wrap; gap: 10px; font-size: 13px; color: var(--muted); padding-bottom: 24px; border-bottom: 1px solid var(--border); margin-bottom: 34px; }
	.page-meta-right:not(:empty)::before { content: '·'; margin-right: 10px; opacity: 0.55; }

	/* ---------- Prose ---------- */
	.markdown-body > *:first-child { margin-top: 0; }
	.markdown-body > *:last-child { margin-bottom: 0; }
	.markdown-body p { margin: 0 0 1.15em; }
	.markdown-body h2 { margin: 2.4em 0 0.7em; font-size: 1.35rem; }
	.markdown-body h3 { margin: 2em 0 0.6em; font-size: 1.1rem; }
	.markdown-body h4 { margin: 1.8em 0 0.5em; font-size: 1rem; }
	.markdown-body h5, .markdown-body h6 { margin: 1.6em 0 0.5em; font-size: 0.92rem; color: var(--muted); text-transform: uppercase; letter-spacing: 0.06em; }
	.markdown-body h1, .markdown-body h2, .markdown-body h3, .markdown-body h4, .markdown-body h5, .markdown-body h6 { scroll-margin-top: 28px; }
	.markdown-body a {
		color: var(--link); text-decoration: none; font-weight: 500;
		border-bottom: 1px solid color-mix(in srgb, var(--link) 32%, transparent);
		transition: color 0.15s, border-color 0.15s;
	}
	.markdown-body a:hover { color: var(--accent-hover); border-bottom-color: var(--accent-hover); }
	.markdown-body ul, .markdown-body ol { margin: 0 0 1.15em; padding-left: 1.35em; }
	.markdown-body li { margin: 0.35em 0; }
	.markdown-body li::marker { color: var(--muted); }
	.markdown-body li:has(> input[type="checkbox"]) { list-style: none; margin-left: -1.35em; }
	.markdown-body input[type="checkbox"] { accent-color: var(--accent); width: 13px; height: 13px; margin-right: 6px; vertical-align: -1px; }
	.markdown-body strong { color: var(--heading); font-weight: 600; }
	.markdown-body hr { border: 0; height: 1px; background: var(--border); margin: 2.6em 0; }
	.markdown-body blockquote { margin: 1.6em 0; padding: 2px 0 2px 18px; border-left: 2px solid var(--border-strong); color: var(--muted); }
	.markdown-body blockquote > *:last-child { margin-bottom: 0; }
	.markdown-body img { max-width: 100%; height: auto; display: block; margin: 0 auto; border-radius: 12px; }
	.markdown-body figure.image-caption { margin: 1.8em 0; text-align: center; }
	.markdown-body figure.image-caption img { margin: 0 auto 10px; }
	.markdown-body figure.image-caption figcaption { color: var(--muted); font-size: 12.5px; }
	/* Code */
	.markdown-body pre {
		margin: 1.6em 0; padding: 16px 18px; overflow-x: auto;
		background: var(--bg-elev); border: 1px solid var(--border); border-radius: 12px;
		font-size: 13.5px; line-height: 1.65; color: var(--text);
	}
	.markdown-body pre code { background: none; border: 0; padding: 0; font-size: inherit; color: inherit; }
	.markdown-body :not(pre) > code {
		background: var(--accent-soft); color: var(--heading); border-radius: 6px;
		padding: 0.15em 0.4em; font-size: 0.86em; overflow-wrap: anywhere;
	}
	/* Tables — hairline rows, no grid */
	.table-wrap { margin: 1.7em 0; overflow-x: auto; border: 1px solid var(--border); border-radius: 12px; }
	.markdown-body table { border-collapse: collapse; width: 100%; margin: 0; font-size: 14px; }
	.markdown-body table th {
		background: var(--bg-elev); color: var(--muted); font-weight: 600; font-size: 11.5px;
		text-transform: uppercase; letter-spacing: 0.07em; text-align: left;
		padding: 11px 15px; border: 0; border-bottom: 1px solid var(--border); white-space: nowrap;
	}
	.markdown-body table td { padding: 12px 15px; border: 0; border-top: 1px solid var(--border); vertical-align: top; }
	.markdown-body table tbody tr:first-child td { border-top: 0; }
	.markdown-body table tbody tr { transition: background 0.12s; }
	.markdown-body table tbody tr:hover { background: var(--hover); }
	/* Page footer */
	.page-footer { margin-top: 64px; padding-top: 20px; border-top: 1px solid var(--border); font-size: 12.5px; color: var(--muted); }
	.page-footer a { color: var(--muted); text-decoration: none; border-bottom: 1px solid var(--border-strong); transition: color 0.15s, border-color 0.15s; }
	.page-footer a:hover { color: var(--accent); border-bottom-color: var(--accent); }

	/* ---------- Right sidebar ---------- */
	.sidebar-right { background: var(--bg); border-left: 1px solid var(--border); padding: 22px 20px 40px; position: sticky; top: 0; height: 100vh; overflow-y: auto; overscroll-behavior: contain; }
	.graph-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 10px; }
	#local-graph {
		width: 100%; height: 190px; margin-bottom: 24px; overflow: hidden;
		background: var(--card-bg); border: 1px solid var(--border); border-radius: 12px;
	}
	#local-graph circle { fill: var(--graph-node); transition: fill 0.15s, opacity 0.15s; }
	#local-graph .node.current circle { fill: var(--accent); }
	#local-graph .node.hovered circle, #local-graph .node.neighbor circle { fill: var(--accent); }
	#local-graph .node.dimmed circle { opacity: 0.15; }
	.sidebar-section { margin-bottom: 24px; }
	.sidebar-section h3 { margin-bottom: 8px; }
	.sidebar-links + .sidebar-links, .sidebar-links h3:not(:first-child) { margin-top: 20px; }
	.sidebar-links ul { list-style: none; margin: 0 0 4px; padding: 0; }
	.sidebar-links li { margin: 0; }
	.sidebar-links a {
		display: block; padding: 5px 8px; margin: 0 -8px; border-radius: 7px;
		color: var(--muted); font-size: 13px; line-height: 1.45; text-decoration: none; overflow-wrap: anywhere;
		transition: background 0.14s, color 0.14s;
	}
	.sidebar-links a:hover { background: var(--hover); color: var(--accent); }
	a.stub-link, a.stub-link:hover { color: var(--stub); }
	/* Tags */
	.tags { margin-bottom: 24px; display: flex; flex-wrap: wrap; gap: 6px; align-items: center; }
	.tags-label { font-size: 11px; font-weight: 600; text-transform: uppercase; letter-spacing: 0.085em; color: var(--muted); width: 100%; margin-bottom: 2px; }
	.tag {
		display: inline-block; padding: 3px 9px; border-radius: 999px; font-size: 12px; font-weight: 500;
		background: var(--accent-soft); color: var(--accent); border: 1px solid transparent; letter-spacing: -0.005em;
	}
	/* Table of contents */
	.toc { margin-bottom: 24px; }
	.toc h3 { margin-bottom: 10px; }
	.toc-list { list-style: none; margin: 0; padding: 0; border-left: 1px solid var(--border); }
	.toc-item { margin: 0; }
	.toc-item a {
		display: block; padding: 5px 0 5px 12px; margin-left: -1px;
		border-left: 2px solid transparent; color: var(--muted); font-size: 13px;
		text-decoration: none; transition: color 0.15s, border-color 0.15s;
	}
	.toc-item a:hover { color: var(--heading); }
	.toc-item.active > a { color: var(--accent); border-left-color: var(--accent); }
	.toc-item.level-2 a { padding-left: 24px; }
	.toc-item.level-3 a { padding-left: 36px; }
	.toc-item.level-4 a { padding-left: 48px; }
	.toc-item.level-5 a { padding-left: 58px; }
	.toc-item.level-6 a { padding-left: 68px; }

	/* ---------- Callouts ---------- */
	.callout {
		--callout-color: var(--accent);
		margin: 1.7em 0; padding: 14px 16px; border-radius: 12px;
		background: color-mix(in srgb, var(--callout-color) var(--callout-tint), var(--bg));
		border: 1px solid color-mix(in srgb, var(--callout-color) var(--callout-line), transparent);
	}
	.callout-title {
		display: flex; align-items: center; gap: 8px; padding: 0 0 6px;
		color: var(--callout-color); font-weight: 600; font-size: 14px; letter-spacing: -0.012em;
	}
	.callout-icon { display: inline-flex; align-items: center; flex-shrink: 0; }
	.callout-icon svg { width: 15px; height: 15px; }
	.callout-content { padding: 0; }
	.callout-content > *:last-child { margin-bottom: 0; }
	.callout-content > *:first-child { margin-top: 0; }
	.callout .table-wrap, .callout pre { background: color-mix(in srgb, var(--callout-color) 6%, var(--bg)); }
	/* Callout accent colors */
	[data-theme="light"] .callout[data-callout="note"], [data-theme="light"] .callout[data-callout="info"] { --callout-color: #1a6fd4; }
	[data-theme="light"] .callout[data-callout="question"], [data-theme="light"] .callout[data-callout="help"], [data-theme="light"] .callout[data-callout="faq"] { --callout-color: #1b62b8; }
	[data-theme="light"] .callout[data-callout="tip"], [data-theme="light"] .callout[data-callout="hint"], [data-theme="light"] .callout[data-callout="important"],
	[data-theme="light"] .callout[data-callout="success"], [data-theme="light"] .callout[data-callout="check"], [data-theme="light"] .callout[data-callout="done"] { --callout-color: #0f8a5f; }
	[data-theme="light"] .callout[data-callout="warning"], [data-theme="light"] .callout[data-callout="caution"], [data-theme="light"] .callout[data-callout="attention"] { --callout-color: #a06b00; }
	[data-theme="light"] .callout[data-callout="danger"], [data-theme="light"] .callout[data-callout="error"] { --callout-color: #cc3a2c; }
	[data-theme="light"] .callout[data-callout="example"] { --callout-color: #8043c8; }
	:root .callout[data-callout="note"], :root .callout[data-callout="info"],
	[data-theme="dark"] .callout[data-callout="note"], [data-theme="dark"] .callout[data-callout="info"] { --callout-color: #5eb1ef; }
	:root .callout[data-callout="question"], :root .callout[data-callout="help"], :root .callout[data-callout="faq"],
	[data-theme="dark"] .callout[data-callout="question"], [data-theme="dark"] .callout[data-callout="help"], [data-theme="dark"] .callout[data-callout="faq"] { --callout-color: #7aa7f2; }
	:root .callout[data-callout="tip"], :root .callout[data-callout="hint"], :root .callout[data-callout="important"],
	:root .callout[data-callout="success"], :root .callout[data-callout="check"], :root .callout[data-callout="done"],
	[data-theme="dark"] .callout[data-callout="tip"], [data-theme="dark"] .callout[data-callout="hint"], [data-theme="dark"] .callout[data-callout="important"],
	[data-theme="dark"] .callout[data-callout="success"], [data-theme="dark"] .callout[data-callout="check"], [data-theme="dark"] .callout[data-callout="done"] { --callout-color: #3ecf8e; }
	:root .callout[data-callout="warning"], :root .callout[data-callout="caution"], :root .callout[data-callout="attention"],
	[data-theme="dark"] .callout[data-callout="warning"], [data-theme="dark"] .callout[data-callout="caution"], [data-theme="dark"] .callout[data-callout="attention"] { --callout-color: #e8b04b; }
	:root .callout[data-callout="danger"], :root .callout[data-callout="error"],
	[data-theme="dark"] .callout[data-callout="danger"], [data-theme="dark"] .callout[data-callout="error"] { --callout-color: #f2726f; }
	:root .callout[data-callout="example"],
	[data-theme="dark"] .callout[data-callout="example"] { --callout-color: #c39bf5; }
	/* Foldable callout */
	.callout summary {
		display: flex; align-items: center; gap: 8px; padding: 0; list-style: none; cursor: pointer;
		color: var(--callout-color); font-weight: 600; font-size: 14px; user-select: none;
	}
	.callout summary::-webkit-details-marker { display: none; }
	.callout[open] summary { padding-bottom: 6px; }
	.callout summary .callout-title { padding: 0; }
	/* CSS-drawn chevron (no glyph font needed) for both foldable callouts and collapsibles */
	.callout summary::before, details.collapsible summary::before {
		content: ''; flex: none; width: 6px; height: 6px; margin-right: 2px;
		border-right: 1.6px solid currentColor; border-bottom: 1.6px solid currentColor;
		transform: rotate(-45deg); transition: transform 0.18s ease; opacity: 0.75;
	}
	.callout[open] summary::before, details.collapsible[open] summary::before { transform: rotate(45deg); }
	/* Collapsible sections */
	details.collapsible { margin: 1.7em 0; border: 1px solid var(--border); border-radius: 12px; background: var(--card-bg); overflow: hidden; }
	details.collapsible summary {
		display: flex; align-items: center; gap: 10px; padding: 12px 16px; cursor: pointer;
		font-weight: 600; font-size: 14.5px; color: var(--heading); list-style: none; user-select: none;
		transition: background 0.14s;
	}
	details.collapsible summary::-webkit-details-marker { display: none; }
	details.collapsible summary:hover { background: var(--hover); }
	details.collapsible summary::before { border-color: var(--muted); }
	details.collapsible .collapsible-content { padding: 4px 16px 14px; }
	details.collapsible .collapsible-content > *:last-child { margin-bottom: 0; }

	/* ---------- Overlays: search + full graph ---------- */
	.search-overlay, .full-graph-overlay {
		position: fixed; inset: 0; z-index: 1000; display: flex;
		background: var(--scrim); backdrop-filter: blur(8px) saturate(140%); -webkit-backdrop-filter: blur(8px) saturate(140%);
		animation: fade 0.16s ease-out;
	}
	.search-overlay { align-items: flex-start; justify-content: center; padding: 12vh 16px 16px; }
	.full-graph-overlay { align-items: center; justify-content: center; padding: 24px; }
	@keyframes fade { from { opacity: 0; } to { opacity: 1; } }
	@keyframes pop { from { opacity: 0; transform: translateY(-10px) scale(0.985); } to { opacity: 1; transform: none; } }
	.search-modal, .full-graph-modal {
		background: var(--bg-elev); border: 1px solid var(--border-strong); border-radius: 16px;
		box-shadow: var(--shadow-lg); display: flex; flex-direction: column; overflow: hidden;
		animation: pop 0.18s cubic-bezier(0.22, 1, 0.36, 1);
	}
	.search-modal { width: 100%; max-width: 620px; max-height: 72vh; }
	.full-graph-modal { width: 100%; max-width: 1400px; height: 86vh; }
	.search-header { display: flex; align-items: center; gap: 11px; padding: 14px 16px; border-bottom: 1px solid var(--border); }
	.search-header .search-icon { width: 15px; height: 15px; flex: none; color: var(--muted); }
	#search-input { flex: 1; min-width: 0; background: none; border: none; color: var(--heading); font-size: 15px; outline: none; }
	#search-input::placeholder { color: var(--muted); }
	#search-results { overflow-y: auto; padding: 8px; }
	.search-result { display: block; padding: 10px 12px; border-radius: 10px; text-decoration: none; color: var(--text); }
	.search-result:hover, .search-result.selected { background: var(--hover); }
	.search-result.selected { box-shadow: inset 0 0 0 1px var(--border-strong); }
	.search-result-title { font-weight: 600; font-size: 14.5px; margin-bottom: 3px; color: var(--heading); letter-spacing: -0.014em; }
	.search-result-tags { margin-bottom: 5px; display: flex; flex-wrap: wrap; gap: 5px; }
	.search-result-snippet { font-size: 13px; color: var(--muted); line-height: 1.55; }
	.search-result-snippet mark { background: var(--accent-soft); color: var(--accent); border-radius: 3px; padding: 0 2px; }
	.search-empty { padding: 28px 20px; text-align: center; color: var(--muted); font-size: 13.5px; }
	.search-footer { display: flex; align-items: center; gap: 14px; padding: 9px 14px; border-top: 1px solid var(--border); font-size: 11.5px; color: var(--muted); }
	.search-footer kbd { font-size: 10.5px; padding: 3px 5px; border-radius: 4px; border: 1px solid var(--border); background: var(--card-bg); margin-right: 4px; }
	.full-graph-header { display: flex; justify-content: space-between; align-items: center; padding: 13px 16px; border-bottom: 1px solid var(--border); }
	.full-graph-header h2 { margin: 0; font-size: 11px; font-weight: 600; text-transform: uppercase; letter-spacing: 0.085em; color: var(--muted); }
	#full-graph-container { flex: 1; overflow: hidden; }
	#full-graph-container iframe { width: 100%; height: 100%; border: none; background: var(--bg); }
	.full-graph-iframe { width: 100%; height: 100%; border: none; }
	/* Embedded graph (via graph directive in a page) */
	.graph-embed { width: 100%; height: 480px; margin: 1.8em 0; border: 1px solid var(--border); border-radius: 14px; overflow: hidden; background: #06060f; }
	.graph-embed iframe { display: block; width: 100%; height: 100%; border: 0; }

	/* ---------- Mobile chrome (hidden on desktop) ---------- */
	.mobile-header, .nav-scrim { display: none; }

	/* ---------- Responsive ---------- */
	@media (max-width: 1180px) {
		.layout { grid-template-columns: 264px minmax(0, 1fr); }
		.sidebar-right { display: none; }
		.content-col { padding-top: 48px; }
	}
	@media (max-width: 900px) {
		.layout { grid-template-columns: 1fr; padding-top: 56px; }
		.mobile-header {
			position: fixed; top: 0; left: 0; right: 0; z-index: 998; height: 56px;
			display: flex; align-items: center; gap: 10px; padding: 8px 12px;
			background: color-mix(in srgb, var(--bg) 82%, transparent);
			backdrop-filter: blur(12px) saturate(160%); -webkit-backdrop-filter: blur(12px) saturate(160%);
			border-bottom: 1px solid var(--border);
		}
		.mobile-header .mobile-site-name { flex: 1; min-width: 0; font-size: 14.5px; font-weight: 600; letter-spacing: -0.02em; color: var(--heading); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
		.sidebar-nav {
			position: fixed; top: 0; left: 0; z-index: 1001; height: 100vh; width: min(320px, 86vw);
			transform: translateX(-100%); transition: transform 0.26s cubic-bezier(0.22, 1, 0.36, 1);
			box-shadow: var(--shadow-lg);
		}
		.sidebar-nav.open { transform: translateX(0); }
		.nav-scrim { display: block; position: fixed; inset: 0; z-index: 1000; background: var(--scrim); opacity: 0; pointer-events: none; transition: opacity 0.22s ease; }
		.nav-scrim.open { opacity: 1; pointer-events: auto; }
		.content-col { padding: 32px 20px 72px; max-width: none; }
		.sidebar-right { display: block; position: static; height: auto; padding: 28px 20px 56px; border-left: none; border-top: 1px solid var(--border); }
		.full-graph-overlay { padding: 0; }
		.full-graph-modal { max-width: none; height: 100%; border-radius: 0; border: none; }
		.search-overlay { padding: 8vh 12px 12px; }
	}
	@media (prefers-reduced-motion: reduce) {
		html { scroll-behavior: auto; }
		*, *::before, *::after { animation-duration: 0.01ms !important; animation-iteration-count: 1 !important; transition-duration: 0.01ms !important; }
	}
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
        <button id="mobile-nav-toggle" class="icon-btn" aria-label="Toggle navigation" aria-expanded="false">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><line x1="3" y1="6" x2="21" y2="6"/><line x1="3" y1="12" x2="21" y2="12"/><line x1="3" y1="18" x2="21" y2="18"/></svg>
        </button>
        <span class="mobile-site-name">%[12]s</span>
        <button id="mobile-search" class="icon-btn" aria-label="Search">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><circle cx="11" cy="11" r="7"/><line x1="16.5" y1="16.5" x2="21" y2="21"/></svg>
        </button>
    </div>
    <div class="nav-scrim" id="nav-scrim"></div>
<div class="layout">
    <aside class="sidebar-nav">
        <div class="site-name">
            <svg class="logo" viewBox="0 0 32 32" fill="none" stroke="currentColor" stroke-linecap="round" aria-hidden="true"><line x1="10" y1="12" x2="16" y2="22" stroke-width="1.5"/><line x1="22" y1="12" x2="16" y2="22" stroke-width="1.5"/><line x1="10" y1="12" x2="22" y2="12" stroke-width="1.5"/><circle cx="10" cy="12" r="2.6" fill="currentColor" stroke="none"/><circle cx="22" cy="12" r="2.6" fill="currentColor" stroke="none"/><circle cx="16" cy="22" r="2.6" fill="currentColor" stroke="none"/></svg>
            <span>%[12]s</span>
        </div>
        <div class="sidebar-header">
            <h2>Browse</h2>
            <button class="icon-btn theme-toggle" id="theme-toggle" title="Toggle dark/light mode" aria-label="Toggle dark/light mode"></button>
        </div>
        <button id="open-search" class="search-bar" type="button">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><circle cx="11" cy="11" r="7"/><line x1="16.5" y1="16.5" x2="21" y2="21"/></svg>
            Search
            <kbd id="search-hint">&#8984;K</kbd>
        </button>
        <nav class="nav-tree" id="nav-tree"></nav>
    </aside>
    <main class="content-col">
        %[16]s
        <h1>%[1]s</h1>
        <div class="page-meta">
            <span class="page-meta-left">%[4]s</span>
            <span class="page-meta-right">%[5]s</span>
        </div>
        <div class="markdown-body">
            %[6]s
        </div>
        <footer class="page-footer">
            Created by <a href="https://nebula-md.j6n.dev" target="_blank" rel="noopener">Nebula</a>
        </footer>
    </main>
    <aside class="sidebar-right">
        <div class="graph-header">
            <h2>Graph</h2>
            <button id="open-full-graph" class="icon-btn" title="Full vault graph" aria-label="Open full vault graph">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="15 3 21 3 21 9"/><polyline points="9 21 3 21 3 15"/><line x1="21" y1="3" x2="14" y2="10"/><line x1="3" y1="21" x2="10" y2="14"/></svg>
            </button>
        </div>
        <div id="local-graph"></div>
        %[7]s
        %[8]s
        %[9]s
    </aside>
</div>
<script>
window.siteName = %[14]s;
    // Mobile nav drawer (toggle + scrim + Escape)
    (function() {
        var navToggle = document.getElementById('mobile-nav-toggle');
        var sidebarNav = document.querySelector('.sidebar-nav');
        var scrim = document.getElementById('nav-scrim');
        if (!navToggle || !sidebarNav) return;
        function setNav(open) {
            sidebarNav.classList.toggle('open', open);
            if (scrim) scrim.classList.toggle('open', open);
            navToggle.setAttribute('aria-expanded', open ? 'true' : 'false');
        }
        navToggle.addEventListener('click', function() { setNav(!sidebarNav.classList.contains('open')); });
        if (scrim) scrim.addEventListener('click', function() { setNav(false); });
        sidebarNav.addEventListener('click', function(e) { if (e.target.closest('a')) setNav(false); });
        document.addEventListener('keydown', function(e) { if (e.key === 'Escape') setNav(false); });
    })();
    // Show the platform-correct search shortcut hint
    (function() {
        var hint = document.getElementById('search-hint');
        if (hint && !/Mac|iPhone|iPad/.test(navigator.platform || navigator.userAgent)) hint.textContent = 'Ctrl K';
    })();
window.siteTheme = "%[13]s";
window.graphMode = %[15]s;
window.pageGraphData = %[10]s;
window.navTree = %[11]s;
</script>
<script>
// ---- Nav: render immediately ----
(function() {
    function escHtml(s) { return s.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;'); }
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
    var CHEVRON = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round"><polyline points="9 6 15 12 9 18"/></svg>';
    var currentHref = (window.pageGraphData && window.pageGraphData.currentHref) || '';
    // A folder holding the current page starts expanded, so landing on a nested
    // page always shows where you are instead of a wall of collapsed folders.
    function holdsCurrent(node) {
        if (!currentHref) return false;
        if (node.href) return node.href === currentHref;
        if (node.indexHref === currentHref) return true;
        if (!node.children) return false;
        for (var i = 0; i < node.children.length; i++) { if (holdsCurrent(node.children[i])) return true; }
        return false;
    }
    function buildNavHTML(nodes, parentPath) {
        var html = '';
        var depth = currentHref ? currentHref.split('/').length - 1 : 0;
        var prefix = depth > 0 ? '../'.repeat(depth) : '';
        var expandedFolders = getExpandedFolders();
        for (var i = 0; i < nodes.length; i++) {
            var node = nodes[i];
            if (node.children) {
                var folderId = 'navf-' + (parentPath ? parentPath + '-' : '') + node.name;
                var isOpen = expandedFolders.indexOf(folderId) >= 0 || holdsCurrent(node);
                var folderLabel = escHtml(node.name);
                var iconClass = isOpen ? 'icon open' : 'icon';
                html += '<div class="nav-folder">';
                if (node.indexHref) {
                    var folderLink = '<a href="' + prefix + node.indexHref + '" onclick="event.stopPropagation()">' + folderLabel + '</a>';
                    html += '<div class="nav-folder-header" onclick="toggleNavFolder(this)">';
                    html += '<span class="' + iconClass + '">' + CHEVRON + '</span>' + folderLink;
                    html += '</div>';
                } else {
                    html += '<div class="nav-folder-header" onclick="toggleNavFolder(this)">';
                    html += '<span class="' + iconClass + '">' + CHEVRON + '</span>' + folderLabel;
                    html += '</div>';
                }
                html += '<div class="nav-folder-children' + (isOpen ? ' open' : '') + '" id="' + folderId + '">';
                html += buildNavHTML(node.children, folderId);
                html += '</div></div>';
            } else {
                var href = prefix + node.href;
                var isActive = currentHref && currentHref === node.href;
                var cls = isActive ? 'nav-page active' : 'nav-page';
                html += '<div class="' + cls + '"><a href="' + href + '">' + escHtml(node.name) + '</a></div>';
            }
        }
        return html;
    }
    function getExpandedFolders() {
        try { return JSON.parse(sessionStorage.getItem('nebula-nav-open') || '[]'); } catch(e) { return []; }
    }
    function saveExpandedFolders(folders) {
        try { sessionStorage.setItem('nebula-nav-open', JSON.stringify(folders)); } catch(e) {}
    }
    var navEl = document.getElementById('nav-tree');
    if (navEl) {
        navEl.innerHTML = buildNavHTML(window.navTree || [], '');
        // Keep the current page visible in a long, deeply nested tree.
        var active = navEl.querySelector('.nav-page.active');
        if (active) {
            var rail = document.querySelector('.sidebar-nav');
            if (rail && active.offsetTop > rail.clientHeight - 120) rail.scrollTop = active.offsetTop - rail.clientHeight / 2;
        }
    }
})();
</script>
<script>
// ---- Theme toggle ----
(function() {
    var html = document.documentElement;
    var toggle = document.getElementById('theme-toggle');
    // Apply saved preference or default to dark
    var saved = localStorage.getItem('nebula-theme');
    if (saved) { html.setAttribute('data-theme', saved); }
    else { html.setAttribute('data-theme', 'dark'); }
    updateIcon();
    toggle.addEventListener('click', function() {
        var current = html.getAttribute('data-theme');
        var next = current === 'dark' ? 'light' : 'dark';
        html.setAttribute('data-theme', next);
        localStorage.setItem('nebula-theme', next);
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
        var w = container.clientWidth || 260;
        var h = container.clientHeight || 190;
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
            // Obsidian-style circular bias: a weak per-node pull toward the center
            // rounds out the silhouette and keeps outliers inside the small canvas.
            .force('x', _d3.forceX(w / 2).strength(0.08))
            .force('y', _d3.forceY(h / 2).strength(0.08))
            .force('collision', _d3.forceCollide().radius(15));
        // Render nodes/links immediately (before sim ticks)
        // Colors come from CSS custom properties so the graph follows the active theme.
        var NODE_FILL = 'var(--graph-node)', CUR_FILL = 'var(--accent)', EDGE = 'var(--graph-edge)';
        function baseFill(n) { return n.current ? CUR_FILL : NODE_FILL; }
        var link = linkG.selectAll('line').data(edges).enter().append('line').style('stroke', EDGE).style('stroke-width', 1.5);
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
            node.selectAll('circle').style('fill', function(n) { return n.id === nid || (n.id === pageId || d.id === pageId) ? CUR_FILL : NODE_FILL; });
            node.selectAll('circle').style('opacity', function(n) { return n.id !== nid && n.id !== pageId && d.id !== pageId ? '0.15' : '1'; });
            link.style('stroke', function(l) { return (l.source.id === nid || l.target.id === nid || l.source.id === pageId || l.target.id === pageId) ? CUR_FILL : EDGE; });
            link.style('stroke-opacity', function(l) { return (l.source.id === nid || l.target.id === nid || l.source.id === pageId || l.target.id === pageId) ? 1 : 0.15; });
        });
        node.on('mouseout', function(e, d) {
            if (draggingNodeId !== null) return;
            node.classed('hovered', false).classed('neighbor', false).classed('dimmed', false);
            node.selectAll('circle').style('fill', baseFill).style('opacity', '1');
            link.style('stroke', EDGE).style('stroke-opacity', 1);
        });
        node.append('circle').attr('r', function(d) { return d.current ? 6 : 3.5 }).style('fill', baseFill);
        node.append('text').attr('dx', 0).attr('dy', function(d) { var r = d.current ? 6 : 3.5; return r + 11; }).attr('text-anchor', 'middle').style('font-size', '9.5px').style('font-weight', '500').style('fill', 'currentColor').style('opacity', '0.65').text(function(d) { return d.title; });
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
            <button id="close-full-graph" class="icon-btn" aria-label="Close">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><line x1="6" y1="6" x2="18" y2="18"/><line x1="18" y1="6" x2="6" y2="18"/></svg>
            </button>
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
    <div class="search-modal" role="dialog" aria-modal="true" aria-label="Search">
        <div class="search-header">
            <svg class="search-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><circle cx="11" cy="11" r="7"/><line x1="16.5" y1="16.5" x2="21" y2="21"/></svg>
            <input id="search-input" type="text" placeholder="Search pages, tags, and content..." autocomplete="off" spellcheck="false" />
            <button id="close-search" class="icon-btn" aria-label="Close search">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><line x1="6" y1="6" x2="18" y2="18"/><line x1="18" y1="6" x2="6" y2="18"/></svg>
            </button>
        </div>
        <div id="search-results"></div>
        <div class="search-footer">
            <span><kbd>&#8593;</kbd><kbd>&#8595;</kbd> navigate</span>
            <span><kbd>&#8629;</kbd> open</span>
            <span><kbd>esc</kbd> close</span>
        </div>
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

    // Index of the keyboard-highlighted result, or -1 when nothing is selected.
    var selected = -1;

    function paintSelection() {
        var rows = results.querySelectorAll('.search-result');
        for (var i = 0; i < rows.length; i++) rows[i].classList.toggle('selected', i === selected);
        if (selected >= 0 && rows[selected]) rows[selected].scrollIntoView({ block: 'nearest' });
    }

    function move(delta) {
        var rows = results.querySelectorAll('.search-result');
        if (!rows.length) return;
        selected = (selected + delta + rows.length) %% rows.length;
        paintSelection();
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
        selected = -1;
        if (term.length === 0) {
            results.innerHTML = '<div class="search-empty">Start typing to search your vault.</div>';
            return;
        }
        if (matches.length === 0) {
            results.innerHTML = '<div class="search-empty">No results for &ldquo;' + escHtml(term) + '&rdquo;</div>';
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
        selected = 0;
        paintSelection();
    }

    function openSearch() {
        overlay.style.display = 'flex';
        input.focus();
        input.select();
        if (!searchIndex) {
            var depth = (window.pageGraphData && window.pageGraphData.currentHref) ? window.pageGraphData.currentHref.split('/').length - 1 : 0;
            var prefix = depth > 0 ? '../'.repeat(depth) : '';
            fetch(prefix + 'search.json').then(function(r) { return r.json(); }).then(function(data) {
                searchIndex = data;
                doSearch(input.value);
            }).catch(function() { searchIndex = []; });
        } else {
            doSearch(input.value);
        }
    }

    function closeSearch() {
        overlay.style.display = 'none';
        input.value = '';
        results.innerHTML = '';
        selected = -1;
    }

    function isOpen() { return overlay.style.display === 'flex'; }

    openBtn.addEventListener('click', openSearch);
    var mobileSearchBtn = document.getElementById('mobile-search');
    if (mobileSearchBtn) mobileSearchBtn.addEventListener('click', openSearch);
    input.addEventListener('input', function() { doSearch(input.value); });
    closeBtn.addEventListener('click', closeSearch);
    results.addEventListener('mousemove', function(e) {
        var row = e.target.closest('.search-result');
        if (!row) return;
        var rows = Array.prototype.slice.call(results.querySelectorAll('.search-result'));
        var idx = rows.indexOf(row);
        if (idx >= 0 && idx !== selected) { selected = idx; paintSelection(); }
    });

    overlay.addEventListener('click', function(e) { if (e.target === overlay) closeSearch(); });

    document.addEventListener('keydown', function(e) {
        // Cmd/Ctrl+K opens the palette from anywhere on the page.
        if ((e.metaKey || e.ctrlKey) && (e.key === 'k' || e.key === 'K')) {
            e.preventDefault();
            isOpen() ? closeSearch() : openSearch();
            return;
        }
        if (!isOpen()) return;
        if (e.key === 'Escape') { closeSearch(); }
        else if (e.key === 'ArrowDown') { e.preventDefault(); move(1); }
        else if (e.key === 'ArrowUp') { e.preventDefault(); move(-1); }
        else if (e.key === 'Enter') {
            var rows = results.querySelectorAll('.search-result');
            if (selected >= 0 && rows[selected]) { e.preventDefault(); window.location.href = rows[selected].getAttribute('href'); }
        }
    });
})();
</script>
<script>
// ---- Table of contents: highlight the section you're reading ----
(function() {
    var items = document.querySelectorAll('.toc-item');
    if (!items.length || !('IntersectionObserver' in window)) return;
    var byId = {};
    var headings = [];
    items.forEach(function(item) {
        var a = item.querySelector('a');
        if (!a) return;
        var id = decodeURIComponent(a.getAttribute('href').slice(1));
        var h = document.getElementById(id);
        if (!h) return;
        byId[id] = item;
        headings.push(h);
    });
    if (!headings.length) return;
    var visible = new Set();
    function paint() {
        // The topmost heading currently on screen wins; when none is visible
        // (mid-section scrolling) keep the last heading scrolled past.
        var active = null;
        for (var i = 0; i < headings.length; i++) {
            if (visible.has(headings[i].id)) { active = headings[i]; break; }
        }
        if (!active) {
            for (var j = 0; j < headings.length; j++) {
                if (headings[j].getBoundingClientRect().top < 120) active = headings[j];
            }
        }
        // At the top of the page nothing has been scrolled past yet — mark the first section.
        if (!active && window.scrollY < 160) active = headings[0];
        items.forEach(function(item) { item.classList.remove('active'); });
        if (active && byId[active.id]) byId[active.id].classList.add('active');
    }
    var obs = new IntersectionObserver(function(entries) {
        entries.forEach(function(en) {
            if (en.isIntersecting) visible.add(en.target.id); else visible.delete(en.target.id);
        });
        paint();
    }, { rootMargin: '-80px 0px -70%% 0px' });
    headings.forEach(function(h) { obs.observe(h); });
    window.addEventListener('scroll', paint, { passive: true });
    paint();
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
		siteCfg.SiteName, siteCfg.SiteTheme, string(siteNameJS), string(graphModeJS),
		breadcrumbsHTML)
}

// wrapTables puts each markdown table in a horizontally scrollable container so
// wide tables scroll on their own instead of stretching the page.
func wrapTables(htmlContent string) string {
	if !strings.Contains(htmlContent, "<table>") {
		return htmlContent
	}
	s := strings.ReplaceAll(htmlContent, "<table>", "<div class=\"table-wrap\"><table>")
	return strings.ReplaceAll(s, "</table>", "</table></div>")
}

// buildBreadcrumbsHTML renders the folder trail above the page title, e.g.
// "documentation / features" for documentation/features/theming.md. Pages at the
// vault root have no trail and render nothing.
func buildBreadcrumbsHTML(sourcePath string) string {
	dir := filepath.ToSlash(filepath.Dir(sourcePath))
	if dir == "." || dir == "/" || dir == "" {
		return ""
	}
	const chevron = "<svg viewBox=\"0 0 24 24\" fill=\"none\" stroke=\"currentColor\" stroke-width=\"2.2\" stroke-linecap=\"round\" stroke-linejoin=\"round\" aria-hidden=\"true\"><polyline points=\"9 6 15 12 9 18\"/></svg>"
	var b strings.Builder
	b.WriteString("<nav class=\"breadcrumbs\" aria-label=\"Breadcrumb\">")
	for i, seg := range strings.Split(dir, "/") {
		if seg == "" {
			continue
		}
		if i > 0 {
			b.WriteString(chevron)
		}
		b.WriteString("<span>" + html.EscapeString(seg) + "</span>")
	}
	b.WriteString("</nav>")
	return b.String()
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
	s := "<div class=\"tags\"><span class=\"tags-label\">Tags</span>"
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
        @import url('https://fonts.googleapis.com/css2?family=Inter:opsz,wght@14..32,400..700&family=Lilex:wght@400&display=swap');
        :root {
            color-scheme: dark light;
            --bg: #0b0d11; --card: #14181f; --border: rgba(255,255,255,0.08);
            --text: #c7d0dc; --heading: #f1f5f9; --muted: #7d8797; --accent: #e8a34d;
        }
        @media (prefers-color-scheme: light) {
            :root { --bg: #ffffff; --card: #ffffff; --border: rgba(15,23,42,0.09); --text: #3f4855; --heading: #0d1420; --muted: #6b7480; --accent: #b8722a; }
        }
        * { box-sizing: border-box; }
        body {
            margin: 0; min-height: 100vh; display: grid; place-items: center; padding: 24px;
            background: var(--bg); color: var(--text);
            font-family: 'Inter', -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
            font-size: 16px; line-height: 1.7; letter-spacing: -0.006em;
            -webkit-font-smoothing: antialiased;
        }
        .stub { width: 100%%; max-width: 480px; text-align: center; }
        .badge {
            display: inline-flex; align-items: center; gap: 7px; margin-bottom: 20px;
            padding: 5px 12px; border-radius: 999px; font-size: 12px; font-weight: 500;
            background: color-mix(in srgb, var(--accent) 12%%, transparent);
            border: 1px solid color-mix(in srgb, var(--accent) 28%%, transparent); color: var(--accent);
        }
        .badge svg { width: 13px; height: 13px; }
        h1 { margin: 0 0 10px; font-size: 1.7rem; font-weight: 650; letter-spacing: -0.03em; color: var(--heading); line-height: 1.25; overflow-wrap: anywhere; }
        p { margin: 0; color: var(--muted); font-size: 14.5px; }
        code {
            font-family: 'Lilex', ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 0.88em;
            background: var(--card); border: 1px solid var(--border); border-radius: 6px; padding: 2px 6px; color: var(--heading);
        }
    </style>
</head>
<body>
    <main class="stub">
        <span class="badge">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>
            Empty note
        </span>
        <h1>%[1]s</h1>
        <p>This page doesn't exist yet. Create <code>%s.md</code> in your vault to fill it in.</p>
    </main>
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
    <!-- Follow the reader's theme choice (same-origin localStorage, set by the site's toggle)
         so the graph doesn't open dark inside a light page. -->
    <script>(function(){try{var t=localStorage.getItem('nebula-theme');if(t)document.documentElement.setAttribute('data-theme',t);}catch(e){}})();</script>
    <style>
        @import url('https://fonts.googleapis.com/css2?family=Inter:opsz,wght@14..32,400..700&display=swap');
        :root, [data-theme="dark"] {
            color-scheme: dark;
            --bg: #0b0d11; --text: #c7d0dc; --border: rgba(255,255,255,0.09); --heading: #f1f5f9;
            --card-bg: rgba(16,19,25,0.82); --muted: #7d8797; --link: #5eb1ef; --stub: #e8a34d;
            --graph-node: #4e586d; --graph-edge: rgba(255,255,255,0.14);
        }
        [data-theme="light"] {
            color-scheme: light;
            --bg: #ffffff; --text: #3f4855; --border: rgba(15,23,42,0.1); --heading: #0d1420;
            --card-bg: rgba(255,255,255,0.82); --muted: #6b7480; --link: #1a6fd4; --stub: #b8722a;
            --graph-node: #c2cad6; --graph-edge: rgba(15,23,42,0.16);
        }
        html, body { overflow: hidden; height: 100%%; margin: 0; }
        body {
            font-family: 'Inter', -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
            background: var(--bg); color: var(--text); letter-spacing: -0.006em; -webkit-font-smoothing: antialiased;
        }
        #graph { width: 100vw; height: 100vh; overflow: hidden; }
        .node { cursor: pointer; }
        .node circle { fill: var(--graph-node); stroke: none; transition: fill 0.18s, opacity 0.18s; }
        .node.stub circle { fill: var(--stub); }
        .node text { font-size: 11px; font-weight: 500; fill: currentColor; opacity: 0.6; pointer-events: none; transition: opacity 0.2s; }
        .link { stroke: var(--graph-edge); stroke-width: 1px; transition: stroke-opacity 0.2s, stroke 0.2s; }
        .node.dimmed circle { opacity: 0.14; }
        .node.dimmed text { opacity: 0.16; }
        .link.dimmed { stroke-opacity: 0.12; }
        .node.hovered circle, .node.neighbor circle { fill: var(--link); }
        .node.hovered text { opacity: 1; }
        .link.connected { stroke: var(--link); stroke-opacity: 0.9; }
        /* Glassy floating panels */
        .panel {
            background: var(--card-bg); border: 1px solid var(--border); border-radius: 12px;
            backdrop-filter: blur(12px) saturate(160%%); -webkit-backdrop-filter: blur(12px) saturate(160%%);
            box-shadow: 0 14px 40px -14px rgba(0,0,0,0.5);
        }
        #legend { position: absolute; top: 20px; right: 20px; padding: 13px 16px; font-size: 12.5px; min-width: 150px; }
        #legend h3 { margin: 0 0 9px; font-size: 10.5px; font-weight: 600; text-transform: uppercase; letter-spacing: 0.09em; color: var(--muted); }
        #legend div { display: flex; align-items: center; gap: 8px; margin: 5px 0; }
        #legend span { display: inline-block; width: 9px; height: 9px; border-radius: 50%%; flex: none; }
        .legend-page { background: var(--link); }
        .legend-stub { background: var(--stub); }
        #mode-toggle { position: absolute; bottom: 20px; right: 20px; display: flex; gap: 5px; }
        #mode-toggle a, #mode-toggle span.current {
            border-radius: 9px; padding: 7px 13px; font-size: 11.5px; font-weight: 500;
            text-decoration: none; color: var(--muted); transition: color 0.15s, border-color 0.15s, background 0.15s;
        }
        #mode-toggle span.current { background: var(--link); color: var(--bg); border-color: transparent; }
        #mode-toggle a:hover { border-color: var(--link); color: var(--heading); }
    </style>
</head>
<body>
    <div id="legend" class="panel">
        <h3>Legend</h3>
        <div><span class="legend-page"></span>Page</div>
        <div><span class="legend-stub"></span>Stub (dead link)</div>
    </div>
    <div id="mode-toggle">
        <a class="panel" href="nebula{{EXT}}">3D</a>
        <span class="current panel">2D</span>
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
    // Obsidian-style layout: strong many-body repulsion spreads nodes evenly, short
    // weak-ish links keep neighbors adjacent without tangling, and per-node forceX/Y
    // (forceCenter only re-centers the mean — it doesn't shape anything) pulls the
    // even spread into a circular silhouette. Isolated notes drift inward until
    // repulsion balances, filling the gaps between clusters uniformly.
    // Tag each node with its connected component so clusters can repel as units.
    var adj = {};
    graph.nodes.forEach(function(n) { adj[n.id] = []; });
    graph.edges.forEach(function(e) { adj[e.source].push(e.target); adj[e.target].push(e.source); });
    var compOf = {}, compSizes = [], compCount = 0;
    graph.nodes.forEach(function(n) {
        if (compOf[n.id] !== undefined) return;
        var stack = [n.id]; compOf[n.id] = compCount; var size = 0;
        while (stack.length) {
            var cur = stack.pop(); size++;
            adj[cur].forEach(function(nb) { if (compOf[nb] === undefined) { compOf[nb] = compCount; stack.push(nb); } });
        }
        compSizes.push(size); compCount++;
    });
    graph.nodes.forEach(function(n) { n.component = compOf[n.id]; });
    // Cluster-separation force: each multi-node component keeps a buffer between
    // its hull and its neighbors' so clusters can't drift into a tangle. Singleton
    // notes are exempt — they still fill the gaps between clusters.
    var CLUSTER_BUFFER = 40;
    function forceClusterSeparation(alpha) {
        var comps = [];
        for (var c = 0; c < compCount; c++) comps.push(compSizes[c] > 1 ? { x: 0, y: 0, r: 0, nodes: [] } : null);
        graph.nodes.forEach(function(n) {
            var cc = comps[n.component];
            if (cc) { cc.x += n.x; cc.y += n.y; cc.nodes.push(n); }
        });
        comps.forEach(function(cc) {
            if (!cc) return;
            cc.x /= cc.nodes.length; cc.y /= cc.nodes.length;
            cc.nodes.forEach(function(n) {
                var dx = n.x - cc.x, dy = n.y - cc.y;
                var d2 = dx * dx + dy * dy;
                if (d2 > cc.r * cc.r) cc.r = Math.sqrt(d2);
            });
        });
        for (var i = 0; i < comps.length; i++) {
            var a = comps[i]; if (!a) continue;
            for (var j = i + 1; j < comps.length; j++) {
                var b = comps[j]; if (!b) continue;
                var dx = b.x - a.x, dy = b.y - a.y;
                var dist = Math.sqrt(dx * dx + dy * dy) || 1e-6;
                var want = a.r + b.r + CLUSTER_BUFFER;
                if (dist >= want) continue;
                // Push both clusters apart along the centroid axis, heavier
                // clusters moving proportionally less.
                var push = (want - dist) / dist * alpha * 0.6;
                var total = a.nodes.length + b.nodes.length;
                var ax = -dx * push * b.nodes.length / total, ay = -dy * push * b.nodes.length / total;
                var bx = dx * push * a.nodes.length / total, by = dy * push * a.nodes.length / total;
                a.nodes.forEach(function(n) { n.vx += ax; n.vy += ay; });
                b.nodes.forEach(function(n) { n.vx += bx; n.vy += by; });
            }
        }
    }
    // Degree-aware links keep hubs untangled within a component: spoke edges to
    // leaves stay short and stiff (d3's default link strength is 1/min-degree,
    // ~1 for a leaf) so each halo hugs its hub, while hub-hub edges are long and
    // loose so two connected hubs sit far enough apart that their halos don't
    // interleave. Hubs also repel harder (charge scales with degree) to push
    // neighboring halos off each other.
    var sim = d3.forceSimulation(graph.nodes)
        .force("link", d3.forceLink(graph.edges).id(function(d) { return d.id; })
            .distance(function(l) {
                var ds = adj[l.source.id || l.source].length, dt = adj[l.target.id || l.target].length;
                // A bridge between two hubs must clear both halos. A halo's
                // radius is its spoke length plus crowding stretch (leaves repel
                // each other outward), so it grows with degree — budget for both
                // ends plus a buffer (oversized a bit since the centering pull
                // compresses long links below their nominal length).
                if (Math.min(ds, dt) >= 3) return (60 + 5 * ds) + (60 + 5 * dt) + 50;
                return 60 + 8 * Math.min(ds, dt);
            })
            .strength(function(l) {
                var ds = adj[l.source.id || l.source].length, dt = adj[l.target.id || l.target].length;
                // Bridges get a firm strength: d3's default (1/min-degree) is so
                // weak for hub-hub links that the centering pull would crush the
                // budgeted distance and fold the halos into each other.
                if (Math.min(ds, dt) >= 3) return 0.5;
                return 1 / Math.min(ds, dt);
            }))
        .force("charge", d3.forceManyBody().strength(function(d) { return -250 - 15 * adj[d.id].length; }))
        .force("center", d3.forceCenter(w / 2, h / 2))
        .force("x", d3.forceX(w / 2).strength(0.2))
        .force("y", d3.forceY(h / 2).strength(0.2))
        .force("collision", d3.forceCollide().radius(function(d) { return 20 + adj[d.id].length; }))
        .force("cluster", forceClusterSeparation)
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
	data = strings.Replace(data, "{{EXT}}", linkExt, 1)
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
            if (mode === '2d') window.location.href = 'index{{EXT}}';
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
        function makeStarLayer(count, minR, maxR, size, opacity, tint) {
            var geo = new THREE.BufferGeometry();
            var pos = new Float32Array(count * 3);
            for (var i = 0; i < count; i++) {
                // Random direction on the unit sphere, then a radius in [minR, maxR] —
                // a hollow shell so stars sit out in the surrounding space, never inside
                // the node/cloud region where they'd crowd or be mistaken for nodes.
                var dx = Math.random() - 0.5, dy = Math.random() - 0.5, dz = Math.random() - 0.5;
                var dl = Math.sqrt(dx * dx + dy * dy + dz * dz) || 1;
                var r = minR + Math.random() * (maxR - minR);
                pos[i * 3]     = dx / dl * r;
                pos[i * 3 + 1] = dy / dl * r;
                pos[i * 3 + 2] = dz / dl * r;
            }
            geo.setAttribute('position', new THREE.BufferAttribute(pos, 3));
            var mat = new THREE.PointsMaterial({
                color: tint, size: size, opacity: opacity,
                // Screen-space sizing: stars stay small dots no matter how close the
                // camera gets, so none ever balloon into a square or read as a node.
                transparent: true, sizeAttenuation: false,
                blending: THREE.AdditiveBlending, depthWrite: false
            });
            var pts = new THREE.Points(geo, mat);
            scene.add(pts);
            return mat;
        }
        // Two depth layers in hollow shells well beyond the graph, each twinkling at its
        // own rate/phase: a white mid-field and a distant blue dust. No close foreground
        // layer — those were the stars that blew up near the camera.
        var starLayers = [
            { mat: makeStarLayer(900, 700, 1300, 2.0, 0.55, 0xffffff), base: 0.55, speed: 1.1, amp: 0.15, phase: 1.7 },
            { mat: makeStarLayer(1300, 1300, 2200, 1.4, 0.35, 0x99aaff), base: 0.35, speed: 0.7, amp: 0.10, phase: 0.0 }
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

        // ---- Simulated physics (repulsion + link springs + spherical bias) ----
        // We use a simple Euler integration to spread nodes
        var positions = nodeMeshes.map(function(m) { return m.position.clone(); });
        var velocities = nodeMeshes.map(function() { return new THREE.Vector3(); });

        // Obsidian-style "mold spore" layout: strong repulsion spaces everything
        // evenly, stiff short springs glue each cluster tightly around its hub so
        // clusters read as separate spores rather than one tangle, and the spherical
        // bias (the 3D analog of the 2D circular bias) pulls the even spread into a
        // round ball, isolated notes filling the gaps. Nodes in different clusters
        // repel each other extra hard so every spore keeps a clear buffer around it.
        // Repulsion is modest because the cluster-aware seeding below starts the
        // layout near equilibrium — it only fine-tunes, rather than exploding a
        // dense ball outward.
        var REPEL = 2500;
        var INTER_CLUSTER_REPEL = 3;
        var LINK_DIST = 25, LINK_STRENGTH = 1.5;
        var SPHERE_BIAS = 0.05;
        var nodeIndex = {};
        nodeMeshes.forEach(function(m, i) { nodeIndex[m.userData.id] = i; });
        var degree = nodeMeshes.map(function() { return 0; });
        var springPairs = []; // [sourceIdx, targetIdx, strength, restLength]
        graph.edges.forEach(function(e) {
            var sid = typeof e.source === 'object' ? e.source.id : e.source;
            var tid = typeof e.target === 'object' ? e.target.id : e.target;
            var si = nodeIndex[sid], ti = nodeIndex[tid];
            if (si !== undefined && ti !== undefined && si !== ti) {
                springPairs.push([si, ti, 0]);
                degree[si]++; degree[ti]++;
            }
        });
        // d3-style degree normalization: a spring is only as stiff as its
        // lower-degree endpoint allows, so leaf spokes stay short and stiff.
        // A bridge between two hubs instead gets a long, firm spring sized to
        // clear both halos (spoke length plus crowding stretch grows with
        // degree), so connected hubs sit apart rather than merging into one
        // tangle — the 3D twin of the 2D hub-bridge rule.
        springPairs.forEach(function(sp) {
            var dS = degree[sp[0]], dT = degree[sp[1]];
            if (Math.min(dS, dT) >= 3) {
                sp[2] = LINK_STRENGTH * 0.7;
                sp[3] = (LINK_DIST + 2 * dS) + (LINK_DIST + 2 * dT) + 25;
            } else {
                sp[2] = LINK_STRENGTH / Math.max(1, Math.min(dS, dT));
                sp[3] = LINK_DIST;
            }
        });
        // Connected components (union-find over the springs) so nodes in different
        // clusters can repel harder. Singletons are exempt so isolated notes still
        // settle into the gaps between spores.
        var compParent = nodeMeshes.map(function(_, i) { return i; });
        function compFind(x) {
            while (compParent[x] !== x) { compParent[x] = compParent[compParent[x]]; x = compParent[x]; }
            return x;
        }
        springPairs.forEach(function(sp) { compParent[compFind(sp[0])] = compFind(sp[1]); });
        var clusterOf = nodeMeshes.map(function(_, i) { return compFind(i); });
        var clusterCounts = {};
        clusterOf.forEach(function(c) { clusterCounts[c] = (clusterCounts[c] || 0) + 1; });
        var inCluster = clusterOf.map(function(c) { return clusterCounts[c] > 1; });

        // Cluster-aware seeding: instead of spawning everything in one dense ball
        // that explodes outward, give each cluster its own anchor on a Fibonacci
        // sphere (evenly spaced directions) and scatter its members around that
        // anchor, sized by cluster population. The layout starts already spread
        // out, so the opening settle is a gentle drift instead of a big bang.
        var clusterMembers = {};
        clusterOf.forEach(function(c, i) {
            if (inCluster[i]) (clusterMembers[c] = clusterMembers[c] || []).push(i);
        });
        var clusterIds = Object.keys(clusterMembers).sort(function(a, b) {
            return clusterMembers[b].length - clusterMembers[a].length;
        });
        var clusterRadii = clusterIds.map(function(cid) {
            return LINK_DIST * (0.5 + 0.5 * Math.cbrt(clusterMembers[cid].length));
        });
        var avgRadius = clusterRadii.length
            ? clusterRadii.reduce(function(a, b) { return a + b; }, 0) / clusterRadii.length : 0;
        // Anchor sphere sized so neighboring anchors sit roughly two cluster radii apart
        var anchorR = Math.max(40, avgRadius * Math.sqrt(clusterIds.length) * 1.2);
        var GOLDEN_ANGLE = 2.399963229728653;
        clusterIds.forEach(function(cid, k) {
            var N = clusterIds.length;
            var gy = N === 1 ? 0 : 1 - 2 * k / (N - 1);
            var gr = Math.sqrt(Math.max(0, 1 - gy * gy));
            var ax = Math.cos(k * GOLDEN_ANGLE) * gr * anchorR;
            var ay = gy * anchorR;
            var az = Math.sin(k * GOLDEN_ANGLE) * gr * anchorR;
            var cr = clusterRadii[k];
            clusterMembers[cid].forEach(function(i) {
                var th = Math.random() * Math.PI * 2, ph = Math.acos(2 * Math.random() - 1);
                var rr = cr * Math.cbrt(Math.random());
                positions[i].set(
                    ax + rr * Math.sin(ph) * Math.cos(th),
                    ay + rr * Math.cos(ph),
                    az + rr * Math.sin(ph) * Math.sin(th));
                nodeMeshes[i].position.copy(positions[i]);
            });
        });
        // Singletons scatter through the same shell, filling gaps between clusters
        nodeMeshes.forEach(function(m, i) {
            if (inCluster[i]) return;
            var th = Math.random() * Math.PI * 2, ph = Math.acos(2 * Math.random() - 1);
            var rr = anchorR * (0.4 + 0.7 * Math.random());
            positions[i].set(
                rr * Math.sin(ph) * Math.cos(th),
                rr * Math.cos(ph),
                rr * Math.sin(ph) * Math.sin(th));
            nodeMeshes[i].position.copy(positions[i]);
        });

        function simulate(dt) {
            var count = positions.length;

            for (var i = 0; i < count; i++) {
                var p = positions[i];
                var v = velocities[i];

                // Repulsion between all pairs (capped at close range so a frame
                // hitch on near-coincident nodes can't launch them)
                for (var j = i + 1; j < count; j++) {
                    var q = positions[j];
                    var dx = p.x - q.x, dy = p.y - q.y, dz = p.z - q.z;
                    var dist = Math.sqrt(dx * dx + dy * dy + dz * dz) + 0.001;
                    var force = REPEL / Math.max(dist * dist, 25);
                    if (inCluster[i] && inCluster[j] && clusterOf[i] !== clusterOf[j]) force *= INTER_CLUSTER_REPEL;
                    var fx = dx / dist * force, fy = dy / dist * force, fz = dz / dist * force;
                    v.x += fx * dt; v.y += fy * dt; v.z += fz * dt;
                    velocities[j].x -= fx * dt; velocities[j].y -= fy * dt; velocities[j].z -= fz * dt;
                }

                // Spherical bias (linear pull to origin, stronger the farther out)
                v.x -= p.x * SPHERE_BIAS * dt;
                v.y -= p.y * SPHERE_BIAS * dt;
                v.z -= p.z * SPHERE_BIAS * dt;
            }

            // Link springs pull each connected pair toward its rest length
            for (var s = 0; s < springPairs.length; s++) {
                var a = positions[springPairs[s][0]], b = positions[springPairs[s][1]];
                var sdx = b.x - a.x, sdy = b.y - a.y, sdz = b.z - a.z;
                var sdist = Math.sqrt(sdx * sdx + sdy * sdy + sdz * sdz) + 0.001;
                var sf = springPairs[s][2] * (sdist - springPairs[s][3]) / sdist;
                var sfx = sdx * sf * dt, sfy = sdy * sf * dt, sfz = sdz * sf * dt;
                var va = velocities[springPairs[s][0]], vb = velocities[springPairs[s][1]];
                va.x += sfx; va.y += sfy; va.z += sfz;
                vb.x -= sfx; vb.y -= sfy; vb.z -= sfz;
            }

            // Damping
            for (var di = 0; di < count; di++) {
                velocities[di].multiplyScalar(0.92);
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
		data = strings.Replace(data, "{{EXT}}", linkExt, 1)
		err := os.WriteFile(filepath.Join(graphDir, "nebula.html"), []byte(data), 0644)
		if err != nil {
			fmt.Printf("Error writing graph nebula.html: %v\n", err)
		}
	}
