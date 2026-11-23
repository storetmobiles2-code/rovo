package geo

import "testing"

func TestStraightM(t *testing.T) {
	cases := []struct {
		name string
		a, b Point
		want int
	}{
		{"same point", Point{16.7488, 78.0035}, Point{16.7488, 78.0035}, 0},
		// One degree of latitude on the reference sphere: R·π/180 = 111,195.08 m.
		{"one degree of latitude", Point{16, 78}, Point{17, 78}, 111195},
		// One degree of longitude at the equator is the same arc.
		{"one degree of longitude at equator", Point{0, 78}, Point{0, 79}, 111195},
		// Longitude shrinks by cos(lat): 111,195.08 × cos(60°) = 55,597.5 m.
		{"one degree of longitude at 60N", Point{60, 10}, Point{60, 11}, 55597},
		// Half the circumference: π·R = 20,015,114.4 m.
		{"antipodes", Point{0, 0}, Point{0, 180}, 20015114},
	}
	for _, c := range cases {
		if got := StraightM(c.a, c.b); got < c.want-1 || got > c.want+1 {
			t.Errorf("%s: StraightM = %d, want %d (±1)", c.name, got, c.want)
		}
		if ab, ba := StraightM(c.a, c.b), StraightM(c.b, c.a); ab != ba {
			t.Errorf("%s: not symmetric: %d vs %d", c.name, ab, ba)
		}
	}
}

func TestRoadM(t *testing.T) {
	// Doc 16 §4.2 worked examples: straight × 1.3, rounded half up.
	for _, c := range []struct{ straight, want int }{{5410, 7033}, {7703, 10014}, {7000, 9100}, {0, 0}, {1, 1}, {5, 7}} {
		if got := RoadM(c.straight, 1300); got != c.want {
			t.Errorf("RoadM(%d, 1300) = %d, want %d", c.straight, got, c.want)
		}
	}
}

func TestPointValid(t *testing.T) {
	for p, want := range map[Point]bool{
		{16.7488, 78.0035}: true, {0, 0}: false, {91, 0}: false, {-91, 10}: false,
		{10, 181}: false, {10, -181}: false, {0, 78}: true, {-90, -180}: true,
	} {
		if got := p.Valid(); got != want {
			t.Errorf("%+v.Valid() = %v, want %v", p, got, want)
		}
	}
}
