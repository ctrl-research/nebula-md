package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMermaidBlocksRender(t *testing.T) {
	src := "# Flow\n\n```mermaid\nflowchart LR\n  %% a mermaid comment\n  A[[Sub]] --> B{\"x < y\"}\n  %% another\n```\n\n~~~mermaid\ngraph TD; C-->D\n~~~\n\n```go\nx := 1\n```\n"
	dir := t.TempDir()
	path := filepath.Join(dir, "flow.md")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	_, body, targets, _, err := NewMarkdownParser().ProcessFile(path, "flow.md")
	if err != nil {
		t.Fatal(err)
	}
	out := string(body)
	want := `<pre class="mermaid">flowchart LR
  %% a mermaid comment
  A[[Sub]] --&gt; B{&#34;x &lt; y&#34;}
  %% another
</pre>`
	if !strings.Contains(out, want) {
		t.Errorf("backtick fence not rendered verbatim, got:\n%s", out)
	}
	if !strings.Contains(out, `<pre class="mermaid">graph TD; C--&gt;D
</pre>`) {
		t.Errorf("tilde fence not rendered, got:\n%s", out)
	}
	if !strings.Contains(out, `<code class="language-go">`) {
		t.Errorf("other code fences should be untouched, got:\n%s", out)
	}
	if strings.Contains(out, "NEBULAMERMAID") || strings.Contains(out, "<p><pre") {
		t.Errorf("sentinel leaked or wrapped in a paragraph:\n%s", out)
	}
	if len(targets) != 0 {
		t.Errorf("[[Sub]] inside a diagram became a wiki link: %v", targets)
	}
}

func TestStripMermaidBlocks(t *testing.T) {
	in := "before [[Real]]\n```mermaid\nA[[Fake]]\n```\nafter\n"
	if got := string(stripMermaidBlocks([]byte(in))); strings.Contains(got, "Fake") || !strings.Contains(got, "[[Real]]") {
		t.Errorf("got %q", got)
	}
}
