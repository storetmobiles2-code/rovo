-- +goose Up
-- Module: payments (doc 10 §7, §7.1). Money-path FKs to orders per R41.

CREATE TABLE payments (
  id                     uuid PRIMARY KEY,
  order_id               uuid NOT NULL REFERENCES orders(id),
  city_id                uuid NOT NULL REFERENCES cities(id),
  provider               text NOT NULL CHECK (provider IN ('RAZORPAY','CASHFREE','PHONEPE','COD','MANUAL','FAKE')),
  status                 text NOT NULL CHECK (status IN ('CREATED','AUTHORIZED','CAPTURED','FAILED','EXPIRED',
                                                         'COD_PENDING','COD_COLLECTED','COD_NOT_COLLECTED')),
  method                 text CHECK (method IN ('UPI','CARD','NETBANKING','WALLET','EMI','PAYLATER','CASH','OTHER')),
  amount_paise           bigint NOT NULL CHECK (amount_paise > 0),
  currency               char(3) NOT NULL DEFAULT 'INR',
  amount_refunded_paise  bigint NOT NULL DEFAULT 0,
  provider_order_id      text,
  provider_payment_id    text,
  provider_fee_paise     bigint,
  provider_tax_paise     bigint,
  failure_code           text,
  expires_at             timestamptz,
  authorized_at          timestamptz,
  captured_at            timestamptz,
  version                int NOT NULL DEFAULT 1,
  created_at             timestamptz NOT NULL DEFAULT now(),
  updated_at             timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ck_payments__refund_bound CHECK (amount_refunded_paise BETWEEN 0 AND amount_paise)
);
CREATE UNIQUE INDEX ux_payments__provider_order   ON payments (provider, provider_order_id)   WHERE provider_order_id IS NOT NULL;
CREATE UNIQUE INDEX ux_payments__provider_payment ON payments (provider, provider_payment_id) WHERE provider_payment_id IS NOT NULL;
CREATE UNIQUE INDEX ux_payments__one_success      ON payments (order_id) WHERE status IN ('CAPTURED','COD_COLLECTED');
CREATE INDEX ix_payments__order   ON payments (order_id);
CREATE INDEX ix_payments__pending ON payments (expires_at) WHERE status IN ('CREATED','AUTHORIZED');
SELECT attach_updated_at_trigger('payments');

CREATE TABLE payment_attempts (
  id                    uuid PRIMARY KEY,
  payment_id            uuid NOT NULL REFERENCES payments(id),
  provider_payment_id   text NOT NULL,
  method                text,
  status                text NOT NULL CHECK (status IN ('CREATED','AUTHORIZED','CAPTURED','FAILED','REFUNDED')),
  error_code            text,
  error_description     text,
  vpa_masked            text,
  card_last4            text,
  created_at            timestamptz NOT NULL DEFAULT now(),
  updated_at            timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ux_payment_attempts__provider UNIQUE (payment_id, provider_payment_id)
);
SELECT attach_updated_at_trigger('payment_attempts');

CREATE TABLE payment_events (
  id                    uuid PRIMARY KEY,
  provider              text NOT NULL,
  provider_event_id     text NOT NULL,
  source                text NOT NULL CHECK (source IN ('WEBHOOK','POLL','CLIENT_CALLBACK')),
  event_type            text NOT NULL,
  signature_verified    boolean NOT NULL,
  payload               jsonb NOT NULL,
  raw_body_sha256       bytea NOT NULL,
  raw_body_object_key   text,
  provider_order_id     text,
  provider_payment_id   text,
  payment_id            uuid REFERENCES payments(id),
  refund_id             uuid,
  received_at           timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX ux_payment_events__dedupe ON payment_events (provider, provider_event_id) WHERE signature_verified;
CREATE INDEX ix_payment_events__payment ON payment_events (payment_id, received_at);
SELECT attach_append_only_trigger('payment_events');

CREATE TABLE refunds (
  id                     uuid PRIMARY KEY,
  payment_id             uuid NOT NULL REFERENCES payments(id),
  order_id               uuid NOT NULL REFERENCES orders(id),
  city_id                uuid NOT NULL REFERENCES cities(id),
  channel                text NOT NULL DEFAULT 'PA' CHECK (channel IN ('PA','MANUAL_UPI','MANUAL_BANK')),
  payee_vpa_enc          bytea,
  payee_vpa_masked       text,
  pii_key_id             text,
  utr_reference          text,
  amount_paise           bigint NOT NULL CHECK (amount_paise > 0),
  currency               char(3) NOT NULL DEFAULT 'INR',
  reason_code            text NOT NULL,
  reason_text            text,
  refund_key             text NOT NULL UNIQUE,
  initiated_by_type      text NOT NULL CHECK (initiated_by_type IN ('SYSTEM','ADMIN')),
  initiated_by_user_id   uuid REFERENCES users(id),
  approval_id            uuid,
  status                 text NOT NULL CHECK (status IN ('PENDING_APPROVAL','REQUESTED','PROCESSING','SUCCEEDED','FAILED','CANCELLED')),
  provider_refund_id     text,
  speed                  text CHECK (speed IN ('NORMAL','OPTIMUM','INSTANT')),
  failure_reason         text,
  requested_at           timestamptz NOT NULL DEFAULT now(),
  processed_at           timestamptz,
  version                int NOT NULL DEFAULT 1,
  created_at             timestamptz NOT NULL DEFAULT now(),
  updated_at             timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ck_refunds__manual CHECK (channel = 'PA' OR status <> 'SUCCEEDED' OR utr_reference IS NOT NULL),
  CONSTRAINT ck_refunds__upi CHECK (channel <> 'MANUAL_UPI' OR payee_vpa_enc IS NOT NULL)
);
CREATE UNIQUE INDEX ux_refunds__provider ON refunds (provider_refund_id) WHERE provider_refund_id IS NOT NULL;
CREATE INDEX ix_refunds__order ON refunds (order_id);
CREATE INDEX ix_refunds__open  ON refunds (status, requested_at) WHERE status IN ('REQUESTED','PROCESSING','FAILED');
SELECT attach_updated_at_trigger('refunds');
ALTER TABLE payment_events ADD CONSTRAINT fk_payment_events__refund FOREIGN KEY (refund_id) REFERENCES refunds(id);

CREATE TABLE transfers (
  id                     uuid PRIMARY KEY,
  order_id               uuid NOT NULL REFERENCES orders(id),
  payment_id             uuid NOT NULL REFERENCES payments(id),
  restaurant_id          uuid NOT NULL,
  city_id                uuid NOT NULL REFERENCES cities(id),
  provider               text NOT NULL CHECK (provider IN ('RAZORPAY','CASHFREE','FAKE')),
  provider_account_ref   text NOT NULL,
  provider_transfer_id   text,
  amount_paise           bigint NOT NULL CHECK (amount_paise > 0),
  currency               char(3) NOT NULL DEFAULT 'INR',
  on_hold                boolean NOT NULL DEFAULT true,
  release_at             timestamptz,
  status                 text NOT NULL DEFAULT 'CREATED' CHECK (status IN ('CREATED','PENDING','ON_HOLD','RELEASED',
                                                 'SETTLED','REVERSED','PARTIALLY_REVERSED','FAILED')),
  reversed_paise         bigint NOT NULL DEFAULT 0,
  payout_id              uuid,
  failure_reason         text,
  version                int NOT NULL DEFAULT 1,
  created_at             timestamptz NOT NULL DEFAULT now(),
  updated_at             timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ck_transfers__reversal CHECK (reversed_paise BETWEEN 0 AND amount_paise)
);
CREATE UNIQUE INDEX ux_transfers__provider ON transfers (provider, provider_transfer_id) WHERE provider_transfer_id IS NOT NULL;
CREATE UNIQUE INDEX ux_transfers__order ON transfers (order_id) WHERE status NOT IN ('FAILED','REVERSED');
CREATE INDEX ix_transfers__held ON transfers (restaurant_id, status) WHERE status IN ('ON_HOLD','PENDING');
SELECT attach_updated_at_trigger('transfers');

CREATE TABLE pa_settlements (
  id                     uuid PRIMARY KEY,
  city_id                uuid REFERENCES cities(id),
  provider               text NOT NULL,
  source                 text NOT NULL CHECK (source IN ('PA_REPORT','BANK_STATEMENT')),
  report_date            date NOT NULL,
  provider_settlement_id text,
  file_id                uuid,
  file_sha256            bytea NOT NULL,
  expected_credit_paise  bigint,
  line_count             int NOT NULL DEFAULT 0,
  status                 text NOT NULL DEFAULT 'INGESTED' CHECK (status IN ('INGESTED','MATCHING','MATCHED','EXCEPTIONS')),
  ingested_at            timestamptz NOT NULL DEFAULT now(),
  completed_at           timestamptz,
  CONSTRAINT ux_pa_settlements__file UNIQUE (provider, source, file_sha256)
);
CREATE UNIQUE INDEX ux_pa_settlements__period ON pa_settlements (provider, source, report_date, provider_settlement_id)
  NULLS NOT DISTINCT;

CREATE TABLE pa_settlement_lines (
  id                     uuid PRIMARY KEY,
  settlement_id          uuid NOT NULL REFERENCES pa_settlements(id),
  line_no                int NOT NULL,
  line_type              text NOT NULL CHECK (line_type IN ('PAYMENT','REFUND','TRANSFER','FEE','ADJUSTMENT','CHARGEBACK','BANK_CREDIT')),
  provider_entity_id     text,
  amount_paise           bigint NOT NULL,
  fee_paise              bigint,
  tax_on_fee_paise       bigint,
  utr_reference          text,
  occurred_at            timestamptz,
  match_status           text NOT NULL DEFAULT 'UNMATCHED' CHECK (match_status IN ('UNMATCHED','MATCHED','EXCEPTION','IGNORED')),
  payment_attempt_id     uuid REFERENCES payment_attempts(id),
  refund_id              uuid REFERENCES refunds(id),
  transfer_id            uuid REFERENCES transfers(id),
  journal_id             uuid,
  raw                    jsonb NOT NULL DEFAULT '{}',
  CONSTRAINT ux_pa_settlement_lines__line UNIQUE (settlement_id, line_no)
);
CREATE INDEX ix_pa_settlement_lines__entity ON pa_settlement_lines (provider_entity_id);
CREATE INDEX ix_pa_settlement_lines__unmatched ON pa_settlement_lines (settlement_id) WHERE match_status = 'UNMATCHED';

CREATE TABLE recon_exceptions (
  id                     uuid PRIMARY KEY,
  city_id                uuid REFERENCES cities(id),
  exception_type         text NOT NULL CHECK (exception_type IN ('MISSING_IN_LEDGER','MISSING_AT_PA','AMOUNT_MISMATCH',
                            'FEE_MISMATCH','CHARGEBACK','REFUND_FAILED','TRANSFER_FAILED','BANK_CREDIT_MISMATCH','OTHER')),
  settlement_line_id     uuid REFERENCES pa_settlement_lines(id),
  payment_id             uuid REFERENCES payments(id),
  refund_id              uuid REFERENCES refunds(id),
  transfer_id            uuid REFERENCES transfers(id),
  order_id               uuid,
  amount_paise           bigint,
  expected_paise         bigint,
  status                 text NOT NULL DEFAULT 'OPEN' CHECK (status IN ('OPEN','INVESTIGATING','RESOLVED','WRITTEN_OFF')),
  resolution_note        text,
  resolution_journal_id  uuid,
  assigned_admin_id      uuid REFERENCES users(id),
  version                int NOT NULL DEFAULT 1,
  raised_at              timestamptz NOT NULL DEFAULT now(),
  resolved_at            timestamptz,
  resolved_by            uuid REFERENCES users(id)
);
CREATE INDEX ix_recon_exceptions__queue ON recon_exceptions (city_id, status, raised_at) WHERE status IN ('OPEN','INVESTIGATING');

-- +goose Down
DROP TABLE IF EXISTS recon_exceptions;
DROP TABLE IF EXISTS pa_settlement_lines;
DROP TABLE IF EXISTS pa_settlements;
DROP TABLE IF EXISTS transfers;
ALTER TABLE IF EXISTS payment_events DROP CONSTRAINT IF EXISTS fk_payment_events__refund;
DROP TABLE IF EXISTS refunds;
DROP TABLE IF EXISTS payment_events;
DROP TABLE IF EXISTS payment_attempts;
DROP TABLE IF EXISTS payments;
