-- +goose Up
-- Modules: pricing (fee_configs, tax_rules, quotes), promotions (coupons, coupon_targets,
-- coupon_redemptions). DDL follows doc 10 §5; default values mirror 16 §6.5 (R18, R48).

CREATE TABLE fee_configs (
  id                               uuid PRIMARY KEY,
  city_id                          uuid NOT NULL REFERENCES cities(id),
  zone_id                          uuid,              -- ref: geo.zones; NULL = city default
  version                          int NOT NULL,
  effective_from                   timestamptz NOT NULL,
  effective_to                     timestamptz,
  delivery_fee_slabs               jsonb NOT NULL,    -- road metres [fromM, toM): [{"fromM":0,"toM":2000,"feePaise":2000}, ...]
  free_delivery_min_order_paise    bigint,
  platform_fee_paise               bigint NOT NULL DEFAULT 500,
  small_cart_threshold_paise       bigint NOT NULL DEFAULT 14900,
  small_cart_fee_paise             bigint NOT NULL DEFAULT 1500,
  road_factor_milli                int NOT NULL DEFAULT 1300 CHECK (road_factor_milli BETWEEN 1000 AND 3000),
  max_serviceable_radius_m         int NOT NULL DEFAULT 7000 CHECK (max_serviceable_radius_m BETWEEN 500 AND 15000),
  cod_enabled                      boolean NOT NULL DEFAULT true,
  cod_max_order_paise              bigint NOT NULL DEFAULT 100000,
  cod_first_order_max_paise        bigint NOT NULL DEFAULT 60000,
  round_payable_to_rupee           boolean NOT NULL DEFAULT true,
  rider_base_pay_paise             bigint NOT NULL DEFAULT 2500,
  rider_base_distance_m            int NOT NULL DEFAULT 2000,
  rider_per_km_paise               bigint NOT NULL DEFAULT 600,
  rider_wait_free_s                int NOT NULL DEFAULT 600,
  rider_wait_per_min_paise         bigint NOT NULL DEFAULT 100,
  rider_cash_limit_paise           bigint NOT NULL DEFAULT 200000,
  rider_cancel_comp_bps            int NOT NULL DEFAULT 5000,
  eta_params                       jsonb NOT NULL,
  status                           text NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT','APPROVED','RETIRED')),
  approval_id                      uuid,
  created_by                       uuid REFERENCES users(id),
  created_at                       timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ux_fee_configs__version UNIQUE NULLS NOT DISTINCT (city_id, zone_id, version),
  CONSTRAINT ck_fee_configs__range CHECK (effective_to IS NULL OR effective_to > effective_from),
  CONSTRAINT ck_fee_configs__slabs CHECK (jsonb_typeof(delivery_fee_slabs) = 'array'
                                          AND jsonb_array_length(delivery_fee_slabs) BETWEEN 1 AND 20),
  CONSTRAINT ex_fee_configs__no_overlap EXCLUDE USING gist (
      city_id WITH =,
      (coalesce(zone_id, '00000000-0000-0000-0000-000000000000'::uuid)) WITH =,
      tstzrange(effective_from, effective_to) WITH &&) WHERE (status = 'APPROVED')
);

CREATE TABLE tax_rules (
  id                     uuid PRIMARY KEY,
  country_code           char(2) NOT NULL DEFAULT 'IN',
  component              text NOT NULL CHECK (component IN ('ITEM_TOTAL','PACKAGING','DELIVERY_FEE',
                                                            'PLATFORM_FEE','SMALL_CART_FEE','COMMISSION')),
  tax_category           text NOT NULL DEFAULT 'DEFAULT',
  rate_bps               int NOT NULL CHECK (rate_bps BETWEEN 0 AND 10000),
  split                  text NOT NULL CHECK (split IN ('CGST_SGST','IGST')),
  liable_party           text NOT NULL CHECK (liable_party IN ('PLATFORM_SEC_9_5','PLATFORM','RESTAURANT')),
  price_inclusive        boolean NOT NULL DEFAULT false,
  sac_code               text,
  legal_basis            text NOT NULL,
  effective_from         timestamptz NOT NULL,
  effective_to           timestamptz,
  created_by             uuid REFERENCES users(id),
  created_at             timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ex_tax_rules__no_overlap EXCLUDE USING gist (
      country_code WITH =, component WITH =, tax_category WITH =, tstzrange(effective_from, effective_to) WITH &&)
);

CREATE TABLE quotes (
  id                     uuid PRIMARY KEY,
  user_id                uuid NOT NULL REFERENCES users(id),
  city_id                uuid NOT NULL REFERENCES cities(id),
  restaurant_id          uuid NOT NULL,              -- ref: catalog.restaurants
  menu_version           int NOT NULL,
  zone_id                uuid NOT NULL,              -- ref: geo.zones
  address_id             uuid NOT NULL,              -- ref: users.customer_addresses
  payment_method         text NOT NULL CHECK (payment_method IN ('ONLINE','COD')),
  coupon_code            citext,
  fee_config_id          uuid NOT NULL REFERENCES fee_configs(id),
  input_lines            jsonb NOT NULL,
  input_hash             bytea NOT NULL,
  priced_lines           jsonb NOT NULL,
  charges                jsonb NOT NULL,
  straight_distance_m    int NOT NULL,
  est_road_distance_m    int NOT NULL,
  eta_min_minutes        smallint NOT NULL,
  eta_max_minutes        smallint NOT NULL,
  total_paise            bigint NOT NULL CHECK (total_paise >= 0),
  currency               char(3) NOT NULL DEFAULT 'INR',
  issues                 jsonb NOT NULL DEFAULT '[]',
  is_orderable           boolean NOT NULL,
  expires_at             timestamptz NOT NULL,
  consumed_at            timestamptz,
  order_id               uuid,                       -- ref: ordering.orders
  created_at             timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ix_quotes__user ON quotes (user_id, created_at DESC);
CREATE INDEX ix_quotes__sweep ON quotes (expires_at) WHERE consumed_at IS NULL;
CREATE UNIQUE INDEX ux_quotes__order ON quotes (order_id) WHERE order_id IS NOT NULL;

CREATE TABLE coupons (
  id                       uuid PRIMARY KEY,
  city_id                  uuid REFERENCES cities(id),
  kind                     text NOT NULL DEFAULT 'CAMPAIGN' CHECK (kind IN ('CAMPAIGN','GOODWILL')),
  owner_user_id            uuid REFERENCES users(id),
  source_ticket_id         uuid,
  source_order_id          uuid,
  code                     citext NOT NULL CHECK (code ~ '^[A-Za-z0-9]{4,24}$'),
  title                    text NOT NULL,
  title_i18n               jsonb NOT NULL DEFAULT '{}',
  terms                    text NOT NULL,
  terms_i18n               jsonb NOT NULL DEFAULT '{}',
  discount_type            text NOT NULL CHECK (discount_type IN ('FLAT','PERCENT')),
  applies_to               text NOT NULL DEFAULT 'ITEM_TOTAL' CHECK (applies_to IN ('ITEM_TOTAL','DELIVERY_FEE')),
  flat_paise               bigint CHECK (flat_paise > 0),
  percent_bps              int CHECK (percent_bps BETWEEN 1 AND 10000),
  max_discount_paise       bigint CHECK (max_discount_paise > 0),
  min_order_paise          bigint NOT NULL DEFAULT 0 CHECK (min_order_paise >= 0),
  valid_from               timestamptz NOT NULL,
  valid_until              timestamptz NOT NULL,
  per_user_limit           int NOT NULL DEFAULT 1 CHECK (per_user_limit >= 1),
  global_limit             int CHECK (global_limit >= 1),
  redemption_count         int NOT NULL DEFAULT 0,
  budget_paise             bigint CHECK (budget_paise > 0),
  budget_used_paise        bigint NOT NULL DEFAULT 0,
  funded_by                text NOT NULL CHECK (funded_by IN ('PLATFORM','RESTAURANT')),
  new_users_only           boolean NOT NULL DEFAULT false,
  payment_methods          text[],
  is_public                boolean NOT NULL DEFAULT true,
  status                   text NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT','ACTIVE','PAUSED','ENDED','ARCHIVED')),
  created_by               uuid REFERENCES users(id),
  version                  int NOT NULL DEFAULT 1,
  created_at               timestamptz NOT NULL DEFAULT now(),
  updated_at               timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ck_coupons__shape CHECK (
       (discount_type = 'FLAT'    AND flat_paise IS NOT NULL AND percent_bps IS NULL)
    OR (discount_type = 'PERCENT' AND percent_bps IS NOT NULL AND flat_paise IS NULL AND max_discount_paise IS NOT NULL)),
  CONSTRAINT ck_coupons__validity CHECK (valid_until > valid_from),
  CONSTRAINT ck_coupons__limits   CHECK (global_limit IS NULL OR redemption_count <= global_limit),
  CONSTRAINT ck_coupons__budget   CHECK (budget_paise IS NULL OR budget_used_paise <= budget_paise),
  CONSTRAINT ck_coupons__goodwill CHECK (kind <> 'GOODWILL' OR (owner_user_id IS NOT NULL AND discount_type = 'FLAT'
                                         AND global_limit = 1 AND per_user_limit = 1 AND is_public = false))
);
CREATE UNIQUE INDEX ux_coupons__code ON coupons (code) WHERE status <> 'ARCHIVED';
CREATE INDEX ix_coupons__live ON coupons (city_id, valid_until) WHERE status = 'ACTIVE';
CREATE INDEX ix_coupons__owner ON coupons (owner_user_id) WHERE kind = 'GOODWILL' AND status = 'ACTIVE';
SELECT attach_updated_at_trigger('coupons');

CREATE TABLE coupon_targets (
  coupon_id     uuid NOT NULL REFERENCES coupons(id) ON DELETE CASCADE,
  target_type   text NOT NULL CHECK (target_type IN ('ZONE','RESTAURANT')),
  target_id     uuid NOT NULL,
  PRIMARY KEY (coupon_id, target_type, target_id)
);
CREATE INDEX ix_coupon_targets__target ON coupon_targets (target_type, target_id);

CREATE TABLE coupon_redemptions (
  id                         uuid PRIMARY KEY,
  coupon_id                  uuid NOT NULL REFERENCES coupons(id),
  user_id                    uuid NOT NULL REFERENCES users(id),
  order_id                   uuid NOT NULL UNIQUE,        -- ref: ordering.orders
  discount_paise             bigint NOT NULL CHECK (discount_paise > 0),
  platform_funded_paise      bigint NOT NULL,
  restaurant_funded_paise    bigint NOT NULL,
  status                     text NOT NULL CHECK (status IN ('RESERVED','APPLIED','RELEASED')),
  reserved_at                timestamptz NOT NULL DEFAULT now(),
  applied_at                 timestamptz,
  released_at                timestamptz,
  CONSTRAINT ck_coupon_redemptions__split CHECK (platform_funded_paise + restaurant_funded_paise = discount_paise)
);
CREATE INDEX ix_coupon_redemptions__user ON coupon_redemptions (coupon_id, user_id) WHERE status IN ('RESERVED','APPLIED');

-- +goose Down
DROP TABLE IF EXISTS coupon_redemptions;
DROP TABLE IF EXISTS coupon_targets;
DROP TABLE IF EXISTS coupons;
DROP TABLE IF EXISTS quotes;
DROP TABLE IF EXISTS tax_rules;
DROP TABLE IF EXISTS fee_configs;
