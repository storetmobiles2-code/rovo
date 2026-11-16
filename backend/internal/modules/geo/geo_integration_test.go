//go:build integration

package geo_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/storetmobiles2-code/rovo/backend/internal/app/apptest"
)

const centre = "lat=16.7488&lng=78.0035"

func exec(t *testing.T, env *apptest.Env, sql string, args ...any) {
	t.Helper()
	if _, err := env.DB.Pool.Exec(context.Background(), sql, args...); err != nil {
		t.Fatalf("exec %q: %v", sql, err)
	}
}

func TestServiceability(t *testing.T) {
	env := apptest.New(t)
	seeded := env.Seed()
	c := env.Client(apptest.CustomerHost)
	get := func(q string) *apptest.Response { return c.Do(http.MethodGet, "/api/v1/public/serviceability?"+q, nil) }

	res := get(centre)
	if res.Code != http.StatusOK || res.JSON["status"] != "SERVICEABLE" ||
		res.JSON["zoneId"] != seeded.ZoneID.String() || res.JSON["cityId"] != seeded.CityID.String() {
		t.Fatalf("centre = %d %v", res.Code, res.JSON)
	}
	if guess, _ := res.JSON["localityGuess"].(map[string]any); guess == nil || guess["name"] == "" {
		t.Errorf("no locality guess at the city centre: %v", res.JSON)
	}

	// Hyderabad is ~100 km away: outside every zone.
	if res := get("lat=17.3850&lng=78.4867"); res.JSON["status"] != "OUTSIDE_SERVICE_AREA" || res.JSON["zoneId"] != nil {
		t.Errorf("outside = %v", res.JSON)
	}
	// Just outside the octagon's east vertex (78.0505) versus just inside it.
	if res := get("lat=16.7488&lng=78.0510"); res.JSON["status"] != "OUTSIDE_SERVICE_AREA" {
		t.Errorf("just outside boundary = %v", res.JSON["status"])
	}
	if res := get("lat=16.7488&lng=78.0500"); res.JSON["status"] != "SERVICEABLE" {
		t.Errorf("just inside boundary = %v", res.JSON["status"])
	}

	for _, q := range []string{"", "lat=16.7", "lat=abc&lng=78", "lat=91&lng=78", "lat=0&lng=0", "lat=16.7&lng=181"} {
		if res := get(q); res.Code != http.StatusBadRequest {
			t.Errorf("query %q = %d, want 400", q, res.Code)
		}
	}

	exec(t, env, `UPDATE zones SET paused_until = now() + interval '1 hour', pause_reason = 'RAIN',
		pause_message_i18n = '{"en":"Paused for rain.","te":"వర్షం"}'`)
	if res := get(centre); res.JSON["status"] != "ZONE_PAUSED" || res.JSON["message"] != "Paused for rain." {
		t.Errorf("paused zone = %v", res.JSON)
	}
	// A pause that has already ended does not block.
	exec(t, env, `UPDATE zones SET paused_until = now() - interval '1 minute'`)
	if res := get(centre); res.JSON["status"] != "SERVICEABLE" {
		t.Errorf("expired pause = %v", res.JSON["status"])
	}

	exec(t, env, `UPDATE cities SET status = 'PAUSED'`)
	if res := get(centre); res.JSON["status"] != "CITY_NOT_LIVE" {
		t.Errorf("city not live = %v", res.JSON["status"])
	}
	exec(t, env, `UPDATE cities SET status = 'LIVE'`)
	exec(t, env, `UPDATE zones SET status = 'INACTIVE'`)
	if res := get(centre); res.JSON["status"] != "OUTSIDE_SERVICE_AREA" {
		t.Errorf("inactive zone = %v", res.JSON["status"])
	}
}

func TestCitiesAndLocalities(t *testing.T) {
	env := apptest.New(t)
	seeded := env.Seed()
	c := env.Client(apptest.CustomerHost)

	res := c.Do(http.MethodGet, "/api/v1/public/cities", nil)
	var cities []map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &cities); err != nil || len(cities) != 1 {
		t.Fatalf("cities = %d %s (err %v)", res.Code, res.Body, err)
	}
	if cities[0]["code"] != "MBNR" || cities[0]["status"] != "LIVE" || cities[0]["timezone"] != "Asia/Kolkata" {
		t.Errorf("city = %v", cities[0])
	}

	list := func(q string) []map[string]any {
		t.Helper()
		res := c.Do(http.MethodGet, "/api/v1/public/localities?cityId="+seeded.CityID.String()+q, nil)
		var out []map[string]any
		if err := json.Unmarshal(res.Body.Bytes(), &out); err != nil || res.Code != http.StatusOK {
			t.Fatalf("localities%s = %d %s (err %v)", q, res.Code, res.Body, err)
		}
		return out
	}
	if all := list(""); len(all) != 12 {
		t.Errorf("all localities = %d, want 12", len(all))
	}
	if got := list("&q=new"); len(got) != 1 || got[0]["name"] != "New Town" {
		t.Errorf("q=new = %v", got)
	}
	// Alias match ("patha town" is an alias of Old Town).
	if got := list("&q=patha"); len(got) != 1 || got[0]["name"] != "Old Town" {
		t.Errorf("q=patha = %v", got)
	}
	// LIKE wildcards in the query are literal.
	if got := list("&q=%25"); len(got) != 0 {
		t.Errorf("q=%% matched %d localities, want 0", len(got))
	}
	if res := c.Do(http.MethodGet, "/api/v1/public/localities?cityId=nope", nil); res.Code != http.StatusBadRequest {
		t.Errorf("bad cityId = %d, want 400", res.Code)
	}
}
