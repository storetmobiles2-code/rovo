package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/storetmobiles2-code/rovo/backend/internal/platform/clock"
)

// Access-token lifetimes (12 §3.4).
const (
	AccessTTLDefault = 10 * time.Minute
	AccessTTLAdmin   = 5 * time.Minute
	maxTokenBytes    = 2048
	leeway           = 30 * time.Second
	tokenType        = "at+jwt"
)

// Token verification errors; middleware maps them to 401 codes.
var (
	ErrTokenExpired          = errors.New("auth: token expired")
	ErrTokenInvalid          = errors.New("auth: token invalid")
	ErrTokenAudienceMismatch = errors.New("auth: token audience mismatch")
)

// Claims are the access-token claims (12 §4.2). No PII.
type Claims struct {
	jwt.RegisteredClaims
	SessionID string   `json:"sid"`
	Context   string   `json:"ctx,omitempty"`
	Roles     []string `json:"roles"`
	Scopes    *Scopes  `json:"scp,omitempty"`
	AMR       []string `json:"amr,omitempty"`
}

// Scopes carries admin city scopes.
type Scopes struct {
	City []string `json:"city,omitempty"`
}

// TokenInput describes an access token to issue.
type TokenInput struct {
	UserID     uuid.UUID
	SessionID  uuid.UUID
	Audience   Audience
	Roles      []Role
	Context    string
	CityScopes []uuid.UUID
	AMR        []string
	TTL        time.Duration
}

// Issuer signs and verifies EdDSA access tokens.
type Issuer struct {
	keys   *Keyset
	issuer string
	clock  clock.Clock
	ids    func() uuid.UUID
}

// NewIssuer creates an issuer; iss is the per-deployment issuer string.
func NewIssuer(keys *Keyset, iss string, clk clock.Clock) *Issuer {
	return &Issuer{keys: keys, issuer: iss, clock: clk, ids: func() uuid.UUID { return uuid.Must(uuid.NewV7()) }}
}

// Issue signs a token and returns it with its expiry.
func (i *Issuer) Issue(in TokenInput) (string, time.Time, error) {
	ttl := in.TTL
	if ttl <= 0 {
		ttl = AccessTTLDefault
		if in.Audience == AudienceAdmin {
			ttl = AccessTTLAdmin
		}
	}
	now := i.clock.Now()
	exp := now.Add(ttl)
	roles := make([]string, len(in.Roles))
	for n, r := range in.Roles {
		roles[n] = string(r)
	}
	c := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    i.issuer,
			Subject:   in.UserID.String(),
			Audience:  jwt.ClaimStrings{string(in.Audience)},
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
			ID:        i.ids().String(),
		},
		SessionID: in.SessionID.String(),
		Context:   in.Context,
		Roles:     roles,
		AMR:       in.AMR,
	}
	if len(in.CityScopes) > 0 {
		c.Scopes = &Scopes{}
		for _, id := range in.CityScopes {
			c.Scopes.City = append(c.Scopes.City, id.String())
		}
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodEdDSA, c)
	tok.Header["kid"] = i.keys.activeKID
	tok.Header["typ"] = tokenType
	s, err := tok.SignedString(i.keys.active)
	if err != nil {
		return "", time.Time{}, err
	}
	return s, exp, nil
}

// Verify checks signature, kid, typ, iss, aud, exp/nbf (±30 s) and size, then returns the
// principal. aud must equal the audience of the host the request arrived on.
func (i *Issuer) Verify(token string, aud Audience) (*Principal, error) {
	if len(token) == 0 || len(token) > maxTokenBytes {
		return nil, ErrTokenInvalid
	}
	var c Claims
	parsed, err := jwt.ParseWithClaims(token, &c, func(t *jwt.Token) (any, error) {
		if typ, _ := t.Header["typ"].(string); typ != tokenType {
			return nil, ErrTokenInvalid
		}
		kid, _ := t.Header["kid"].(string)
		pub, ok := i.keys.verify[kid]
		if !ok {
			return nil, ErrTokenInvalid
		}
		return pub, nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodEdDSA.Alg()}),
		jwt.WithIssuer(i.issuer),
		jwt.WithLeeway(leeway),
		jwt.WithTimeFunc(i.clock.Now),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
	)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrTokenInvalid
	}
	if !parsed.Valid {
		return nil, ErrTokenInvalid
	}
	if len(c.Audience) != 1 || c.Audience[0] != string(aud) {
		return nil, ErrTokenAudienceMismatch
	}
	uid, err := uuid.Parse(c.Subject)
	if err != nil {
		return nil, ErrTokenInvalid
	}
	sid, err := uuid.Parse(c.SessionID)
	if err != nil {
		return nil, ErrTokenInvalid
	}
	p := &Principal{UserID: uid, SessionID: sid, Audience: aud, Context: c.Context, AMR: c.AMR}
	for _, r := range c.Roles {
		p.Roles = append(p.Roles, Role(r))
	}
	if c.Scopes != nil {
		for _, s := range c.Scopes.City {
			id, err := uuid.Parse(s)
			if err != nil {
				return nil, ErrTokenInvalid
			}
			p.CityScopes = append(p.CityScopes, id)
		}
	}
	return p, nil
}
