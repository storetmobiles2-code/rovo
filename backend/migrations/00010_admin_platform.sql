-- +goose Up
-- Modules: adminaudit (approval_requests, reason_codes, feature_flags, app_config,
-- report_exports, audit_logs) and shared platform tables (processed_events, idempotency_keys,
-- file_objects). DDL follows doc 10 §11.

CREATE TABLE approval_requests (
  id                 uuid PRIMARY KEY,
  city_id            uuid REFERENCES cities(id),
  action_type        text NOT NULL CHECK (action_type IN (
                        'REFUND','GOODWILL_COUPON','LEDGER_ADJUSTMENT','RIDER_CASH_ADJUSTMENT',
                        'PAYOUT_BATCH',
                        'COMMISSION_CHANGE','FEE_CONFIG_CHANGE',
                        'PAYOUT_ACCOUNT_CHANGE',
                        'ROLE_GRANT')),
  target_type        text,
  target_id          uuid,
  payload            jsonb NOT NULL,
  payload_sha256     bytea NOT NULL,
  amount_paise       bigint,
  maker_id           uuid NOT NULL REFERENCES users(id),
  maker_reason       text NOT NULL CHECK (char_length(maker_reason) >= 15),
  status             text NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING','APPROVED','REJECTED','EXPIRED','EXECUTED','FAILED')),
  checker_id         uuid REFERENCES users(id),
  checker_reason     text,
  decided_at         timestamptz,
  executed_at        timestamptz,
  execution_error    text,
  expires_at         timestamptz NOT NULL,
  is_break_glass     boolean NOT NULL DEFAULT false,
  post_review_due_at timestamptz,
  post_reviewed_by   uuid REFERENCES users(id),
  post_reviewed_at   timestamptz,
  post_review_note   text,
  version            int NOT NULL DEFAULT 1,
  created_at         timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ck_approval_requests__four_eyes CHECK (checker_id IS NULL OR checker_id <> maker_id OR is_break_glass),
  CONSTRAINT ck_approval_requests__break_glass CHECK (NOT is_break_glass OR post_review_due_at IS NOT NULL),
  CONSTRAINT ck_approval_requests__post_review CHECK (post_reviewed_by IS NULL OR post_reviewed_by <> maker_id)
);
CREATE INDEX ix_approval_requests__queue ON approval_requests (city_id, action_type, created_at) WHERE status = 'PENDING';
CREATE INDEX ix_approval_requests__post_review ON approval_requests (post_review_due_at) WHERE is_break_glass AND post_reviewed_at IS NULL;

CREATE TABLE reason_codes (
  code                   text PRIMARY KEY,
  category               text NOT NULL CHECK (category IN ('ORDER_CANCEL','ORDER_REJECT','DELIVERY_FAIL','OFFER_DECLINE',
                            'RIDER_RELEASE','REFUND','RESTAURANT_PAUSE','ZONE_PAUSE','TICKET_RESOLUTION','ACCOUNT_BLOCK')),
  allowed_actor_types    text[] NOT NULL,
  default_fault_party    text CHECK (default_fault_party IN ('CUSTOMER','RESTAURANT','RIDER','PLATFORM','NONE')),
  requires_note          boolean NOT NULL DEFAULT false,
  label                  text NOT NULL,
  label_i18n             jsonb NOT NULL DEFAULT '{}',
  customer_message_i18n  jsonb NOT NULL DEFAULT '{}',
  is_active              boolean NOT NULL DEFAULT true,
  sort_order             int NOT NULL DEFAULT 0
);

CREATE TABLE feature_flags (
  key            text PRIMARY KEY,
  description    text NOT NULL,
  enabled        boolean NOT NULL DEFAULT false,
  rules          jsonb NOT NULL DEFAULT '{}',
  owner          text,
  version        int NOT NULL DEFAULT 1,
  updated_by     uuid REFERENCES users(id),
  updated_at     timestamptz NOT NULL DEFAULT now()
);
SELECT attach_updated_at_trigger('feature_flags');

CREATE TABLE app_config (
  id             uuid PRIMARY KEY,
  scope_type     text NOT NULL CHECK (scope_type IN ('GLOBAL','CITY','ZONE','RESTAURANT')),
  scope_id       uuid,
  key            text NOT NULL CHECK (key ~ '^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$'),
  value          jsonb NOT NULL,
  version        int NOT NULL DEFAULT 1,
  updated_by     uuid REFERENCES users(id),
  updated_at     timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ux_app_config__key UNIQUE NULLS NOT DISTINCT (scope_type, scope_id, key),
  CONSTRAINT ck_app_config__scope CHECK ((scope_type = 'GLOBAL') = (scope_id IS NULL))
);
SELECT attach_updated_at_trigger('app_config');

CREATE TABLE report_exports (
  id             uuid PRIMARY KEY,
  city_id        uuid REFERENCES cities(id),
  report_type    text NOT NULL CHECK (report_type IN ('ORDERS','SETTLEMENT_STATEMENT','GST_SUMMARY','RIDER_EARNINGS',
                                                      'COD_CASH','REFUNDS','GIG_WORKER_REGISTRATION','DATA_EXPORT')),
  params         jsonb NOT NULL,
  status         text NOT NULL DEFAULT 'QUEUED' CHECK (status IN ('QUEUED','RUNNING','SUCCEEDED','FAILED','EXPIRED')),
  file_id        uuid,
  row_count      int,
  contains_pii   boolean NOT NULL DEFAULT false,
  requested_by   uuid NOT NULL REFERENCES users(id),
  error          text,
  created_at     timestamptz NOT NULL DEFAULT now(),
  completed_at   timestamptz,
  expires_at     timestamptz
);
CREATE INDEX ix_report_exports__requester ON report_exports (requested_by, created_at DESC);

CREATE TABLE audit_logs (
  id               uuid PRIMARY KEY,
  occurred_at      timestamptz NOT NULL,
  actor_type       text NOT NULL CHECK (actor_type IN ('USER','ADMIN','SYSTEM','PROVIDER_WEBHOOK','CLI')),
  actor_id         uuid,
  actor_roles      text[] NOT NULL DEFAULT '{}',
  session_id       uuid,
  request_id       text,
  trace_id         text,
  ip               inet,
  user_agent_hash  bytea,
  city_id          uuid,
  action           text NOT NULL,
  resource_type    text NOT NULL,
  resource_id      uuid,
  outcome          text NOT NULL CHECK (outcome IN ('SUCCESS','DENIED','ERROR')),
  reason           text,
  approval_id      uuid,
  changes          jsonb,
  retention_class  text NOT NULL DEFAULT 'SECURITY_1Y' CHECK (retention_class IN ('SECURITY_1Y','MONEY_8Y','OPS_2Y')),
  created_at       timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ix_audit_logs__resource ON audit_logs (resource_type, resource_id, occurred_at DESC);
CREATE INDEX ix_audit_logs__actor    ON audit_logs (actor_id, occurred_at DESC);
CREATE INDEX ix_audit_logs__action   ON audit_logs (action, occurred_at DESC);
CREATE INDEX ix_audit_logs__retention ON audit_logs (retention_class, occurred_at);
SELECT attach_append_only_trigger('audit_logs');

CREATE TABLE processed_events (
  handler        text NOT NULL,
  event_id       uuid NOT NULL,
  processed_at   timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (handler, event_id)
);
CREATE INDEX ix_processed_events__sweep ON processed_events (processed_at);

CREATE TABLE idempotency_keys (
  principal_id      text NOT NULL,
  key               text NOT NULL CHECK (char_length(key) BETWEEN 8 AND 128),
  operation_id      text NOT NULL,
  request_hash      bytea NOT NULL,
  status            text NOT NULL CHECK (status IN ('IN_PROGRESS','COMPLETED')),
  locked_until      timestamptz,
  response_status   smallint,
  response_headers  jsonb,
  response_body     jsonb,
  resource_type     text,
  resource_id       uuid,
  created_at        timestamptz NOT NULL DEFAULT now(),
  completed_at      timestamptz,
  expires_at        timestamptz NOT NULL,
  PRIMARY KEY (principal_id, key)
);
CREATE INDEX ix_idempotency_keys__expiry ON idempotency_keys (expires_at);

CREATE TABLE file_objects (
  id                 uuid PRIMARY KEY,
  bucket             text NOT NULL,
  object_key         text NOT NULL,
  purpose            text NOT NULL CHECK (purpose IN ('MENU_IMAGE','RESTAURANT_LOGO','RESTAURANT_COVER','KYC_DOC',
                                                     'TICKET_ATTACHMENT','COD_DEPOSIT_PROOF','INVOICE_PDF','REPORT_EXPORT',
                                                     'MENU_IMPORT_CSV','SETTLEMENT_FILE')),
  visibility         text NOT NULL CHECK (visibility IN ('PUBLIC_CDN','PRIVATE')),
  content_type       text NOT NULL CHECK (content_type IN ('image/jpeg','image/png','image/webp','application/pdf','text/csv')),
  size_bytes         bigint NOT NULL CHECK (size_bytes BETWEEN 1 AND 10485760),
  sha256             bytea,
  width_px           int,
  height_px          int,
  dominant_color     text CHECK (dominant_color ~ '^#[0-9a-f]{6}$'),
  variants           jsonb NOT NULL DEFAULT '[]',
  encryption         text NOT NULL DEFAULT 'PROVIDER_SSE' CHECK (encryption IN ('PROVIDER_SSE','APP_ENVELOPE')),
  dek_wrapped        bytea,
  kek_id             text,
  owner_user_id      uuid REFERENCES users(id),
  status             text NOT NULL DEFAULT 'PENDING_UPLOAD' CHECK (status IN ('PENDING_UPLOAD','QUARANTINE','READY','REJECTED','DELETED')),
  scan_result        text,
  created_at         timestamptz NOT NULL DEFAULT now(),
  updated_at         timestamptz NOT NULL DEFAULT now(),
  deleted_at         timestamptz,
  CONSTRAINT ux_file_objects__key UNIQUE (bucket, object_key),
  CONSTRAINT ck_file_objects__envelope CHECK ((encryption = 'APP_ENVELOPE') = (dek_wrapped IS NOT NULL AND kek_id IS NOT NULL)),
  CONSTRAINT ck_file_objects__kyc_private CHECK (purpose <> 'KYC_DOC'
    OR (visibility = 'PRIVATE' AND content_type IN ('image/jpeg','image/png','image/webp')))
);
CREATE INDEX ix_file_objects__pending ON file_objects (created_at) WHERE status IN ('PENDING_UPLOAD','QUARANTINE');
SELECT attach_updated_at_trigger('file_objects');

-- +goose Down
DROP TABLE IF EXISTS file_objects;
DROP TABLE IF EXISTS idempotency_keys;
DROP TABLE IF EXISTS processed_events;
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS report_exports;
DROP TABLE IF EXISTS app_config;
DROP TABLE IF EXISTS feature_flags;
DROP TABLE IF EXISTS reason_codes;
DROP TABLE IF EXISTS approval_requests;
