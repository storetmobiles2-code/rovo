# SPDX-License-Identifier: Apache-2.0
variable "name_prefix" { type = string }
variable "tags" {
  type    = map(string)
  default = {}
}
variable "domain" {
  description = "Base domain, e.g. rovo.example."
  type        = string
}
variable "web_bucket_regional_domain" { type = string }
variable "media_bucket_regional_domain" { type = string }
variable "alb_domain" {
  description = "origin-api.<domain> alias to the ALB (never advertised)."
  type        = string
}
variable "acm_certificate_arn" {
  description = "*.<domain> cert in us-east-1 (CloudFront). Optional for validate."
  type        = string
  default     = null
}
variable "origin_verify_secret_value" {
  description = "Sent to the ALB as X-Rovo-Origin-Verify. Marked sensitive."
  type        = string
  default     = "placeholder-set-out-of-band"
  sensitive   = true
}
variable "web_acl_arn" {
  description = "CloudFront-scoped WAF web ACL (from aws.us_east_1)."
  type        = string
  default     = null
}
