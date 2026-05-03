package main

import (
	"fmt"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"strings"
)

func main() {
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithXHTML(),
		),
	)

	testCases := []string{
		// Case 1: Multi-line with blank line between
		`> [!info]
> Line 1
>
> Line 2`,

		// Case 2: Multi-line NO blank line
		`> [!info]
> Line 1
> Line 2
> Line 3`,

		// Case 3: Single line content
		`> [!info]
> Single line`,

		// Case 4: Multi-line on same line (with br)
		`> [!info]
> Line 1
> Line 2
> Line 3`,
	}

	for i, tc := range testCases {
		fmt.Printf("\n=== Case %d ===\nInput:\n%s\n\nOutput HTML:\n", i+1, tc)
		var buf strings.Builder
		if err := md.Convert([]byte(tc), &buf); err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}
		fmt.Printf("%s\n", buf.String())
	}
}
