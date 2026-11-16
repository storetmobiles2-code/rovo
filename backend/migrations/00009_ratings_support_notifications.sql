-- +goose Up
-- Modules: ratings (ratings, reviews, rating_aggregates), support (support_tickets,
-- ticket_messages), notifications (notification_templates, notifications,
-- notification_deliveries, push_subscriptions). DDL follows doc 10 §10.

CREATE TABLE ratings (
  id                uuid PRIMARY KEY,
  order_id          uuid NOT NULL,
  city_id           uuid NOT NULL REFERENCES cities(id),
  rater_user_id     uuid REFERENCES users(id),     -- nulled on erasure (doc 10 §13.1)
  target_type       text NOT NULL CHECK (target_type IN ('RESTAURANT','RIDER')),
  restaurant_id     uuid,
  rider_id          uuid,
  stars             smallint CHECK (stars BETWEEN 1 AND 5),
  thumbs_up         boolean,
  tags              text[] NOT NULL DEFAULT '{}',
  is_excluded       boolean NOT NULL DEFAULT false,
  excluded_reason   text,
  created_at        timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ux_ratings__order_target UNIQUE (order_id, target_type),
  CONSTRAINT ck_ratings__target CHECK (
       (target_type = 'RESTAURANT' AND restaurant_id IS NOT NULL AND rider_id IS NULL AND stars IS NOT NULL AND thumbs_up IS NULL)
    OR (target_type = 'RIDER'      AND rider_id IS NOT NULL AND restaurant_id IS NULL AND thumbs_up IS NOT NULL AND stars IS NULL))
);
CREATE INDEX ix_ratings__restaurant ON ratings (restaurant_id, created_at DESC) WHERE target_type = 'RESTAURANT';
CREATE INDEX ix_ratings__rider      ON ratings (rider_id, created_at DESC)      WHERE target_type = 'RIDER';

CREATE TABLE reviews (
  id                  uuid PRIMARY KEY,
  rating_id           uuid NOT NULL UNIQUE REFERENCES ratings(id),
  restaurant_id       uuid NOT NULL,
  body                text NOT NULL CHECK (char_length(body) BETWEEN 1 AND 500),
  language            text,
  profanity_flagged   boolean NOT NULL DEFAULT false,
  is_hidden           boolean NOT NULL DEFAULT false,
  hidden_by           uuid REFERENCES users(id),
  hidden_at           timestamptz,
  hidden_reason       text,
  created_at          timestamptz NOT NULL DEFAULT now(),
  updated_at          timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ix_reviews__restaurant_public ON reviews (restaurant_id, created_at DESC) WHERE NOT is_hidden AND NOT profanity_flagged;
SELECT attach_updated_at_trigger('reviews');

CREATE TABLE rating_aggregates (
  target_type     text NOT NULL CHECK (target_type IN ('RESTAURANT','RIDER')),
  target_id       uuid NOT NULL,
  rating_count    int NOT NULL DEFAULT 0,
  stars_sum       bigint NOT NULL DEFAULT 0,
  thumbs_up_count int NOT NULL DEFAULT 0,
  avg_30d         numeric(3,2),
  updated_at      timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (target_type, target_id)
);
SELECT attach_updated_at_trigger('rating_aggregates');

CREATE TABLE support_tickets (
  id                  uuid PRIMARY KEY,
  code                text NOT NULL UNIQUE CHECK (code ~ '^TK-[0-9A-HJKMNP-TV-Z]{6}$'),
  city_id             uuid NOT NULL REFERENCES cities(id),
  requester_user_id   uuid REFERENCES users(id),
  requester_app       text NOT NULL CHECK (requester_app IN ('CUSTOMER','RESTAURANT','RIDER','SYSTEM','ADMIN')),
  order_id            uuid,
  restaurant_id       uuid,
  rider_id            uuid,
  category            text NOT NULL CHECK (category IN ('MISSING_ITEMS','WRONG_ITEMS','QUALITY','LATE_DELIVERY','NOT_DELIVERED',
                        'PAYMENT_ISSUE','REFUND_STATUS','RIDER_BEHAVIOUR','RESTAURANT_CANNOT_FULFIL','UNDELIVERABLE_REQUEST',
                        'ACCOUNT','PAYOUT','COD_DEPOSIT','APP_ISSUE','DPDP_GRIEVANCE','OTHER')),
  is_dispute          boolean NOT NULL DEFAULT false,
  priority            text NOT NULL DEFAULT 'NORMAL' CHECK (priority IN ('LOW','NORMAL','HIGH','URGENT')),
  status              text NOT NULL DEFAULT 'OPEN' CHECK (status IN
                        ('OPEN','IN_PROGRESS','AWAITING_REQUESTER','AWAITING_APPROVAL','RESOLVED','CLOSED','REOPENED')),
  assigned_admin_id   uuid REFERENCES users(id),
  subject             text NOT NULL CHECK (char_length(subject) BETWEEN 1 AND 140),
  fault_party         text CHECK (fault_party IN ('CUSTOMER','RESTAURANT','RIDER','PLATFORM','NONE','UNDETERMINED')),
  resolution_code     text,
  resolution_note     text,
  compensation_offered boolean NOT NULL DEFAULT false,   -- R29: COD compensation choice is open to the customer
  compensation_amount_paise bigint CHECK (compensation_amount_paise > 0),
  refund_id           uuid,
  goodwill_coupon_id  uuid,
  sla_due_at          timestamptz NOT NULL,
  first_response_at   timestamptz,
  resolved_at         timestamptz,
  closed_at           timestamptz,
  version             int NOT NULL DEFAULT 1,
  created_at          timestamptz NOT NULL DEFAULT now(),
  updated_at          timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ix_support_tickets__queue ON support_tickets (city_id, priority, sla_due_at)
  WHERE status NOT IN ('RESOLVED','CLOSED');
CREATE INDEX ix_support_tickets__order ON support_tickets (order_id) WHERE order_id IS NOT NULL;
CREATE INDEX ix_support_tickets__requester ON support_tickets (requester_user_id, created_at DESC);
SELECT attach_updated_at_trigger('support_tickets');

CREATE TABLE ticket_messages (
  id                    uuid PRIMARY KEY,
  ticket_id             uuid NOT NULL REFERENCES support_tickets(id),
  kind                  text NOT NULL DEFAULT 'MESSAGE' CHECK (kind IN ('MESSAGE','INTERNAL_NOTE','ACTION','STATUS_CHANGE')),
  author_user_id        uuid REFERENCES users(id),
  author_type           text NOT NULL CHECK (author_type IN ('REQUESTER','AGENT','SYSTEM')),
  body                  text NOT NULL CHECK (char_length(body) <= 4000),
  attachment_file_ids   uuid[] NOT NULL DEFAULT '{}',
  action                jsonb,
  created_at            timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ix_ticket_messages__ticket ON ticket_messages (ticket_id, created_at);

CREATE TABLE notification_templates (
  id                     uuid PRIMARY KEY,
  key                    text NOT NULL,
  channel                text NOT NULL CHECK (channel IN ('IN_APP','WEB_PUSH','SMS','EMAIL','VOICE')),
  locale                 text NOT NULL,
  version                int NOT NULL,
  title                  text,
  body                   text NOT NULL,
  variables              text[] NOT NULL DEFAULT '{}',
  dlt_template_id        text,
  dlt_sender_id          text,
  provider_template_name text,
  status                 text NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('DRAFT','ACTIVE','RETIRED')),
  created_at             timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ux_notification_templates__key UNIQUE (key, channel, locale, version)
);

CREATE TABLE notifications (
  id                uuid PRIMARY KEY,
  user_id           uuid NOT NULL REFERENCES users(id),
  audience          text NOT NULL CHECK (audience IN ('customer','restaurant','rider','admin')),
  city_id           uuid REFERENCES cities(id),
  category          text NOT NULL CHECK (category IN ('ORDER','DELIVERY','PAYMENT','PROMO','ACCOUNT','SUPPORT','OPS_ALERT','PAYOUT')),
  template_key      text NOT NULL,
  params            jsonb NOT NULL DEFAULT '{}',
  locale            text NOT NULL,
  title             text NOT NULL,
  body              text NOT NULL,
  deep_link         text CHECK (deep_link IS NULL OR deep_link ~ '^/'),
  priority          text NOT NULL DEFAULT 'NORMAL' CHECK (priority IN ('HIGH','NORMAL','LOW')),
  dedupe_key        text,
  source_event_id   uuid,
  read_at           timestamptz,
  expires_at        timestamptz,
  created_at        timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX ux_notifications__dedupe ON notifications (user_id, dedupe_key) WHERE dedupe_key IS NOT NULL;
CREATE INDEX ix_notifications__inbox  ON notifications (user_id, audience, created_at DESC);
CREATE INDEX ix_notifications__unread ON notifications (user_id, audience) WHERE read_at IS NULL;

CREATE TABLE notification_deliveries (
  id                     uuid PRIMARY KEY,
  notification_id        uuid NOT NULL REFERENCES notifications(id),
  channel                text NOT NULL CHECK (channel IN ('SSE','WEB_PUSH','SMS','EMAIL','VOICE')),
  push_subscription_id   uuid,
  to_masked              text,
  provider               text,
  provider_message_id    text,
  status                 text NOT NULL DEFAULT 'QUEUED' CHECK (status IN ('QUEUED','SENT','DELIVERED','FAILED','SKIPPED','EXPIRED')),
  attempts               smallint NOT NULL DEFAULT 0,
  last_error             text,
  cost_micro_inr         bigint,
  queued_at              timestamptz NOT NULL DEFAULT now(),
  sent_at                timestamptz,
  delivered_at           timestamptz,
  created_at             timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ix_notification_deliveries__notification ON notification_deliveries (notification_id);
CREATE UNIQUE INDEX ux_notification_deliveries__provider_msg ON notification_deliveries (provider, provider_message_id)
  WHERE provider_message_id IS NOT NULL;

CREATE TABLE push_subscriptions (
  id                 uuid PRIMARY KEY,
  user_id            uuid NOT NULL REFERENCES users(id),
  device_id          uuid,
  audience           text NOT NULL CHECK (audience IN ('customer','restaurant','rider','admin')),
  kind               text NOT NULL DEFAULT 'WEB_PUSH' CHECK (kind IN ('WEB_PUSH','FCM','APNS')),
  endpoint           text NOT NULL CHECK (endpoint ~ '^https://'),
  p256dh             text,
  auth_secret        text,
  locale             text,
  failure_count      int NOT NULL DEFAULT 0,
  last_success_at    timestamptz,
  revoked_at         timestamptz,
  created_at         timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX ux_push_subscriptions__endpoint ON push_subscriptions (endpoint) WHERE revoked_at IS NULL;
CREATE INDEX ix_push_subscriptions__user ON push_subscriptions (user_id, audience) WHERE revoked_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS push_subscriptions;
DROP TABLE IF EXISTS notification_deliveries;
DROP TABLE IF EXISTS notifications;
DROP TABLE IF EXISTS notification_templates;
DROP TABLE IF EXISTS ticket_messages;
DROP TABLE IF EXISTS support_tickets;
DROP TABLE IF EXISTS rating_aggregates;
DROP TABLE IF EXISTS reviews;
DROP TABLE IF EXISTS ratings;
