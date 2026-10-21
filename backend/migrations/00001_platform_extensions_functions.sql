-- +goose Up
-- Extensions (doc 10 DB-D01). On managed services these are created once by IaC with the
-- provider admin role; IF NOT EXISTS keeps this migration a no-op there.
CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS btree_gist;
CREATE EXTENSION IF NOT EXISTS citext;
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- +goose StatementBegin
CREATE FUNCTION set_updated_at() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  NEW.updated_at := now();
  RETURN NEW;
END $$;
-- +goose StatementEnd

-- Append-only guard (doc 10 §1.2). The only sanctioned delete path is the retention sweep, which
-- runs as the table owner and disables the trigger for its own transaction (only owners can).
-- +goose StatementBegin
CREATE FUNCTION forbid_mutation() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  RAISE EXCEPTION '% is append-only', TG_TABLE_NAME USING ERRCODE = 'insufficient_privilege';
END $$;
-- +goose StatementEnd

-- Helpers so every migration attaches the standard triggers the same way.
-- +goose StatementBegin
CREATE FUNCTION attach_updated_at_trigger(tbl regclass) RETURNS void LANGUAGE plpgsql AS $$
BEGIN
  EXECUTE format('CREATE TRIGGER trg_%s_updated_at BEFORE UPDATE ON %s FOR EACH ROW EXECUTE FUNCTION set_updated_at()',
                 replace(tbl::text, '.', '_'), tbl);
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION attach_append_only_trigger(tbl regclass) RETURNS void LANGUAGE plpgsql AS $$
BEGIN
  EXECUTE format('CREATE TRIGGER trg_%s_immutable BEFORE UPDATE OR DELETE ON %s FOR EACH ROW EXECUTE FUNCTION forbid_mutation()',
                 replace(tbl::text, '.', '_'), tbl);
END $$;
-- +goose StatementEnd

-- +goose Down
DROP FUNCTION IF EXISTS attach_append_only_trigger(regclass);
DROP FUNCTION IF EXISTS attach_updated_at_trigger(regclass);
DROP FUNCTION IF EXISTS forbid_mutation();
DROP FUNCTION IF EXISTS set_updated_at();
-- Extensions are left installed: they are owned by IaC on managed services and other
-- databases on the same cluster may rely on them.
