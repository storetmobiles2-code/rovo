# SPDX-License-Identifier: Apache-2.0
# KMS CMKs (one per data class) and Secrets Manager "shells" — the secret
# resources exist in IaC, but their VALUES are set out-of-band by an Admin or
# rotated by RDS, so no secret material is ever in state (doc 22 §7).
variable "name_prefix" { type = string }
variable "tags" {
  type    = map(string)
  default = {}
}
variable "kms_deletion_window_days" {
  type    = number
  default = 30
}
# App secrets created as empty shells (doc 22 §7 lists these).
variable "app_secret_names" {
  type = list(string)
  default = [
    "jwt-signing-key", "quote-signing-key", "field-encryption-key",
    "refresh-token-pepper", "trusted-origin-secret", "pa-key", "pa-webhook-secret",
    "sms-key", "push-vapid-private-key", "grafana-otlp-token", "admin-bootstrap-password",
  ]
}

locals {
  kms_keys = {
    rds     = "RDS storage + snapshots"
    kyc     = "KYC private bucket (SSE-KMS)"
    secrets = "App secrets in Secrets Manager"
    backups = "Logical dumps + backup replicas"
  }
}

resource "aws_kms_key" "this" {
  for_each                = local.kms_keys
  description             = "rovo ${var.name_prefix} ${each.value}"
  deletion_window_in_days = var.kms_deletion_window_days
  enable_key_rotation     = true
  tags                    = merge(var.tags, { Name = "${var.name_prefix}-${each.key}" })
}

resource "aws_kms_alias" "this" {
  for_each      = local.kms_keys
  name          = "alias/rovo/${var.name_prefix}/${each.key}"
  target_key_id = aws_kms_key.this[each.key].key_id
}

resource "aws_secretsmanager_secret" "app" {
  for_each    = toset(var.app_secret_names)
  name        = "rovo/${var.name_prefix}/${each.value}"
  description = "rovo ${var.name_prefix} ${each.value} (value set out-of-band / rotated)"
  kms_key_id  = aws_kms_key.this["secrets"].arn
  tags        = merge(var.tags, { Name = "${var.name_prefix}-${each.value}" })
}
