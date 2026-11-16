-- +goose Up
-- Module: ledger & settlement (doc 10 §9). Chart of accounts per 14 §10.2 (incl. ROUND_OFF, R8).
-- payout_batches is an implementation addition: one row per settlement run so the maker-checker
-- approval (R31 family 2) and the catch-up marker (13 §5.2) have a single target row.

CREATE TABLE ledger_accounts (
  id             uuid PRIMARY KEY,
  city_id        uuid NOT NULL REFERENCES cities(id),
  code           text NOT NULL CHECK (code IN (
                   'PA_CLEARING','BANK','RIDER_CASH_IN_HAND','CUSTOMER_ADVANCES','REFUNDS_PAYABLE','RESTAURANT_PAYABLE',
                   'RIDER_PAYABLE','GST_OUTPUT_9_5_RESTAURANT','GST_OUTPUT_9_5_DELIVERY','GST_OUTPUT_OWN','GST_INPUT_CREDIT',
                   'TDS_194O_PAYABLE','REVENUE_COMMISSION','REVENUE_DELIVERY_FEE','REVENUE_PLATFORM_FEE','EXPENSE_RIDER_PAY',
                   'EXPENSE_PA_FEES','EXPENSE_PROMOTIONS','EXPENSE_GOODWILL','ROUND_OFF','SUSPENSE')),
  account_type   text NOT NULL CHECK (account_type IN ('ASSET','LIABILITY','REVENUE','EXPENSE','EQUITY')),
  normal_side    char(1) NOT NULL CHECK (normal_side IN ('D','C')),
  owner_type     text NOT NULL CHECK (owner_type IN ('PLATFORM','RESTAURANT','RIDER')),
  owner_id       uuid,
  gstin_state    char(2) CHECK (gstin_state ~ '^[0-9]{2}$'),
  currency       char(3) NOT NULL DEFAULT 'INR',
  is_active      boolean NOT NULL DEFAULT true,
  created_at     timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ux_ledger_accounts__key UNIQUE NULLS NOT DISTINCT (city_id, code, owner_type, owner_id, gstin_state, currency),
  CONSTRAINT ck_ledger_accounts__gst CHECK ((code LIKE 'GST\_%') = (gstin_state IS NOT NULL)),
  CONSTRAINT ck_ledger_accounts__owner CHECK ((owner_type = 'PLATFORM') = (owner_id IS NULL))
);

CREATE TABLE ledger_journals (
  id                  uuid PRIMARY KEY,
  city_id             uuid NOT NULL REFERENCES cities(id),
  entry_type          text NOT NULL CHECK (entry_type IN ('PAYMENT_CAPTURED','ORDER_SETTLED','COD_COLLECTED','RIDER_EARNING',
                        'REFUND','PA_FEE','PA_SETTLEMENT','TRANSFER','PAYOUT','COD_DEPOSIT','GOODWILL_REDEEMED',
                        'CANCELLATION_COMPENSATION','ADJUSTMENT','REVERSAL')),
  adjustment_type     text CHECK (adjustment_type IN ('MG_TOPUP','PEAK_BONUS','RECOVERY','CASH_CORRECTION','WRITE_OFF','OTHER')),
  source_type         text NOT NULL,
  source_id           uuid NOT NULL,
  rule                text NOT NULL,
  idempotency_key     text NOT NULL UNIQUE,
  reverses_journal_id uuid REFERENCES ledger_journals(id),
  approval_id         uuid,
  description         text,
  occurred_at         timestamptz NOT NULL,
  accounting_date     date NOT NULL,
  created_by_user_id  uuid REFERENCES users(id),
  created_at          timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ck_ledger_journals__adjustment CHECK ((entry_type = 'ADJUSTMENT') = (adjustment_type IS NOT NULL))
);
CREATE INDEX ix_ledger_journals__source ON ledger_journals (source_type, source_id);
CREATE INDEX ix_ledger_journals__date   ON ledger_journals (city_id, accounting_date);
SELECT attach_append_only_trigger('ledger_journals');

CREATE TABLE ledger_postings (
  id             uuid PRIMARY KEY,
  journal_id     uuid NOT NULL REFERENCES ledger_journals(id),
  account_id     uuid NOT NULL REFERENCES ledger_accounts(id),
  amount_paise   bigint NOT NULL CHECK (amount_paise <> 0),   -- + debit, − credit
  currency       char(3) NOT NULL DEFAULT 'INR',
  order_id       uuid REFERENCES orders(id),
  memo           text,
  created_at     timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ix_ledger_postings__account ON ledger_postings (account_id, created_at);
CREATE INDEX ix_ledger_postings__journal ON ledger_postings (journal_id);
CREATE INDEX ix_ledger_postings__order   ON ledger_postings (order_id) WHERE order_id IS NOT NULL;
SELECT attach_append_only_trigger('ledger_postings');

-- Every journal must have >= 2 postings summing to zero at COMMIT (DB-D09). Fired from both
-- tables so a journal inserted without postings is rejected as well.
-- +goose StatementBegin
CREATE FUNCTION assert_journal_balanced() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
  jid uuid;
  s bigint;
  n int;
BEGIN
  IF TG_TABLE_NAME = 'ledger_journals' THEN
    jid := NEW.id;
  ELSE
    jid := NEW.journal_id;
  END IF;
  SELECT COALESCE(SUM(amount_paise), 0), COUNT(*) INTO s, n FROM ledger_postings WHERE journal_id = jid;
  IF s <> 0 OR n < 2 THEN
    RAISE EXCEPTION 'journal % unbalanced (sum %, lines %)', jid, s, n USING ERRCODE = 'check_violation';
  END IF;
  RETURN NULL;
END $$;
-- +goose StatementEnd
CREATE CONSTRAINT TRIGGER trg_ledger_postings_balanced AFTER INSERT ON ledger_postings
  DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION assert_journal_balanced();
CREATE CONSTRAINT TRIGGER trg_ledger_journals_balanced AFTER INSERT ON ledger_journals
  DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION assert_journal_balanced();

CREATE TABLE ledger_account_balances (
  account_id      uuid PRIMARY KEY REFERENCES ledger_accounts(id),
  balance_paise   bigint NOT NULL DEFAULT 0,
  last_journal_id uuid,
  version         bigint NOT NULL DEFAULT 0,
  updated_at      timestamptz NOT NULL DEFAULT now()
);
SELECT attach_updated_at_trigger('ledger_account_balances');

CREATE TABLE commission_plans (
  id               uuid PRIMARY KEY,
  restaurant_id    uuid NOT NULL,
  city_id          uuid NOT NULL REFERENCES cities(id),
  commission_bps   int NOT NULL CHECK (commission_bps BETWEEN 0 AND 3000),
  basis            text NOT NULL DEFAULT 'ITEM_TOTAL_NET_OF_RESTAURANT_DISCOUNT'
                     CHECK (basis IN ('ITEM_TOTAL_NET_OF_RESTAURANT_DISCOUNT','ITEM_TOTAL_PLUS_PACKAGING_NET')),
  effective_from   timestamptz NOT NULL,
  effective_to     timestamptz,
  contract_ref     text,
  approval_id      uuid,
  created_by       uuid REFERENCES users(id),
  created_at       timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ck_commission_plans__range CHECK (effective_to IS NULL OR effective_to > effective_from),
  CONSTRAINT ex_commission_plans__no_overlap EXCLUDE USING gist (restaurant_id WITH =, tstzrange(effective_from, effective_to) WITH &&)
);

CREATE TABLE payout_accounts (
  id                     uuid PRIMARY KEY,
  owner_type             text NOT NULL CHECK (owner_type IN ('RESTAURANT','RIDER')),
  owner_id               uuid NOT NULL,
  method                 text NOT NULL CHECK (method IN ('BANK_ACCOUNT','UPI')),
  account_holder_name    text NOT NULL,
  account_number_enc     bytea,
  account_number_last4   text,
  ifsc                   text CHECK (ifsc ~ '^[A-Z]{4}0[A-Z0-9]{6}$'),
  upi_vpa_enc            bytea,
  upi_vpa_masked         text,
  pii_key_id             text NOT NULL,
  verification_status    text NOT NULL DEFAULT 'UNVERIFIED' CHECK (verification_status IN ('UNVERIFIED','PENNY_DROP_OK','MANUAL_OK','FAILED')),
  is_primary             boolean NOT NULL DEFAULT false,
  status                 text NOT NULL DEFAULT 'PENDING_APPROVAL' CHECK (status IN ('PENDING_APPROVAL','ACTIVE','RETIRED')),
  approval_id            uuid,
  cooling_off_until      timestamptz,
  version                int NOT NULL DEFAULT 1,
  created_at             timestamptz NOT NULL DEFAULT now(),
  updated_at             timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ck_payout_accounts__shape CHECK (
       (method = 'BANK_ACCOUNT' AND account_number_enc IS NOT NULL AND ifsc IS NOT NULL)
    OR (method = 'UPI' AND upi_vpa_enc IS NOT NULL))
);
CREATE UNIQUE INDEX ux_payout_accounts__primary ON payout_accounts (owner_type, owner_id) WHERE is_primary AND status = 'ACTIVE';
SELECT attach_updated_at_trigger('payout_accounts');

CREATE TABLE payout_batches (
  id                 uuid PRIMARY KEY,
  city_id            uuid NOT NULL REFERENCES cities(id),
  payee_type         text NOT NULL CHECK (payee_type IN ('RESTAURANT','RIDER')),
  period_start       timestamptz NOT NULL,
  period_end         timestamptz NOT NULL,
  status             text NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT','PENDING_APPROVAL','APPROVED',
                                                  'PARTIALLY_PAID','PAID','CANCELLED')),
  payout_count       int NOT NULL DEFAULT 0,
  total_net_paise    bigint NOT NULL DEFAULT 0,
  currency           char(3) NOT NULL DEFAULT 'INR',
  approval_id        uuid,
  created_by_type    text NOT NULL CHECK (created_by_type IN ('SYSTEM','ADMIN')),
  created_by_user_id uuid REFERENCES users(id),
  version            int NOT NULL DEFAULT 1,
  created_at         timestamptz NOT NULL DEFAULT now(),
  updated_at         timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ck_payout_batches__period CHECK (period_end > period_start)
);
CREATE UNIQUE INDEX ux_payout_batches__period ON payout_batches (city_id, payee_type, period_end) WHERE status <> 'CANCELLED';
CREATE INDEX ix_payout_batches__status ON payout_batches (city_id, status, created_at DESC);
SELECT attach_updated_at_trigger('payout_batches');

CREATE TABLE payouts (
  id                       uuid PRIMARY KEY,
  city_id                  uuid NOT NULL REFERENCES cities(id),
  batch_id                 uuid NOT NULL REFERENCES payout_batches(id),
  payee_type               text NOT NULL CHECK (payee_type IN ('RESTAURANT','RIDER')),
  payee_id                 uuid NOT NULL,
  ledger_account_id        uuid NOT NULL REFERENCES ledger_accounts(id),
  period_start             timestamptz NOT NULL,
  period_end               timestamptz NOT NULL,
  gross_paise              bigint NOT NULL,
  deductions_paise         bigint NOT NULL DEFAULT 0,
  net_paise                bigint NOT NULL CHECK (net_paise > 0),
  currency                 char(3) NOT NULL DEFAULT 'INR',
  payout_account_id        uuid REFERENCES payout_accounts(id),
  payout_account_snapshot  jsonb,
  status                   text NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT','APPROVED','PAID','FAILED','CANCELLED')),
  approval_id              uuid,
  method                   text CHECK (method IN ('NEFT','IMPS','UPI','RTGS','PA_PAYOUT','PA_TRANSFER_RELEASE')),
  utr_reference            text,
  paid_at                  timestamptz,
  paid_by_user_id          uuid REFERENCES users(id),
  journal_id               uuid REFERENCES ledger_journals(id),
  version                  int NOT NULL DEFAULT 1,
  created_at               timestamptz NOT NULL DEFAULT now(),
  updated_at               timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ck_payouts__net CHECK (net_paise = gross_paise - deductions_paise),
  CONSTRAINT ck_payouts__paid CHECK (status <> 'PAID' OR (utr_reference IS NOT NULL AND paid_at IS NOT NULL AND journal_id IS NOT NULL))
);
CREATE UNIQUE INDEX ux_payouts__payee_period ON payouts (payee_type, payee_id, period_end) WHERE status <> 'CANCELLED';
CREATE INDEX ix_payouts__batch ON payouts (batch_id);
SELECT attach_updated_at_trigger('payouts');

CREATE TABLE payout_items (
  id                 uuid PRIMARY KEY,
  payout_id          uuid NOT NULL REFERENCES payouts(id) ON DELETE CASCADE,
  ledger_posting_id  uuid NOT NULL REFERENCES ledger_postings(id),
  order_id           uuid REFERENCES orders(id),
  item_type          text NOT NULL CHECK (item_type IN ('ORDER_EARNING','COMMISSION','COMMISSION_GST','TDS','REFUND_RECOVERY',
                                                        'COD_NETTING','CANCELLATION_COMPENSATION','ADJUSTMENT','PENALTY','DELIVERY_EARNING',
                                                        'MG_TOPUP','PEAK_BONUS')),
  amount_paise       bigint NOT NULL,
  created_at         timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX ux_payout_items__posting ON payout_items (ledger_posting_id);
CREATE INDEX ix_payout_items__payout ON payout_items (payout_id);

CREATE TABLE cod_deposits (
  id                 uuid PRIMARY KEY,
  rider_id           uuid NOT NULL,
  city_id            uuid NOT NULL REFERENCES cities(id),
  amount_paise       bigint NOT NULL CHECK (amount_paise > 0),
  currency           char(3) NOT NULL DEFAULT 'INR',
  method             text NOT NULL CHECK (method IN ('UPI_TO_COMPANY','BANK_DEPOSIT','CASH_AT_HUB','PAYOUT_NETTING')),
  reference          text,
  proof_file_id      uuid,
  status             text NOT NULL DEFAULT 'DECLARED' CHECK (status IN ('DECLARED','VERIFIED','REJECTED')),
  declared_at        timestamptz NOT NULL DEFAULT now(),
  verified_by        uuid REFERENCES users(id),
  verified_at        timestamptz,
  rejection_reason   text,
  journal_id         uuid REFERENCES ledger_journals(id),
  created_at         timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ck_cod_deposits__verified CHECK (status <> 'VERIFIED' OR (verified_by IS NOT NULL AND journal_id IS NOT NULL))
);
CREATE INDEX ix_cod_deposits__queue ON cod_deposits (city_id, status, declared_at) WHERE status = 'DECLARED';
CREATE INDEX ix_cod_deposits__rider ON cod_deposits (rider_id, declared_at DESC);

CREATE TABLE invoice_sequences (
  series          text NOT NULL,
  financial_year  text NOT NULL,
  next_value      bigint NOT NULL DEFAULT 1,
  PRIMARY KEY (series, financial_year)
);

CREATE TABLE invoices (
  id               uuid PRIMARY KEY,
  city_id          uuid NOT NULL REFERENCES cities(id),
  series           text NOT NULL,
  financial_year   text NOT NULL,
  number           text NOT NULL CHECK (char_length(number) <= 16),
  kind             text NOT NULL CHECK (kind IN ('CUSTOMER_FOOD','CUSTOMER_SERVICES','RESTAURANT_COMMISSION','CREDIT_NOTE')),
  order_id         uuid REFERENCES orders(id),
  payee_type       text,
  payee_id         uuid,
  total_paise      bigint NOT NULL,
  tax_paise        bigint NOT NULL,
  document         jsonb NOT NULL,
  file_id          uuid,
  issued_at        timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ux_invoices__number UNIQUE (series, financial_year, number)
);
CREATE INDEX ix_invoices__order ON invoices (order_id);

-- +goose Down
DROP TABLE IF EXISTS invoices;
DROP TABLE IF EXISTS invoice_sequences;
DROP TABLE IF EXISTS cod_deposits;
DROP TABLE IF EXISTS payout_items;
DROP TABLE IF EXISTS payouts;
DROP TABLE IF EXISTS payout_batches;
DROP TABLE IF EXISTS payout_accounts;
DROP TABLE IF EXISTS commission_plans;
DROP TABLE IF EXISTS ledger_account_balances;
DROP TABLE IF EXISTS ledger_postings;
DROP TABLE IF EXISTS ledger_journals;
DROP FUNCTION IF EXISTS assert_journal_balanced();
DROP TABLE IF EXISTS ledger_accounts;
