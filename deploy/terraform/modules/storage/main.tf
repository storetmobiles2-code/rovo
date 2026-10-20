# SPDX-License-Identifier: Apache-2.0
# Secure S3 buckets (doc 22 §6.3). Block Public Access on every bucket; media is
# reachable only through CloudFront OAC; kyc is SSE-KMS and private; logs/exports
# follow CERT-In retention (R36). CRR to ap-south-2 is wired by a separate DR
# apply after Gate B (C10) and omitted here.
variable "name_prefix" { type = string }
variable "tags" {
  type    = map(string)
  default = {}
}
variable "kyc_kms_key_arn" { type = string }
variable "backups_kms_key_arn" { type = string }
variable "log_retention_days" {
  type    = number
  default = 180 # CERT-In app/access logs (R36)
}
variable "security_log_retention_days" {
  type    = number
  default = 400 # security events >= 1 year (R36)
}
variable "cloudfront_distribution_arn" {
  description = "Set once the edge module exists, to scope the OAC bucket policy."
  type        = string
  default     = null
}

locals {
  buckets = {
    web     = { versioned = true, sse = "AES256", kms = null }
    media   = { versioned = true, sse = "AES256", kms = null }
    kyc     = { versioned = true, sse = "aws:kms", kms = var.kyc_kms_key_arn }
    exports = { versioned = true, sse = "aws:kms", kms = var.backups_kms_key_arn }
    logs    = { versioned = false, sse = "AES256", kms = null }
  }
}

resource "aws_s3_bucket" "this" {
  for_each      = local.buckets
  bucket        = "${var.name_prefix}-${each.key}"
  force_destroy = false
  tags          = merge(var.tags, { Name = "${var.name_prefix}-${each.key}" })
}

resource "aws_s3_bucket_public_access_block" "this" {
  for_each                = aws_s3_bucket.this
  bucket                  = each.value.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_versioning" "this" {
  for_each = { for k, v in local.buckets : k => v if v.versioned }
  bucket   = aws_s3_bucket.this[each.key].id
  versioning_configuration { status = "Enabled" }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "this" {
  for_each = aws_s3_bucket.this
  bucket   = each.value.id
  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm     = local.buckets[each.key].sse
      kms_master_key_id = local.buckets[each.key].kms
    }
    bucket_key_enabled = local.buckets[each.key].sse == "aws:kms"
  }
}

# TLS-only bucket policies; media/web additionally restricted to CloudFront OAC.
data "aws_iam_policy_document" "bucket" {
  for_each = aws_s3_bucket.this
  statement {
    sid       = "DenyInsecureTransport"
    effect    = "Deny"
    actions   = ["s3:*"]
    resources = [each.value.arn, "${each.value.arn}/*"]
    principals {
      type        = "*"
      identifiers = ["*"]
    }
    condition {
      test     = "Bool"
      variable = "aws:SecureTransport"
      values   = ["false"]
    }
  }
  dynamic "statement" {
    for_each = (contains(["web", "media"], each.key) && var.cloudfront_distribution_arn != null) ? [1] : []
    content {
      sid       = "AllowCloudFrontOAC"
      effect    = "Allow"
      actions   = ["s3:GetObject"]
      resources = ["${each.value.arn}/*"]
      principals {
        type        = "Service"
        identifiers = ["cloudfront.amazonaws.com"]
      }
      condition {
        test     = "StringEquals"
        variable = "AWS:SourceArn"
        values   = [var.cloudfront_distribution_arn]
      }
    }
  }
}

resource "aws_s3_bucket_policy" "this" {
  for_each = aws_s3_bucket.this
  bucket   = each.value.id
  policy   = data.aws_iam_policy_document.bucket[each.key].json
}

# Lifecycle: expire noncurrent app assets; expire logs at the CERT-In horizon.
resource "aws_s3_bucket_lifecycle_configuration" "web" {
  bucket = aws_s3_bucket.this["web"].id
  rule {
    id     = "expire-noncurrent"
    status = "Enabled"
    filter {}
    noncurrent_version_expiration { noncurrent_days = 30 }
  }
}
resource "aws_s3_bucket_lifecycle_configuration" "media" {
  bucket = aws_s3_bucket.this["media"].id
  rule {
    id     = "expire-noncurrent"
    status = "Enabled"
    filter {}
    noncurrent_version_expiration { noncurrent_days = 90 }
  }
}
resource "aws_s3_bucket_lifecycle_configuration" "kyc" {
  bucket = aws_s3_bucket.this["kyc"].id
  rule {
    id     = "expire-noncurrent"
    status = "Enabled"
    filter {}
    noncurrent_version_expiration { noncurrent_days = 30 }
  }
}
resource "aws_s3_bucket_lifecycle_configuration" "logs" {
  bucket = aws_s3_bucket.this["logs"].id
  rule {
    id     = "cert-in-archive"
    status = "Enabled"
    filter {}
    transition {
      days          = 30
      storage_class = "STANDARD_IA"
    }
    expiration { days = var.log_retention_days }
  }
}
