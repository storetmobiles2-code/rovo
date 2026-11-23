// Package geo owns cities, localities, zone polygons (PostGIS), serviceability (zone ∩ 7 km
// straight-line, road distance ×1.3) and the fee slab lookup (charter §3, docs 16/18).
package geo

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/storetmobiles2-code/rovo/backend/internal/platform/app"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/apperr"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/db"
	pgeo "github.com/storetmobiles2-code/rovo/backend/internal/platform/geo"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/queue"
)

// Service is the geo module's public API (charter §5).
type Service interface {
	// Resolve maps a drop pin to its zone, delivery status and the fee config in force. A pin
	// outside every zone is a normal result (StatusOutsideService), not an error.
	Resolve(ctx context.Context, p pgeo.Point) (Coverage, error)
	// Serviceability is Resolve reduced to the summary other modules usually need.
	Serviceability(ctx context.Context, lat, lng float64) (Serviceability, error)
	// Cities lists cities that are live or announced.
	Cities(ctx context.Context) ([]City, error)
	// Localities searches a city's active localities by name, alias or pin code.
	Localities(ctx context.Context, cityID uuid.UUID, q string) ([]Locality, error)
	// PausedZones returns the ids of zones in a city that are paused right now.
	PausedZones(ctx context.Context, cityID uuid.UUID) (map[uuid.UUID]bool, error)
}

// ServiceStatus is the serviceability outcome.
type ServiceStatus string

// Serviceability statuses (doc 16 §3.2, schema Serviceability).
const (
	StatusServiceable    ServiceStatus = "SERVICEABLE"
	StatusOutsideService ServiceStatus = "OUTSIDE_SERVICE_AREA"
	StatusCityNotLive    ServiceStatus = "CITY_NOT_LIVE"
	StatusZonePaused     ServiceStatus = "ZONE_PAUSED"
	StatusTooFar         ServiceStatus = "TOO_FAR"
)

// Serviceability is the resolved coverage summary for a point.
type Serviceability struct {
	Status ServiceStatus
	CityID uuid.UUID
	ZoneID uuid.UUID
}

// Coverage is everything known about a drop pin: its zone, whether orders can be delivered
// there, and the pricing parameters that apply.
type Coverage struct {
	Status   ServiceStatus
	Point    pgeo.Point
	CityID   uuid.UUID
	ZoneID   uuid.UUID
	Timezone *time.Location
	// PauseMessage is the ops message for a paused zone, keyed by language.
	PauseMessage map[string]string
	// Fees is nil when the pin is outside every zone or no approved fee config is in force.
	Fees          *FeeConfig
	LocalityGuess *LocalityRef
}

// InZone reports whether the pin resolved to a zone of a live city, paused or not. Browsing is
// allowed in a paused zone; ordering is not (doc 16 §3.1).
func (c Coverage) InZone() bool {
	return c.Status == StatusServiceable || c.Status == StatusZonePaused
}

// LocalityRef names the locality nearest a pin.
type LocalityRef struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

// Slab is one delivery-fee band on road metres, [FromM, ToM).
type Slab struct {
	FromM    int   `json:"fromM"`
	ToM      int   `json:"toM"`
	FeePaise int64 `json:"feePaise"`
}

// SpeedBand is the assumed rider speed for a local time-of-day window. A band whose To is not
// after its From wraps past midnight.
type SpeedBand struct {
	From string  `json:"from"`
	To   string  `json:"to"`
	Kmph float64 `json:"kmph"`
}

// ETAParams are the ETA model inputs (doc 16 §5).
type ETAParams struct {
	PickupBufferMin float64     `json:"pickupBufferMin"`
	HandoverMin     float64     `json:"handoverMin"`
	Bands           []SpeedBand `json:"speedKmphByBand"`
}

// FeeConfig is the approved pricing configuration in force for a zone.
type FeeConfig struct {
	ID                    uuid.UUID
	Slabs                 []Slab
	RoadFactorMilli       int
	MaxServiceableRadiusM int
	ETA                   ETAParams
}

// DeliveryFee returns the fee for a road distance, or false when no slab covers it.
func (f *FeeConfig) DeliveryFee(roadM int) (int64, bool) {
	for _, s := range f.Slabs {
		if roadM >= s.FromM && roadM < s.ToM {
			return s.FeePaise, true
		}
	}
	return 0, false
}

// defaultSpeedKmph is used when no band matches (a config gap), so an ETA is still shown.
const defaultSpeedKmph = 20

func minuteOfDay(hhmm string) (int, bool) {
	t, err := time.Parse("15:04", hhmm)
	if err != nil {
		return 0, false
	}
	return t.Hour()*60 + t.Minute(), true
}

func (e ETAParams) speedAt(local time.Time) float64 {
	m := local.Hour()*60 + local.Minute()
	for _, b := range e.Bands {
		from, ok1 := minuteOfDay(b.From)
		to, ok2 := minuteOfDay(b.To)
		if !ok1 || !ok2 || b.Kmph <= 0 {
			continue
		}
		if (from < to && m >= from && m < to) || (from >= to && (m >= from || m < to)) {
			return b.Kmph
		}
	}
	return defaultSpeedKmph
}

// ETARange returns the displayed delivery window in minutes: prep + pickup buffer + travel +
// handover, as a 10-minute range on 5-minute steps (doc 16 §5). local is the time in the city.
func (f *FeeConfig) ETARange(prepMin, roadM int, local time.Time) (minMinutes, maxMinutes int) {
	travel := float64(roadM) / 1000 / f.ETA.speedAt(local) * 60
	mid := float64(prepMin) + f.ETA.PickupBufferMin + travel + f.ETA.HandoverMin
	lo := int(math.Floor((mid-2.5)/5+0.5)) * 5
	if lo < 5 {
		lo = 5
	}
	return lo, lo + 10
}

// City is a public city record (schema City).
type City struct {
	ID       uuid.UUID         `json:"id"`
	Code     string            `json:"code"`
	Name     string            `json:"name"`
	NameI18n map[string]string `json:"nameI18n"`
	Status   string            `json:"status"`
	Centroid pgeo.Point        `json:"centroid"`
	Timezone string            `json:"timezone"`
}

// Locality is a public locality record (schema Locality).
type Locality struct {
	ID       uuid.UUID         `json:"id"`
	Name     string            `json:"name"`
	NameI18n map[string]string `json:"nameI18n"`
	PinCodes []string          `json:"pinCodes"`
	Centroid pgeo.Point        `json:"centroid"`
}

type service struct{ deps *app.Deps }

// New constructs the geo module.
func New(deps *app.Deps) *app.Module {
	s := &service{deps: deps}
	return &app.Module{Name: app.ServiceGeo, Service: Service(s), Routes: s.registerRoutes, Jobs: s.registerJobs}
}

// localityGuessRadiusM is how near a locality centroid must be to pre-fill the address form.
const localityGuessRadiusM = 2000

func (s *service) Resolve(ctx context.Context, p pgeo.Point) (Coverage, error) {
	if !p.Valid() {
		return Coverage{}, apperr.BadRequest(apperr.CodeInvalidLocation, "Location is not valid.")
	}
	now := s.deps.Clock.Now()
	cov := Coverage{Status: StatusOutsideService, Point: p, Timezone: time.UTC}

	var (
		cityStatus, tz string
		pausedUntil    *time.Time
		pauseMsg       []byte
	)
	err := s.deps.Pool.QueryRow(ctx, `SELECT z.id, z.city_id, c.status, c.timezone, z.paused_until, z.pause_message_i18n
		FROM zones z JOIN cities c ON c.id = z.city_id
		WHERE z.status = 'ACTIVE' AND ST_Covers(z.boundary, ST_SetSRID(ST_MakePoint($1, $2), 4326))
		ORDER BY z.priority DESC, ST_Area(z.boundary) ASC, z.id LIMIT 1`, p.Lng, p.Lat).
		Scan(&cov.ZoneID, &cov.CityID, &cityStatus, &tz, &pausedUntil, &pauseMsg)
	if db.IsNoRows(err) {
		return cov, nil
	}
	if err != nil {
		return Coverage{}, err
	}
	if loc, err := time.LoadLocation(tz); err == nil {
		cov.Timezone = loc
	}
	switch {
	case cityStatus != "LIVE":
		cov.Status = StatusCityNotLive
		return cov, nil
	case pausedUntil != nil && pausedUntil.After(now):
		cov.Status = StatusZonePaused
		_ = json.Unmarshal(pauseMsg, &cov.PauseMessage)
	default:
		cov.Status = StatusServiceable
	}

	fees, err := s.feeConfig(ctx, cov.CityID, cov.ZoneID, now)
	if err != nil {
		return Coverage{}, err
	}
	cov.Fees = fees

	var guess LocalityRef
	err = s.deps.Pool.QueryRow(ctx, `SELECT id, name FROM localities
		WHERE city_id = $1 AND is_active AND ST_DWithin(centroid, ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography, $4)
		ORDER BY centroid <-> ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography LIMIT 1`,
		cov.CityID, p.Lng, p.Lat, localityGuessRadiusM).Scan(&guess.ID, &guess.Name)
	switch {
	case err == nil:
		cov.LocalityGuess = &guess
	case !db.IsNoRows(err):
		return Coverage{}, err
	}
	return cov, nil
}

// feeConfig returns the approved config in force: the zone's own if it has one, else the city
// default (zone_id NULL). It returns nil when none is in force.
func (s *service) feeConfig(ctx context.Context, cityID, zoneID uuid.UUID, now time.Time) (*FeeConfig, error) {
	var (
		f          FeeConfig
		slabs, eta []byte
	)
	err := s.deps.Pool.QueryRow(ctx, `SELECT id, delivery_fee_slabs, road_factor_milli, max_serviceable_radius_m, eta_params
		FROM fee_configs
		WHERE city_id = $1 AND (zone_id = $2 OR zone_id IS NULL) AND status = 'APPROVED'
		  AND effective_from <= $3 AND (effective_to IS NULL OR effective_to > $3)
		ORDER BY (zone_id IS NULL), effective_from DESC LIMIT 1`, cityID, zoneID, now).
		Scan(&f.ID, &slabs, &f.RoadFactorMilli, &f.MaxServiceableRadiusM, &eta)
	if db.IsNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(slabs, &f.Slabs); err != nil {
		return nil, fmt.Errorf("geo: fee config %s slabs: %w", f.ID, err)
	}
	if err := json.Unmarshal(eta, &f.ETA); err != nil {
		return nil, fmt.Errorf("geo: fee config %s eta params: %w", f.ID, err)
	}
	return &f, nil
}

func (s *service) Serviceability(ctx context.Context, lat, lng float64) (Serviceability, error) {
	cov, err := s.Resolve(ctx, pgeo.Point{Lat: lat, Lng: lng})
	if err != nil {
		return Serviceability{}, err
	}
	return Serviceability{Status: cov.Status, CityID: cov.CityID, ZoneID: cov.ZoneID}, nil
}

func (s *service) PausedZones(ctx context.Context, cityID uuid.UUID) (map[uuid.UUID]bool, error) {
	rows, err := s.deps.Pool.Query(ctx, `SELECT id FROM zones WHERE city_id = $1 AND paused_until > $2`,
		cityID, s.deps.Clock.Now())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[uuid.UUID]bool{}
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[id] = true
	}
	return out, rows.Err()
}

func (s *service) Cities(ctx context.Context) ([]City, error) {
	rows, err := s.deps.Pool.Query(ctx, `SELECT id, code, name, name_i18n, status, timezone,
		ST_Y(centroid::geometry), ST_X(centroid::geometry)
		FROM cities WHERE status IN ('LIVE','PAUSED','PLANNED') ORDER BY (status = 'LIVE') DESC, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []City{}
	for rows.Next() {
		var c City
		var i18n []byte
		if err := rows.Scan(&c.ID, &c.Code, &c.Name, &i18n, &c.Status, &c.Timezone, &c.Centroid.Lat, &c.Centroid.Lng); err != nil {
			return nil, err
		}
		c.NameI18n = map[string]string{}
		_ = json.Unmarshal(i18n, &c.NameI18n)
		out = append(out, c)
	}
	return out, rows.Err()
}

// maxLocalities caps a locality listing; a V1 city has a few dozen.
const maxLocalities = 50

func (s *service) Localities(ctx context.Context, cityID uuid.UUID, q string) ([]Locality, error) {
	q = strings.ToLower(strings.TrimSpace(q))
	if len(q) > 60 {
		q = q[:60]
	}
	// LIKE wildcards in user input are escaped so "%" cannot widen the match.
	like := "%" + strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(q) + "%"
	rows, err := s.deps.Pool.Query(ctx, `SELECT id, name, name_i18n, pin_codes, ST_Y(centroid::geometry), ST_X(centroid::geometry)
		FROM localities l
		WHERE is_active AND ($1::uuid IS NULL OR city_id = $1)
		  AND ($2 = '' OR lower(name) LIKE $3 OR lower(name_i18n::text) LIKE $3 OR $2 = ANY(pin_codes)
		       OR EXISTS (SELECT 1 FROM unnest(search_aliases) a WHERE lower(a) LIKE $3))
		ORDER BY sort_order, name LIMIT $4`, nullUUID(cityID), q, like, maxLocalities)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Locality{}
	for rows.Next() {
		var l Locality
		var i18n []byte
		if err := rows.Scan(&l.ID, &l.Name, &i18n, &l.PinCodes, &l.Centroid.Lat, &l.Centroid.Lng); err != nil {
			return nil, err
		}
		l.NameI18n = map[string]string{}
		_ = json.Unmarshal(i18n, &l.NameI18n)
		if l.PinCodes == nil {
			l.PinCodes = []string{}
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func nullUUID(id uuid.UUID) *uuid.UUID {
	if id == uuid.Nil {
		return nil
	}
	return &id
}

func (s *service) registerJobs(_ *queue.Registry, _ *app.Deps) {}
