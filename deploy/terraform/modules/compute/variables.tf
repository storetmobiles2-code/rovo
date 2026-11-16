# SPDX-License-Identifier: Apache-2.0
variable "name_prefix" { type = string }
variable "region" { type = string }
variable "tags" {
  type    = map(string)
  default = {}
}
variable "vpc_id" { type = string }
variable "task_subnet_ids" { type = list(string) }
variable "task_assign_public_ip" {
  description = "true at closed pilot (public subnets, egress-only public IP, R28)."
  type        = bool
  default     = true
}
variable "public_subnet_ids" {
  description = "For the ALB (always public)."
  type        = list(string)
}
variable "image" {
  description = "Backend image reference, pinned by @sha256 digest in deploys."
  type        = string
}
variable "database_secret_arn" {
  description = "RDS-managed master secret (JSON with host/port/username/password)."
  type        = string
}
variable "app_secret_arns" {
  description = "map(name => secret arn) injected into the task as env vars."
  type        = map(string)
  default     = {}
}
variable "kms_key_arns" {
  type    = map(string)
  default = {}
}
variable "media_bucket_arn" { type = string }
variable "kyc_bucket_arn" { type = string }
variable "log_group_name" { type = string }
variable "app_environment" {
  description = "Non-secret ROVO_* env (ROVO_ENV, ROVO_APP_HOSTS, ...)."
  type        = map(string)
  default     = {}
}
variable "api_desired_count" {
  type    = number
  default = 2
}
variable "worker_desired_count" {
  type    = number
  default = 1
}
variable "api_cpu" {
  type    = number
  default = 256
}
variable "api_memory" {
  type    = number
  default = 512
}
variable "worker_cpu" {
  type    = number
  default = 256
}
variable "worker_memory" {
  type    = number
  default = 512
}
variable "api_max_count" {
  type    = number
  default = 4
}
variable "acm_certificate_arn" {
  description = "ALB TLS cert (origin-api.<domain>, ap-south-1). Optional for validate."
  type        = string
  default     = null
}
variable "origin_verify_secret_arn" {
  description = "Secret whose value CloudFront sends as X-Rovo-Origin-Verify."
  type        = string
  default     = null
}
