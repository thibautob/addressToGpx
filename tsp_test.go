package main

import (
	"math"
	"testing"
)

func names(pts []wpt) []string {
	out := make([]string, len(pts))
	for i, p := range pts {
		out[i] = p.Name
	}
	return out
}

func TestHaversine(t *testing.T) {
	// Paris -> Lyon, ~392 km great-circle.
	paris := wpt{Lat: 48.8566, Lon: 2.3522}
	lyon := wpt{Lat: 45.7640, Lon: 4.8357}
	if d := haversine(paris, lyon); math.Abs(d-392000) > 5000 {
		t.Errorf("haversine(Paris, Lyon) = %.0f m, want ~392000", d)
	}
}

func TestOrderOpenSortsCollinearPoints(t *testing.T) {
	// Shuffled points on a meridian; the optimal open path from A sweeps north.
	pts := []wpt{
		{Name: "A", Lat: 45.0, Lon: 5.0},
		{Name: "D", Lat: 45.3, Lon: 5.0},
		{Name: "B", Lat: 45.1, Lon: 5.0},
		{Name: "E", Lat: 45.4, Lon: 5.0},
		{Name: "C", Lat: 45.2, Lon: 5.0},
	}
	got := names(orderOpen(pts))
	want := []string{"A", "B", "C", "D", "E"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order = %v, want %v", got, want)
		}
	}
}

func TestOrderOpenKeepsFixedStartAndImproves(t *testing.T) {
	// A layout where greedy nearest-neighbor alone is suboptimal: 2-opt must
	// still return a path starting at index 0 that is no longer than input order.
	pts := []wpt{
		{Name: "start", Lat: 45.00, Lon: 5.00},
		{Name: "far", Lat: 45.20, Lon: 5.00},
		{Name: "near1", Lat: 45.01, Lon: 5.01},
		{Name: "mid", Lat: 45.10, Lon: 5.02},
		{Name: "near2", Lat: 45.02, Lon: 5.00},
	}
	got := orderOpen(pts)
	if got[0].Name != "start" {
		t.Fatalf("start moved: order = %v", names(got))
	}
	if pathLen(got) > pathLen(pts) {
		t.Errorf("optimized path (%.0f m) longer than input order (%.0f m)", pathLen(got), pathLen(pts))
	}
}

func TestOrderOpenSmallInputsUntouched(t *testing.T) {
	pts := []wpt{{Name: "A"}, {Name: "B"}}
	got := orderOpen(pts)
	if len(got) != 2 || got[0].Name != "A" || got[1].Name != "B" {
		t.Errorf("2-point input changed: %v", names(got))
	}
}
