package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type geoResp struct {
	Features []struct {
		Geometry struct {
			Coordinates [2]float64 `json:"coordinates"` // [lon, lat]
		} `json:"geometry"`
		Properties struct {
			Label string  `json:"label"`
			Score float64 `json:"score"`
		} `json:"properties"`
	} `json:"features"`
}

type wpt struct {
	Lat  float64 `xml:"lat,attr"`
	Lon  float64 `xml:"lon,attr"`
	Name string  `xml:"name"`
	Desc string  `xml:"desc,omitempty"`
}

type gpx struct {
	XMLName xml.Name `xml:"gpx"`
	Xmlns   string   `xml:"xmlns,attr"`
	Version string   `xml:"version,attr"`
	Creator string   `xml:"creator,attr"`
	Wpts    []wpt    `xml:"wpt"`
}

// requestInterval paces requests well under the 50 req/s API limit.
const requestInterval = 50 * time.Millisecond

// maxRetries is the number of extra attempts after a 429 response.
const maxRetries = 3

// retryDelay honors the Retry-After header (seconds) when present, otherwise
// falls back to exponential backoff: 1s, 2s, 4s.
// Note: in the browser (WASM), Retry-After is usually not CORS-exposed, so the
// fallback is the common path there.
func retryDelay(resp *http.Response, attempt int) time.Duration {
	if s := resp.Header.Get("Retry-After"); s != "" {
		if secs, err := strconv.Atoi(s); err == nil && secs > 0 {
			return time.Duration(secs) * time.Second
		}
	}
	return time.Second << attempt
}

// geocodeBase is a variable so tests can point it at a local server.
var geocodeBase = "https://data.geopf.fr/geocodage/search?index=address&limit=1&q="

func geocode(c *http.Client, addr string, logf func(string)) (*wpt, error) {
	u := geocodeBase + url.QueryEscape(addr)
	var resp *http.Response
	for attempt := 0; ; attempt++ {
		var err error
		resp, err = c.Get(u)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusTooManyRequests {
			break
		}
		resp.Body.Close()
		if attempt >= maxRetries {
			return nil, fmt.Errorf("HTTP 429 après %d tentatives", attempt+1)
		}
		d := retryDelay(resp, attempt)
		logf(fmt.Sprintf("...  %q: HTTP 429 (rate limit), nouvelle tentative dans %s", addr, d))
		time.Sleep(d)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	var g geoResp
	if err := json.NewDecoder(resp.Body).Decode(&g); err != nil {
		return nil, err
	}
	if len(g.Features) == 0 {
		return nil, fmt.Errorf("aucun résultat")
	}
	f := g.Features[0]
	return &wpt{
		Lat:  f.Geometry.Coordinates[1],
		Lon:  f.Geometry.Coordinates[0],
		Name: addr,
		Desc: fmt.Sprintf("%s (score %.2f)", f.Properties.Label, f.Properties.Score),
	}, nil
}

// generate geocodes each non-empty, non-comment line of input and returns the
// GPX document. logf receives one progress line per address; failures are
// logged and skipped, they do not abort the run.
func generate(c *http.Client, input string, logf func(string)) (string, error) {
	out := gpx{Xmlns: "http://www.topografix.com/GPX/1/1", Version: "1.1", Creator: "addr2gpx"}

	first := true
	for _, raw := range strings.Split(input, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Pace every request (success or failure), not just successful ones.
		if !first {
			time.Sleep(requestInterval)
		}
		first = false
		w, err := geocode(c, line, logf)
		if err != nil {
			logf(fmt.Sprintf("NOK  %q: %v", line, err))
			continue
		}
		logf(fmt.Sprintf("OK   %q -> %.5f, %.5f (%s)", line, w.Lat, w.Lon, w.Desc))
		out.Wpts = append(out.Wpts, *w)
	}

	if len(out.Wpts) == 0 {
		return "", fmt.Errorf("aucune adresse géocodée")
	}

	var sb strings.Builder
	sb.WriteString(xml.Header)
	enc := xml.NewEncoder(&sb)
	enc.Indent("", "  ")
	if err := enc.Encode(out); err != nil {
		return "", err
	}
	sb.WriteString("\n")
	return sb.String(), nil
}
