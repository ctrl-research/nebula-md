package main

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

func main() {
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM, extension.TaskList, extension.Typographer),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		// NOTE: Basalt's actual code does NOT use WithXHTML() or WithHardWraps()
		// Let me check what the DEFAULT output is...
	)

	testCases := []struct {
		name string
		md   string
	}{
		{
			name: "Multi-line NO blank line",
			md: `> [!info]
> Line 1
> Line 2
> Line 3`,
		},
		{
			name: "Single line",
			md: `> [!info]
> Single line`,
		},
		{
			name: "Multi-line with blank line between",
			md: `> [!info]
> Line 1
>
> Line 2`,
		},
	}

	for _, tc := range testCases {
		fmt.Printf("\n=== %s ===\nInput:\n%s\n\nOutput HTML:\n", tc.name, tc.md)
		var buf bytes.Buffer
		if err := md.Convert([]byte(tc.md), &buf); err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}
		html := buf.String()
		fmt.Println(html)

		// Check for <br> tags
		if strings.Contains(html, "<br>") {
			fmt.Println("^^ Contains <br>")
		}
		if strings.Contains(html, "<br/>") {
			fmt.Println("^^ Contains <br/>")
		}
		if strings.Contains(html, "<br />") {
			fmt.Println("^^ Contains <br />")
		}
	}
}