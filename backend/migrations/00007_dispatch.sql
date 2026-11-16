-- +goose Up
-- Module: dispatch (doc 10 §8). delivery_offers.status includes REVOKED (R16).

CREATE TABLE riders (
  id                       uuid PRIMARY KEY,
  user_id                  uuid NOT NULL UNIQUE REFERENCES users(id),
  city_id                  uuid NOT NULL REFERENCES cities(id),
  home_zone_id             uuid,
  status                   text NOT NULL DEFAULT 'APPLIED' CHECK (status IN
                             ('APPLIED','UNDER_REVIEW','CHANGES_REQUESTED','ACTIVE','SUSPENDED','REJECTED','OFFBOARDED')),
  status_reason            text,
  kyc_status               text NOT NULL DEFAULT 'NOT_SUBMITTED' CHECK (kyc_status IN
                             ('NOT_SUBMITTED','PENDING','VERIFIED','REJECTED','EXPIRED')),
  vehicle_type             text NOT NULL CHECK (vehicle_type IN ('BICYCLE','MOTORCYCLE','SCOOTER','EV_SCOOTER','EV_CYCLE')),
  vehicle_reg_no           text CHECK (vehicle_reg_no ~ '^[A-Z]{2}[0-9]{1,2}[A-Z]{0,3}[0-9]{1,4}$'),
  dl_number_enc            bytea,
  dl_last4                 text,
  dl_valid_until           date,
  pii_key_id               text,
  emergency_contact_name   text,
  emergency_contact_phone  text CHECK (emergency_contact_phone ~ '^\+[1-9][0-9]{7,14}$'),
  cash_limit_paise         bigint CHECK (cash_limit_paise >= 0),
  legal_name               text,
  date_of_birth            date,
  gender                   text CHECK (gender IN ('FEMALE','MALE','TRANSGENDER','UNDISCLOSED')),
  residential_state        text,
  residential_district     text,
  ss_portal_id_enc         bytea,
  ss_portal_id_last4       text,
  ss_registration_status   text NOT NULL DEFAULT 'PENDING' CHECK (ss_registration_status IN ('NOT_REQUIRED','PENDING','EXPORTED','REGISTERED','FAILED')),
  ss_last_exported_at      timestamptz,
  approved_at              timestamptz,
  approved_by              uuid REFERENCES users(id),
  version                  int NOT NULL DEFAULT 1,
  created_at               timestamptz NOT NULL DEFAULT now(),
  updated_at               timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ck_riders__motor_needs_reg CHECK (vehicle_type IN ('BICYCLE','EV_CYCLE') OR status <> 'ACTIVE' OR vehicle_reg_no IS NOT NULL)
);
CREATE INDEX ix_riders__city_status ON riders (city_id, status);
SELECT attach_updated_at_trigger('riders');

CREATE TABLE rider_availability (
  rider_id                    uuid PRIMARY KEY REFERENCES riders(id),
  city_id                     uuid NOT NULL REFERENCES cities(id),
  state                       text NOT NULL DEFAULT 'OFFLINE' CHECK (state IN ('OFFLINE','AVAILABLE','ON_DELIVERY')),
  state_changed_at            timestamptz NOT NULL DEFAULT now(),
  offline_reason              text CHECK (offline_reason IN ('RIDER','STALE_LOCATION','MISSED_OFFERS','ADMIN_FORCED',
                                                             'SUSPENDED','LOGOUT')),
  online_since                timestamptz,
  last_location               geography(Point,4326),
  last_location_at            timestamptz,
  last_seen_at                timestamptz,
  last_location_accuracy_m    int,
  active_delivery_count       smallint NOT NULL DEFAULT 0 CHECK (active_delivery_count BETWEEN 0 AND 1),
  consecutive_missed_offers   smallint NOT NULL DEFAULT 0,
  cash_in_hand_paise          bigint NOT NULL DEFAULT 0,
  cod_blocked                 boolean NOT NULL DEFAULT false,
  updated_at                  timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ck_rider_availability__delivery CHECK ((state = 'ON_DELIVERY') = (active_delivery_count > 0))
) WITH (fillfactor = 70);
CREATE INDEX ix_rider_availability__dispatch ON rider_availability USING gist (last_location) WHERE state = 'AVAILABLE';
CREATE INDEX ix_rider_availability__city_state ON rider_availability (city_id, state);
SELECT attach_updated_at_trigger('rider_availability');

CREATE TABLE rider_kyc_documents (
  id                 uuid PRIMARY KEY,
  rider_id           uuid NOT NULL REFERENCES riders(id),
  doc_type           text NOT NULL CHECK (doc_type IN ('DRIVING_LICENSE','VEHICLE_RC','PAN_CARD','AADHAAR_MASKED','OTHER_OVD',
                                                       'BANK_PROOF','SELFIE','VEHICLE_INSURANCE','POLICE_VERIFICATION')),
  file_id            uuid NOT NULL,
  doc_number_enc     bytea,
  doc_number_last4   text,
  pii_key_id         text,
  valid_until        date,
  status             text NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING','APPROVED','REJECTED','EXPIRED','SUPERSEDED')),
  reviewed_by        uuid REFERENCES users(id),
  reviewed_at        timestamptz,
  rejection_reason   text,
  created_at         timestamptz NOT NULL DEFAULT now(),
  updated_at         timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ck_rider_kyc__no_aadhaar_number CHECK (doc_type <> 'AADHAAR_MASKED' OR doc_number_enc IS NULL)
);
CREATE UNIQUE INDEX ux_rider_kyc__current ON rider_kyc_documents (rider_id, doc_type) WHERE status IN ('PENDING','APPROVED');
CREATE INDEX ix_rider_kyc__expiry ON rider_kyc_documents (valid_until) WHERE status = 'APPROVED';
SELECT attach_updated_at_trigger('rider_kyc_documents');

CREATE TABLE rider_location_pings (
  rider_id        uuid NOT NULL,
  recorded_at     timestamptz NOT NULL,
  received_at     timestamptz NOT NULL DEFAULT now(),
  location        geography(Point,4326) NOT NULL,
  accuracy_m      int,
  speed_mps       real,
  heading_deg     smallint,
  delivery_id     uuid,
  PRIMARY KEY (rider_id, recorded_at)
);
CREATE INDEX ix_rider_location_pings__recorded ON rider_location_pings USING brin (recorded_at);
CREATE INDEX ix_rider_location_pings__delivery ON rider_location_pings (delivery_id, recorded_at) WHERE delivery_id IS NOT NULL;

CREATE TABLE deliveries (
  id                          uuid PRIMARY KEY,
  order_id                    uuid NOT NULL UNIQUE REFERENCES orders(id),
  city_id                     uuid NOT NULL REFERENCES cities(id),
  zone_id                     uuid NOT NULL,
  restaurant_id               uuid NOT NULL,
  rider_id                    uuid REFERENCES riders(id),
  status                      text NOT NULL DEFAULT 'UNASSIGNED' CHECK (status IN ('UNASSIGNED','OFFERED','ASSIGNED',
                                'AT_RESTAURANT','PICKED_UP','AT_DROP','DELIVERED','FAILED','CANCELLED')),
  version                     int NOT NULL DEFAULT 1,
  pickup_location             geography(Point,4326) NOT NULL,
  drop_location               geography(Point,4326) NOT NULL,
  est_road_distance_m         int NOT NULL,
  dispatch_after              timestamptz NOT NULL,
  dispatch_round              smallint NOT NULL DEFAULT 0,
  search_radius_m             int,
  dispatch_exhausted_at       timestamptz,
  cod_amount_paise            bigint NOT NULL DEFAULT 0 CHECK (cod_amount_paise >= 0),
  cod_collected_paise         bigint CHECK (cod_collected_paise >= 0),
  requires_delivery_code      boolean NOT NULL DEFAULT false,
  delivery_code_attempts      smallint NOT NULL DEFAULT 0 CHECK (delivery_code_attempts <= 5),
  restaurant_skipped_ready    boolean NOT NULL DEFAULT false,
  undeliverable_requested_at  timestamptz,
  undeliverable_reason_code   text,
  undeliverable_ticket_id     uuid,
  call_attempts               smallint NOT NULL DEFAULT 0,
  waiting_seconds             int,
  rider_pay_paise             bigint,
  rider_pay_breakdown         jsonb,
  assigned_at                 timestamptz,
  at_restaurant_at            timestamptz,
  picked_up_at                timestamptz,
  at_drop_at                  timestamptz,
  delivered_at                timestamptz,
  failed_at                   timestamptz,
  cancelled_at                timestamptz,
  close_reason_code           text,
  created_at                  timestamptz NOT NULL DEFAULT now(),
  updated_at                  timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ck_deliveries__rider_required CHECK (
    status NOT IN ('ASSIGNED','AT_RESTAURANT','PICKED_UP','AT_DROP','DELIVERED','FAILED') OR rider_id IS NOT NULL),
  CONSTRAINT ck_deliveries__cod_collected CHECK (status <> 'DELIVERED' OR cod_amount_paise = 0 OR cod_collected_paise = cod_amount_paise)
);
CREATE UNIQUE INDEX ux_deliveries__rider_active ON deliveries (rider_id)
  WHERE status IN ('ASSIGNED','AT_RESTAURANT','PICKED_UP','AT_DROP');
CREATE INDEX ix_deliveries__dispatch_queue ON deliveries (city_id, dispatch_after) WHERE status IN ('UNASSIGNED','OFFERED');
CREATE INDEX ix_deliveries__rider_history ON deliveries (rider_id, created_at DESC);
CREATE INDEX ix_deliveries__undeliverable ON deliveries (undeliverable_requested_at) WHERE status = 'AT_DROP' AND undeliverable_requested_at IS NOT NULL;
SELECT attach_updated_at_trigger('deliveries');

CREATE TABLE delivery_offers (
  id                   uuid PRIMARY KEY,
  delivery_id          uuid NOT NULL REFERENCES deliveries(id),
  rider_id             uuid NOT NULL REFERENCES riders(id),
  round                smallint NOT NULL,
  status               text NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING','ACCEPTED','DECLINED','EXPIRED','REVOKED')),
  close_reason         text CHECK (close_reason IN ('RIDER_ACCEPTED','RIDER_DECLINED','TIMEOUT','REVOKED_MANUAL_ASSIGN',
                                                    'REVOKED_ORDER_CANCELLED','RIDER_WENT_OFFLINE')),
  decline_reason_code  text,
  offered_at           timestamptz NOT NULL DEFAULT now(),
  expires_at           timestamptz NOT NULL,
  responded_at         timestamptz,
  rider_location       geography(Point,4326),
  rider_distance_m     int,
  score                numeric(10,3),
  est_earnings_paise   bigint NOT NULL,
  tier                 smallint NOT NULL DEFAULT 1 CHECK (tier IN (1, 2)),
  CONSTRAINT ck_delivery_offers__closed CHECK ((status = 'PENDING') = (close_reason IS NULL)),
  CONSTRAINT ck_delivery_offers__reason CHECK (
       status = 'PENDING'
    OR (status = 'ACCEPTED' AND close_reason = 'RIDER_ACCEPTED')
    OR (status = 'DECLINED' AND close_reason = 'RIDER_DECLINED')
    OR (status = 'EXPIRED'  AND close_reason = 'TIMEOUT')
    OR (status = 'REVOKED'  AND close_reason IN ('REVOKED_MANUAL_ASSIGN','REVOKED_ORDER_CANCELLED','RIDER_WENT_OFFLINE')))
);
CREATE UNIQUE INDEX ux_delivery_offers__pending_delivery ON delivery_offers (delivery_id) WHERE status = 'PENDING';
CREATE UNIQUE INDEX ux_delivery_offers__pending_rider    ON delivery_offers (rider_id)    WHERE status = 'PENDING';
CREATE INDEX ix_delivery_offers__delivery ON delivery_offers (delivery_id, offered_at);
CREATE INDEX ix_delivery_offers__rider    ON delivery_offers (rider_id, offered_at DESC);

CREATE TABLE delivery_status_history (
  id                 uuid PRIMARY KEY,
  delivery_id        uuid NOT NULL REFERENCES deliveries(id),
  seq                int NOT NULL,
  from_status        text,
  to_status          text NOT NULL,
  command            text NOT NULL,
  actor_type         text NOT NULL CHECK (actor_type IN ('RIDER','ADMIN','SYSTEM','RESTAURANT')),
  actor_user_id      uuid REFERENCES users(id),
  rider_id           uuid REFERENCES riders(id),
  location           geography(Point,4326),
  geofence_ok        boolean,
  reason_code        text,
  command_id         uuid,
  client_occurred_at timestamptz,
  metadata           jsonb NOT NULL DEFAULT '{}',
  occurred_at        timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ux_delivery_status_history__seq UNIQUE (delivery_id, seq)
);
CREATE UNIQUE INDEX ux_delivery_status_history__command ON delivery_status_history (delivery_id, command_id) WHERE command_id IS NOT NULL;
SELECT attach_append_only_trigger('delivery_status_history');

CREATE TABLE sos_events (
  id                 uuid PRIMARY KEY,
  rider_id           uuid NOT NULL REFERENCES riders(id),
  city_id            uuid NOT NULL REFERENCES cities(id),
  delivery_id        uuid REFERENCES deliveries(id),
  kind               text NOT NULL CHECK (kind IN ('ACCIDENT','UNSAFE','HARASSMENT','MEDICAL','OTHER')),
  location           geography(Point,4326),
  location_at        timestamptz,
  note               text CHECK (char_length(note) <= 500),
  status             text NOT NULL DEFAULT 'OPEN' CHECK (status IN ('OPEN','ACKNOWLEDGED','RESOLVED','FALSE_ALARM')),
  acknowledged_by    uuid REFERENCES users(id),
  acknowledged_at    timestamptz,
  resolved_by        uuid REFERENCES users(id),
  resolved_at        timestamptz,
  resolution_note    text,
  created_at         timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ix_sos_events__open ON sos_events (city_id, created_at) WHERE status IN ('OPEN','ACKNOWLEDGED');
CREATE INDEX ix_sos_events__rider ON sos_events (rider_id, created_at DESC);

CREATE TABLE contact_tap_log (
  id                 uuid PRIMARY KEY,
  delivery_id        uuid NOT NULL REFERENCES deliveries(id),
  order_id           uuid NOT NULL,
  actor_type         text NOT NULL CHECK (actor_type IN ('RIDER','CUSTOMER','RESTAURANT','ADMIN')),
  actor_user_id      uuid REFERENCES users(id),
  target_type        text NOT NULL CHECK (target_type IN ('CUSTOMER','RIDER','RESTAURANT','SUPPORT')),
  channel            text NOT NULL DEFAULT 'TEL_LINK' CHECK (channel IN ('TEL_LINK','MASKED_CALL')),
  location           geography(Point,4326),
  tapped_at          timestamptz NOT NULL,
  created_at         timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ix_contact_tap_log__delivery ON contact_tap_log (delivery_id, tapped_at);

-- +goose Down
DROP TABLE IF EXISTS contact_tap_log;
DROP TABLE IF EXISTS sos_events;
DROP TABLE IF EXISTS delivery_status_history;
DROP TABLE IF EXISTS delivery_offers;
DROP TABLE IF EXISTS deliveries;
DROP TABLE IF EXISTS rider_location_pings;
DROP TABLE IF EXISTS rider_kyc_documents;
DROP TABLE IF EXISTS rider_availability;
DROP TABLE IF EXISTS riders;
