package main

import "math"

const earthRadiusM = 6371000

// haversine returns the great-circle distance between two waypoints in meters.
func haversine(a, b wpt) float64 {
	la1, lo1 := a.Lat*math.Pi/180, a.Lon*math.Pi/180
	la2, lo2 := b.Lat*math.Pi/180, b.Lon*math.Pi/180
	sdla := math.Sin((la2 - la1) / 2)
	sdlo := math.Sin((lo2 - lo1) / 2)
	h := sdla*sdla + math.Cos(la1)*math.Cos(la2)*sdlo*sdlo
	return 2 * earthRadiusM * math.Asin(math.Sqrt(h))
}

// pathLen returns the total length of the open path visiting pts in order.
func pathLen(pts []wpt) float64 {
	var total float64
	for i := 1; i < len(pts); i++ {
		total += haversine(pts[i-1], pts[i])
	}
	return total
}

// orderOpen reorders pts as an open path (no return to start) beginning at
// pts[0], minimizing total haversine distance. Heuristic: nearest neighbor
// then 2-opt until no improving move remains — near-optimal for the few dozen
// points this tool handles.
func orderOpen(pts []wpt) []wpt {
	n := len(pts)
	if n < 3 {
		return pts
	}

	d := make([][]float64, n)
	for i := range d {
		d[i] = make([]float64, n)
		for j := range d[i] {
			d[i][j] = haversine(pts[i], pts[j])
		}
	}

	// Nearest neighbor from the fixed start.
	order := make([]int, 1, n)
	visited := make([]bool, n)
	visited[0] = true
	cur := 0
	for len(order) < n {
		best, bestD := -1, math.Inf(1)
		for j := 0; j < n; j++ {
			if !visited[j] && d[cur][j] < bestD {
				best, bestD = j, d[cur][j]
			}
		}
		visited[best] = true
		order = append(order, best)
		cur = best
	}

	// 2-opt: reverse order[i..j] whenever it shortens the path. The start is
	// fixed (i >= 1) and the path is open, so when j is the last index the
	// edge after it does not exist and only the incoming edge changes.
	for improved := true; improved; {
		improved = false
		for i := 1; i < n-1; i++ {
			for j := i + 1; j < n; j++ {
				delta := d[order[i-1]][order[j]] - d[order[i-1]][order[i]]
				if j < n-1 {
					delta += d[order[i]][order[j+1]] - d[order[j]][order[j+1]]
				}
				if delta < -1e-9 {
					for l, r := i, j; l < r; l, r = l+1, r-1 {
						order[l], order[r] = order[r], order[l]
					}
					improved = true
				}
			}
		}
	}

	out := make([]wpt, n)
	for i, idx := range order {
		out[i] = pts[idx]
	}
	return out
}
