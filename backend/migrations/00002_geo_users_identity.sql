-- +goose Up
-- Modules: geo (cities, localities, zones, waitlist), users (users, customer_profiles,
-- customer_addresses, user_consents, erasure_requests), identity (user_roles, otp_challenges,
-- devices, sessions, refresh_tokens, admin_credentials, admin_recovery_codes,
-- phone_change_requests, rate_limit_buckets). DDL follows doc 10 §3.

CREATE TABLE cities (
  id                 uuid PRIMARY KEY,
  code               text NOT NULL UNIQUE CHECK (code ~ '^[A-Z]{3,8}$'),
  slug               text NOT NULL UNIQUE CHECK (slug ~ '^[a-z0-9-]{2,40}$'),
  name               text NOT NULL,
  name_i18n          jsonb NOT NULL DEFAULT '{}' CHECK (jsonb_typeof(name_i18n) = 'object'),
  state_name         text NOT NULL,
  gst_state_code     char(2) NOT NULL CHECK (gst_state_code ~ '^[0-9]{2}$'),
  country_code       char(2) NOT NULL DEFAULT 'IN',
  timezone           text NOT NULL DEFAULT 'Asia/Kolkata',
  currency           char(3) NOT NULL DEFAULT 'INR',
  default_locale     text NOT NULL DEFAULT 'en',
  supported_locales  text[] NOT NULL DEFAULT '{en,te}',
  centroid           geography(Point,4326) NOT NULL,
  status             text NOT NULL DEFAULT 'PLANNED' CHECK (status IN ('PLANNED','LIVE','PAUSED','RETIRED')),
  launched_at        timestamptz,
  created_at         timestamptz NOT NULL DEFAULT now(),
  updated_at         timestamptz NOT NULL DEFAULT now()
);
SELECT attach_updated_at_trigger('cities');

CREATE TABLE users (
  id                    uuid PRIMARY KEY,
  kind                  text NOT NULL DEFAULT 'MEMBER' CHECK (kind IN ('MEMBER','STAFF')),
  phone_e164            text CHECK (phone_e164 ~ '^\+[1-9][0-9]{7,14}$'),
  phone_verified_at     timestamptz,
  email                 citext,
  email_verified_at     timestamptz,
  full_name             text CHECK (char_length(full_name) <= 100),
  preferred_locale      text NOT NULL DEFAULT 'en' CHECK (preferred_locale ~ '^[a-z]{2}(-[A-Z]{2})?$'),
  home_city_id          uuid REFERENCES cities(id),
  status                text NOT NULL DEFAULT 'ACTIVE'
                          CHECK (status IN ('ACTIVE','BLOCKED','DELETION_PENDING','DETACHED','ANONYMIZED')),
  status_reason         text,
  adult_declared_at     timestamptz,
  last_active_at        timestamptz,
  anonymized_at         timestamptz,
  created_at            timestamptz NOT NULL DEFAULT now(),
  updated_at            timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ck_users__identifier CHECK (
       status IN ('ANONYMIZED','DETACHED')
    OR (kind = 'MEMBER' AND phone_e164 IS NOT NULL)
    OR (kind = 'STAFF'  AND email IS NOT NULL))
);
CREATE UNIQUE INDEX ux_users__phone ON users (phone_e164) WHERE kind = 'MEMBER' AND status NOT IN ('DETACHED','ANONYMIZED');
CREATE UNIQUE INDEX ux_users__email ON users (email) WHERE kind = 'STAFF' AND status <> 'ANONYMIZED';
CREATE INDEX ix_users__name_trgm ON users USING gin (lower(full_name) gin_trgm_ops);
SELECT attach_updated_at_trigger('users');

CREATE TABLE localities (
  id               uuid PRIMARY KEY,
  city_id          uuid NOT NULL REFERENCES cities(id),
  slug             text NOT NULL,
  name             text NOT NULL,
  name_i18n        jsonb NOT NULL DEFAULT '{}',
  search_aliases   text[] NOT NULL DEFAULT '{}',
  pin_codes        text[] NOT NULL DEFAULT '{}',
  centroid         geography(Point,4326) NOT NULL,
  boundary         geometry(MultiPolygon,4326),
  is_active        boolean NOT NULL DEFAULT true,
  sort_order       int NOT NULL DEFAULT 0,
  created_at       timestamptz NOT NULL DEFAULT now(),
  updated_at       timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ux_localities__city_slug UNIQUE (city_id, slug)
);
CREATE INDEX ix_localities__name_trgm ON localities USING gin (lower(name) gin_trgm_ops);
CREATE INDEX ix_localities__aliases   ON localities USING gin (search_aliases);
CREATE INDEX ix_localities__centroid  ON localities USING gist (centroid);
SELECT attach_updated_at_trigger('localities');

CREATE TABLE zones (
  id                      uuid PRIMARY KEY,
  city_id                 uuid NOT NULL REFERENCES cities(id),
  code                    text NOT NULL CHECK (code ~ '^[A-Z0-9-]{2,20}$'),
  name                    text NOT NULL,
  name_i18n               jsonb NOT NULL DEFAULT '{}',
  boundary                geometry(MultiPolygon,4326) NOT NULL,
  priority                int NOT NULL DEFAULT 0,
  status                  text NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT','ACTIVE','INACTIVE')),
  paused_until            timestamptz,
  pause_reason            text CHECK (pause_reason IN ('RAIN','RIDER_SHORTAGE','LAW_AND_ORDER','FESTIVAL','TECHNICAL','OTHER')),
  pause_message_i18n      jsonb NOT NULL DEFAULT '{}',
  paused_by               uuid REFERENCES users(id),
  version                 int NOT NULL DEFAULT 1,
  created_by              uuid REFERENCES users(id),
  created_at              timestamptz NOT NULL DEFAULT now(),
  updated_at              timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ux_zones__city_code UNIQUE (city_id, code),
  CONSTRAINT ck_zones__valid     CHECK (ST_IsValid(boundary) AND ST_SRID(boundary) = 4326)
);
CREATE INDEX ix_zones__boundary ON zones USING gist (boundary);
CREATE INDEX ix_zones__city_active ON zones (city_id) WHERE status = 'ACTIVE';
SELECT attach_updated_at_trigger('zones');

CREATE TABLE waitlist (
  id                     uuid PRIMARY KEY,
  city_id                uuid REFERENCES cities(id),
  user_id                uuid REFERENCES users(id),
  phone_e164             text CHECK (phone_e164 ~ '^\+[1-9][0-9]{7,14}$'),
  location               geography(Point,4326) NOT NULL,
  locality_id            uuid REFERENCES localities(id),
  pin_code               char(6) CHECK (pin_code ~ '^[1-9][0-9]{5}$'),
  consent_notice_version text NOT NULL,
  status                 text NOT NULL DEFAULT 'WAITING' CHECK (status IN ('WAITING','NOTIFIED','UNSUBSCRIBED')),
  notified_at            timestamptz,
  created_at             timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ck_waitlist__contact CHECK (user_id IS NOT NULL OR phone_e164 IS NOT NULL)
);
CREATE INDEX ix_waitlist__city ON waitlist (city_id, status);
CREATE INDEX ix_waitlist__location ON waitlist USING gist (location) WHERE status = 'WAITING';

CREATE TABLE customer_profiles (
  user_id                  uuid PRIMARY KEY REFERENCES users(id),
  cod_status               text NOT NULL DEFAULT 'ENABLED' CHECK (cod_status IN ('ENABLED','DISABLED')),
  cod_disabled_reason      text,
  cod_strike_count         int NOT NULL DEFAULT 0,
  delivered_order_count    int NOT NULL DEFAULT 0,
  first_order_at           timestamptz,
  default_address_id       uuid,
  created_at               timestamptz NOT NULL DEFAULT now(),
  updated_at               timestamptz NOT NULL DEFAULT now()
);
SELECT attach_updated_at_trigger('customer_profiles');

CREATE TABLE customer_addresses (
  id                      uuid PRIMARY KEY,
  user_id                 uuid NOT NULL REFERENCES users(id),
  city_id                 uuid NOT NULL REFERENCES cities(id),
  locality_id             uuid,                                  -- ref: geo.localities
  label                   text NOT NULL CHECK (label IN ('HOME','WORK','OTHER')),
  label_custom            text CHECK (char_length(label_custom) <= 30),
  house_no                text NOT NULL CHECK (char_length(house_no) BETWEEN 1 AND 100),
  building_street         text CHECK (char_length(building_street) <= 150),
  area_text               text CHECK (char_length(area_text) <= 100),
  landmark                text NOT NULL CHECK (char_length(landmark) BETWEEN 3 AND 150),
  city_name               text NOT NULL,
  state_name              text NOT NULL,
  pin_code                char(6) NOT NULL CHECK (pin_code ~ '^[1-9][0-9]{5}$'),
  location                geography(Point,4326) NOT NULL,
  location_accuracy_m     int CHECK (location_accuracy_m >= 0),
  contact_name            text CHECK (char_length(contact_name) <= 100),
  contact_phone_e164      text CHECK (contact_phone_e164 ~ '^\+[1-9][0-9]{7,14}$'),
  delivery_instructions   text CHECK (char_length(delivery_instructions) <= 200),
  is_default              boolean NOT NULL DEFAULT false,
  last_used_at            timestamptz,
  created_at              timestamptz NOT NULL DEFAULT now(),
  updated_at              timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ix_customer_addresses__user ON customer_addresses (user_id, last_used_at DESC);
CREATE UNIQUE INDEX ux_customer_addresses__default ON customer_addresses (user_id) WHERE is_default;
SELECT attach_updated_at_trigger('customer_addresses');
ALTER TABLE customer_profiles ADD CONSTRAINT fk_customer_profiles__default_address
  FOREIGN KEY (default_address_id) REFERENCES customer_addresses(id) ON DELETE SET NULL;

CREATE TABLE user_consents (
  id               uuid PRIMARY KEY,
  user_id          uuid NOT NULL REFERENCES users(id),
  purpose          text NOT NULL CHECK (purpose IN ('TERMS','PRIVACY_NOTICE','MARKETING_SMS','MARKETING_WHATSAPP',
                                                    'MARKETING_PUSH','PRECISE_LOCATION','PARTNER_AGREEMENT')),
  notice_version   text NOT NULL,
  granted          boolean NOT NULL,
  locale           text NOT NULL,
  source           text NOT NULL,
  ip               inet,
  user_agent       text,
  created_at       timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ix_user_consents__current ON user_consents (user_id, purpose, created_at DESC);
SELECT attach_append_only_trigger('user_consents');

CREATE TABLE erasure_requests (
  id                 uuid PRIMARY KEY,
  user_id            uuid NOT NULL REFERENCES users(id),
  source             text NOT NULL CHECK (source IN ('SELF_SERVICE','SUPPORT','GRIEVANCE_OFFICER')),
  status             text NOT NULL DEFAULT 'RECEIVED' CHECK (status IN ('RECEIVED','ON_HOLD','EXECUTING','COMPLETED','REJECTED')),
  hold_reason        text CHECK (hold_reason IN ('OPEN_ORDER','OPEN_PAYOUT','OPEN_DISPUTE','LEGAL_HOLD','CASH_IN_HAND')),
  due_at             timestamptz NOT NULL,
  executed_at        timestamptz,
  executed_by        text,
  steps_completed    jsonb NOT NULL DEFAULT '[]',
  rejection_reason   text,
  created_at         timestamptz NOT NULL DEFAULT now(),
  updated_at         timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX ux_erasure_requests__open ON erasure_requests (user_id) WHERE status IN ('RECEIVED','ON_HOLD','EXECUTING');
CREATE INDEX ix_erasure_requests__due ON erasure_requests (due_at) WHERE status IN ('RECEIVED','ON_HOLD');
SELECT attach_updated_at_trigger('erasure_requests');

CREATE TABLE user_roles (
  id               uuid PRIMARY KEY,
  user_id          uuid NOT NULL REFERENCES users(id),
  role             text NOT NULL CHECK (role IN ('CUSTOMER','RESTAURANT_OWNER','RESTAURANT_STAFF','RIDER',
                                                 'ADMIN_SUPER','ADMIN_OPS','ADMIN_SUPPORT','ADMIN_FINANCE')),
  city_id          uuid REFERENCES cities(id),
  restaurant_id    uuid,                          -- ref: catalog.restaurants
  granted_by       uuid REFERENCES users(id),
  approval_id      uuid,                          -- ref: adminaudit.approval_requests
  granted_at       timestamptz NOT NULL DEFAULT now(),
  revoked_at       timestamptz,
  revoked_by       uuid REFERENCES users(id),
  revoke_reason    text,
  CONSTRAINT ck_user_roles__scope CHECK (
       (role IN ('RESTAURANT_OWNER','RESTAURANT_STAFF') AND restaurant_id IS NOT NULL)
    OR (role LIKE 'ADMIN\_%' AND restaurant_id IS NULL)
    OR (role = 'RIDER'    AND restaurant_id IS NULL AND city_id IS NOT NULL)
    OR (role = 'CUSTOMER' AND restaurant_id IS NULL AND city_id IS NULL))
);
CREATE UNIQUE INDEX ux_user_roles__active ON user_roles (user_id, role, city_id, restaurant_id)
  NULLS NOT DISTINCT WHERE revoked_at IS NULL;
CREATE INDEX ix_user_roles__restaurant ON user_roles (restaurant_id) WHERE revoked_at IS NULL AND restaurant_id IS NOT NULL;
CREATE INDEX ix_user_roles__user_live ON user_roles (user_id) WHERE revoked_at IS NULL;

-- 12 AUTH-D01/D02: admin roles only on STAFF users, member roles only on MEMBER users,
-- RIDER and RESTAURANT_* mutually exclusive on one user.
-- +goose StatementBegin
CREATE FUNCTION enforce_user_role_rules() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE k text;
BEGIN
  IF NEW.revoked_at IS NOT NULL THEN RETURN NEW; END IF;
  SELECT kind INTO k FROM users WHERE id = NEW.user_id;
  IF NEW.role LIKE 'ADMIN\_%' AND k <> 'STAFF' THEN
    RAISE EXCEPTION 'admin role % requires a STAFF user', NEW.role USING ERRCODE = 'check_violation';
  END IF;
  IF NEW.role NOT LIKE 'ADMIN\_%' AND k <> 'MEMBER' THEN
    RAISE EXCEPTION 'member role % requires a MEMBER user', NEW.role USING ERRCODE = 'check_violation';
  END IF;
  IF NEW.role = 'RIDER' AND EXISTS (SELECT 1 FROM user_roles WHERE user_id = NEW.user_id AND revoked_at IS NULL
                                    AND role IN ('RESTAURANT_OWNER','RESTAURANT_STAFF') AND id <> NEW.id) THEN
    RAISE EXCEPTION 'RIDER and RESTAURANT_* roles are mutually exclusive' USING ERRCODE = 'check_violation';
  END IF;
  IF NEW.role IN ('RESTAURANT_OWNER','RESTAURANT_STAFF') AND EXISTS (SELECT 1 FROM user_roles WHERE user_id = NEW.user_id
                                    AND revoked_at IS NULL AND role = 'RIDER' AND id <> NEW.id) THEN
    RAISE EXCEPTION 'RIDER and RESTAURANT_* roles are mutually exclusive' USING ERRCODE = 'check_violation';
  END IF;
  RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER trg_user_roles_rules BEFORE INSERT OR UPDATE ON user_roles
  FOR EACH ROW EXECUTE FUNCTION enforce_user_role_rules();

CREATE TABLE otp_challenges (
  id                    uuid PRIMARY KEY,
  phone_e164            text NOT NULL,
  purpose               text NOT NULL CHECK (purpose IN ('LOGIN','PHONE_CHANGE_OLD','PHONE_CHANGE_NEW','STEP_UP','PARTNER_AGREEMENT')),
  audience              text NOT NULL CHECK (audience IN ('customer','restaurant','rider')),
  channel               text NOT NULL CHECK (channel IN ('SMS')),
  code_hmac             bytea NOT NULL,
  attempts              smallint NOT NULL DEFAULT 0,
  max_attempts          smallint NOT NULL DEFAULT 5,
  resend_count          smallint NOT NULL DEFAULT 0,
  expires_at            timestamptz NOT NULL,
  verified_at           timestamptz,
  invalidated_at        timestamptz,
  provider              text,
  provider_message_id   text,
  request_ip            inet,
  device_id             uuid,
  created_at            timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ck_otp__attempts CHECK (attempts <= max_attempts)
);
CREATE INDEX ix_otp_challenges__phone ON otp_challenges (phone_e164, created_at DESC);
CREATE INDEX ix_otp_challenges__ip    ON otp_challenges (request_ip, created_at DESC);

CREATE TABLE devices (
  id               uuid PRIMARY KEY,
  install_id       text NOT NULL UNIQUE,
  user_id          uuid REFERENCES users(id),
  platform         text NOT NULL CHECK (platform IN ('WEB_ANDROID','WEB_IOS','WEB_DESKTOP','ANDROID','IOS')),
  app              text NOT NULL CHECK (app IN ('customer','restaurant','rider','admin')),
  app_version      text,
  user_agent       text,
  first_seen_at    timestamptz NOT NULL DEFAULT now(),
  last_seen_at     timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ix_devices__user ON devices (user_id);

CREATE TABLE sessions (
  id                    uuid PRIMARY KEY,
  user_id               uuid NOT NULL REFERENCES users(id),
  audience              text NOT NULL CHECK (audience IN ('customer','restaurant','rider','admin',
                                                     'customer_native','restaurant_native','rider_native')),
  active_context        text,
  device_id             uuid REFERENCES devices(id),
  binding               text NOT NULL DEFAULT 'STANDARD' CHECK (binding IN ('STANDARD','DEVICE_BOUND')),
  idle_timeout_s        int NOT NULL,
  absolute_timeout_s    int NOT NULL,
  device_label          text,
  amr                   text[] NOT NULL,
  step_up_at            timestamptz,
  created_ip            inet,
  last_ip               inet,
  last_seen_at          timestamptz NOT NULL DEFAULT now(),
  idle_expires_at       timestamptz NOT NULL,
  absolute_expires_at   timestamptz NOT NULL,
  revoked_at            timestamptz,
  revoke_reason         text CHECK (revoke_reason IN ('LOGOUT','LOGOUT_OTHERS','REFRESH_REUSE','ADMIN_REVOKED',
                                                      'USER_BLOCKED','PHONE_CHANGED','PASSWORD_CHANGED','GLOBAL_NOT_BEFORE',
                                                      'DEVICE_REVOKED')),
  created_at            timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ck_sessions__device_bound CHECK (binding = 'STANDARD' OR (device_id IS NOT NULL AND audience = 'restaurant'))
);
CREATE INDEX ix_sessions__user_live ON sessions (user_id) WHERE revoked_at IS NULL;

CREATE TABLE refresh_tokens (
  id              uuid PRIMARY KEY,
  session_id      uuid NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
  token_sha256    bytea NOT NULL UNIQUE,
  parent_id       uuid REFERENCES refresh_tokens(id),
  issued_at       timestamptz NOT NULL DEFAULT now(),
  used_at         timestamptz,
  expires_at      timestamptz NOT NULL
);
CREATE INDEX ix_refresh_tokens__session ON refresh_tokens (session_id);

CREATE TABLE admin_credentials (
  user_id                uuid PRIMARY KEY REFERENCES users(id),
  password_hash          text NOT NULL,
  totp_secret_enc        bytea,
  pii_key_id             text,
  totp_enrolled_at       timestamptz,
  totp_last_step         bigint,
  failed_attempts        int NOT NULL DEFAULT 0,
  locked_until           timestamptz,
  must_change_password   boolean NOT NULL DEFAULT true,
  password_changed_at    timestamptz NOT NULL DEFAULT now(),
  setup_token_sha256     bytea UNIQUE,
  setup_token_expires_at timestamptz,
  created_at             timestamptz NOT NULL DEFAULT now(),
  updated_at             timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ck_admin_credentials__totp CHECK ((totp_secret_enc IS NULL) = (pii_key_id IS NULL))
);
SELECT attach_updated_at_trigger('admin_credentials');

CREATE TABLE admin_recovery_codes (
  id            uuid PRIMARY KEY,
  user_id       uuid NOT NULL REFERENCES users(id),
  code_hash     text NOT NULL,
  used_at       timestamptz,
  created_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ix_admin_recovery_codes__user ON admin_recovery_codes (user_id) WHERE used_at IS NULL;

CREATE TABLE phone_change_requests (
  id                 uuid PRIMARY KEY,
  user_id            uuid NOT NULL REFERENCES users(id),
  old_phone_e164     text NOT NULL,
  new_phone_e164     text NOT NULL,
  old_otp_id         uuid REFERENCES otp_challenges(id),
  new_otp_id         uuid REFERENCES otp_challenges(id),
  status             text NOT NULL CHECK (status IN ('PENDING','COMPLETED','EXPIRED','CANCELLED','REVIEW')),
  expires_at         timestamptz NOT NULL,
  completed_at       timestamptz,
  created_at         timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX ux_phone_change__pending ON phone_change_requests (user_id) WHERE status = 'PENDING';

CREATE TABLE rate_limit_buckets (
  bucket_key      text NOT NULL,
  window_start    timestamptz NOT NULL,
  window_s        int NOT NULL CHECK (window_s > 0),
  hits            int NOT NULL DEFAULT 0,
  expires_at      timestamptz NOT NULL,
  PRIMARY KEY (bucket_key, window_start)
) WITH (fillfactor = 70);
CREATE INDEX ix_rate_limit_buckets__expiry ON rate_limit_buckets (expires_at);

-- +goose Down
DROP TABLE IF EXISTS rate_limit_buckets;
DROP TABLE IF EXISTS phone_change_requests;
DROP TABLE IF EXISTS admin_recovery_codes;
DROP TABLE IF EXISTS admin_credentials;
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS devices;
DROP TABLE IF EXISTS otp_challenges;
DROP TABLE IF EXISTS user_roles;
DROP FUNCTION IF EXISTS enforce_user_role_rules();
DROP TABLE IF EXISTS erasure_requests;
DROP TABLE IF EXISTS user_consents;
ALTER TABLE IF EXISTS customer_profiles DROP CONSTRAINT IF EXISTS fk_customer_profiles__default_address;
DROP TABLE IF EXISTS customer_addresses;
DROP TABLE IF EXISTS customer_profiles;
DROP TABLE IF EXISTS waitlist;
DROP TABLE IF EXISTS zones;
DROP TABLE IF EXISTS localities;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS cities;
