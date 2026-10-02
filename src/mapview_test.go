package main

import (
	"strings"
	"testing"
)

func TestExtractLocation(t *testing.T) {
	cases := []struct {
		name     string
		in       string
		lat, lng float64
		ok       bool
	}{
		{"list", "---\ntitle: a\nlocation: [43.65, -79.38]\n---\nbody", 43.65, -79.38, true},
		{"quoted string", "---\nlocation: \"43.65,-79.38\"\n---\n", 43.65, -79.38, true},
		{"integers", "---\nlocation: [1, 2]\n---\n", 1, 2, true},
		{"missing", "---\ntitle: a\n---\n", 0, 0, false},
		{"body only", "---\ntitle: a\n---\nlocation: [1, 2]\n", 0, 0, false},
		{"out of range", "---\nlocation: [143.65, -79.38]\n---\n", 0, 0, false},
		{"no frontmatter", "location: [1, 2]\n", 0, 0, false},
	}
	for _, c := range cases {
		lat, lng, ok := extractLocation([]byte(c.in))
		if ok != c.ok || lat != c.lat || lng != c.lng {
			t.Errorf("%s: got (%v, %v, %v), want (%v, %v, %v)", c.name, lat, lng, ok, c.lat, c.lng, c.ok)
		}
	}
}

func TestMapEmbedRoundTrip(t *testing.T) {
	p := NewMarkdownParser()
	cases := []struct {
		directive string
		wantSrc   string
	}{
		{"%% map %%", `src="../map/map.html?embed=1"`},
		{"%% map tag=review %%", `src="../map/map.html?embed=1&amp;tag=review"`},
		{"%% MAP tag=new_york %%", `src="../map/map.html?embed=1&amp;tag=new_york"`},
		{"%% map tag=review exclude=gauntlet icons=cafe:☕,*:🍴 %%",
			`src="../map/map.html?embed=1&amp;exclude=gauntlet&amp;icons=cafe%3A%E2%98%95%2C%2A%3A%F0%9F%8D%B4&amp;tag=review"`},
		{"%% map bogus=1 tag=review %%", `src="../map/map.html?embed=1&amp;tag=review"`},
		{"%% map tag=review hide=WIP,toronto %%", `src="../map/map.html?embed=1&amp;hide=WIP%2Ctoronto&amp;tag=review"`},
		{"<!-- map tag=review -->", `src="../map/map.html?embed=1&amp;tag=review"`},
	}
	for _, c := range cases {
		md := []byte("# Title\n\nIntro\n\n" + c.directive + "\n\nOutro\n")
		raw := protectMapEmbeds(md)
		raw = stripObsidianComments(raw)
		var buf strings.Builder
		if err := p.markdown.Convert(raw, &buf); err != nil {
			t.Fatal(err)
		}
		html := string(injectMapEmbeds([]byte(buf.String()), "../"))
		if !strings.Contains(html, c.wantSrc) {
			t.Errorf("%q: missing %s in\n%s", c.directive, c.wantSrc, html)
		}
		if strings.Contains(html, "NEBULAMAPEMBED") || strings.Contains(html, "<p><div") {
			t.Errorf("%q: sentinel or stray paragraph left in\n%s", c.directive, html)
		}
	}
}
