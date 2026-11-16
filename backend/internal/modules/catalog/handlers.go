package catalog

import (
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/storetmobiles2-code/rovo/backend/internal/modules/geo"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/app"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/apperr"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/auth"
	pgeo "github.com/storetmobiles2-code/rovo/backend/internal/platform/geo"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/httpx"
)

func (s *service) registerRoutes(mux *httpx.Mux, _ *app.Deps) {
	mux.Handle("GET /api/v1/public/cuisines", s.handleCuisines)
	mux.Handle("GET /api/v1/public/restaurants", s.handleDiscover)
	mux.Handle("GET /api/v1/public/restaurants/{restaurantId}", s.handleRestaurant)
	mux.Handle("GET /api/v1/public/restaurants/{restaurantId}/menu", s.handleMenu)
	mux.Handle("GET /api/v1/public/search", s.handleSearch)
}

const (
	defaultPageSize = 20
	maxPageSize     = 100
)

// Cursors are opaque offsets signed with a derived key, so a client cannot craft one.
func (s *service) encodeCursor(offset int) string {
	body := strconv.Itoa(offset)
	return base64.RawURLEncoding.EncodeToString([]byte(body)) + "." + hex.EncodeToString(auth.MAC(s.cursorKey, []byte(body))[:8])
}

func (s *service) decodeCursor(cursor string) (int, error) {
	bad := apperr.BadRequest(apperr.CodeInvalidCursor, "Cursor is not valid.")
	b64, sig, ok := strings.Cut(cursor, ".")
	if !ok {
		return 0, bad
	}
	body, err := base64.RawURLEncoding.DecodeString(b64)
	if err != nil {
		return 0, bad
	}
	got, err := hex.DecodeString(sig)
	if err != nil || !auth.EqualMAC(got, auth.MAC(s.cursorKey, body)[:8]) {
		return 0, bad
	}
	n, err := strconv.Atoi(string(body))
	if err != nil || n < 0 {
		return 0, bad
	}
	return n, nil
}

func fieldErr(field, code, detail string) error {
	return apperr.Validation(apperr.FieldError{Field: field, Code: code, Detail: detail})
}

var sortOrders = map[string]bool{"": true, "relevance": true, "rating": true, "eta": true, "costLow": true, "costHigh": true}

func (s *service) parseDiscover(r *http.Request) (DiscoverQuery, error) {
	pin, err := geo.PointFromQuery(r)
	if err != nil {
		return DiscoverQuery{}, err
	}
	v := r.URL.Query()
	q := DiscoverQuery{Pin: pin, Cuisine: strings.TrimSpace(v.Get("cuisine")), Sort: v.Get("sort"), Limit: defaultPageSize, Locale: geo.Locale(r)}
	if !sortOrders[q.Sort] {
		return q, fieldErr("sort", "INVALID_ENUM", "must be relevance, rating, eta, costLow or costHigh")
	}
	for name, dst := range map[string]*bool{"veg": &q.VegOnly, "hasOffer": &q.HasOffer} {
		if raw := v.Get(name); raw != "" {
			b, err := strconv.ParseBool(raw)
			if err != nil {
				return q, fieldErr(name, "INVALID_BOOLEAN", "must be true or false")
			}
			*dst = b
		}
	}
	if raw := v.Get("minRating"); raw != "" {
		f, err := strconv.ParseFloat(raw, 64)
		if err != nil || f < 0 || f > 5 {
			return q, fieldErr("minRating", "OUT_OF_RANGE", "must be between 0 and 5")
		}
		q.MinRating = f
	}
	if raw := v.Get("maxEtaMin"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			return q, fieldErr("maxEtaMin", "OUT_OF_RANGE", "must be a positive integer")
		}
		q.MaxETAMin = n
	}
	if raw := v.Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > maxPageSize {
			return q, fieldErr("limit", "OUT_OF_RANGE", "must be between 1 and 100")
		}
		q.Limit = n
	}
	if raw := v.Get("cursor"); raw != "" {
		if q.Offset, err = s.decodeCursor(raw); err != nil {
			return q, err
		}
	}
	return q, nil
}

type discoverResponse struct {
	Items      []RestaurantCard `json:"items"`
	NextCursor *string          `json:"nextCursor"`
	Zone       *ZoneNotice      `json:"zone"`
}

func (s *service) handleDiscover(w http.ResponseWriter, r *http.Request) {
	q, err := s.parseDiscover(r)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	res, err := s.Discover(r.Context(), q)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	out := discoverResponse{Items: res.Items, Zone: res.Zone}
	if res.NextOffset >= 0 {
		c := s.encodeCursor(res.NextOffset)
		out.NextCursor = &c
	}
	httpx.JSON(w, http.StatusOK, out)
}

func restaurantID(r *http.Request) (uuid.UUID, error) {
	id, err := uuid.Parse(r.PathValue("restaurantId"))
	if err != nil {
		// A malformed id cannot name a restaurant.
		return uuid.Nil, apperr.NotFound("Restaurant not found.")
	}
	return id, nil
}

func (s *service) handleRestaurant(w http.ResponseWriter, r *http.Request) {
	id, err := restaurantID(r)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	// The pin is optional here so a shared link opens before the customer has set a location.
	var pin *pgeo.Point
	if q := r.URL.Query(); q.Has("lat") || q.Has("lng") {
		p, err := geo.PointFromQuery(r)
		if err != nil {
			httpx.Error(w, r, err)
			return
		}
		pin = &p
	}
	d, err := s.Restaurant(r.Context(), id, pin)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, d)
}

func (s *service) handleMenu(w http.ResponseWriter, r *http.Request) {
	id, err := restaurantID(r)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	m, err := s.Menu(r.Context(), id)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	// The ETag tracks the menu version. Availability can change without a version bump, so the
	// cache lifetime is short and a matching ETag is only honoured for that long.
	etag := `"m` + strconv.Itoa(m.MenuVersion) + `"`
	h := w.Header()
	h.Set("ETag", etag)
	h.Set("Cache-Control", "public, max-age=60")
	if match := r.Header.Get("If-None-Match"); match != "" && strings.Contains(match, etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	httpx.JSON(w, http.StatusOK, m)
}

func (s *service) handleCuisines(w http.ResponseWriter, r *http.Request) {
	var cityID uuid.UUID
	if v := r.URL.Query().Get("cityId"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			httpx.Error(w, r, fieldErr("cityId", "INVALID_UUID", "must be a UUID"))
			return
		}
		cityID = id
	}
	out, err := s.Cuisines(r.Context(), cityID)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=300")
	httpx.JSON(w, http.StatusOK, out)
}

func (s *service) handleSearch(w http.ResponseWriter, r *http.Request) {
	pin, err := geo.PointFromQuery(r)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	res, err := s.Search(r.Context(), r.URL.Query().Get("q"), pin)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}
