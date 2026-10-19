-- +goose Up
-- Least privilege for the runtime role (doc 10 §1.3). Guarded: a no-op when rovo_app does not
-- exist (local dev and CI run as the owner). The role itself is created by IaC/DevOps.
-- +goose StatementBegin
DO $$
DECLARE t text;
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'rovo_app') THEN
    RAISE NOTICE 'role rovo_app not found; skipping grants';
    RETURN;
  END IF;
  EXECUTE 'GRANT USAGE ON SCHEMA public TO rovo_app';
  EXECUTE 'GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO rovo_app';
  EXECUTE 'GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO rovo_app';
  EXECUTE 'GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO rovo_app';
  FOREACH t IN ARRAY ARRAY['order_status_history','delivery_status_history','ledger_journals','ledger_postings',
                           'audit_logs','payment_events','user_consents'] LOOP
    EXECUTE format('REVOKE UPDATE, DELETE, TRUNCATE ON %I FROM rovo_app', t);
  END LOOP;
  -- Reference data and schema bookkeeping are owner-managed.
  EXECUTE 'REVOKE INSERT, UPDATE, DELETE ON goose_db_version FROM rovo_app';
  EXECUTE 'REVOKE INSERT, UPDATE, DELETE ON tax_rules FROM rovo_app';
  EXECUTE 'ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO rovo_app';
  EXECUTE 'ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT USAGE, SELECT ON SEQUENCES TO rovo_app';
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'rovo_app') THEN
    RETURN;
  END IF;
  EXECUTE 'ALTER DEFAULT PRIVILEGES IN SCHEMA public REVOKE SELECT, INSERT, UPDATE, DELETE ON TABLES FROM rovo_app';
  EXECUTE 'ALTER DEFAULT PRIVILEGES IN SCHEMA public REVOKE USAGE, SELECT ON SEQUENCES FROM rovo_app';
  EXECUTE 'REVOKE ALL ON ALL TABLES IN SCHEMA public FROM rovo_app';
  EXECUTE 'REVOKE ALL ON ALL SEQUENCES IN SCHEMA public FROM rovo_app';
  EXECUTE 'REVOKE EXECUTE ON ALL FUNCTIONS IN SCHEMA public FROM rovo_app';
END $$;
-- +goose StatementEnd
