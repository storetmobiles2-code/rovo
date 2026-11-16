# SPDX-License-Identifier: Apache-2.0
variable "env" {
  type    = string
  default = "prod"
}
variable "region" {
  type    = string
  default = "ap-south-1"
}
variable "azs" {
  type    = list(string)
  default = ["ap-south-1a", "ap-south-1b"]
}
variable "domain" {
  type    = string
  default = "rovo.example" # placeholder (doc 22); real domain [OPEN]
}

# --- profile-driven (set by profiles/*.tfvars, R32) ---
variable "nat_gateway_count" {
  type    = number
  default = 0
}
variable "db_instance_class" {
  type    = string
  default = "db.t4g.small"
}
variable "db_multi_az" {
  type    = bool
  default = false
}
variable "db_allocated_storage" {
  type    = number
  default = 20
}
variable "api_desired_count" {
  type    = number
  default = 2
}
variable "api_cpu" {
  type    = number
  default = 256
}
variable "api_memory" {
  type    = number
  default = 512
}
variable "api_max_count" {
  type    = number
  default = 4
}
variable "worker_desired_count" {
  type    = number
  default = 1
}

variable "image" {
  description = "Backend image, pinned by digest at deploy time."
  type        = string
  default     = "ghcr.io/storetmobiles2-code/rovo:latest"
}
variable "alarm_email" {
  type    = string
  default = null
}
variable "acm_cloudfront_cert_arn" {
  description = "*.<domain> in us-east-1; null during validate."
  type        = string
  default     = null
}
variable "acm_alb_cert_arn" {
  description = "origin-api.<domain> in ap-south-1; null during validate."
  type        = string
  default     = null
}
