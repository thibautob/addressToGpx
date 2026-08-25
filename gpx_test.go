//go:build !js

package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const okBody = `{"features":[{"geometry":{"coordinates":[2.29,49.89]},"properties":{"label":"8 Bd du Port","score":0.97}}]}`

func TestGeocodeRetriesOn429(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls <= 2 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		fmt.Fprint(w, okBody)
	}))
	defer srv.Close()

	old := geocodeBase
	geocodeBase = srv.URL + "/?q="
	defer func() { geocodeBase = old }()

	var logs []string
	start := time.Now()
	w, err := geocode(srv.Client(), "8 boulevard du Port", func(m string) { logs = append(logs, m) })
	if err != nil {
		t.Fatalf("geocode: %v", err)
	}
	if calls != 3 {
		t.Errorf("calls = %d, want 3", calls)
	}
	if w.Lat != 49.89 || w.Lon != 2.29 {
		t.Errorf("got %v,%v want 49.89,2.29", w.Lat, w.Lon)
	}
	// Two retries honoring Retry-After: 1s => at least ~2s total.
	if d := time.Since(start); d < 2*time.Second {
		t.Errorf("retries too fast (%s), Retry-After not honored", d)
	}
	if len(logs) != 2 || !strings.Contains(logs[0], "429") {
		t.Errorf("unexpected retry logs: %v", logs)
	}
}

func TestGeocodeGivesUpAfterMaxRetries(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Retry-After", "1")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	old := geocodeBase
	geocodeBase = srv.URL + "/?q="
	defer func() { geocodeBase = old }()

	_, err := geocode(srv.Client(), "x", func(string) {})
	if err == nil || !strings.Contains(err.Error(), "429") {
		t.Fatalf("want 429 error, got %v", err)
	}
	if calls != maxRetries+1 {
		t.Errorf("calls = %d, want %d", calls, maxRetries+1)
	}
}

func TestGenerateEmitsOptimizedRoute(t *testing.T) {
	// Three collinear addresses given out of order: C is between A and B, so
	// the optimized open path from A must visit C before B.
	coords := map[string][2]float64{ // name -> [lon, lat]
		"A": {5.0, 45.0},
		"B": {5.0, 45.2},
		"C": {5.0, 45.1},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := coords[r.URL.Query().Get("q")]
		fmt.Fprintf(w, `{"features":[{"geometry":{"coordinates":[%f,%f]},"properties":{"label":"x","score":1}}]}`, c[0], c[1])
	}))
	defer srv.Close()

	old := geocodeBase
	geocodeBase = srv.URL + "/?q="
	defer func() { geocodeBase = old }()

	out, err := generate(srv.Client(), "A\nB\nC\n", true, func(string) {})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if !strings.Contains(out, "<rte>") {
		t.Fatal("output has no <rte> element")
	}
	iA := strings.Index(out, "<name>A</name>")
	iB := strings.Index(out, "<name>B</name>")
	iC := strings.Index(out, "<name>C</name>")
	if iA == -1 || iB == -1 || iC == -1 || !(iA < iC && iC < iB) {
		t.Errorf("waypoints not in optimized order A,C,B:\n%s", out)
	}

	// Without optimization: input order preserved, no route element.
	out, err = generate(srv.Client(), "A\nB\nC\n", false, func(string) {})
	if err != nil {
		t.Fatalf("generate (optimize=false): %v", err)
	}
	if strings.Contains(out, "<rte>") {
		t.Error("optimize=false must not emit a <rte> element")
	}
	iA = strings.Index(out, "<name>A</name>")
	iB = strings.Index(out, "<name>B</name>")
	iC = strings.Index(out, "<name>C</name>")
	if !(iA < iB && iB < iC) {
		t.Errorf("optimize=false must keep input order A,B,C:\n%s", out)
	}
}

func TestRetryDelayFallbackIsExponential(t *testing.T) {
	resp := &http.Response{Header: http.Header{}}
	for attempt, want := range []time.Duration{time.Second, 2 * time.Second, 4 * time.Second} {
		if got := retryDelay(resp, attempt); got != want {
			t.Errorf("attempt %d: got %s, want %s", attempt, got, want)
		}
	}
}
