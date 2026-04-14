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

type MarkdownParser struct {
	markdown goldmark.Markdown
}

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

func (p *MarkdownParser) ProcessFile(filePath, sourceRelPath string) (title string, htmlBody []byte, linkTargets []string, linkHrefs []string, err error) {
	rawContent, err := os.ReadFile(filePath)
	if err != nil {
		return "", nil, nil, nil, err
	}

	// Strip comments first so frontmatter/title extraction ignores comment content
	contentNoComments := stripObsidianComments(rawContent)
	title = extractTitle(contentNoComments)
	contentWithLinks, targets, rels := extractWikiLinks(contentNoComments, sourceRelPath)
	linkTargets = targets
	linkHrefs = rels

	contentToRender := removeFrontmatter(contentWithLinks)
	contentToRender = stripObsidianComments(contentToRender)

	var buf bytes.Buffer
	if err := p.markdown.Convert(contentToRender, &buf); err != nil {
		return "", nil, nil, nil, err
	}

	htmlBody = buf.Bytes()

	htmlBody = processCallouts(htmlBody)
	htmlBody = processCollapsibles(htmlBody)

	reLink := regexp.MustCompile(`\(([^)]*?\.md)\)`)
	htmlBody = reLink.ReplaceAll(htmlBody, []byte("($1html)"))

	return title, htmlBody, linkTargets, linkHrefs, nil
}

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

func toHTMLName(mdPath string) string {
	return strings.TrimSuffix(filepath.Base(mdPath), ".md")
}

func removeFrontmatter(data []byte) []byte {
	re := regexp.MustCompile(`(?s)^---\s*\n.*?\n---\n?`)
	return re.ReplaceAll(data, []byte{})
}

// stripObsidianComments removes Obsidian-style comments (%%...%%) from markdown.
// Handles inline comments like "text %%comment%% more text" and line-based comments.
func stripObsidianComments(data []byte) []byte {
	// Match %%...%% content - handles multi-line as well
	re := regexp.MustCompile(`(?s)%%.*?%%`)
	return re.ReplaceAll(data, []byte{})
}

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

// processCallouts converts Obsidian-style callouts to styled divs.
// Syntax: > [!TYPE] or > [!TYPE|TITLE] or > [!TYPE]+/- or > [!TYPE]
// Handles: custom titles, foldable (+/-), multi-line content
func processCallouts(htmlBody []byte) []byte {
	// Match blockquotes starting with <p>[!TYPE] - capture TYPE letters only
	typeRe := regexp.MustCompile(`(?s)<blockquote>\s*<p>\[!([a-zA-Z]+)(\|([^\]\+]+))?\](\+|\-)?`)

	type calloutInfo struct {
		fullStart int
		fullEnd   int
		typ       string
		title     string
		fold      string
		content   string
	}
	var infos []*calloutInfo

	matches := typeRe.FindAllSubmatchIndex(htmlBody, -1)
	for _, mi := range matches {
		m := mi
		// m[0],m[1]: full match span
		// m[2],m[3]: capture group 1 (the TYPE letters)
		if len(m) < 4 {
			continue
		}

		typ := strings.ToLower(string(htmlBody[m[2]:m[3]]))

		// Extract custom title from group 3 (after | in header)
		title := ""
		fold := ""
		// Group 2 is `|title` (whole), group 3 is `title` (without |)
		if len(m) >= 8 && m[6] >= 0 && m[7] >= 0 {
			title = strings.TrimSpace(string(htmlBody[m[6]:m[7]]))
		}
		// Extract fold indicator from group 4
		if len(m) >= 10 && m[8] >= 0 && m[9] >= 0 {
			fold = string(htmlBody[m[8]:m[9]])
		}
		if title == "" {
			title = defaultTitle(typ)
		}

		// Find end of this blockquote
		// If there are nested blockquotes, find the </blockquote> that closes THIS blockquote
		s := string(htmlBody[m[1]:])
		bqEndIdx := -1
		bqDepth := 1 // We're already inside one <blockquote> (this one, starting at m[0])
		for i := 0; i < len(s); i++ {
			if strings.HasPrefix(s[i:], "<blockquote>") {
				bqDepth++
				i += 11 // skip past <blockquote>
			} else if strings.HasPrefix(s[i:], "</blockquote>") {
				bqDepth--
				if bqDepth == 0 {
					bqEndIdx = i
					break
				}
				i += 12 // skip past </blockquote>
			}
		}
		if bqEndIdx < 0 {
			continue
		}
		bqEnd := m[1] + bqEndIdx + len("</blockquote>")

		// Find first </p> after the header - this separates header from content
		inner := string(htmlBody[m[0]:bqEnd])
		closeP := strings.Index(inner, "</p>")
		if closeP < 0 {
			continue
		}

		// Content starts after the closing ] (m[1]) until </p>
		// If afterClose has only whitespace/newlines, the content is inline
		// If there are <p> tags between m[1]+closeP and bqEnd, those are the content paragraphs
		contentStart := m[0] + closeP + 5 // after </p>
		
		// Content extraction - stop at nested blockquotes
		// Find where content ends (before any nested blockquote)
		contentEnd := bqEnd
		s2 := string(htmlBody[contentStart:bqEnd])
		if strings.Contains(s2, "<blockquote>") {
			// There's a nested blockquote - content ends before it
			nestedBQ := strings.Index(s2, "<blockquote>")
			contentEnd = contentStart + nestedBQ
		}
		
		afterClose := strings.TrimLeft(string(htmlBody[contentStart:contentEnd]), "\n\r ")
		
		// Extract inline content between ] and first </p> (before any separate <p> blocks)
		// This handles cases where content is on the same line as the header
		inlineContent := strings.TrimSpace(string(htmlBody[m[1] : m[0]+closeP]))
		// Strip fold indicator from inline content if present
		if strings.HasSuffix(inlineContent, "+") || strings.HasSuffix(inlineContent, "-") {
			inlineContent = strings.TrimSpace(inlineContent[:len(inlineContent)-1])
		}
		
		fullContent := ""
		remaining := afterClose
		for {
			pStart := strings.Index(remaining, "<p>")
			if pStart < 0 {
				break
			}
			remaining = remaining[pStart+3:]
			pEnd := strings.Index(remaining, "</p>")
			if pEnd < 0 {
				break
			}
			pText := strings.TrimRight(remaining[:pEnd], "\n\r ")
			if pText != "" {
				if fullContent != "" {
					fullContent += "\n"
				}
				fullContent += "<p>" + pText + "</p>"
			}
			remaining = remaining[pEnd+4:]
		}

		// If we have both inline content and separate paragraph content, combine them
		// If we only have inline content, use it
		// If we only have paragraph content, use it
		if inlineContent != "" && fullContent != "" {
			// Both exist - combine (inline first, then paragraphs)
			fullContent = "<p>" + inlineContent + "</p>\n" + fullContent
		} else if inlineContent != "" {
			fullContent = "<p>" + inlineContent + "</p>"
		} else if fullContent == "" {
			// No content at all - skip
			continue
		}

		infos = append(infos, &calloutInfo{
			fullStart: m[0],
			fullEnd:   bqEnd,
			typ:       typ,
			title:     title,
			fold:      fold,
			content:   fullContent,
		})
	}

	if len(infos) == 0 {
		return htmlBody
	}

	// Replace from end to start to preserve indices
	result := string(htmlBody)
	for i := len(infos) - 1; i >= 0; i-- {
		info := infos[i]
		icon := calloutIcon(info.typ)
		var replacement string
		if info.fold != "" {
			openAttr := ""
			if info.fold == "+" {
				openAttr = " open"
			}
			replacement = fmt.Sprintf(`<details class="callout" data-callout="%s"%s><summary class="callout-title"><span class="callout-icon">%s</span>%s</summary><div class="callout-content">%s</div></details>`, info.typ, openAttr, icon, info.title, info.content)
		} else {
			replacement = fmt.Sprintf(`<div class="callout" data-callout="%s"><div class="callout-title"><span class="callout-icon">%s</span>%s</div><div class="callout-content">%s</div></div>`, info.typ, icon, info.title, info.content)
		}
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

func defaultTitle(typ string) string {
	switch typ {
	case "note":
		return "Note"
	case "abstract", "summary", "tldr":
		return "Abstract"
	case "info":
		return "Info"
	case "todo":
		return "To Do"
	case "tip", "hint", "important":
		return "Tip"
	case "success", "check", "done":
		return "Success"
	case "question", "help", "faq":
		return "Question"
	case "warning", "caution", "attention":
		return "Warning"
	case "failure", "fail", "missing":
		return "Failure"
	case "danger", "error":
		return "Danger"
	case "bug":
		return "Bug"
	case "example":
		return "Example"
	case "quote", "cite":
		return "Quote"
	default:
		return strings.Title(typ)
	}
}

func calloutIcon(typ string) string {
	switch typ {
	case "note":
		return `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="12" y1="16" x2="12" y2="12"/><line x1="12" y1="8" x2="12.01" y2="8"/></svg>`
	case "abstract", "summary", "tldr":
		return `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/><polyline points="10 9 9 9 8 9"/></svg>`
	case "info":
		return `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="12" y1="16" x2="12" y2="12"/><line x1="12" y1="8" x2="12.01" y2="8"/></svg>`
	case "todo", "check":
		return `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="9 11 12 14 22 4"/><path d="M21 12v7a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11"/></svg>`
	case "tip", "hint", "important":
		return `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="2" x2="12" y2="6"/><line x1="12" y1="18" x2="12" y2="22"/><line x1="4.93" y1="4.93" x2="7.76" y2="7.76"/><line x1="16.24" y1="16.24" x2="19.07" y2="19.07"/><line x1="2" y1="12" x2="6" y2="12"/><line x1="18" y1="12" x2="22" y2="12"/><line x1="4.93" y1="19.07" x2="7.76" y2="16.24"/><line x1="16.24" y1="7.76" x2="19.07" y2="4.93"/></svg>`
	case "success", "done":
		return `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg>`
	case "question", "help", "faq":
		return `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>`
	case "warning", "caution", "attention":
		return `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>`
	case "failure", "fail", "missing":
		return `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="15" y1="9" x2="9" y2="15"/><line x1="9" y1="9" x2="15" y2="15"/></svg>`
	case "danger", "error":
		return `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polygon points="7.86 2 16.14 2 22 7.86 22 16.14 16.14 22 7.86 22 2 16.14 2 7.86 7.86 2"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>`
	case "bug":
		return `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="8" y="6" width="8" height="14" rx="4"/><path d="m19 6-4 4m-6 0 4 4m-4-4-4-4m0 12-4 4m6 0 4-4"/></svg>`
	case "example":
		return `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="10" y1="12" x2="14" y2="12"/></svg>`
	case "quote", "cite":
		return `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 21c3 0 7-1 7-8V5c0-1.25-.756-2.017-2-2H4c-1.25 0-2 .75-2 1.972V11c0 1.25.75 2 2 2 1 0 1 0 1 1v1c0 1-1 2-2 2s-1 .008-1 1.031V21z"/><path d="M15 21c3 0 7-1 7-8V5c0-1.25-.757-2.017-2-2h-4c-1.25 0-2 .75-2 1.972V11c0 1.25.75 2 2 2h.75c0 2.25.25 4-2.75 4v3z"/></svg>`
	default:
		return `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="12" y1="16" x2="12" y2="12"/><line x1="12" y1="8" x2="12.01" y2="8"/></svg>`
	}
}
