package catalog

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/storetmobiles2-code/rovo/backend/internal/platform/apperr"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/db"
	pgeo "github.com/storetmobiles2-code/rovo/backend/internal/platform/geo"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/money"
)

// restaurantRow is a live restaurant with everything a card needs.
type restaurantRow struct {
	id, cityID, zoneID uuid.UUID
	name               string
	nameI18n           map[string]string
	description        *string
	dietType           string
	location           pgeo.Point
	maxRadiusM         int
	prepMin            int
	costForTwoPaise    *int64
	acceptingOrders    bool
	pausedUntil        *time.Time
	menuVersion        int
	fssai              *string
	addressLine        string
	landmark           *string
	pinCode            string
	cuisines           []string
	cuisineCodes       []string
	ratingCount        int
	starsSum           int64
	closedNow          bool // inside a planned closure
	tz                 *time.Location
	slots              []slot
}

// restaurantFilter selects live restaurants. Zero fields do not filter.
type restaurantFilter struct {
	ids        []uuid.UUID
	cityID     *uuid.UUID
	near       *pgeo.Point // with maxRadiusM: index prefilter by delivery radius
	maxRadiusM int
}

func i18nMap(raw []byte) map[string]string {
	m := map[string]string{}
	_ = json.Unmarshal(raw, &m)
	return m
}

// loadRestaurants returns ACTIVE restaurants matching f, with cuisines, rating and hours. The
// radius filter is a superset (2% margin, GEO-D03); the exact check is done in Go by the caller.
func (s *service) loadRestaurants(ctx context.Context, f restaurantFilter, now time.Time) ([]restaurantRow, error) {
	var lat, lng *float64
	if f.near != nil {
		lat, lng = &f.near.Lat, &f.near.Lng
	}
	rows, err := s.deps.Pool.Query(ctx, `
		SELECT r.id, r.city_id, r.zone_id, r.name, r.name_i18n, r.description, r.diet_type,
		       ST_Y(r.location::geometry), ST_X(r.location::geometry), r.max_delivery_radius_m, r.avg_prep_time_min,
		       r.cost_for_two_paise, r.accepting_orders, r.paused_until, r.menu_version, r.fssai_license_no,
		       r.address_line, r.landmark, r.pin_code::text, c.timezone,
		       COALESCE((SELECT array_agg(cu.name ORDER BY rc.is_primary DESC, cu.sort_order, cu.name)
		                   FROM restaurant_cuisines rc JOIN cuisines cu ON cu.id = rc.cuisine_id
		                  WHERE rc.restaurant_id = r.id AND cu.is_active), '{}'),
		       COALESCE((SELECT array_agg(cu.code ORDER BY rc.is_primary DESC, cu.sort_order, cu.name)
		                   FROM restaurant_cuisines rc JOIN cuisines cu ON cu.id = rc.cuisine_id
		                  WHERE rc.restaurant_id = r.id AND cu.is_active), '{}'),
		       COALESCE(ra.rating_count, 0), COALESCE(ra.stars_sum, 0),
		       EXISTS (SELECT 1 FROM restaurant_closures cl
		                WHERE cl.restaurant_id = r.id AND cl.starts_at <= $1 AND cl.ends_at > $1)
		  FROM restaurants r
		  JOIN cities c ON c.id = r.city_id
		  LEFT JOIN rating_aggregates ra ON ra.target_type = 'RESTAURANT' AND ra.target_id = r.id
		 WHERE r.status = 'ACTIVE'
		   AND ($2::uuid[] IS NULL OR r.id = ANY($2))
		   AND ($3::uuid IS NULL OR r.city_id = $3)
		   AND ($4::float8 IS NULL OR ST_DWithin(r.location, ST_SetSRID(ST_MakePoint($5::float8, $4::float8), 4326)::geography,
		                                         LEAST(r.max_delivery_radius_m, $6::int) * 1.02))
		 ORDER BY r.id`,
		now, f.ids, f.cityID, lat, lng, f.maxRadiusM)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var (
		out   []restaurantRow
		index = map[uuid.UUID]int{}
		ids   []uuid.UUID
	)
	for rows.Next() {
		var (
			r    restaurantRow
			zone *uuid.UUID
			i18n []byte
			tz   string
		)
		if err := rows.Scan(&r.id, &r.cityID, &zone, &r.name, &i18n, &r.description, &r.dietType,
			&r.location.Lat, &r.location.Lng, &r.maxRadiusM, &r.prepMin, &r.costForTwoPaise, &r.acceptingOrders,
			&r.pausedUntil, &r.menuVersion, &r.fssai, &r.addressLine, &r.landmark, &r.pinCode, &tz,
			&r.cuisines, &r.cuisineCodes, &r.ratingCount, &r.starsSum, &r.closedNow); err != nil {
			return nil, err
		}
		if zone != nil {
			r.zoneID = *zone
		}
		r.nameI18n = i18nMap(i18n)
		r.tz = time.UTC
		if loc, err := time.LoadLocation(tz); err == nil {
			r.tz = loc
		}
		index[r.id] = len(out)
		ids = append(ids, r.id)
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, nil
	}

	hours, err := s.deps.Pool.Query(ctx, `SELECT restaurant_id, day_of_week,
		       EXTRACT(HOUR FROM opens_at)::int * 60 + EXTRACT(MINUTE FROM opens_at)::int,
		       EXTRACT(HOUR FROM closes_at)::int * 60 + EXTRACT(MINUTE FROM closes_at)::int
		  FROM restaurant_operating_hours WHERE restaurant_id = ANY($1) ORDER BY day_of_week, opens_at`, ids)
	if err != nil {
		return nil, err
	}
	defer hours.Close()
	for hours.Next() {
		var id uuid.UUID
		var sl slot
		if err := hours.Scan(&id, &sl.day, &sl.opens, &sl.closes); err != nil {
			return nil, err
		}
		r := &out[index[id]]
		r.slots = append(r.slots, sl)
	}
	return out, hours.Err()
}

func (s *service) Cuisines(ctx context.Context, cityID uuid.UUID) ([]Cuisine, error) {
	var city *uuid.UUID
	if cityID != uuid.Nil {
		city = &cityID
	}
	rows, err := s.deps.Pool.Query(ctx, `SELECT cu.code, cu.name, cu.name_i18n FROM cuisines cu
		WHERE cu.is_active AND ($1::uuid IS NULL OR EXISTS (
		        SELECT 1 FROM restaurant_cuisines rc JOIN restaurants r ON r.id = rc.restaurant_id
		         WHERE rc.cuisine_id = cu.id AND r.city_id = $1 AND r.status = 'ACTIVE'))
		ORDER BY cu.sort_order, cu.name`, city)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Cuisine{}
	for rows.Next() {
		var c Cuisine
		var i18n []byte
		if err := rows.Scan(&c.Code, &c.Name, &i18n); err != nil {
			return nil, err
		}
		c.NameI18n = i18nMap(i18n)
		out = append(out, c)
	}
	return out, rows.Err()
}

// itemColumns is the menu_items select list scanned by scanItem (table alias mi).
const itemColumns = `mi.id, mi.restaurant_id, mi.category_id, mi.name, mi.name_i18n, mi.description, mi.veg_type,
	mi.price_paise, mi.packaging_paise, mi.is_available, mi.unavailable_until, mi.is_recommended, mi.spice_level, mi.serves`

type scanner interface{ Scan(dest ...any) error }

func scanItem(row scanner, now time.Time) (MenuItem, error) {
	var (
		it               MenuItem
		i18n             []byte
		price, packaging int64
		available        bool
		until            *time.Time
	)
	if err := row.Scan(&it.ID, &it.restaurantID, &it.categoryID, &it.Name, &i18n, &it.Description, &it.VegType,
		&price, &packaging, &available, &until, &it.IsRecommended, &it.SpiceLevel, &it.Serves); err != nil {
		return MenuItem{}, err
	}
	it.NameI18n = i18nMap(i18n)
	it.Price, it.Packaging = money.INR(money.Paise(price)), money.INR(money.Paise(packaging))
	// An item switched off "until 6 pm" is orderable again once that time has passed.
	it.IsAvailable = available || (until != nil && !until.After(now))
	if !it.IsAvailable {
		it.UnavailableUntil = until
	}
	it.Variants, it.AddonGroups = []ItemVariant{}, []AddonGroup{}
	return it, nil
}

func (s *service) Menu(ctx context.Context, restaurantID uuid.UUID) (Menu, error) {
	now := s.deps.Clock.Now()
	pool := s.deps.Pool
	var m Menu
	err := pool.QueryRow(ctx, `SELECT menu_version FROM restaurants WHERE id = $1 AND status = 'ACTIVE'`, restaurantID).
		Scan(&m.MenuVersion)
	if db.IsNoRows(err) {
		return Menu{}, apperr.NotFound("Restaurant not found.")
	}
	if err != nil {
		return Menu{}, err
	}

	// Add-on groups with their add-ons, keyed by group id.
	groups := map[uuid.UUID]*AddonGroup{}
	grows, err := pool.Query(ctx, `SELECT id, name, name_i18n, min_select, max_select FROM addon_groups
		WHERE restaurant_id = $1 AND archived_at IS NULL`, restaurantID)
	if err != nil {
		return Menu{}, err
	}
	for grows.Next() {
		g := &AddonGroup{Addons: []Addon{}}
		var i18n []byte
		if err := grows.Scan(&g.ID, &g.Name, &i18n, &g.MinSelect, &g.MaxSelect); err != nil {
			grows.Close()
			return Menu{}, err
		}
		g.NameI18n = i18nMap(i18n)
		groups[g.ID] = g
	}
	grows.Close()
	if err := grows.Err(); err != nil {
		return Menu{}, err
	}
	arows, err := pool.Query(ctx, `SELECT a.addon_group_id, a.id, a.name, a.name_i18n, a.price_paise, a.veg_type, a.is_available
		FROM addons a JOIN addon_groups g ON g.id = a.addon_group_id
		WHERE g.restaurant_id = $1 AND g.archived_at IS NULL AND a.archived_at IS NULL
		ORDER BY a.sort_order, a.name`, restaurantID)
	if err != nil {
		return Menu{}, err
	}
	for arows.Next() {
		var (
			gid   uuid.UUID
			a     Addon
			i18n  []byte
			price int64
		)
		if err := arows.Scan(&gid, &a.ID, &a.Name, &i18n, &price, &a.VegType, &a.IsAvailable); err != nil {
			arows.Close()
			return Menu{}, err
		}
		a.NameI18n, a.Price = i18nMap(i18n), money.INR(money.Paise(price))
		if g := groups[gid]; g != nil {
			g.Addons = append(g.Addons, a)
		}
	}
	arows.Close()
	if err := arows.Err(); err != nil {
		return Menu{}, err
	}

	// Items in display order.
	var items []*MenuItem
	byItem := map[uuid.UUID]*MenuItem{}
	irows, err := pool.Query(ctx, `SELECT `+itemColumns+` FROM menu_items mi
		WHERE mi.restaurant_id = $1 AND mi.archived_at IS NULL ORDER BY mi.sort_order, mi.name`, restaurantID)
	if err != nil {
		return Menu{}, err
	}
	for irows.Next() {
		it, err := scanItem(irows, now)
		if err != nil {
			irows.Close()
			return Menu{}, err
		}
		items = append(items, &it)
		byItem[it.ID] = &it
	}
	irows.Close()
	if err := irows.Err(); err != nil {
		return Menu{}, err
	}

	vrows, err := pool.Query(ctx, `SELECT v.menu_item_id, v.id, v.name, v.name_i18n, v.price_paise, v.is_default, v.is_available
		FROM item_variants v JOIN menu_items mi ON mi.id = v.menu_item_id
		WHERE mi.restaurant_id = $1 AND mi.archived_at IS NULL AND v.archived_at IS NULL
		ORDER BY v.sort_order, v.name`, restaurantID)
	if err != nil {
		return Menu{}, err
	}
	for vrows.Next() {
		var (
			itemID uuid.UUID
			v      ItemVariant
			i18n   []byte
			price  int64
		)
		if err := vrows.Scan(&itemID, &v.ID, &v.Name, &i18n, &price, &v.IsDefault, &v.IsAvailable); err != nil {
			vrows.Close()
			return Menu{}, err
		}
		v.NameI18n, v.Price = i18nMap(i18n), money.INR(money.Paise(price))
		if it := byItem[itemID]; it != nil {
			it.Variants = append(it.Variants, v)
		}
	}
	vrows.Close()
	if err := vrows.Err(); err != nil {
		return Menu{}, err
	}

	lrows, err := pool.Query(ctx, `SELECT menu_item_id, addon_group_id FROM menu_item_addon_groups
		WHERE restaurant_id = $1 ORDER BY sort_order, addon_group_id`, restaurantID)
	if err != nil {
		return Menu{}, err
	}
	for lrows.Next() {
		var itemID, groupID uuid.UUID
		if err := lrows.Scan(&itemID, &groupID); err != nil {
			lrows.Close()
			return Menu{}, err
		}
		if it, g := byItem[itemID], groups[groupID]; it != nil && g != nil {
			it.AddonGroups = append(it.AddonGroups, *g)
		}
	}
	lrows.Close()
	if err := lrows.Err(); err != nil {
		return Menu{}, err
	}

	crows, err := pool.Query(ctx, `SELECT id, name, name_i18n FROM menu_categories
		WHERE restaurant_id = $1 AND is_active AND archived_at IS NULL ORDER BY sort_order, name`, restaurantID)
	if err != nil {
		return Menu{}, err
	}
	defer crows.Close()
	m.Categories = []MenuCategory{}
	for crows.Next() {
		c := MenuCategory{Items: []MenuItem{}}
		var i18n []byte
		if err := crows.Scan(&c.ID, &c.Name, &i18n); err != nil {
			return Menu{}, err
		}
		c.NameI18n = i18nMap(i18n)
		for _, it := range items {
			if it.categoryID == c.ID {
				c.Items = append(c.Items, *it)
			}
		}
		// A category with nothing to show is left out.
		if len(c.Items) > 0 {
			m.Categories = append(m.Categories, c)
		}
	}
	return m, crows.Err()
}

// searchItems finds dishes by English or Telugu name among the given restaurants. Names that
// start with the text rank first. Hits carry no variants or add-ons; the menu endpoint has those.
func (s *service) searchItems(ctx context.Context, restaurantIDs []uuid.UUID, lowered string, limit int, now time.Time) ([]MenuItem, error) {
	if len(restaurantIDs) == 0 {
		return nil, nil
	}
	esc := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(lowered)
	rows, err := s.deps.Pool.Query(ctx, `SELECT `+itemColumns+` FROM menu_items mi
		JOIN menu_categories mc ON mc.id = mi.category_id
		WHERE mi.restaurant_id = ANY($1) AND mi.archived_at IS NULL AND mc.is_active AND mc.archived_at IS NULL
		  AND (lower(mi.name) LIKE $2 OR mi.name_i18n->>'te' LIKE $2)
		ORDER BY (lower(mi.name) LIKE $3) DESC, mi.is_available DESC, mi.name, mi.id LIMIT $4`,
		restaurantIDs, "%"+esc+"%", esc+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []MenuItem
	for rows.Next() {
		it, err := scanItem(rows, now)
		if err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}
