-- +goose Up
-- Module: catalog (doc 10 §4). Restaurant onboarding applications are restaurants rows in
-- DRAFT/SUBMITTED/CHANGES_REQUESTED (doc 10 §1.5), not a separate table.

CREATE TABLE restaurants (
  id                          uuid PRIMARY KEY,
  city_id                     uuid NOT NULL REFERENCES cities(id),
  zone_id                     uuid,              -- ref: geo.zones
  locality_id                 uuid,              -- ref: geo.localities
  slug                        text NOT NULL CHECK (slug ~ '^[a-z0-9-]{3,80}$'),
  name                        text NOT NULL CHECK (char_length(name) BETWEEN 2 AND 80),
  name_i18n                   jsonb NOT NULL DEFAULT '{}',
  description                 text CHECK (char_length(description) <= 500),
  description_i18n            jsonb NOT NULL DEFAULT '{}',
  legal_name                  text,
  status                      text NOT NULL DEFAULT 'DRAFT' CHECK (status IN
                                ('DRAFT','SUBMITTED','CHANGES_REQUESTED','REJECTED','ACTIVE','SUSPENDED','OFFBOARDED')),
  status_reason               text,
  diet_type                   text NOT NULL DEFAULT 'MIXED' CHECK (diet_type IN ('PURE_VEG','MIXED')),
  phone_e164                  text NOT NULL CHECK (phone_e164 ~ '^\+[1-9][0-9]{7,14}$'),
  email                       citext,
  address_line                text NOT NULL,
  landmark                    text,
  pin_code                    char(6) NOT NULL CHECK (pin_code ~ '^[1-9][0-9]{5}$'),
  location                    geography(Point,4326) NOT NULL,
  max_delivery_radius_m       int NOT NULL DEFAULT 7000 CHECK (max_delivery_radius_m BETWEEN 500 AND 15000),
  avg_prep_time_min           smallint NOT NULL DEFAULT 20 CHECK (avg_prep_time_min BETWEEN 5 AND 90),
  min_order_paise             bigint NOT NULL DEFAULT 0 CHECK (min_order_paise >= 0),
  cost_for_two_paise          bigint CHECK (cost_for_two_paise >= 0),
  packaging_mode              text NOT NULL DEFAULT 'PER_ITEM' CHECK (packaging_mode IN ('PER_ITEM','PER_ORDER','NONE')),
  packaging_per_order_paise   bigint NOT NULL DEFAULT 0 CHECK (packaging_per_order_paise BETWEEN 0 AND 10000),
  fssai_license_no            text CHECK (fssai_license_no ~ '^[0-9]{14}$'),
  fssai_valid_until           date,
  gstin                       text CHECK (gstin ~ '^[0-9]{2}[A-Z0-9]{10}[0-9A-Z]Z[0-9A-Z]$'),
  pan_enc                     bytea,
  pan_last4                   text,
  pii_key_id                  text,
  accepting_orders            boolean NOT NULL DEFAULT false,
  paused_until                timestamptz,
  pause_reason                text CHECK (pause_reason IN ('TOO_MANY_ORDERS','STAFF_SHORTAGE','KITCHEN_ISSUE',
                                                           'AUTO_MISSED_ORDERS','DEVICE_OFFLINE','ADMIN','OTHER')),
  paused_by_user_id           uuid REFERENCES users(id),
  consecutive_missed_orders   smallint NOT NULL DEFAULT 0,
  menu_version                int NOT NULL DEFAULT 1,
  logo_file_id                uuid,
  cover_file_id               uuid,
  approved_at                 timestamptz,
  approved_by                 uuid REFERENCES users(id),
  version                     int NOT NULL DEFAULT 1,
  created_at                  timestamptz NOT NULL DEFAULT now(),
  updated_at                  timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ux_restaurants__city_slug UNIQUE (city_id, slug),
  CONSTRAINT ck_restaurants__live_requirements CHECK (
    status <> 'ACTIVE' OR (fssai_license_no IS NOT NULL AND fssai_valid_until IS NOT NULL AND zone_id IS NOT NULL))
);
CREATE INDEX ix_restaurants__live_location ON restaurants USING gist (location) WHERE status = 'ACTIVE';
CREATE INDEX ix_restaurants__city_status   ON restaurants (city_id, status);
CREATE INDEX ix_restaurants__name_trgm     ON restaurants USING gin (lower(name) gin_trgm_ops);
CREATE INDEX ix_restaurants__fssai_expiry  ON restaurants (fssai_valid_until) WHERE status = 'ACTIVE';
SELECT attach_updated_at_trigger('restaurants');

CREATE TABLE staff_invites (
  id                   uuid PRIMARY KEY,
  restaurant_id        uuid NOT NULL REFERENCES restaurants(id),
  phone_hmac           bytea NOT NULL,
  phone_last4          text NOT NULL,
  display_name         text,
  permissions          text[] NOT NULL DEFAULT '{}',
  token_sha256         bytea NOT NULL UNIQUE,
  status               text NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING','ACCEPTED','REVOKED','EXPIRED')),
  expires_at           timestamptz NOT NULL,
  invited_by           uuid NOT NULL REFERENCES users(id),
  accepted_by          uuid REFERENCES users(id),
  accepted_at          timestamptz,
  created_at           timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX ux_staff_invites__pending ON staff_invites (restaurant_id, phone_hmac) WHERE status = 'PENDING';

CREATE TABLE restaurant_users (
  id                   uuid PRIMARY KEY,
  restaurant_id        uuid NOT NULL REFERENCES restaurants(id),
  user_id              uuid NOT NULL REFERENCES users(id),
  role                 text NOT NULL CHECK (role IN ('RESTAURANT_OWNER','RESTAURANT_STAFF')),
  permissions          text[] NOT NULL DEFAULT '{}',
  display_name         text,
  status               text NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE','REMOVED')),
  staff_invite_id      uuid REFERENCES staff_invites(id),
  user_role_id         uuid,                                -- ref: identity.user_roles
  removed_at           timestamptz,
  created_at           timestamptz NOT NULL DEFAULT now(),
  updated_at           timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX ux_restaurant_users__member ON restaurant_users (restaurant_id, user_id) WHERE status = 'ACTIVE';
CREATE INDEX ix_restaurant_users__user ON restaurant_users (user_id) WHERE status = 'ACTIVE';
SELECT attach_updated_at_trigger('restaurant_users');

CREATE TABLE leads (
  id                      uuid PRIMARY KEY,
  lead_type               text NOT NULL CHECK (lead_type IN ('RESTAURANT','RIDER')),
  city_id                 uuid REFERENCES cities(id),
  locality_id             uuid,
  business_name           text CHECK (char_length(business_name) <= 80),
  contact_name            text NOT NULL CHECK (char_length(contact_name) <= 100),
  phone_e164              text NOT NULL CHECK (phone_e164 ~ '^\+[1-9][0-9]{7,14}$'),
  cuisine_codes           text[] NOT NULL DEFAULT '{}',
  fssai_state             text CHECK (fssai_state IN ('YES','NO','APPLIED')),
  preferred_call_time     text,
  status                  text NOT NULL DEFAULT 'NEW' CHECK (status IN ('NEW','CONTACTED','CONVERTED','DISQUALIFIED')),
  assigned_admin_id       uuid REFERENCES users(id),
  converted_restaurant_id uuid,
  converted_rider_id      uuid,
  notes                   text,
  consent_notice_version  text NOT NULL,
  version                 int NOT NULL DEFAULT 1,
  created_at              timestamptz NOT NULL DEFAULT now(),
  updated_at              timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ix_leads__queue ON leads (city_id, lead_type, status, created_at);
SELECT attach_updated_at_trigger('leads');

CREATE TABLE restaurant_devices (
  id                    uuid PRIMARY KEY,
  restaurant_id         uuid NOT NULL REFERENCES restaurants(id),
  device_id             uuid NOT NULL,                       -- ref: identity.devices
  user_id               uuid REFERENCES users(id),
  label                 text,
  is_order_receiver     boolean NOT NULL DEFAULT true,
  session_id            uuid,                                -- ref: identity.sessions (DEVICE_BOUND)
  registered_by         uuid REFERENCES users(id),
  revoked_at            timestamptz,
  last_heartbeat_at     timestamptz NOT NULL DEFAULT now(),
  last_sse_connected_at timestamptz,
  push_ok               boolean NOT NULL DEFAULT false,
  app_version           text,
  created_at            timestamptz NOT NULL DEFAULT now(),
  updated_at            timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ux_restaurant_devices__device UNIQUE (restaurant_id, device_id)
) WITH (fillfactor = 70);
CREATE INDEX ix_restaurant_devices__liveness ON restaurant_devices (restaurant_id, last_heartbeat_at DESC) WHERE is_order_receiver;
SELECT attach_updated_at_trigger('restaurant_devices');

CREATE TABLE restaurant_kyc_documents (
  id                 uuid PRIMARY KEY,
  restaurant_id      uuid NOT NULL REFERENCES restaurants(id),
  doc_type           text NOT NULL CHECK (doc_type IN ('FSSAI_LICENSE','GST_CERTIFICATE','PAN_CARD','BANK_PROOF',
                                                       'SHOP_ESTABLISHMENT','OWNER_ID_PROOF','TRADE_LICENSE','MENU_CARD','OUTLET_PHOTO')),
  file_id            uuid NOT NULL,                     -- ref: platform.file_objects
  doc_number_enc     bytea,
  doc_number_last4   text,
  pii_key_id         text,
  valid_until        date,
  status             text NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING','APPROVED','REJECTED','EXPIRED','SUPERSEDED')),
  reviewed_by        uuid REFERENCES users(id),
  reviewed_at        timestamptz,
  rejection_reason   text,
  created_at         timestamptz NOT NULL DEFAULT now(),
  updated_at         timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX ux_restaurant_kyc__current ON restaurant_kyc_documents (restaurant_id, doc_type)
  WHERE status IN ('PENDING','APPROVED') AND doc_type <> 'OUTLET_PHOTO';
CREATE INDEX ix_restaurant_kyc__expiry ON restaurant_kyc_documents (valid_until) WHERE status = 'APPROVED';
SELECT attach_updated_at_trigger('restaurant_kyc_documents');

CREATE TABLE restaurant_operating_hours (
  id              uuid PRIMARY KEY,
  restaurant_id   uuid NOT NULL REFERENCES restaurants(id) ON DELETE CASCADE,
  day_of_week     smallint NOT NULL CHECK (day_of_week BETWEEN 1 AND 7),
  opens_at        time NOT NULL,
  closes_at       time NOT NULL,
  created_at      timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ck_hours__nonzero CHECK (opens_at <> closes_at)
);
CREATE INDEX ix_restaurant_hours__restaurant ON restaurant_operating_hours (restaurant_id, day_of_week);

CREATE TABLE restaurant_closures (
  id              uuid PRIMARY KEY,
  restaurant_id   uuid NOT NULL REFERENCES restaurants(id),
  starts_at       timestamptz NOT NULL,
  ends_at         timestamptz NOT NULL,
  reason          text NOT NULL CHECK (reason IN ('HOLIDAY','FESTIVAL','RENOVATION','LICENCE_ISSUE','OTHER')),
  note            text,
  created_by      uuid REFERENCES users(id),
  created_at      timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ck_closures__range CHECK (ends_at > starts_at),
  CONSTRAINT ex_closures__overlap EXCLUDE USING gist (restaurant_id WITH =, tstzrange(starts_at, ends_at) WITH &&)
);

CREATE TABLE cuisines (
  id          uuid PRIMARY KEY,
  code        text NOT NULL UNIQUE CHECK (code ~ '^[A-Z][A-Z0-9_]{1,39}$'),
  name        text NOT NULL,
  name_i18n   jsonb NOT NULL DEFAULT '{}',
  sort_order  int NOT NULL DEFAULT 0,
  is_active   boolean NOT NULL DEFAULT true
);

CREATE TABLE restaurant_cuisines (
  restaurant_id  uuid NOT NULL REFERENCES restaurants(id) ON DELETE CASCADE,
  cuisine_id     uuid NOT NULL REFERENCES cuisines(id),
  is_primary     boolean NOT NULL DEFAULT false,
  PRIMARY KEY (restaurant_id, cuisine_id)
);
CREATE INDEX ix_restaurant_cuisines__cuisine ON restaurant_cuisines (cuisine_id);

CREATE TABLE menu_categories (
  id              uuid PRIMARY KEY,
  restaurant_id   uuid NOT NULL REFERENCES restaurants(id),
  name            text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 60),
  name_i18n       jsonb NOT NULL DEFAULT '{}',
  sort_order      int NOT NULL DEFAULT 0,
  is_active       boolean NOT NULL DEFAULT true,
  archived_at     timestamptz,
  version         int NOT NULL DEFAULT 1,
  created_at      timestamptz NOT NULL DEFAULT now(),
  updated_at      timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ux_menu_categories__id_restaurant UNIQUE (id, restaurant_id)
);
CREATE UNIQUE INDEX ux_menu_categories__name ON menu_categories (restaurant_id, lower(name)) WHERE archived_at IS NULL;
SELECT attach_updated_at_trigger('menu_categories');

CREATE TABLE menu_items (
  id                  uuid PRIMARY KEY,
  restaurant_id       uuid NOT NULL REFERENCES restaurants(id),
  category_id         uuid NOT NULL,
  name                text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 80),
  name_i18n           jsonb NOT NULL DEFAULT '{}',
  description         text CHECK (char_length(description) <= 300),
  description_i18n    jsonb NOT NULL DEFAULT '{}',
  veg_type            text NOT NULL CHECK (veg_type IN ('VEG','NON_VEG','EGG')),
  price_paise         bigint NOT NULL CHECK (price_paise BETWEEN 0 AND 10000000),
  packaging_paise     bigint NOT NULL DEFAULT 0 CHECK (packaging_paise BETWEEN 0 AND 10000),
  has_variants        boolean NOT NULL DEFAULT false,
  image_file_id       uuid,
  is_available        boolean NOT NULL DEFAULT true,
  unavailable_until   timestamptz,
  is_recommended      boolean NOT NULL DEFAULT false,
  spice_level         smallint CHECK (spice_level BETWEEN 0 AND 3),
  serves              smallint CHECK (serves BETWEEN 1 AND 20),
  tax_category        text NOT NULL DEFAULT 'RESTAURANT_SERVICE',
  sort_order          int NOT NULL DEFAULT 0,
  archived_at         timestamptz,
  version             int NOT NULL DEFAULT 1,
  created_at          timestamptz NOT NULL DEFAULT now(),
  updated_at          timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ux_menu_items__id_restaurant UNIQUE (id, restaurant_id),
  CONSTRAINT fk_menu_items__category FOREIGN KEY (category_id, restaurant_id)
    REFERENCES menu_categories (id, restaurant_id)
);
CREATE INDEX ix_menu_items__menu ON menu_items (restaurant_id, category_id, sort_order) WHERE archived_at IS NULL;
CREATE INDEX ix_menu_items__name_trgm ON menu_items USING gin (lower(name) gin_trgm_ops) WHERE archived_at IS NULL;
CREATE INDEX ix_menu_items__name_te_trgm ON menu_items USING gin ((name_i18n->>'te') gin_trgm_ops) WHERE archived_at IS NULL;
SELECT attach_updated_at_trigger('menu_items');

CREATE TABLE item_variants (
  id               uuid PRIMARY KEY,
  menu_item_id     uuid NOT NULL REFERENCES menu_items(id),
  name             text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 40),
  name_i18n        jsonb NOT NULL DEFAULT '{}',
  price_paise      bigint NOT NULL CHECK (price_paise BETWEEN 0 AND 10000000),
  packaging_paise  bigint CHECK (packaging_paise BETWEEN 0 AND 10000),
  is_default       boolean NOT NULL DEFAULT false,
  is_available     boolean NOT NULL DEFAULT true,
  sort_order       int NOT NULL DEFAULT 0,
  archived_at      timestamptz,
  created_at       timestamptz NOT NULL DEFAULT now(),
  updated_at       timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX ux_item_variants__default ON item_variants (menu_item_id) WHERE is_default AND archived_at IS NULL;
CREATE INDEX ix_item_variants__item ON item_variants (menu_item_id) WHERE archived_at IS NULL;
SELECT attach_updated_at_trigger('item_variants');

CREATE TABLE addon_groups (
  id             uuid PRIMARY KEY,
  restaurant_id  uuid NOT NULL REFERENCES restaurants(id),
  name           text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 60),
  name_i18n      jsonb NOT NULL DEFAULT '{}',
  min_select     smallint NOT NULL DEFAULT 0,
  max_select     smallint NOT NULL DEFAULT 1,
  archived_at    timestamptz,
  version        int NOT NULL DEFAULT 1,
  created_at     timestamptz NOT NULL DEFAULT now(),
  updated_at     timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ux_addon_groups__id_restaurant UNIQUE (id, restaurant_id),
  CONSTRAINT ck_addon_groups__minmax CHECK (min_select >= 0 AND max_select >= 1 AND min_select <= max_select AND max_select <= 20)
);
SELECT attach_updated_at_trigger('addon_groups');

CREATE TABLE addons (
  id              uuid PRIMARY KEY,
  addon_group_id  uuid NOT NULL REFERENCES addon_groups(id),
  name            text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 60),
  name_i18n       jsonb NOT NULL DEFAULT '{}',
  price_paise     bigint NOT NULL DEFAULT 0 CHECK (price_paise BETWEEN 0 AND 1000000),
  veg_type        text NOT NULL CHECK (veg_type IN ('VEG','NON_VEG','EGG')),
  is_available    boolean NOT NULL DEFAULT true,
  sort_order      int NOT NULL DEFAULT 0,
  archived_at     timestamptz,
  created_at      timestamptz NOT NULL DEFAULT now(),
  updated_at      timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ix_addons__group ON addons (addon_group_id) WHERE archived_at IS NULL;
SELECT attach_updated_at_trigger('addons');

CREATE TABLE menu_item_addon_groups (
  menu_item_id    uuid NOT NULL,
  addon_group_id  uuid NOT NULL,
  restaurant_id   uuid NOT NULL,
  sort_order      int NOT NULL DEFAULT 0,
  PRIMARY KEY (menu_item_id, addon_group_id),
  FOREIGN KEY (menu_item_id, restaurant_id)   REFERENCES menu_items (id, restaurant_id) ON DELETE CASCADE,
  FOREIGN KEY (addon_group_id, restaurant_id) REFERENCES addon_groups (id, restaurant_id)
);
CREATE INDEX ix_menu_item_addon_groups__group ON menu_item_addon_groups (addon_group_id);

-- +goose Down
DROP TABLE IF EXISTS menu_item_addon_groups;
DROP TABLE IF EXISTS addons;
DROP TABLE IF EXISTS addon_groups;
DROP TABLE IF EXISTS item_variants;
DROP TABLE IF EXISTS menu_items;
DROP TABLE IF EXISTS menu_categories;
DROP TABLE IF EXISTS restaurant_cuisines;
DROP TABLE IF EXISTS cuisines;
DROP TABLE IF EXISTS restaurant_closures;
DROP TABLE IF EXISTS restaurant_operating_hours;
DROP TABLE IF EXISTS restaurant_kyc_documents;
DROP TABLE IF EXISTS restaurant_devices;
DROP TABLE IF EXISTS leads;
DROP TABLE IF EXISTS restaurant_users;
DROP TABLE IF EXISTS staff_invites;
DROP TABLE IF EXISTS restaurants;
