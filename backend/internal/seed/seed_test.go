package seed

import (
	"regexp"
	"strings"
	"testing"
)

func TestValidateSlabs(t *testing.T) {
	if err := ValidateSlabs(feeSlabs, MaxServiceableRadiusM, RoadFactorMilli); err != nil {
		t.Fatalf("default slabs: %v", err)
	}
	bad := map[string][]slab{
		"empty":        {},
		"not at zero":  {{100, 2000, 1}},
		"gap":          {{0, 2000, 1}, {2500, 10000, 1}},
		"overlap":      {{0, 2000, 1}, {1500, 10000, 1}},
		"zero width":   {{0, 0, 1}},
		"too short":    {{0, 2000, 1}, {2000, 9100, 1}}, // 7,000 m × 1.3 = 9,100 m is not < 9,100
		"negative fee": {{0, 10000, -1}},
	}
	for name, s := range bad {
		if err := ValidateSlabs(s, MaxServiceableRadiusM, RoadFactorMilli); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

// TestR18Slabs pins the R18 table: ₹20/30/40/50/60 on [0,2) [2,4) [4,6) [6,8) [8,10) km.
func TestR18Slabs(t *testing.T) {
	want := []slab{{0, 2000, 2000}, {2000, 4000, 3000}, {4000, 6000, 4000}, {6000, 8000, 5000}, {8000, 10000, 6000}}
	if len(feeSlabs) != len(want) {
		t.Fatalf("slabs = %d, want %d", len(feeSlabs), len(want))
	}
	for i := range want {
		if feeSlabs[i] != want[i] {
			t.Errorf("slab %d = %+v, want %+v", i, feeSlabs[i], want[i])
		}
	}
	if PlatformFeePaise != 0 {
		t.Errorf("R59: platform fee must be 0, got %d", PlatformFeePaise)
	}
}

var fakePhone = regexp.MustCompile(`^\+9199999000[0-9]{2}$`)

func hasTelugu(s string) bool {
	for _, r := range s {
		if r >= 0x0C00 && r <= 0x0C7F {
			return true
		}
	}
	return false
}

func TestDatasetShape(t *testing.T) {
	if len(restaurants) != 4 || len(riders) != 4 || len(coupons) != 2 {
		t.Fatalf("want 4 restaurants, 4 riders, 2 coupons; got %d, %d, %d", len(restaurants), len(riders), len(coupons))
	}
	if n := len(localities); n < 10 {
		t.Errorf("want about 10 localities, got %d", n)
	}
	phones := map[string]bool{}
	checkPhone := func(p string) {
		t.Helper()
		if !fakePhone.MatchString(p) {
			t.Errorf("phone %q is outside the reserved fake range", p)
		}
		if phones[p] {
			t.Errorf("phone %q is reused", p)
		}
		phones[p] = true
	}
	for _, rd := range riders {
		checkPhone(rd.Phone)
	}

	slugs := map[string]bool{}
	for _, l := range localities {
		if slugs[l.Slug] {
			t.Errorf("duplicate locality %q", l.Slug)
		}
		slugs[l.Slug] = true
		if !hasTelugu(l.NameTe) {
			t.Errorf("locality %q has no Telugu name", l.Slug)
		}
	}

	for _, rs := range restaurants {
		checkPhone(rs.Phone)
		checkPhone(rs.OwnerPhone)
		if !strings.HasPrefix(rs.Name, "Demo ") {
			t.Errorf("%s: restaurant names must be visibly fictional (Demo …)", rs.Slug)
		}
		if !slugs[rs.Locality] {
			t.Errorf("%s: unknown locality %q", rs.Slug, rs.Locality)
		}
		groups := map[string]bool{}
		for _, g := range rs.AddonGroups {
			groups[g.Key] = true
			if g.Min < 0 || g.Max < 1 || g.Min > g.Max || g.Max > len(g.Addons) {
				t.Errorf("%s: add-on group %q has min/max %d/%d for %d add-ons", rs.Slug, g.Key, g.Min, g.Max, len(g.Addons))
			}
		}
		seen := map[string]int{}
		var withVariants, withAddons int
		names := map[string]bool{}
		for _, c := range rs.Categories {
			for _, it := range c.Items {
				if names[it.Name] {
					t.Errorf("%s: duplicate item %q", rs.Slug, it.Name)
				}
				names[it.Name] = true
				seen[it.Veg]++
				if !hasTelugu(it.NameTe) {
					t.Errorf("%s: item %q has no Telugu name", rs.Slug, it.Name)
				}
				if it.Rupees <= 0 {
					t.Errorf("%s: item %q has no price", rs.Slug, it.Name)
				}
				if len(it.Variants) > 0 {
					withVariants++
					defaults := 0
					for _, v := range it.Variants {
						if v.Default {
							defaults++
							if v.Rupees != it.Rupees {
								t.Errorf("%s: item %q base price %d differs from its default variant %d", rs.Slug, it.Name, it.Rupees, v.Rupees)
							}
						}
					}
					if defaults != 1 {
						t.Errorf("%s: item %q has %d default variants", rs.Slug, it.Name, defaults)
					}
				}
				if len(it.AddonGroups) > 0 {
					withAddons++
				}
				for _, g := range it.AddonGroups {
					if !groups[g] {
						t.Errorf("%s: item %q references unknown group %q", rs.Slug, it.Name, g)
					}
				}
			}
		}
		if withVariants == 0 || withAddons == 0 {
			t.Errorf("%s: menu needs items with variants and with add-ons (got %d, %d)", rs.Slug, withVariants, withAddons)
		}
		if rs.Diet == "PURE_VEG" {
			if seen[nonVeg]+seen[egg] > 0 {
				t.Errorf("%s: PURE_VEG restaurant lists non-veg or egg items", rs.Slug)
			}
		} else if seen[veg] == 0 || seen[nonVeg] == 0 || seen[egg] == 0 {
			t.Errorf("%s: MIXED menu should cover veg, non-veg and egg: %v", rs.Slug, seen)
		}
	}
}

func TestIDIsStable(t *testing.T) {
	if ID("city", CityCode) != ID("city", CityCode) || ID("city", "A") == ID("zone", "A") {
		t.Fatal("ID must be deterministic and scoped by kind")
	}
}
