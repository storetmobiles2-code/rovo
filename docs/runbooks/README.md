# rovo runbooks

Operational runbooks for running rovo. They assume the Phase-2 deployment target
(the production-like Docker Compose stack, ADR-P2-003); where cloud (AWS ECS/RDS)
differs, the AWS step is noted. The full DR runbooks (region failover, account
compromise) live in `docs/planning/23-backup-disaster-recovery.md` §8.

| Runbook | When |
|---|---|
| [deploy.md](deploy.md) | Shipping a release to the stack |
| [rollback.md](rollback.md) | Backing out a bad release |
| [backup-restore.md](backup-restore.md) | Taking and restoring database backups |
| [incident.md](incident.md) | Responding to a page/alert (one section per alert id) |

**On-call quick reference**

- Stack control: `make prod-up` / `make prod-down` / `make prod-logs`
- Health: `make smoke`, or `curl http://127.0.0.1:9090/healthz` (internal port)
- Dashboards: Grafana (`make obs-up` → <http://localhost:3000>), Prometheus
  <http://localhost:9091>
- Alert definitions: `deploy/observability/prometheus/rules/alerts.yml`
- Severity: **P1** pages on-call, **P2** on-call (business-hours ops for business
  alerts), **P3** email.
