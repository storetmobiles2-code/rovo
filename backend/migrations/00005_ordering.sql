-- +goose Up
-- Module: ordering (doc 10 §6). The bill is order_charges (11 calls the lines BillLine).

CREATE TABLE orders (
  id                            uuid PRIMARY KEY,
  code                          text NOT NULL UNIQUE CHECK (code ~ '^RV-[0-9A-HJKMNP-TV-Z]{6}$'),
  city_id                       uuid NOT NULL REFERENCES cities(id),
  zone_id                       uuid NOT NULL,
  customer_id                   uuid NOT NULL REFERENCES users(id),
  restaurant_id                 uuid NOT NULL,
  quote_id                      uuid NOT NULL,
  status                        text NOT NULL CHECK (status IN ('PENDING_PAYMENT','PLACED','ACCEPTED','PREPARING',
                                  'READY_FOR_PICKUP','PICKED_UP','DELIVERED','PAYMENT_FAILED','REJECTED','CANCELLED','UNDELIVERABLE')),
  version                       int NOT NULL DEFAULT 1,
  payment_method                text NOT NULL CHECK (payment_method IN ('ONLINE','COD')),
  payment_status                text NOT NULL CHECK (payment_status IN ('PENDING','PAID','COD_DUE','COD_COLLECTED',
                                  'COD_NOT_COLLECTED','PARTIALLY_REFUNDED','REFUNDED','UNPAID')),
  currency                      char(3) NOT NULL DEFAULT 'INR',
  item_total_paise              bigint NOT NULL CHECK (item_total_paise >= 0),
  packaging_paise               bigint NOT NULL DEFAULT 0 CHECK (packaging_paise >= 0),
  delivery_fee_paise            bigint NOT NULL DEFAULT 0 CHECK (delivery_fee_paise >= 0),
  platform_fee_paise            bigint NOT NULL DEFAULT 0 CHECK (platform_fee_paise >= 0),
  small_cart_fee_paise          bigint NOT NULL DEFAULT 0 CHECK (small_cart_fee_paise >= 0),
  discount_paise                bigint NOT NULL DEFAULT 0 CHECK (discount_paise >= 0),
  discount_platform_paise       bigint NOT NULL DEFAULT 0,
  discount_restaurant_paise     bigint NOT NULL DEFAULT 0,
  tax_paise                     bigint NOT NULL DEFAULT 0 CHECK (tax_paise >= 0),
  tax_included_paise            bigint NOT NULL DEFAULT 0 CHECK (tax_included_paise >= 0),
  round_off_paise               bigint NOT NULL DEFAULT 0 CHECK (round_off_paise BETWEEN -50 AND 50),
  total_paise                   bigint NOT NULL CHECK (total_paise >= 0),
  coupon_id                     uuid,
  coupon_code                   citext,
  fee_config_id                 uuid NOT NULL,
  commission_plan_id            uuid,
  commission_bps                int NOT NULL CHECK (commission_bps BETWEEN 0 AND 3000),
  placed_via                    text NOT NULL DEFAULT 'CUSTOMER_APP' CHECK (placed_via IN ('CUSTOMER_APP','OPS_ASSISTED')),
  placed_by_admin_id            uuid REFERENCES users(id),
  restaurant_snapshot           jsonb NOT NULL,
  delivery_address_snapshot     jsonb NOT NULL,
  pickup_location               geography(Point,4326) NOT NULL,
  drop_location                 geography(Point,4326) NOT NULL,
  straight_distance_m           int NOT NULL CHECK (straight_distance_m >= 0),
  est_road_distance_m           int NOT NULL CHECK (est_road_distance_m >= 0),
  customer_note                 text CHECK (char_length(customer_note) <= 200),
  delivery_instructions         text CHECK (char_length(delivery_instructions) <= 200),
  requires_delivery_code        boolean NOT NULL DEFAULT false,
  delivery_code_enc             bytea,
  delivery_code_key_id          text,
  prep_time_min                 smallint CHECK (prep_time_min BETWEEN 5 AND 90),
  promised_eta_at               timestamptz,
  eta_at                        timestamptz,
  placed_at                     timestamptz,
  accepted_at                   timestamptz,
  preparing_at                  timestamptz,
  ready_at                      timestamptz,
  picked_up_at                  timestamptz,
  delivered_at                  timestamptz,
  closed_at                     timestamptz,
  cancelled_by_role             text CHECK (cancelled_by_role IN ('CUSTOMER','RESTAURANT','RIDER','ADMIN','SYSTEM')),
  cancelled_by_user_id          uuid REFERENCES users(id),
  cancel_reason_code            text,
  cancel_reason_text            text CHECK (char_length(cancel_reason_text) <= 500),
  fault_party                   text CHECK (fault_party IN ('CUSTOMER','RESTAURANT','RIDER','PLATFORM','NONE')),
  commission_paise              bigint,
  restaurant_net_paise          bigint,
  rider_pay_paise               bigint,
  client_app                    text,
  created_at                    timestamptz NOT NULL DEFAULT now(),
  updated_at                    timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ck_orders__total CHECK (total_paise = item_total_paise + packaging_paise + delivery_fee_paise
                                     + platform_fee_paise + small_cart_fee_paise - discount_paise + tax_paise + round_off_paise),
  CONSTRAINT ck_orders__whole_rupee CHECK (round_off_paise = 0 OR total_paise % 100 = 0),
  CONSTRAINT ck_orders__discount_split CHECK (discount_paise = discount_platform_paise + discount_restaurant_paise),
  CONSTRAINT ck_orders__terminal_reason CHECK (
    status NOT IN ('CANCELLED','REJECTED','UNDELIVERABLE','PAYMENT_FAILED')
    OR (cancel_reason_code IS NOT NULL AND cancelled_by_role IS NOT NULL AND closed_at IS NOT NULL)),
  CONSTRAINT ck_orders__cod_never_pending_payment CHECK (NOT (payment_method = 'COD' AND status = 'PENDING_PAYMENT')),
  CONSTRAINT ck_orders__delivery_code CHECK (NOT requires_delivery_code OR (payment_method = 'ONLINE' AND delivery_code_enc IS NOT NULL)),
  CONSTRAINT ck_orders__assisted CHECK ((placed_via = 'OPS_ASSISTED') = (placed_by_admin_id IS NOT NULL))
);
CREATE INDEX ix_orders__customer        ON orders (customer_id, created_at DESC);
CREATE INDEX ix_orders__restaurant      ON orders (restaurant_id, created_at DESC);
CREATE INDEX ix_orders__restaurant_live ON orders (restaurant_id, status, placed_at)
  WHERE status IN ('PLACED','ACCEPTED','PREPARING','READY_FOR_PICKUP','PICKED_UP');
CREATE INDEX ix_orders__city_live       ON orders (city_id, status, created_at)
  WHERE status IN ('PENDING_PAYMENT','PLACED','ACCEPTED','PREPARING','READY_FOR_PICKUP','PICKED_UP');
CREATE INDEX ix_orders__city_created    ON orders (city_id, created_at DESC);
CREATE UNIQUE INDEX ux_orders__quote    ON orders (quote_id);
SELECT attach_updated_at_trigger('orders');

CREATE TABLE order_items (
  id                   uuid PRIMARY KEY,
  order_id             uuid NOT NULL REFERENCES orders(id),
  line_no              smallint NOT NULL,
  menu_item_id         uuid,
  variant_id           uuid,
  name                 text NOT NULL,
  name_i18n            jsonb NOT NULL DEFAULT '{}',
  variant_name         text,
  veg_type             text NOT NULL CHECK (veg_type IN ('VEG','NON_VEG','EGG')),
  addons_snapshot      jsonb NOT NULL DEFAULT '[]',
  unit_price_paise     bigint NOT NULL CHECK (unit_price_paise >= 0),
  quantity             smallint NOT NULL CHECK (quantity BETWEEN 1 AND 20),
  packaging_paise      bigint NOT NULL DEFAULT 0 CHECK (packaging_paise >= 0),
  line_total_paise     bigint NOT NULL,
  note                 text,
  tax_category         text NOT NULL DEFAULT 'RESTAURANT_SERVICE',
  CONSTRAINT ux_order_items__line UNIQUE (order_id, line_no),
  CONSTRAINT ck_order_items__total CHECK (line_total_paise = unit_price_paise * quantity)
);

CREATE TABLE order_charges (
  id                   uuid PRIMARY KEY,
  order_id             uuid NOT NULL REFERENCES orders(id),
  line_no              smallint NOT NULL,
  component            text NOT NULL CHECK (component IN ('ITEM_TOTAL','PACKAGING','DELIVERY_FEE',
                                         'PLATFORM_FEE','SMALL_CART_FEE','DISCOUNT','TAX','ROUND_OFF')),
  label                text NOT NULL,
  label_i18n           jsonb NOT NULL DEFAULT '{}',
  amount_paise         bigint NOT NULL,
  tax_type             text CHECK (tax_type IN ('CGST','SGST','IGST','CESS')),
  taxed_component      text,
  tax_rate_bps         int,
  taxable_base_paise   bigint,
  tax_rule_id          uuid,
  is_included          boolean NOT NULL DEFAULT false,
  collected_for        text NOT NULL CHECK (collected_for IN ('RESTAURANT','PLATFORM','GOVERNMENT')),
  funded_by            text CHECK (funded_by IN ('PLATFORM','RESTAURANT')),
  CONSTRAINT ux_order_charges__line UNIQUE (order_id, line_no),
  CONSTRAINT ck_order_charges__tax CHECK ((component = 'TAX') = (tax_type IS NOT NULL AND tax_rate_bps IS NOT NULL)),
  CONSTRAINT ck_order_charges__discount CHECK (component <> 'DISCOUNT' OR (amount_paise <= 0 AND funded_by IS NOT NULL)),
  CONSTRAINT ck_order_charges__included CHECK (NOT is_included OR component = 'TAX')
);
CREATE INDEX ix_order_charges__order ON order_charges (order_id);

-- Σ order_charges (excluding informational inclusive-tax lines) must equal orders.total_paise at commit.
-- +goose StatementBegin
CREATE FUNCTION assert_order_bill_balanced() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
  oid uuid;
  s bigint;
  t bigint;
BEGIN
  IF TG_TABLE_NAME = 'orders' THEN
    oid := NEW.id;
  ELSE
    oid := COALESCE(NEW.order_id, OLD.order_id);
  END IF;
  SELECT COALESCE(SUM(amount_paise), 0) INTO s FROM order_charges WHERE order_id = oid AND NOT is_included;
  SELECT total_paise INTO t FROM orders WHERE id = oid;
  IF t IS NOT NULL AND s <> t THEN
    RAISE EXCEPTION 'order % bill (%) <> total (%)', oid, s, t USING ERRCODE = 'check_violation';
  END IF;
  RETURN NULL;
END $$;
-- +goose StatementEnd
CREATE CONSTRAINT TRIGGER trg_order_charges_balanced AFTER INSERT OR UPDATE OR DELETE ON order_charges
  DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION assert_order_bill_balanced();
CREATE CONSTRAINT TRIGGER trg_orders_bill_balanced AFTER INSERT OR UPDATE OF total_paise ON orders
  DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION assert_order_bill_balanced();

CREATE TABLE order_status_history (
  id                 uuid PRIMARY KEY,
  order_id           uuid NOT NULL REFERENCES orders(id),
  seq                int NOT NULL,
  from_status        text,
  to_status          text NOT NULL,
  command            text NOT NULL,
  actor_type         text NOT NULL CHECK (actor_type IN ('CUSTOMER','RESTAURANT','RIDER','ADMIN','SYSTEM','PAYMENT_PROVIDER')),
  actor_user_id      uuid REFERENCES users(id),
  actor_role         text,
  reason_code        text,
  reason_text        text,
  command_id         uuid,
  metadata           jsonb NOT NULL DEFAULT '{}',
  occurred_at        timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT ux_order_status_history__seq UNIQUE (order_id, seq)
);
CREATE UNIQUE INDEX ux_order_status_history__command ON order_status_history (order_id, command_id) WHERE command_id IS NOT NULL;
CREATE INDEX ix_order_status_history__occurred ON order_status_history USING brin (occurred_at);
SELECT attach_append_only_trigger('order_status_history');

-- +goose Down
DROP TABLE IF EXISTS order_status_history;
DROP TABLE IF EXISTS order_charges;
DROP TABLE IF EXISTS order_items;
DROP TABLE IF EXISTS orders;
DROP FUNCTION IF EXISTS assert_order_bill_balanced();
