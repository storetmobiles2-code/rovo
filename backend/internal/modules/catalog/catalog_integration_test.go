//go:build integration

package catalog_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/storetmobiles2-code/rovo/backend/internal/app/apptest"
	pgeo "github.com/storetmobiles2-code/rovo/backend/internal/platform/geo"
)

var centrePin = pgeo.Point{Lat: 16.7488, Lng: 78.0035}

const (
	centre  = "lat=16.7488&lng=78.0035"
	biryani = "Demo Biryani House"   // at the centre, radius 7 km
	tiffin  = "Demo Tiffin Centre"   // PURE_VEG, ~3.3 km from the centre
	meals   = "Demo Telangana Meals" // radius 3 km, ~3.1 km from the centre: too far
	chinese = "Demo Chinese Corner"
)

type env struct {
	*apptest.Env
	t *testing.T
	c *apptest.Client
}

// setup seeds the demo city and makes every restaurant open around the clock, so the tests do
// not depend on the time of day they run at.
func setup(t *testing.T) *env {
	e := &env{Env: apptest.New(t), t: t}
	e.Seed()
	e.c = e.Client(apptest.CustomerHost)
	e.exec(`DELETE FROM restaurant_operating_hours`)
	e.exec(`INSERT INTO restaurant_operating_hours (id, restaurant_id, day_of_week, opens_at, closes_at)
		SELECT gen_random_uuid(), r.id, d, s.o::time, s.c::time FROM restaurants r, generate_series(1, 7) d,
		       (VALUES ('00:00', '12:00'), ('12:00', '00:00')) s(o, c)`)
	e.exec(`UPDATE restaurants SET accepting_orders = true, paused_until = NULL`)
	return e
}

func (e *env) exec(sql string, args ...any) {
	e.t.Helper()
	if _, err := e.DB.Pool.Exec(context.Background(), sql, args...); err != nil {
		e.t.Fatalf("exec %q: %v", sql, err)
	}
}

func (e *env) id(name string) string {
	e.t.Helper()
	var id string
	if err := e.DB.Pool.QueryRow(context.Background(), `SELECT id::text FROM restaurants WHERE name = $1`, name).Scan(&id); err != nil {
		e.t.Fatalf("restaurant %q: %v", name, err)
	}
	return id
}

// discover returns the cards by name, in response order, plus the raw response.
func (e *env) discover(query string) ([]string, map[string]map[string]any, *apptest.Response) {
	e.t.Helper()
	res := e.c.Do(http.MethodGet, "/api/v1/public/restaurants?"+query, nil)
	if res.Code != http.StatusOK {
		e.t.Fatalf("discover %q = %d %s", query, res.Code, res.Body)
	}
	var names []string
	cards := map[string]map[string]any{}
	for _, it := range res.JSON["items"].([]any) {
		card := it.(map[string]any)
		names = append(names, card["name"].(string))
		cards[card["name"].(string)] = card
	}
	return names, cards, res
}

func paise(v any) float64 { return v.(map[string]any)["amountPaise"].(float64) }

func TestDiscoverShowsOnlyRestaurantsThatDeliverToThePin(t *testing.T) {
	e := setup(t)
	names, cards, res := e.discover(centre)
	if len(names) != 3 || cards[meals] != nil {
		t.Fatalf("restaurants at the centre = %v, want the three that deliver there (not %q)", names, meals)
	}
	if res.JSON["nextCursor"] != nil || res.JSON["zone"] != nil {
		t.Errorf("nextCursor/zone = %v/%v, want null", res.JSON["nextCursor"], res.JSON["zone"])
	}
	// Relevance: all open, so nearest first.
	if names[0] != biryani {
		t.Errorf("first card = %q, want the nearest (%q)", names[0], biryani)
	}

	b := cards[biryani]
	if b["distanceM"] != float64(0) || paise(b["deliveryFee"]) != 2000 || b["availability"] != "OPEN" || b["isOpen"] != true {
		t.Errorf("biryani card = %v", b)
	}
	if b["dietType"] != "MIXED" || paise(b["costForTwo"]) != 40000 || b["nameI18n"].(map[string]any)["te"] == "" {
		t.Errorf("biryani card details = %v", b)
	}
	if cs := b["cuisines"].([]any); len(cs) != 3 || cs[0] != "Biryani" {
		t.Errorf("cuisines = %v, want 3 with the primary first", cs)
	}
	// Prep 25 + pickup 5 + 0 km travel + handover 2 = 32 → shown as 30–40.
	if eta := b["eta"].(map[string]any); eta["minMinutes"] != float64(30) || eta["maxMinutes"] != float64(40) {
		t.Errorf("biryani eta = %v, want 30–40", eta)
	}

	// The tiffin centre: fee comes from the road-distance slab (straight × 1.3).
	straight := pgeo.StraightM(centrePin, pgeo.Point{Lat: 16.7300, Lng: 77.9800})
	road := pgeo.RoadM(straight, 1300)
	if road < 4000 || road >= 6000 {
		t.Fatalf("fixture drifted: road distance %d m is not in the 4–6 km slab", road)
	}
	tc := cards[tiffin]
	if tc["distanceM"] != float64(straight) || paise(tc["deliveryFee"]) != 4000 {
		t.Errorf("tiffin card distance/fee = %v/%v, want %d m / 4000 paise", tc["distanceM"], tc["deliveryFee"], straight)
	}
}

func TestDiscoverFiltersAndSort(t *testing.T) {
	e := setup(t)
	if names, _, _ := e.discover(centre + "&veg=true"); len(names) != 1 || names[0] != tiffin {
		t.Errorf("veg=true = %v, want only %q", names, tiffin)
	}
	if names, _, _ := e.discover(centre + "&cuisine=INDO_CHINESE"); len(names) != 1 || names[0] != chinese {
		t.Errorf("cuisine filter = %v", names)
	}
	if names, _, _ := e.discover(centre + "&hasOffer=true"); len(names) != 0 {
		t.Errorf("hasOffer=true = %v, want none (no offers exist)", names)
	}
	if names, _, _ := e.discover(centre + "&minRating=4"); len(names) != 0 {
		t.Errorf("minRating=4 = %v, want none (nothing is rated yet)", names)
	}
	if names, _, _ := e.discover(centre + "&maxEtaMin=40"); len(names) == 0 || names[0] != biryani {
		t.Errorf("maxEtaMin=40 = %v, want to include %q", names, biryani)
	}
	if names, _, _ := e.discover(centre + "&sort=costLow"); names[0] != tiffin || names[2] != biryani {
		t.Errorf("sort=costLow = %v", names)
	}
	if names, _, _ := e.discover(centre + "&sort=costHigh"); names[0] != biryani || names[2] != tiffin {
		t.Errorf("sort=costHigh = %v", names)
	}
	for _, q := range []string{"&sort=price", "&veg=maybe", "&limit=0", "&limit=101", "&minRating=6", "&cursor=abc"} {
		if res := e.c.Do(http.MethodGet, "/api/v1/public/restaurants?"+centre+q, nil); res.Code != http.StatusBadRequest {
			t.Errorf("query %q = %d, want 400", q, res.Code)
		}
	}
}

func TestDiscoverPagination(t *testing.T) {
	e := setup(t)
	names, _, res := e.discover(centre + "&limit=2")
	cursor, _ := res.JSON["nextCursor"].(string)
	if len(names) != 2 || cursor == "" {
		t.Fatalf("page 1 = %v cursor %q", names, cursor)
	}
	more, _, res2 := e.discover(centre + "&limit=2&cursor=" + cursor)
	if len(more) != 1 || res2.JSON["nextCursor"] != nil || more[0] == names[0] || more[0] == names[1] {
		t.Fatalf("page 2 = %v (page 1 %v), nextCursor %v", more, names, res2.JSON["nextCursor"])
	}
	// A cursor with a changed offset fails its signature check.
	forged := "OTk" + cursor[len("Mg"):] // base64("99") with the signature of "2"
	if res := e.c.Do(http.MethodGet, "/api/v1/public/restaurants?"+centre+"&cursor="+forged, nil); res.Code != http.StatusBadRequest || res.JSON["code"] != "INVALID_CURSOR" {
		t.Errorf("forged cursor = %d %v", res.Code, res.JSON)
	}
}

func TestDiscoverOutsideServiceArea(t *testing.T) {
	e := setup(t)
	res := e.c.Do(http.MethodGet, "/api/v1/public/restaurants?lat=17.3850&lng=78.4867", nil)
	if res.Code != http.StatusUnprocessableEntity || res.JSON["code"] != "OUTSIDE_SERVICE_AREA" {
		t.Fatalf("outside = %d %v", res.Code, res.JSON)
	}
	e.exec(`UPDATE cities SET status = 'PLANNED'`)
	if res := e.c.Do(http.MethodGet, "/api/v1/public/restaurants?"+centre, nil); res.JSON["code"] != "CITY_NOT_LIVE" {
		t.Fatalf("city not live = %d %v", res.Code, res.JSON)
	}
}

func TestAvailabilityStates(t *testing.T) {
	e := setup(t)
	// Not accepting orders → PAUSED; no hours → CLOSED; planned closure → CLOSED.
	e.exec(`UPDATE restaurants SET accepting_orders = false WHERE name = $1`, tiffin)
	e.exec(`DELETE FROM restaurant_operating_hours WHERE restaurant_id = (SELECT id FROM restaurants WHERE name = $1)`, chinese)
	names, cards, _ := e.discover(centre)
	if cards[tiffin]["availability"] != "PAUSED" || cards[tiffin]["isOpen"] != false {
		t.Errorf("not accepting orders = %v", cards[tiffin]["availability"])
	}
	if cards[chinese]["availability"] != "CLOSED" || cards[chinese]["opensAt"] != nil {
		t.Errorf("no hours = %v opensAt %v", cards[chinese]["availability"], cards[chinese]["opensAt"])
	}
	if names[0] != biryani {
		t.Errorf("order = %v, want the open restaurant first", names)
	}

	e.exec(`INSERT INTO restaurant_closures (id, restaurant_id, starts_at, ends_at, reason)
		SELECT gen_random_uuid(), id, now() - interval '1 hour', now() + interval '1 hour', 'HOLIDAY' FROM restaurants WHERE name = $1`, biryani)
	e.exec(`UPDATE restaurants SET accepting_orders = true, paused_until = now() + interval '30 minutes' WHERE name = $1`, tiffin)
	_, cards, _ = e.discover(centre)
	if cards[biryani]["availability"] != "CLOSED" {
		t.Errorf("planned closure = %v, want CLOSED", cards[biryani]["availability"])
	}
	if cards[tiffin]["availability"] != "PAUSED" {
		t.Errorf("paused_until in future = %v, want PAUSED", cards[tiffin]["availability"])
	}

	// A paused zone pauses its restaurants and is flagged on the listing, but browsing still works.
	e.exec(`DELETE FROM restaurant_closures`)
	e.exec(`UPDATE zones SET paused_until = now() + interval '1 hour', pause_message_i18n = '{"en":"Rain."}'`)
	_, cards, res := e.discover(centre)
	zone, _ := res.JSON["zone"].(map[string]any)
	if zone == nil || zone["paused"] != true || zone["message"] != "Rain." {
		t.Errorf("zone notice = %v", res.JSON["zone"])
	}
	if cards[biryani]["availability"] != "PAUSED" {
		t.Errorf("restaurant in paused zone = %v, want PAUSED", cards[biryani]["availability"])
	}
}

func TestRestaurantDetail(t *testing.T) {
	e := setup(t)
	path := "/api/v1/public/restaurants/" + e.id(biryani)

	res := e.c.Do(http.MethodGet, path+"?"+centre, nil)
	if res.Code != http.StatusOK || res.JSON["name"] != biryani || res.JSON["availability"] != "OPEN" {
		t.Fatalf("detail = %d %v", res.Code, res.JSON)
	}
	if sv := res.JSON["serviceability"].(map[string]any); sv["status"] != "SERVICEABLE" {
		t.Errorf("serviceability = %v", sv)
	}
	if hours := res.JSON["hours"].([]any); len(hours) != 7 || len(hours[0].(map[string]any)["slots"].([]any)) != 2 {
		t.Errorf("hours = %v", res.JSON["hours"])
	}
	if res.JSON["addressLine"] == "" || res.JSON["fssaiLicenseNo"] == nil || res.JSON["location"] == nil {
		t.Errorf("profile fields missing: %v", res.JSON)
	}

	// Without a pin there is no fee, ETA or serviceability.
	res = e.c.Do(http.MethodGet, path, nil)
	for _, k := range []string{"deliveryFee", "eta", "distanceM", "serviceability"} {
		if _, ok := res.JSON[k]; ok {
			t.Errorf("detail without pin has %q", k)
		}
	}

	// Beyond the restaurant's own radius: shown, but marked TOO_FAR with no fee.
	res = e.c.Do(http.MethodGet, "/api/v1/public/restaurants/"+e.id(meals)+"?"+centre, nil)
	if res.JSON["availability"] != "TOO_FAR" || res.JSON["isOpen"] != false || res.JSON["serviceability"].(map[string]any)["status"] != "TOO_FAR" {
		t.Errorf("too far = %v / %v", res.JSON["availability"], res.JSON["serviceability"])
	}
	if _, ok := res.JSON["deliveryFee"]; ok {
		t.Error("TOO_FAR restaurant has a delivery fee")
	}
	// The same restaurant is reachable from next door.
	res = e.c.Do(http.MethodGet, "/api/v1/public/restaurants/"+e.id(meals)+"?lat=16.7590&lng=78.0290", nil)
	if res.JSON["availability"] != "OPEN" {
		t.Errorf("meals from nearby = %v, want OPEN", res.JSON["availability"])
	}

	for _, p := range []string{"/api/v1/public/restaurants/not-a-uuid", "/api/v1/public/restaurants/00000000-0000-0000-0000-000000000001"} {
		if res := e.c.Do(http.MethodGet, p, nil); res.Code != http.StatusNotFound {
			t.Errorf("%s = %d, want 404", p, res.Code)
		}
	}
	// A suspended restaurant is not public.
	e.exec(`UPDATE restaurants SET status = 'SUSPENDED' WHERE name = $1`, biryani)
	if res := e.c.Do(http.MethodGet, path, nil); res.Code != http.StatusNotFound {
		t.Errorf("suspended restaurant = %d, want 404", res.Code)
	}
	if res := e.c.Do(http.MethodGet, path+"/menu", nil); res.Code != http.StatusNotFound {
		t.Errorf("suspended restaurant menu = %d, want 404", res.Code)
	}
}

func findItem(t *testing.T, menu map[string]any, name string) map[string]any {
	t.Helper()
	for _, c := range menu["categories"].([]any) {
		for _, it := range c.(map[string]any)["items"].([]any) {
			if it.(map[string]any)["name"] == name {
				return it.(map[string]any)
			}
		}
	}
	t.Fatalf("item %q not in menu", name)
	return nil
}

func TestMenu(t *testing.T) {
	e := setup(t)
	path := "/api/v1/public/restaurants/" + e.id(biryani) + "/menu"
	res := e.c.Do(http.MethodGet, path, nil)
	if res.Code != http.StatusOK || res.JSON["menuVersion"] != float64(1) || len(res.JSON["categories"].([]any)) == 0 {
		t.Fatalf("menu = %d %s", res.Code, res.Body)
	}
	etag := res.Header().Get("ETag")
	if etag != `"m1"` {
		t.Errorf("ETag = %q", etag)
	}

	it := findItem(t, res.JSON, "Veg Dum Biryani")
	if paise(it["price"]) != 18000 || paise(it["packaging"]) != 1000 || it["vegType"] != "VEG" || it["isAvailable"] != true {
		t.Errorf("item = %v", it)
	}
	variants := it["variants"].([]any)
	if len(variants) != 2 || variants[0].(map[string]any)["name"] != "Half" || paise(variants[0].(map[string]any)["price"]) != 12000 ||
		variants[1].(map[string]any)["isDefault"] != true {
		t.Errorf("variants = %v", variants)
	}
	groups := it["addonGroups"].([]any)
	if len(groups) != 2 || len(groups[0].(map[string]any)["addons"].([]any)) == 0 {
		t.Errorf("addon groups = %v", groups)
	}
	if it["nameI18n"].(map[string]any)["te"] == "" {
		t.Error("Telugu item name missing")
	}

	// Conditional request.
	if res := e.c.DoWithHeader(http.MethodGet, path, "If-None-Match", etag); res.Code != http.StatusNotModified || res.Body.Len() != 0 {
		t.Errorf("If-None-Match = %d with %d body bytes, want 304 and none", res.Code, res.Body.Len())
	}
	if res := e.c.DoWithHeader(http.MethodGet, path, "If-None-Match", `"m0"`); res.Code != http.StatusOK {
		t.Errorf("stale If-None-Match = %d, want 200", res.Code)
	}

	// Availability windows and archiving.
	e.exec(`UPDATE menu_items SET is_available = false, unavailable_until = now() + interval '2 hours' WHERE name = 'Veg Dum Biryani'`)
	it = findItem(t, e.c.Do(http.MethodGet, path, nil).JSON, "Veg Dum Biryani")
	if it["isAvailable"] != false || it["unavailableUntil"] == nil {
		t.Errorf("switched-off item = available %v until %v", it["isAvailable"], it["unavailableUntil"])
	}
	e.exec(`UPDATE menu_items SET unavailable_until = now() - interval '1 minute' WHERE name = 'Veg Dum Biryani'`)
	it = findItem(t, e.c.Do(http.MethodGet, path, nil).JSON, "Veg Dum Biryani")
	if it["isAvailable"] != true || it["unavailableUntil"] != nil {
		t.Errorf("item past its unavailable-until = available %v until %v", it["isAvailable"], it["unavailableUntil"])
	}
	e.exec(`UPDATE menu_items SET archived_at = now() WHERE name = 'Veg Dum Biryani'`)
	if body := e.c.Do(http.MethodGet, path, nil).Body.String(); strings.Contains(body, "Veg Dum Biryani") {
		t.Error("archived item still on the menu")
	}
}

func TestSearch(t *testing.T) {
	e := setup(t)
	search := func(q string) *apptest.Response {
		return e.c.Do(http.MethodGet, "/api/v1/public/search?"+centre+"&q="+q, nil)
	}
	res := search("biryani")
	if res.Code != http.StatusOK {
		t.Fatalf("search = %d %s", res.Code, res.Body)
	}
	rs := res.JSON["restaurants"].([]any)
	if len(rs) != 1 || rs[0].(map[string]any)["name"] != biryani {
		t.Errorf("restaurants for 'biryani' = %v", rs)
	}
	dishes := res.JSON["dishes"].([]any)
	if len(dishes) == 0 {
		t.Fatal("no dishes for 'biryani'")
	}
	for _, d := range dishes {
		d := d.(map[string]any)
		if d["restaurant"].(map[string]any)["deliveryFee"] == nil || d["item"].(map[string]any)["price"] == nil {
			t.Errorf("dish hit incomplete: %v", d)
		}
	}
	// Case-insensitive, and by cuisine name.
	if res := search("TIFFIN"); len(res.JSON["restaurants"].([]any)) != 1 {
		t.Errorf("'TIFFIN' restaurants = %v", res.JSON["restaurants"])
	}
	// Dishes of a restaurant that does not deliver to the pin are left out.
	if res := search("jonna"); len(res.JSON["dishes"].([]any)) != 0 {
		t.Errorf("'jonna' from the centre = %v, want none (only the too-far restaurant sells it)", res.JSON["dishes"])
	}
	// Wildcards are literal.
	if res := search("%25"); len(res.JSON["dishes"].([]any)) != 0 || len(res.JSON["restaurants"].([]any)) != 0 {
		t.Errorf("'%%' matched: %v", res.JSON)
	}
	if res := e.c.Do(http.MethodGet, "/api/v1/public/search?"+centre, nil); res.Code != http.StatusBadRequest {
		t.Errorf("missing q = %d, want 400", res.Code)
	}
	if res := e.c.Do(http.MethodGet, "/api/v1/public/search?q=biryani&lat=17.3850&lng=78.4867", nil); res.Code != http.StatusUnprocessableEntity {
		t.Errorf("search outside the service area = %d, want 422", res.Code)
	}
}

func TestCuisines(t *testing.T) {
	e := setup(t)
	var cityID string
	if err := e.DB.Pool.QueryRow(context.Background(), `SELECT id::text FROM cities`).Scan(&cityID); err != nil {
		t.Fatal(err)
	}
	count := func(path string) int {
		t.Helper()
		res := e.c.Do(http.MethodGet, path, nil)
		var out []map[string]any
		if err := json.Unmarshal(res.Body.Bytes(), &out); err != nil || res.Code != http.StatusOK {
			t.Fatalf("%s = %d %s", path, res.Code, res.Body)
		}
		return len(out)
	}
	all, inCity := count("/api/v1/public/cuisines"), count("/api/v1/public/cuisines?cityId="+cityID)
	// The four demo restaurants use 9 distinct cuisines between them.
	if inCity != 9 || all < inCity {
		t.Errorf("cuisines: all=%d in city=%d, want 9 in city and all ≥ that", all, inCity)
	}
}
