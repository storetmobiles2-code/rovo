// Package geo holds the canonical distance maths (doc 16 §4, GEO-D03): haversine on a sphere and
// the road-distance adjustment. Serviceability and pricing decisions use these functions, never
// a PostGIS distance, so every caller agrees to the metre.
package geo

import "math"

// EarthRadiusM is the mean Earth radius used by the canonical formula.
const EarthRadiusM = 6_371_008.8

// Point is a WGS-84 position in degrees.
type Point struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

// Valid reports whether p is a usable pin: inside the coordinate ranges, finite and not the
// (0,0) null island that clients send for an unset location (doc 16 §3.1 step 1).
func (p Point) Valid() bool {
	if math.IsNaN(p.Lat) || math.IsNaN(p.Lng) || math.IsInf(p.Lat, 0) || math.IsInf(p.Lng, 0) {
		return false
	}
	if p.Lat < -90 || p.Lat > 90 || p.Lng < -180 || p.Lng > 180 {
		return false
	}
	return p.Lat != 0 || p.Lng != 0
}

// StraightM is the haversine straight-line distance between a and b in whole metres, rounded
// half up. It is the distance checked against the serviceability radius (R18).
func StraightM(a, b Point) int {
	const rad = math.Pi / 180
	dLat := (b.Lat - a.Lat) * rad
	dLng := (b.Lng - a.Lng) * rad
	sLat, sLng := math.Sin(dLat/2), math.Sin(dLng/2)
	h := sLat*sLat + math.Cos(a.Lat*rad)*math.Cos(b.Lat*rad)*sLng*sLng
	return int(math.Floor(2*EarthRadiusM*math.Asin(math.Min(1, math.Sqrt(h))) + 0.5))
}

// RoadM converts straight-line metres to road metres with a per-zone factor in thousandths
// (1300 = ×1.3), rounded half up. Road metres drive the fee slab, rider pay and ETA only.
func RoadM(straightM, roadFactorMilli int) int {
	return (straightM*roadFactorMilli + 500) / 1000
}
