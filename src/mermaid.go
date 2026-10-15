package main

import (
	"encoding/hex"
	"html"
	"regexp"
)

// mermaidFence matches a fenced ```mermaid (or ~~~mermaid) block; the source is in
// group 1 or 2. Go regexps lack backreferences, so each fence character gets its own
// alternative to keep the closing fence matching the opening one.
var mermaidFence = regexp.MustCompile("(?ms)" +
	"^[ \t]{0,3}```+[ \t]*mermaid[ \t]*\r?\n(.*?)^[ \t]{0,3}```+[ \t]*$" +
	"|^[ \t]{0,3}~~~+[ \t]*mermaid[ \t]*\r?\n(.*?)^[ \t]{0,3}~~~+[ \t]*$")

// The diagram source is hex-encoded into the sentinel so nothing upstream can mangle
// it: Mermaid's own `%%` comments would be eaten as Obsidian comments, and its
// `[[subroutine]]` node shape would be rewritten as a wiki link.
var mermaidSentinelRe = regexp.MustCompile(`NEBULAMERMAIDxZ9([0-9a-f]*)END`)

// protectMermaidBlocks swaps each mermaid fence for a plain-text sentinel on its own
// paragraph, mirroring protectMapEmbeds.
func protectMermaidBlocks(data []byte) []byte {
	return mermaidFence.ReplaceAllFunc(data, func(match []byte) []byte {
		sub := mermaidFence.FindSubmatch(match)
		src := append(sub[1], sub[2]...)
		return []byte("\n\nNEBULAMERMAIDxZ9" + hex.EncodeToString(src) + "END\n\n")
	})
}

// stripMermaidBlocks drops mermaid fences entirely, for scanners (graph edges, search)
// that should not see diagram source.
func stripMermaidBlocks(data []byte) []byte {
	return mermaidFence.ReplaceAll(data, nil)
}

// injectMermaidBlocks swaps sentinels for <pre class="mermaid"> blocks, which the page
// script renders into diagrams.
func injectMermaidBlocks(body []byte) []byte {
	if !mermaidSentinelRe.Match(body) {
		return body
	}
	block := func(hexSrc []byte) []byte {
		src, _ := hex.DecodeString(string(hexSrc))
		return []byte(`<pre class="mermaid">` + html.EscapeString(string(src)) + `</pre>`)
	}
	// Goldmark wraps a lone sentinel in its own paragraph; replace the whole block first.
	wrapped := regexp.MustCompile(`<p>NEBULAMERMAIDxZ9([0-9a-f]*)END</p>`)
	body = wrapped.ReplaceAllFunc(body, func(m []byte) []byte {
		return block(wrapped.FindSubmatch(m)[1])
	})
	return mermaidSentinelRe.ReplaceAllFunc(body, func(m []byte) []byte {
		return block(mermaidSentinelRe.FindSubmatch(m)[1])
	})
}
