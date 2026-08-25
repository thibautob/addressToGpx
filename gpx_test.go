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

func TestRetryDelayFallbackIsExponential(t *testing.T) {
	resp := &http.Response{Header: http.Header{}}
	for attempt, want := range []time.Duration{time.Second, 2 * time.Second, 4 * time.Second} {
		if got := retryDelay(resp, attempt); got != want {
			t.Errorf("attempt %d: got %s, want %s", attempt, got, want)
		}
	}
}
