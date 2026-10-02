package main

import (
	"encoding/hex"
	"encoding/json"
	"net/url"
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

// mapEmbedDirective matches `%% map %%` with optional key=value options, e.g.
// `%% map tag=review exclude=gauntlet icons=cafe:☕,*:🍴 %%` (or the HTML comment
// equivalent). Options are passed through to the viewer as query parameters:
//
//	tag=a,b      only pages with at least one of these tags
//	exclude=a,b  drop pages with any of these tags
//	icons=t:i,…  draw pins as icon i for pages tagged t; the first match in the
//	             list wins and `*` is the fallback. Without it, pins are dots.
var mapEmbedDirective = regexp.MustCompile(`(?i)%%\s*map((?:\s+[a-z]+=[^\s%]+)*)\s*%%|<!--\s*map((?:\s+[a-z]+=\S+?)*)\s*-->`)

var mapEmbedOptions = map[string]bool{"tag": true, "exclude": true, "icons": true}

// The options are hex-encoded into the sentinel so markdown conversion can't mangle
// them (underscores, typographer quotes, emoji, etc).
var mapEmbedSentinelRe = regexp.MustCompile(`NEBULAMAPEMBEDxZ9([0-9a-f]*)END`)

// protectMapEmbeds replaces map directives with a plain-text sentinel, mirroring
// protectGraphEmbeds, so they survive comment-stripping and raw-HTML omission.
func protectMapEmbeds(data []byte) []byte {
	return mapEmbedDirective.ReplaceAllFunc(data, func(match []byte) []byte {
		sub := mapEmbedDirective.FindSubmatch(match)
		opts := string(sub[1]) + string(sub[2])
		return []byte("\n\nNEBULAMAPEMBEDxZ9" + hex.EncodeToString([]byte(opts)) + "END\n\n")
	})
}

// mapEmbedQuery turns the directive's options into the viewer's query string.
// Unknown options are ignored.
func mapEmbedQuery(opts string) string {
	q := url.Values{"embed": {"1"}}
	for _, field := range strings.Fields(opts) {
		key, val, ok := strings.Cut(field, "=")
		key = strings.ToLower(key)
		if ok && val != "" && mapEmbedOptions[key] {
			q.Set(key, val)
		}
	}
	return q.Encode()
}

// injectMapEmbeds swaps map sentinels for an iframe embedding the map viewer.
func injectMapEmbeds(html []byte, prefix string) []byte {
	if !mapEmbedSentinelRe.Match(html) {
		return html
	}
	iframe := func(hexOpts []byte) []byte {
		opts, _ := hex.DecodeString(string(hexOpts))
		src := prefix + "map/map.html?" + strings.ReplaceAll(mapEmbedQuery(string(opts)), "&", "&amp;")
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

// writeMapViewer writes map/map.html: a chrome-free MapLibre map of every pin,
// configured by query parameters (see mapEmbedDirective). Pins link back into the site (target=_top so clicking
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
    <link rel="stylesheet" href="https://unpkg.com/maplibre-gl@5.24.0/dist/maplibre-gl.css" crossorigin="">
    <style>
        @import url('https://fonts.googleapis.com/css2?family=Inter:opsz,wght@14..32,400..700&display=swap');
        /* Site tokens, plus the map palette derived from them. The map style reads
           these at load, so the basemap always matches the site theme. */
        :root, [data-theme="dark"] {
            color-scheme: dark;
            --bg: #0b0d11; --card-bg: #14181f; --text: #c7d0dc; --heading: #f1f5f9; --muted: #7d8797;
            --link: #5eb1ef; --border: rgba(255,255,255,0.09);
            --map-land: #0b0d11; --map-water: #0f1620; --map-park: #0d1112;
            --map-road-major: #343c4c; --map-road-mid: #1e232c; --map-road-minor: #171b22;
            --map-label: #6b7585; --map-label-strong: #9aa4b2; --map-halo: #0b0d11;
        }
        [data-theme="light"] {
            color-scheme: light;
            --bg: #ffffff; --card-bg: #ffffff; --text: #3f4855; --heading: #0d1420; --muted: #6b7480;
            --link: #1a6fd4; --border: rgba(15,23,42,0.1);
            --map-land: #f6f8fa; --map-water: #dfe7f0; --map-park: #eef2f0;
            --map-road-major: #c4ccd6; --map-road-mid: #e1e6ec; --map-road-minor: #e9edf1;
            --map-label: #8a929c; --map-label-strong: #5b6470; --map-halo: #f6f8fa;
        }
        html, body { height: 100%; margin: 0; }
        body { font-family: 'Inter', -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; background: var(--map-land); color: var(--text); }
        #map { position: absolute; inset: 0; }
        .pin { cursor: pointer; }
        .pin-dot { width: 14px; height: 14px; box-sizing: border-box; border-radius: 50%; background: var(--link); border: 2px solid var(--bg); box-shadow: 0 1px 4px rgba(0,0,0,0.35); }
        .pin-icon {
            display: flex; align-items: center; justify-content: center; width: 30px; height: 30px; box-sizing: border-box;
            border-radius: 50%; background: var(--card-bg); border: 2px solid var(--link); font-size: 15px; line-height: 1;
            box-shadow: 0 2px 8px rgba(0,0,0,0.35);
        }
        .pin-dot, .pin-icon { transition: transform 0.12s; }
        .pin:hover .pin-dot, .pin:hover .pin-icon { transform: scale(1.15); }
        .maplibregl-popup-content { background: var(--card-bg); color: var(--text); border: 1px solid var(--border); border-radius: 10px; padding: 11px 26px 11px 14px; font-family: inherit; font-size: 13px; line-height: 1.4; box-shadow: 0 10px 30px -10px rgba(0,0,0,0.5); }
        .maplibregl-popup-anchor-bottom .maplibregl-popup-tip, .maplibregl-popup-anchor-bottom-left .maplibregl-popup-tip, .maplibregl-popup-anchor-bottom-right .maplibregl-popup-tip { border-top-color: var(--card-bg); }
        .maplibregl-popup-anchor-top .maplibregl-popup-tip, .maplibregl-popup-anchor-top-left .maplibregl-popup-tip, .maplibregl-popup-anchor-top-right .maplibregl-popup-tip { border-bottom-color: var(--card-bg); }
        .maplibregl-popup-anchor-left .maplibregl-popup-tip { border-right-color: var(--card-bg); }
        .maplibregl-popup-anchor-right .maplibregl-popup-tip { border-left-color: var(--card-bg); }
        .maplibregl-popup-close-button { color: var(--muted); font-size: 16px; right: 4px; top: 2px; }
        .maplibregl-popup-close-button:hover { background: none; color: var(--heading); }
        a.pin-title { color: var(--heading); font-weight: 600; text-decoration: none; font-size: 14px; }
        a.pin-title:hover { color: var(--link); }
        .pin-tags { margin-top: 4px; color: var(--muted); font-size: 12px; }
        .maplibregl-ctrl-group { background: var(--card-bg); border: 1px solid var(--border); box-shadow: none !important; }
        .maplibregl-ctrl-group button + button { border-top-color: var(--border); }
        [data-theme="dark"] .maplibregl-ctrl-icon { filter: invert(1) brightness(0.8); }
        .maplibregl-ctrl-attrib { background: color-mix(in srgb, var(--bg) 75%, transparent) !important; color: var(--muted); font-size: 10.5px; }
        .maplibregl-ctrl-attrib a { color: var(--muted); }
        [data-theme="dark"] .maplibregl-ctrl-attrib-button { filter: invert(1); }
        #empty { position: absolute; inset: 0; display: none; align-items: center; justify-content: center; color: var(--muted); font-size: 14px; z-index: 5; pointer-events: none; }
    </style>
</head>
<body>
    <div id="map"></div>
    <div id="empty">No pages with a location yet.</div>
    <script src="https://unpkg.com/maplibre-gl@5.24.0/dist/maplibre-gl.js" crossorigin=""></script>
    <script>
    var pins = {{PINS}};
    var params = new URLSearchParams(location.search);
    function list(name) { return (params.get(name) || '').split(',').filter(Boolean); }
    function hasAny(p, tags) { return tags.some(function(t) { return (p.tags || []).indexOf(t) !== -1; }); }
    var only = list('tag'), exclude = list('exclude');
    pins = pins.filter(function(p) { return (!only.length || hasAny(p, only)) && !hasAny(p, exclude); });

    // icons=cafe:☕,bar:🍸,*:🍴 — the first rule whose tag the page has wins.
    var iconRules = list('icons').map(function(r) {
        var i = r.indexOf(':');
        return i > 0 ? { tag: r.slice(0, i), icon: r.slice(i + 1) } : null;
    }).filter(Boolean);
    function iconFor(p) {
        for (var i = 0; i < iconRules.length; i++) {
            var r = iconRules[i];
            if (r.tag === '*' || (p.tags || []).indexOf(r.tag) !== -1) return r.icon;
        }
        return null;
    }

    // A deliberately sparse basemap on OpenFreeMap's vector tiles: water, parks,
    // roads that appear by importance as you zoom in, and place names. No
    // buildings, POIs, rail or boundaries.
    var css = getComputedStyle(document.documentElement);
    function c(name) { return css.getPropertyValue(name).trim(); }
    var lines = ['match', ['geometry-type'], ['LineString', 'MultiLineString'], true, false];
    function roads(id, classes, minzoom, color, widths) {
        return {
            id: id, type: 'line', source: 'omt', 'source-layer': 'transportation', minzoom: minzoom,
            // Ramps add spiky clutter at a glance; they only matter once zoomed in.
            filter: ['all', lines, ['match', ['get', 'class'], classes, true, false],
                ['any', ['>=', ['zoom'], 14], ['!=', ['get', 'ramp'], 1]]],
            layout: { 'line-cap': 'round', 'line-join': 'round' },
            paint: { 'line-color': color, 'line-width': ['interpolate', ['exponential', 1.6], ['zoom']].concat(widths) }
        };
    }
    function places(id, classes, minzoom, size, color) {
        return {
            id: id, type: 'symbol', source: 'omt', 'source-layer': 'place', minzoom: minzoom,
            filter: ['match', ['get', 'class'], classes, true, false],
            layout: {
                'text-field': ['coalesce', ['get', 'name:en'], ['get', 'name:latin'], ['get', 'name']],
                'text-font': ['Noto Sans Regular'], 'text-size': size, 'text-max-width': 8,
                'text-letter-spacing': 0.02
            },
            paint: { 'text-color': color, 'text-halo-color': c('--map-halo'), 'text-halo-width': 1.4 }
        };
    }
    var style = {
        version: 8,
        glyphs: 'https://tiles.openfreemap.org/fonts/{fontstack}/{range}.pbf',
        sources: { omt: { type: 'vector', url: 'https://tiles.openfreemap.org/planet',
            attribution: '<a href="https://openfreemap.org" target="_blank">OpenFreeMap</a> © <a href="https://www.openmaptiles.org/" target="_blank">OpenMapTiles</a> © <a href="https://www.openstreetmap.org/copyright" target="_blank">OpenStreetMap</a>' } },
        layers: [
            { id: 'land', type: 'background', paint: { 'background-color': c('--map-land') } },
            { id: 'park', type: 'fill', source: 'omt', 'source-layer': 'park', minzoom: 9, paint: { 'fill-color': c('--map-park') } },
            { id: 'water', type: 'fill', source: 'omt', 'source-layer': 'water',
              filter: ['!=', ['get', 'brunnel'], 'tunnel'], paint: { 'fill-color': c('--map-water') } },
            roads('roads-minor', ['minor', 'service'], 14, c('--map-road-minor'), [14, 0.5, 16, 2.5, 19, 12]),
            roads('roads-tertiary', ['tertiary'], 12.5, c('--map-road-mid'), [12.5, 0.5, 15, 2.2, 19, 14]),
            roads('roads-arterial', ['primary', 'secondary'], 10.5, c('--map-road-mid'), [10.5, 0.4, 13, 1.4, 16, 5, 19, 18]),
            roads('roads-highway', ['motorway', 'trunk'], 5, c('--map-road-major'), [5, 0.6, 9, 1.4, 12, 2.6, 15, 6, 19, 22]),
            { id: 'road-names', type: 'symbol', source: 'omt', 'source-layer': 'transportation_name', minzoom: 14,
              filter: ['match', ['get', 'class'], ['motorway', 'trunk', 'primary', 'secondary', 'tertiary', 'minor'], true, false],
              layout: { 'symbol-placement': 'line', 'text-field': ['coalesce', ['get', 'name:en'], ['get', 'name']],
                        'text-font': ['Noto Sans Regular'], 'text-size': 11 },
              paint: { 'text-color': c('--map-label'), 'text-halo-color': c('--map-halo'), 'text-halo-width': 1.4 } },
            places('places-minor', ['suburb', 'neighbourhood', 'quarter'], 13, 11, c('--map-label')),
            places('places-village', ['village'], 12, 11, c('--map-label')),
            places('places-town', ['town'], 9, ['interpolate', ['linear'], ['zoom'], 9, 11, 14, 13], c('--map-label')),
            places('places-city', ['city'], 4, ['interpolate', ['linear'], ['zoom'], 4, 11, 12, 16], c('--map-label-strong'))
        ]
    };

    var map = new maplibregl.Map({ container: 'map', style: style, center: [0, 20], zoom: 1.5,
        attributionControl: { compact: true }, dragRotate: false, pitchWithRotate: false });
    map.touchZoomRotate.disableRotation();
    map.addControl(new maplibregl.NavigationControl({ showCompass: false }), 'top-left');

    function esc(s) { var d = document.createElement('div'); d.textContent = s; return d.innerHTML; }
    pins.forEach(function(p) {
        var tags = (p.tags || []).filter(function(t) { return only.indexOf(t) === -1; });
        var icon = iconFor(p);
        var el = document.createElement('div');
        el.className = 'pin';
        el.title = p.title;
        el.innerHTML = icon ? '<div class="pin-icon">' + esc(icon) + '</div>' : '<div class="pin-dot"></div>';
        var popup = new maplibregl.Popup({ offset: icon ? 18 : 10, maxWidth: '260px' }).setHTML(
            '<a class="pin-title" target="_top" href="../' + encodeURI(p.href) + '">' + esc(p.title) + '</a>' +
            (tags.length ? '<div class="pin-tags">' + tags.map(esc).join(' · ') + '</div>' : ''));
        new maplibregl.Marker({ element: el }).setLngLat([p.lng, p.lat]).setPopup(popup).addTo(map);
    });

    // Open on the main cluster rather than every pin, so a couple of far-flung
    // pages (travel) don't zoom the map out to a continent. Zoom out to see the rest.
    function km(a, b) {
        var r = Math.PI / 180, dLat = (b.lat - a.lat) * r, dLng = (b.lng - a.lng) * r;
        var h = Math.sin(dLat / 2) * Math.sin(dLat / 2) + Math.cos(a.lat * r) * Math.cos(b.lat * r) * Math.sin(dLng / 2) * Math.sin(dLng / 2);
        return 12742 * Math.asin(Math.sqrt(h));
    }
    function mainCluster(ps) {
        var best = [];
        ps.forEach(function(a) {
            var near = ps.filter(function(b) { return km(a, b) < 150; });
            if (near.length > best.length) best = near;
        });
        return best;
    }
    if (pins.length) {
        var cluster = mainCluster(pins), bounds = new maplibregl.LngLatBounds();
        cluster.forEach(function(p) { bounds.extend([p.lng, p.lat]); });
        map.fitBounds(bounds, { padding: 48, maxZoom: 15, duration: 0 });
    } else {
        document.getElementById('empty').style.display = 'flex';
    }
    </script>
</body>
</html>
`
