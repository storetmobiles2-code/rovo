# rovo infrastructure (OpenTofu)

IaC for staging (`rovo-nonprod`) and production (`rovo-prod`) on AWS
`ap-south-1`, per doc 22. **Authored and validated in Phase 2, not applied**
(ADR-P2-003: the Phase-2 deployment target is the production-like Docker Compose
stack; AWS needs accounts, a payment method and M6 approval).

## Layout

```
modules/
  network/        VPC, subnets (public/private-app/private-db), IGW, NAT toggle
                  (0 at closed pilot, 1 before Gate B; R28), S3 gateway +
                  interface endpoints, VPC flow logs (all traffic)
  database/       RDS PostgreSQL 17 + PostGIS via CREATE EXTENSION, param group,
                  subnet group, KMS, PITR, cross-region backup, multi_az per profile
  compute/        ECS Fargate (ARM64) cluster, api + worker services, migrate
                  one-off task, autoscaling, ALB, SGs
  edge/           CloudFront (4 app hosts + api. webhooks), S3 web bucket, OAC,
                  WAF (admin geo-IN + rate rules), /api/* behaviour with
                  origin-verify header
  storage/        secure S3 buckets (media public-read via CF, kyc private
                  SSE-KMS, exports, logs); Block Public Access on all
  secrets/        Secrets Manager shells + KMS CMKs (rds, kyc, secrets, backups)
  observability/  CloudWatch log groups (180 d app / 400 d security, CERT-In,
                  R36) + S3 archive, last-resort alarms, SNS
  iam/            GitHub OIDC provider + deploy/plan/apply roles (no long-lived keys)
envs/
  staging/        account rovo-nonprod; closed-pilot profile + staging overrides
  prod/           account rovo-prod; profile selected by -var-file
profiles/
  closed-pilot.tfvars    RDS t4g.small Single-AZ, 2x api + 1 worker, nat=0
  public-launch.tfvars   Multi-AZ, nat=1, 2x api (larger) + 2 worker
policies/         checkov config for plan-time policy checks
```

## Validate (never apply)

```
make tf-validate            # fmt -check + init -backend=false + validate per env
make tf-lint                # tflint + checkov
```

`tofu validate` downloads the AWS provider; `scripts/tf-validate.sh` routes that
through the agent proxy. No backend is configured during validation, so no
state or credentials are touched.

## Applying later (M6, when AWS exists)

1. Bootstrap state buckets + lock, GitHub OIDC provider (per account).
2. `tofu init` with the env's `backend.tf` (S3 state, versioned, SSE-KMS).
3. `tofu plan -var-file=../../profiles/closed-pilot.tfvars` (R32), reviewed.
4. Apply from CI behind the `production-infra` environment approval (doc 21 §7.3).

Promotion is by image **digest**; app deploys never change infra except task
revisions. Move `closed-pilot` -> `public-launch` before Gate B or >100 orders/day.
