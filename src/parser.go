package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
)

// MarkdownParser handles conversion of markdown files to HTML
type MarkdownParser struct {
	markdown goldmark.Markdown
}

// NewMarkdownParser initializes the goldmark engine with GFM + task lists + typographer
func NewMarkdownParser() *MarkdownParser {
	return &MarkdownParser{
		markdown: goldmark.New(
			goldmark.WithExtensions(extension.GFM, extension.TaskList, extension.Typographer),
			goldmark.WithParserOptions(
				parser.WithAutoHeadingID(),
			),
		),
	}
}

// ProcessFile reads a markdown file, extracts metadata and wiki-links,
// then returns the title, rendered HTML body, link targets, and computed rel hrefs.
func (p *MarkdownParser) ProcessFile(filePath, sourceRelPath string) (title string, htmlBody []byte, linkTargets []string, linkHrefs []string, err error) {
	rawContent, err := os.ReadFile(filePath)
	if err != nil {
		return "", nil, nil, nil, err
	}

	title = extractTitle(rawContent)
	contentWithLinks, targets, rels := extractWikiLinks(rawContent, sourceRelPath)
	linkTargets = targets
	linkHrefs = rels

	contentToRender := removeFrontmatter(contentWithLinks)

	var buf bytes.Buffer
	if err := p.markdown.Convert(contentToRender, &buf); err != nil {
		return "", nil, nil, nil, err
	}

	htmlBody = buf.Bytes()

	// Post-process: handle callouts and collapsible sections
	htmlBody = processCallouts(htmlBody)
	htmlBody = processCollapsibles(htmlBody)

	// Rewrite .md links to .html
	reLink := regexp.MustCompile(`\(([^)]*?\.md)\)`)
	htmlBody = reLink.ReplaceAll(htmlBody, []byte("($1html)"))

	return title, htmlBody, linkTargets, linkHrefs, nil
}

// extractTitle gets the title from frontmatter, first H1, or falls back to filename
func extractTitle(data []byte) string {
	re := regexp.MustCompile(`(?s)^---\s*\n(.*?)\n---\n?`)
	if matches := re.FindSubmatch(data); len(matches) > 0 {
		yamlContent := string(matches[1])
		for _, line := range strings.Split(yamlContent, "\n") {
			if strings.HasPrefix(line, "title:") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					return strings.TrimSpace(parts[1])
				}
			}
		}
	}
	h1Re := regexp.MustCompile(`(?m)^#\s+(.*)$`)
	if matches := h1Re.FindSubmatch(data); len(matches) == 2 {
		return strings.TrimSpace(string(matches[1]))
	}
	return "Untitled"
}

// toHTMLName converts a markdown file path to an HTML-safe name (strips .md extension)
func toHTMLName(mdPath string) string {
	return strings.TrimSuffix(filepath.Base(mdPath), ".md")
}

// removeFrontmatter strips YAML frontmatter from raw content
func removeFrontmatter(data []byte) []byte {
	re := regexp.MustCompile(`(?s)^---\s*\n.*?\n---\n?`)
	return re.ReplaceAll(data, []byte{})
}

// extractTags gets the tags list from frontmatter.
func extractTags(data []byte) []string {
	re := regexp.MustCompile(`(?s)^---\s*\n(.*?)\n---\n?`)
	if matches := re.FindSubmatch(data); len(matches) > 0 {
		yamlContent := string(matches[1])
		inlineRe := regexp.MustCompile(`(?m)^tags:\s*\[([^\]]*)\]`)
		if m := inlineRe.FindSubmatch([]byte(yamlContent)); len(m) > 0 {
			parts := strings.Split(string(m[1]), ",")
			var tags []string
			for _, p := range parts {
				t := strings.TrimSpace(p)
				if t != "" {
					tags = append(tags, t)
				}
			}
			return tags
		}
		multilineRe := regexp.MustCompile(`(?m)^tags:\s*\n((?:\s+-\s*[^\n]+\n?)+)`)
		if m := multilineRe.FindSubmatch([]byte(yamlContent)); len(m) > 0 {
			lines := strings.Split(strings.TrimSpace(string(m[1])), "\n")
			var tags []string
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "-") {
					tag := strings.TrimSpace(strings.TrimPrefix(line, "-"))
					if tag != "" {
						tags = append(tags, tag)
					}
				}
			}
			return tags
		}
	}
	return nil
}

// extractDate gets the date from frontmatter if present.
func extractDate(data []byte) string {
	re := regexp.MustCompile(`(?s)^---\s*\n(.*?)\n---\n?`)
	if matches := re.FindSubmatch(data); len(matches) > 0 {
		yamlContent := string(matches[1])
		dateRe := regexp.MustCompile(`(?m)^date:\s*["']?([^"'\n]+)["']?`)
		if m := dateRe.FindSubmatch([]byte(yamlContent)); len(m) > 0 {
			return strings.TrimSpace(string(m[1]))
		}
	}
	return ""
}

// computeReadingTime estimates reading time from word count.
func computeReadingTime(htmlBody []byte) string {
	words := 0
	inTag := false
	for _, b := range htmlBody {
		switch b {
		case '<':
			inTag = true
		case '>':
			inTag = false
		case ' ', '\n', '\r', '\t':
			if !inTag {
				words++
			}
		}
	}
	minutes := (words + 199) / 200
	if minutes < 1 {
		minutes = 1
	}
	if minutes == 1 {
		return "1 min read"
	}
	return fmt.Sprintf("%d min read", minutes)
}

// processCallouts converts Obsidian-style callout blockquotes to styled divs.
// Syntax: > [!NOTE] followed by blockquote content
func processCallouts(htmlBody []byte) []byte {
	typeRe := regexp.MustCompile(`(?s)<blockquote>\s*<p>\[!([^\]]+)\]`)

	type calloutInfo struct {
		fullStart, fullEnd int
		typ, content       string
	}
	var infos []*calloutInfo

	for _, idx := range typeRe.FindAllIndex(htmlBody, -1) {
		typ := strings.ToLower(strings.TrimSpace(string(htmlBody[idx[0]:idx[1]])))
		typ = strings.TrimPrefix(typ, "<blockquote>")
		typ = strings.TrimSpace(typ)
		typ = strings.TrimPrefix(typ, "<p>")
		typ = strings.TrimPrefix(typ, "[!")
		typ = strings.TrimSuffix(typ, "]")

		s := string(htmlBody[idx[1]:])
		bqEndIdx := strings.Index(s, "</blockquote>")
		if bqEndIdx < 0 {
			continue
		}
		bqEnd := idx[1] + bqEndIdx + len("</blockquote>")

		inner := string(htmlBody[idx[0]:bqEnd])
		firstCloseP := strings.Index(inner, "</p>")
		if firstCloseP < 0 {
			continue
		}
		contentAfterClose := strings.TrimLeft(inner[firstCloseP+5:], "\n\r ")

		inlineContent := ""
		if bracketIdx := strings.Index(inner, "]"); bracketIdx >= 0 && bracketIdx < firstCloseP {
			inlineContent = strings.TrimSpace(inner[bracketIdx+1:firstCloseP])
		}

		fullContent := ""
		if inlineContent != "" {
			fullContent = "<p>" + inlineContent + "</p>"
		}
		for {
			pStart := strings.Index(contentAfterClose, "<p>")
			if pStart < 0 {
				break
			}
			contentAfterClose = contentAfterClose[pStart+3:]
			pEnd := strings.Index(contentAfterClose, "</p>")
			if pEnd < 0 {
				break
			}
			pText := strings.TrimRight(contentAfterClose[:pEnd], "\n\r ")
			if pText != "" {
				if fullContent != "" {
					fullContent += "\n"
				}
				fullContent += "<p>" + pText + "</p>"
			}
			contentAfterClose = contentAfterClose[pEnd+4:]
		}

		if fullContent == "" {
			continue
		}
		infos = append(infos, &calloutInfo{fullStart: idx[0], fullEnd: bqEnd, typ: typ, content: fullContent})
	}

	if len(infos) == 0 {
		return htmlBody
	}

	result := string(htmlBody)
	for i := len(infos) - 1; i >= 0; i-- {
		info := infos[i]
		icon := calloutIcon(info.typ)
		replacement := fmt.Sprintf(`<div class="callout callout-%s"><div class="callout-header">%s %s</div><div class="callout-body">%s</div></div>`, info.typ, icon, info.typ, info.content)
		result = result[:info.fullStart] + replacement + result[info.fullEnd:]
	}
	return []byte(result)
}

// processCollapsibles converts :::details blocks to <details>/<summary> elements.
func processCollapsibles(htmlBody []byte) []byte {
	re := regexp.MustCompile(`(?s):::details(?:\s+([^\n]*?))?\n(.*?)\n:::`)
	return re.ReplaceAllFunc(htmlBody, func(match []byte) []byte {
		parts := re.FindSubmatch(match)
		if len(parts) < 3 {
			return match
		}
		title := strings.TrimSpace(string(parts[1]))
		content := strings.TrimSpace(string(parts[2]))
		if title == "" {
			title = "Details"
		}
		return []byte(fmt.Sprintf(`<details class="collapsible"><summary>%s</summary><div class="collapsible-content">%s</div></details>`, title, content))
	})
}

// calloutIcon returns the emoji icon for a given callout type.
func calloutIcon(typ string) string {
	switch typ {
	case "note":
		return "ℹ️"
	case "tip", "hint":
		return "💡"
	case "warning", "caution":
		return "⚠️"
	case "danger", "error":
		return "🚨"
	case "example":
		return "📋"
	case "info":
		return "ℹ️"
	case "success", "check":
		return "✅"
	default:
		return "📌"
	}
}
