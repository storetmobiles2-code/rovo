package geo

import (
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"github.com/storetmobiles2-code/rovo/backend/internal/platform/app"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/apperr"
	pgeo "github.com/storetmobiles2-code/rovo/backend/internal/platform/geo"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/httpx"
)

func (s *service) registerRoutes(mux *httpx.Mux, _ *app.Deps) {
	mux.Handle("GET /api/v1/public/cities", s.handleCities)
	mux.Handle("GET /api/v1/public/serviceability", s.handleServiceability)
	mux.Handle("GET /api/v1/public/localities", s.handleLocalities)
}

// PointFromQuery reads the required lat and lng query parameters. Other modules use it so every
// pin-taking endpoint validates the same way.
func PointFromQuery(r *http.Request) (pgeo.Point, error) {
	q := r.URL.Query()
	var fields []apperr.FieldError
	parse := func(name string) float64 {
		v, err := strconv.ParseFloat(q.Get(name), 64)
		if q.Get(name) == "" {
			fields = append(fields, apperr.FieldError{Field: name, Code: "REQUIRED", Detail: "is required"})
		} else if err != nil {
			fields = append(fields, apperr.FieldError{Field: name, Code: "INVALID_NUMBER", Detail: "must be a number"})
		}
		return v
	}
	p := pgeo.Point{Lat: parse("lat"), Lng: parse("lng")}
	if len(fields) > 0 {
		return pgeo.Point{}, apperr.Validation(fields...)
	}
	if !p.Valid() {
		return pgeo.Point{}, apperr.BadRequest(apperr.CodeInvalidLocation, "Location is not valid.")
	}
	return p, nil
}

// ServiceabilityView is schema Serviceability.
type ServiceabilityView struct {
	Status        ServiceStatus `json:"status"`
	CityID        *uuid.UUID    `json:"cityId"`
	ZoneID        *uuid.UUID    `json:"zoneId"`
	Message       *string       `json:"message"`
	LocalityGuess *LocalityRef  `json:"localityGuess"`
}

// View renders the coverage for the API. locale picks the paused-zone message language, falling
// back to English.
func (c Coverage) View(locale string) ServiceabilityView {
	v := ServiceabilityView{Status: c.Status, LocalityGuess: c.LocalityGuess}
	if c.CityID != uuid.Nil {
		v.CityID = &c.CityID
	}
	if c.ZoneID != uuid.Nil {
		v.ZoneID = &c.ZoneID
	}
	if c.Status == StatusZonePaused {
		for _, k := range []string{locale, "en"} {
			if m := c.PauseMessage[k]; m != "" {
				v.Message = &m
				break
			}
		}
	}
	return v
}

// Locale returns the request's UI language: "te" when Accept-Language asks for Telugu first,
// otherwise "en".
func Locale(r *http.Request) string {
	al := r.Header.Get("Accept-Language")
	if len(al) >= 2 && (al[:2] == "te" || al[:2] == "TE") {
		return "te"
	}
	return "en"
}

func (s *service) handleCities(w http.ResponseWriter, r *http.Request) {
	cities, err := s.Cities(r.Context())
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=300")
	httpx.JSON(w, http.StatusOK, cities)
}

func (s *service) handleServiceability(w http.ResponseWriter, r *http.Request) {
	p, err := PointFromQuery(r)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	cov, err := s.Resolve(r.Context(), p)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, cov.View(Locale(r)))
}

func (s *service) handleLocalities(w http.ResponseWriter, r *http.Request) {
	var cityID uuid.UUID
	if v := r.URL.Query().Get("cityId"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			httpx.Error(w, r, apperr.Validation(apperr.FieldError{Field: "cityId", Code: "INVALID_UUID", Detail: "must be a UUID"}))
			return
		}
		cityID = id
	}
	out, err := s.Localities(r.Context(), cityID, r.URL.Query().Get("q"))
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, out)
}
