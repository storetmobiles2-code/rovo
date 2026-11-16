// Package users owns the member profile, saved addresses, consents and the /me surface
// (charter §3, doc 12).
package users

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/storetmobiles2-code/rovo/backend/internal/platform/app"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/apperr"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/db"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/httpx"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/queue"
)

// Service is the users module's public API (charter §5).
type Service interface {
	// Me returns the user's profile and live role grants.
	Me(ctx context.Context, userID uuid.UUID) (Me, error)
	// UpdateMe changes the profile fields the member may edit and returns the new profile.
	UpdateMe(ctx context.Context, userID uuid.UUID, patch MePatch) (Me, error)
	// ListAddresses returns a member's saved delivery addresses.
	ListAddresses(ctx context.Context, userID uuid.UUID) ([]Address, error)
}

// Me is the /me response shape (doc 11 §2.2, schema Me). It never carries the full phone number.
type Me struct {
	ID              uuid.UUID   `json:"id"`
	Kind            string      `json:"kind"`
	FullName        *string     `json:"fullName"`
	PhoneMasked     *string     `json:"phoneMasked"`
	Email           *string     `json:"email"`
	PreferredLocale string      `json:"preferredLocale"`
	Roles           []RoleGrant `json:"roles"`
	ActiveContext   *string     `json:"activeContext"`
}

// RoleGrant is one live row of user_roles.
type RoleGrant struct {
	Role         string     `json:"role"`
	CityID       *uuid.UUID `json:"cityId"`
	RestaurantID *uuid.UUID `json:"restaurantId"`
}

// MePatch is a partial profile update. A nil field is left unchanged; FullName set to a pointer
// to "" clears the name.
type MePatch struct {
	FullName        *string
	PreferredLocale *string
}

// Address is a saved delivery address.
type Address struct {
	ID             uuid.UUID
	HouseNo        string
	BuildingStreet string
	Landmark       string
	LocalityID     uuid.UUID
	PinCode        string
	Lat, Lng       float64
}

type service struct{ deps *app.Deps }

// New constructs the users module.
func New(deps *app.Deps) *app.Module {
	s := &service{deps: deps}
	return &app.Module{Name: app.ServiceUsers, Service: Service(s), Routes: s.registerRoutes, Jobs: s.registerJobs}
}

func (s *service) Me(ctx context.Context, userID uuid.UUID) (Me, error) {
	me := Me{ID: userID, Roles: []RoleGrant{}}
	var phone *string
	err := s.deps.Pool.QueryRow(ctx, `SELECT kind, full_name, phone_e164, email::text, preferred_locale
		FROM users WHERE id = $1 AND status NOT IN ('DETACHED','ANONYMIZED')`, userID).
		Scan(&me.Kind, &me.FullName, &phone, &me.Email, &me.PreferredLocale)
	if db.IsNoRows(err) {
		return Me{}, apperr.NotFound("User not found.")
	}
	if err != nil {
		return Me{}, err
	}
	if phone != nil {
		m := MaskPhone(*phone)
		me.PhoneMasked = &m
	}
	rows, err := s.deps.Pool.Query(ctx, `SELECT role, city_id, restaurant_id FROM user_roles
		WHERE user_id = $1 AND revoked_at IS NULL ORDER BY granted_at, id`, userID)
	if err != nil {
		return Me{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var g RoleGrant
		if err := rows.Scan(&g.Role, &g.CityID, &g.RestaurantID); err != nil {
			return Me{}, err
		}
		me.Roles = append(me.Roles, g)
	}
	return me, rows.Err()
}

var localeRe = regexp.MustCompile(`^[a-z]{2}(-[A-Z]{2})?$`)

// supportedLocales are the V1 UI languages (R: en + te).
var supportedLocales = map[string]bool{"en": true, "te": true}

func (s *service) UpdateMe(ctx context.Context, userID uuid.UUID, patch MePatch) (Me, error) {
	var fields []apperr.FieldError
	if patch.FullName != nil {
		n := strings.TrimSpace(*patch.FullName)
		if utf8.RuneCountInString(n) > 100 {
			fields = append(fields, apperr.FieldError{Field: "fullName", Code: "TOO_LONG", Detail: "at most 100 characters"})
		}
		patch.FullName = &n
	}
	if patch.PreferredLocale != nil {
		if !localeRe.MatchString(*patch.PreferredLocale) || !supportedLocales[*patch.PreferredLocale] {
			return Me{}, apperr.Unprocessable(apperr.CodeUnsupportedLocale, "This language is not supported.")
		}
	}
	if len(fields) > 0 {
		return Me{}, apperr.Validation(fields...)
	}
	// NULLIF keeps "clear the name" and "leave unchanged" distinct: $2 is whether to touch it.
	tag, err := s.deps.Pool.Exec(ctx, `UPDATE users SET
			full_name        = CASE WHEN $2 THEN NULLIF($3, '') ELSE full_name END,
			preferred_locale = COALESCE($4, preferred_locale)
		WHERE id = $1 AND status = 'ACTIVE'`,
		userID, patch.FullName != nil, deref(patch.FullName), patch.PreferredLocale)
	if err != nil {
		return Me{}, err
	}
	if tag.RowsAffected() == 0 {
		return Me{}, apperr.NotFound("User not found.")
	}
	return s.Me(ctx, userID)
}

func (s *service) ListAddresses(context.Context, uuid.UUID) ([]Address, error) {
	return nil, apperr.NotImplemented("users.ListAddresses")
}

// MaskPhone renders an E.164 number with only the country code and last two digits visible,
// e.g. +91••••••••12 (doc 12: no full phone in API responses).
func MaskPhone(e164 string) string {
	if len(e164) < 6 {
		return "••••"
	}
	return e164[:3] + strings.Repeat("•", len(e164)-5) + e164[len(e164)-2:]
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func (s *service) registerRoutes(mux *httpx.Mux, _ *app.Deps) {
	mux.Handle("GET /api/v1/me", httpx.RequireAuth(s.handleGetMe))
	mux.Handle("PATCH /api/v1/me", httpx.RequireAuth(s.handlePatchMe))
}

func (s *service) registerJobs(_ *queue.Registry, _ *app.Deps) {}

func (s *service) handleGetMe(w http.ResponseWriter, r *http.Request) {
	p := httpx.Principal(r)
	me, err := s.Me(r.Context(), p.UserID)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	if p.Context != "" {
		me.ActiveContext = &p.Context
	}
	httpx.JSON(w, http.StatusOK, me)
}

func (s *service) handlePatchMe(w http.ResponseWriter, r *http.Request) {
	p := httpx.Principal(r)
	// fullName is nullable in the contract, so presence and null must be told apart.
	raw, err := httpx.DecodeJSON[map[string]json.RawMessage](w, r, 4<<10)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	var patch MePatch
	for k, v := range raw {
		switch k {
		case "fullName":
			var n *string
			if err := json.Unmarshal(v, &n); err != nil {
				httpx.Error(w, r, apperr.Validation(apperr.FieldError{Field: k, Code: "INVALID_TYPE", Detail: "must be a string or null"}))
				return
			}
			if n == nil {
				n = new(string)
			}
			patch.FullName = n
		case "preferredLocale":
			var l string
			if err := json.Unmarshal(v, &l); err != nil {
				httpx.Error(w, r, apperr.Validation(apperr.FieldError{Field: k, Code: "INVALID_TYPE", Detail: "must be a string"}))
				return
			}
			patch.PreferredLocale = &l
		default:
			httpx.Error(w, r, apperr.Validation(apperr.FieldError{Field: k, Code: "UNKNOWN_FIELD", Detail: "unknown field"}))
			return
		}
	}
	me, err := s.UpdateMe(r.Context(), p.UserID, patch)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	if p.Context != "" {
		me.ActiveContext = &p.Context
	}
	httpx.JSON(w, http.StatusOK, me)
}
