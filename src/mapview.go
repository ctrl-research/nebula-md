package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// MapPin is a page with a `location: [lat, lng]` frontmatter field.
type MapPin struct {
	Title string   `json:"title"`
	Href  string   `json:"href"` // relative to the output root
	Lat   float64  `json:"lat"`
	Lng   float64  `json:"lng"`
	Tags  []string `json:"tags,omitempty"`
}

// locationRe matches `location: [lat, lng]` or `location: "lat, lng"` — the same
// field the Obsidian Map View plugin reads, so pins show up in both places.
var locationRe = regexp.MustCompile(`(?m)^location:\s*["'\[]?\s*(-?\d+(?:\.\d+)?)\s*,\s*(-?\d+(?:\.\d+)?)\s*["'\]]?\s*$`)

func extractLocation(data []byte) (lat, lng float64, ok bool) {
	re := regexp.MustCompile(`(?s)^---\s*\n(.*?)\n---\n?`)
	matches := re.FindSubmatch(data)
	if len(matches) == 0 {
		return 0, 0, false
	}
	m := locationRe.FindSubmatch(matches[1])
	if len(m) == 0 {
		return 0, 0, false
	}
	lat, err1 := strconv.ParseFloat(string(m[1]), 64)
	lng, err2 := strconv.ParseFloat(string(m[2]), 64)
	if err1 != nil || err2 != nil || lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		return 0, 0, false
	}
	return lat, lng, true
}

// buildMapPins walks the vault and collects every page that has a location.
func buildMapPins(vaultDir string) []MapPin {
	pins := []MapPin{}
	filepath.Walk(vaultDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || !strings.HasSuffix(path, ".md") || isIgnored(path) {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		lat, lng, ok := extractLocation(data)
		if !ok {
			return nil
		}
		rel, _ := filepath.Rel(vaultDir, path)
		pins = append(pins, MapPin{
			Title: extractTitle(stripObsidianComments(data)),
			Href:  filepath.ToSlash(pageIDFromRelPath(rel)) + linkExt,
			Lat:   lat,
			Lng:   lng,
			Tags:  extractTags(data),
		})
		return nil
	})
	return pins
}

// mapEmbedDirective matches `%% map %%` or `%% map tag=review %%` (or the HTML
// comment equivalents). The optional tag limits the map to pages with that tag.
var mapEmbedDirective = regexp.MustCompile(`(?i)%%\s*map(?:\s+tag=([^\s%]+))?\s*%%|<!--\s*map(?:\s+tag=(\S+?))?\s*-->`)

// The tag is hex-encoded into the sentinel so markdown conversion can't mangle it
// (underscores, typographer quotes, etc).
var mapEmbedSentinelRe = regexp.MustCompile(`NEBULAMAPEMBEDxZ9([0-9a-f]*)END`)

// protectMapEmbeds replaces map directives with a plain-text sentinel, mirroring
// protectGraphEmbeds, so they survive comment-stripping and raw-HTML omission.
func protectMapEmbeds(data []byte) []byte {
	return mapEmbedDirective.ReplaceAllFunc(data, func(match []byte) []byte {
		sub := mapEmbedDirective.FindSubmatch(match)
		tag := string(sub[1]) + string(sub[2])
		return []byte("\n\nNEBULAMAPEMBEDxZ9" + hex.EncodeToString([]byte(tag)) + "END\n\n")
	})
}

// injectMapEmbeds swaps map sentinels for an iframe embedding the map viewer.
func injectMapEmbeds(html []byte, prefix string) []byte {
	if !mapEmbedSentinelRe.Match(html) {
		return html
	}
	iframe := func(hexTag []byte) []byte {
		src := prefix + "map/map.html?embed=1"
		if tag, err := hex.DecodeString(string(hexTag)); err == nil && len(tag) > 0 {
			src += "&amp;tag=" + urlQueryEscape(string(tag))
		}
		return []byte(`<div class="map-embed"><iframe src="` + src +
			`" loading="lazy" title="Map" allowfullscreen></iframe></div>`)
	}
	// Goldmark wraps a lone sentinel in its own paragraph; replace the whole block first.
	wrapped := regexp.MustCompile(`<p>NEBULAMAPEMBEDxZ9([0-9a-f]*)END</p>`)
	html = wrapped.ReplaceAllFunc(html, func(m []byte) []byte {
		return iframe(wrapped.FindSubmatch(m)[1])
	})
	return mapEmbedSentinelRe.ReplaceAllFunc(html, func(m []byte) []byte {
		return iframe(mapEmbedSentinelRe.FindSubmatch(m)[1])
	})
}

func urlQueryEscape(s string) string {
	var b bytes.Buffer
	for _, c := range []byte(s) {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.' {
			b.WriteByte(c)
		} else {
			fmt.Fprintf(&b, "%%%02X", c)
		}
	}
	return b.String()
}

// writeMapViewer writes map/map.html: a chrome-free Leaflet map of every pin,
// filterable with ?tag=. Pins link back into the site (target=_top so clicking
// from an embed navigates the page, not the iframe).
func writeMapViewer(outputDir string, pins []MapPin, siteTheme string, siteName string) error {
	mapDir := filepath.Join(outputDir, "map")
	if err := os.MkdirAll(mapDir, 0755); err != nil {
		return err
	}
	pinsJSON, err := json.Marshal(pins)
	if err != nil {
		return err
	}
	html := strings.NewReplacer(
		"{{THEME}}", siteTheme,
		"{{SITE_NAME}}", siteName,
		"{{PINS}}", string(pinsJSON),
	).Replace(mapViewerTemplate)
	return os.WriteFile(filepath.Join(mapDir, "map.html"), []byte(html), 0644)
}

const mapViewerTemplate = `<!DOCTYPE html>
<html lang="en" data-theme="{{THEME}}">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Map — {{SITE_NAME}}</title>
    <script>(function(){try{var t=localStorage.getItem('nebula-theme');if(t)document.documentElement.setAttribute('data-theme',t);}catch(e){}})();</script>
    <link rel="stylesheet" href="https://unpkg.com/leaflet@1.9.4/dist/leaflet.css" crossorigin="">
    <style>
        @import url('https://fonts.googleapis.com/css2?family=Inter:opsz,wght@14..32,400..700&display=swap');
        :root, [data-theme="dark"] {
            color-scheme: dark;
            --bg: #0b0d11; --text: #c7d0dc; --heading: #f1f5f9; --muted: #7d8797;
            --link: #5eb1ef; --card-bg: #14181f; --border: rgba(255,255,255,0.09);
        }
        [data-theme="light"] {
            color-scheme: light;
            --bg: #ffffff; --text: #3f4855; --heading: #0d1420; --muted: #6b7480;
            --link: #1a6fd4; --card-bg: #ffffff; --border: rgba(15,23,42,0.1);
        }
        html, body { height: 100%; margin: 0; }
        body { font-family: 'Inter', -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; background: var(--bg); color: var(--text); }
        #map { width: 100vw; height: 100vh; background: var(--bg); }
        /* OSM only serves light tiles; invert them for dark mode. */
        [data-theme="dark"] .leaflet-tile-pane { filter: invert(1) hue-rotate(180deg) brightness(0.95) contrast(0.9); }
        .leaflet-popup-content-wrapper, .leaflet-popup-tip { background: var(--card-bg); color: var(--text); box-shadow: 0 10px 30px -10px rgba(0,0,0,0.5); border: 1px solid var(--border); }
        .leaflet-popup-content { margin: 11px 14px; font-size: 13px; line-height: 1.4; }
        .leaflet-popup-content a.pin-title { color: var(--heading); font-weight: 600; text-decoration: none; font-size: 14px; }
        .leaflet-popup-content a.pin-title:hover { color: var(--link); }
        .pin-tags { margin-top: 4px; color: var(--muted); font-size: 12px; }
        .leaflet-container a.leaflet-popup-close-button { color: var(--muted); }
        #empty { position: absolute; inset: 0; display: none; align-items: center; justify-content: center; color: var(--muted); font-size: 14px; z-index: 1000; pointer-events: none; }
    </style>
</head>
<body>
    <div id="map"></div>
    <div id="empty">No pages with a location yet.</div>
    <script src="https://unpkg.com/leaflet@1.9.4/dist/leaflet.js" crossorigin=""></script>
    <script>
    var pins = {{PINS}};
    var params = new URLSearchParams(location.search);
    var tag = params.get('tag');
    if (tag) pins = pins.filter(function(p) { return (p.tags || []).indexOf(tag) !== -1; });

    var dark = document.documentElement.getAttribute('data-theme') !== 'light';
    var accent = getComputedStyle(document.documentElement).getPropertyValue('--link').trim();
    var map = L.map('map', { zoomControl: true, worldCopyJump: true });
    L.tileLayer('https://tile.openstreetmap.org/{z}/{x}/{y}.png', {
        maxZoom: 19,
        attribution: '&copy; <a href="https://www.openstreetmap.org/copyright" target="_blank">OpenStreetMap</a> contributors'
    }).addTo(map);

    function esc(s) { var d = document.createElement('div'); d.textContent = s; return d.innerHTML; }
    var markers = pins.map(function(p) {
        var tags = (p.tags || []).filter(function(t) { return t !== tag; });
        var html = '<a class="pin-title" target="_top" href="../' + encodeURI(p.href) + '">' + esc(p.title) + '</a>' +
            (tags.length ? '<div class="pin-tags">' + tags.map(esc).join(' · ') + '</div>' : '');
        return L.circleMarker([p.lat, p.lng], {
            radius: 7, weight: 2, color: dark ? '#0b0d11' : '#ffffff', fillColor: accent, fillOpacity: 0.95
        }).bindPopup(html).bindTooltip(esc(p.title), { direction: 'top', offset: [0, -6] }).addTo(map);
    });

    // Open on the main cluster rather than every pin, so a couple of far-flung
    // pages (travel) don't zoom the map out to a continent. Zoom out to see the rest.
    function mainCluster(ms) {
        var best = ms;
        var bestCount = 0;
        ms.forEach(function(a) {
            var near = ms.filter(function(b) { return a.getLatLng().distanceTo(b.getLatLng()) < 150000; });
            if (near.length > bestCount) { best = near; bestCount = near.length; }
        });
        return best;
    }
    if (markers.length) {
        map.fitBounds(L.featureGroup(mainCluster(markers)).getBounds().pad(0.15), { maxZoom: 15 });
    } else {
        map.setView([20, 0], 2);
        document.getElementById('empty').style.display = 'flex';
    }
    </script>
</body>
</html>
`
