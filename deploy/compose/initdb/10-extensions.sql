-- SPDX-License-Identifier: Apache-2.0
-- Runs once when the local data volume is first created (docker-entrypoint-initdb.d).
-- Migrations still run `CREATE EXTENSION IF NOT EXISTS ...` themselves (RDS has
-- no init hook); template1 is deliberately left clean so a migration that forgets
-- an extension fails in tests instead of only in the cloud.
CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;
