package identity

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/storetmobiles2-code/rovo/backend/internal/platform/apperr"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/auth"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/db"
)

// refreshRaceWindow is how long after use a replayed refresh token counts as a benign
// multi-tab race rather than theft (12 §4.3).
const refreshRaceWindow = 15 * time.Second

// sessionPolicy is the idle (sliding) and absolute lifetime of a session (12 §3.4).
type sessionPolicy struct{ idle, absolute time.Duration }

const day = 24 * time.Hour

func policyFor(aud auth.Audience) sessionPolicy {
	switch aud {
	case auth.AudienceRestaurant:
		return sessionPolicy{30 * day, 90 * day}
	case auth.AudienceRider:
		return sessionPolicy{30 * day, 180 * day}
	case auth.AudienceAdmin:
		return sessionPolicy{30 * time.Minute, 12 * time.Hour}
	default:
		return sessionPolicy{7 * day, 30 * day}
	}
}

// grant is one live user_roles row.
type grant struct {
	role         auth.Role
	restaurantID *uuid.UUID
}

// claimsFor returns the roles and active context an access token for aud may carry. A member
// can hold roles for several apps; each app's token only carries its own.
func claimsFor(aud auth.Audience, grants []grant) ([]auth.Role, string) {
	var roles []auth.Role
	activeCtx := ""
	add := func(r auth.Role) {
		for _, have := range roles {
			if have == r {
				return
			}
		}
		roles = append(roles, r)
	}
	for _, g := range grants {
		switch {
		case aud == auth.AudienceCustomer && g.role == auth.RoleCustomer:
			add(g.role)
		case aud == auth.AudienceRider && g.role == auth.RoleRider:
			add(g.role)
			activeCtx = "rider"
		case aud == auth.AudienceRestaurant && (g.role == auth.RoleRestaurantOwner || g.role == auth.RoleRestaurantStaff):
			// The first restaurant (oldest grant) is the default context; only its role is claimed.
			if activeCtx == "" && g.restaurantID != nil {
				activeCtx = "restaurant:" + g.restaurantID.String()
				add(g.role)
			}
		}
	}
	return roles, activeCtx
}

func loadGrants(ctx context.Context, q db.Querier, userID uuid.UUID) ([]grant, error) {
	rows, err := q.Query(ctx, `SELECT role, restaurant_id FROM user_roles
		WHERE user_id = $1 AND revoked_at IS NULL ORDER BY granted_at, id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []grant
	for rows.Next() {
		var g grant
		var role string
		if err := rows.Scan(&role, &g.restaurantID); err != nil {
			return nil, err
		}
		g.role = auth.Role(role)
		out = append(out, g)
	}
	return out, rows.Err()
}

func devicePlatform(userAgent string) string {
	switch {
	case strings.Contains(userAgent, "Android"):
		return "WEB_ANDROID"
	case strings.Contains(userAgent, "iPhone"), strings.Contains(userAgent, "iPad"):
		return "WEB_IOS"
	default:
		return "WEB_DESKTOP"
	}
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

// startSession registers the device, inserts the session with its first refresh token and
// issues the access token. It runs inside the OTP verification transaction.
func (s *service) startSession(ctx context.Context, tx pgx.Tx, userID uuid.UUID, in VerifyOTPInput, now time.Time) (Tokens, error) {
	var deviceID uuid.UUID
	ua := truncate(in.UserAgent, 300)
	if err := tx.QueryRow(ctx, `INSERT INTO devices (id, install_id, user_id, platform, app, user_agent, first_seen_at, last_seen_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
		ON CONFLICT (install_id) DO UPDATE SET user_id = EXCLUDED.user_id, user_agent = EXCLUDED.user_agent,
		  last_seen_at = EXCLUDED.last_seen_at
		RETURNING id`,
		s.deps.IDs.NewID(), in.DeviceInstallID, userID, devicePlatform(ua), string(in.Audience), nullIfEmpty(ua), now).
		Scan(&deviceID); err != nil {
		return Tokens{}, err
	}
	grants, err := loadGrants(ctx, tx, userID)
	if err != nil {
		return Tokens{}, err
	}
	roles, activeCtx := claimsFor(in.Audience, grants)
	pol := policyFor(in.Audience)
	sessionID := s.deps.IDs.NewID()
	absolute := now.Add(pol.absolute)
	idle := now.Add(pol.idle)
	amr := []string{"otp"}
	if _, err := tx.Exec(ctx, `INSERT INTO sessions
		(id, user_id, audience, active_context, device_id, idle_timeout_s, absolute_timeout_s, amr,
		 created_ip, last_ip, last_seen_at, idle_expires_at, absolute_expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::inet, $9::inet, $10, $11, $12, $10)`,
		sessionID, userID, string(in.Audience), nullIfEmpty(activeCtx), deviceID,
		int(pol.idle/time.Second), int(pol.absolute/time.Second), amr, nullIfEmpty(in.IP), now, idle, absolute); err != nil {
		return Tokens{}, err
	}
	refresh, hash := auth.NewRefreshToken()
	if _, err := tx.Exec(ctx, `INSERT INTO refresh_tokens (id, session_id, token_sha256, issued_at, expires_at)
		VALUES ($1, $2, $3, $4, $5)`, s.deps.IDs.NewID(), sessionID, hash, now, idle); err != nil {
		return Tokens{}, err
	}
	access, exp, err := s.tokens.Issue(auth.TokenInput{UserID: userID, SessionID: sessionID, Audience: in.Audience,
		Roles: roles, Context: activeCtx, AMR: amr})
	if err != nil {
		return Tokens{}, err
	}
	return Tokens{
		AccessToken: access, AccessExpiresAt: exp, RefreshToken: refresh, RefreshMaxAge: absolute.Sub(now),
		Principal: &auth.Principal{UserID: userID, SessionID: sessionID, Audience: in.Audience, Roles: roles, Context: activeCtx, AMR: amr},
	}, nil
}

func sessionEnded() *apperr.Error {
	return apperr.Unauthenticated(apperr.CodeSessionRevoked, "Your session has ended. Please sign in again.")
}

// Refresh consumes the presented refresh token and issues a child in the same session, sliding
// the idle expiry up to the absolute limit. A token replayed after the race window revokes the
// whole session (reuse detection); within the window it returns 409 REFRESH_RACE.
func (s *service) Refresh(ctx context.Context, refreshToken string, aud auth.Audience, ip string) (Tokens, error) {
	if !auth.LooksLikeRefreshToken(refreshToken) {
		return Tokens{}, apperr.Unauthenticated(apperr.CodeUnauthenticated, "Authentication required.")
	}
	now := s.deps.Clock.Now()
	type result struct {
		fail *apperr.Error
		tok  Tokens
	}
	out, err := db.InTx(ctx, s.deps.Pool, func(tx pgx.Tx) (result, error) {
		var (
			tokenID, sessionID, userID uuid.UUID
			usedAt, revokedAt          *time.Time
			tokenExp, idleExp, absExp  time.Time
			audience, userStatus       string
			idleTimeoutS               int
			amr                        []string
		)
		err := tx.QueryRow(ctx, `SELECT rt.id, rt.used_at, rt.expires_at, s.id, s.user_id, s.audience, s.revoked_at,
			       s.idle_expires_at, s.absolute_expires_at, s.idle_timeout_s, s.amr, u.status
			FROM refresh_tokens rt JOIN sessions s ON s.id = rt.session_id JOIN users u ON u.id = s.user_id
			WHERE rt.token_sha256 = $1 FOR UPDATE OF rt, s`, auth.HashToken(refreshToken)).
			Scan(&tokenID, &usedAt, &tokenExp, &sessionID, &userID, &audience, &revokedAt, &idleExp, &absExp, &idleTimeoutS, &amr, &userStatus)
		if db.IsNoRows(err) {
			return result{fail: apperr.Unauthenticated(apperr.CodeUnauthenticated, "Authentication required.")}, nil
		}
		if err != nil {
			return result{}, err
		}
		if revokedAt != nil || audience != string(aud) {
			return result{fail: sessionEnded()}, nil
		}
		if usedAt != nil {
			if now.Sub(*usedAt) <= refreshRaceWindow {
				return result{fail: apperr.Conflict(apperr.CodeRefreshRace, "Session was refreshed in another tab. Retry.")}, nil
			}
			if _, err := tx.Exec(ctx, `UPDATE sessions SET revoked_at = $2, revoke_reason = 'REFRESH_REUSE' WHERE id = $1`,
				sessionID, now); err != nil {
				return result{}, err
			}
			s.deps.Logger.WarnContext(ctx, "auth.refresh.reuse_detected", "session_id", sessionID, "user_id", userID)
			return result{fail: sessionEnded()}, nil
		}
		if !tokenExp.After(now) || !idleExp.After(now) || !absExp.After(now) {
			return result{fail: sessionEnded()}, nil
		}
		if userStatus != "ACTIVE" {
			if _, err := tx.Exec(ctx, `UPDATE sessions SET revoked_at = $2, revoke_reason = 'USER_BLOCKED' WHERE id = $1`,
				sessionID, now); err != nil {
				return result{}, err
			}
			return result{fail: sessionEnded()}, nil
		}

		newIdle := now.Add(time.Duration(idleTimeoutS) * time.Second)
		if newIdle.After(absExp) {
			newIdle = absExp
		}
		if _, err := tx.Exec(ctx, `UPDATE refresh_tokens SET used_at = $2 WHERE id = $1`, tokenID, now); err != nil {
			return result{}, err
		}
		refresh, hash := auth.NewRefreshToken()
		if _, err := tx.Exec(ctx, `INSERT INTO refresh_tokens (id, session_id, token_sha256, parent_id, issued_at, expires_at)
			VALUES ($1, $2, $3, $4, $5, $6)`, s.deps.IDs.NewID(), sessionID, hash, tokenID, now, newIdle); err != nil {
			return result{}, err
		}
		grants, err := loadGrants(ctx, tx, userID)
		if err != nil {
			return result{}, err
		}
		roles, activeCtx := claimsFor(aud, grants)
		if _, err := tx.Exec(ctx, `UPDATE sessions SET last_seen_at = $2, idle_expires_at = $3, active_context = $4,
			last_ip = COALESCE($5::inet, last_ip) WHERE id = $1`, sessionID, now, newIdle, nullIfEmpty(activeCtx), nullIfEmpty(ip)); err != nil {
			return result{}, err
		}
		access, exp, err := s.tokens.Issue(auth.TokenInput{UserID: userID, SessionID: sessionID, Audience: aud,
			Roles: roles, Context: activeCtx, AMR: amr})
		if err != nil {
			return result{}, err
		}
		return result{tok: Tokens{
			AccessToken: access, AccessExpiresAt: exp, RefreshToken: refresh, RefreshMaxAge: absExp.Sub(now),
			Principal: &auth.Principal{UserID: userID, SessionID: sessionID, Audience: aud, Roles: roles, Context: activeCtx, AMR: amr},
		}}, nil
	})
	if err != nil {
		return Tokens{}, err
	}
	if out.fail != nil {
		return Tokens{}, out.fail
	}
	return out.tok, nil
}

func (s *service) Logout(ctx context.Context, sessionID uuid.UUID) error {
	_, err := s.deps.Pool.Exec(ctx, `UPDATE sessions SET revoked_at = $2, revoke_reason = 'LOGOUT'
		WHERE id = $1 AND revoked_at IS NULL`, sessionID, s.deps.Clock.Now())
	return err
}

func (s *service) LogoutByRefreshToken(ctx context.Context, refreshToken string) (bool, error) {
	if !auth.LooksLikeRefreshToken(refreshToken) {
		return false, nil
	}
	var sessionID uuid.UUID
	err := s.deps.Pool.QueryRow(ctx, `SELECT session_id FROM refresh_tokens WHERE token_sha256 = $1`,
		auth.HashToken(refreshToken)).Scan(&sessionID)
	if db.IsNoRows(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, s.Logout(ctx, sessionID)
}
