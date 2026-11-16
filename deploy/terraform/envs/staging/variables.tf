# SPDX-License-Identifier: Apache-2.0
# Staging (account rovo-nonprod): closed-pilot profile scaled down (doc 22 §9).
variable "env" {
  type    = string
  default = "staging"
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
  default = "staging.rovo.example"
}
variable "nat_gateway_count" {
  type    = number
  default = 0
}
variable "db_instance_class" {
  type    = string
  default = "db.t4g.micro"
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
variable "api_max_count" {
  type    = number
  default = 2
}
variable "worker_desired_count" {
  type    = number
  default = 1
}
variable "image" {
  type    = string
  default = "ghcr.io/storetmobiles2-code/rovo:main"
}
variable "alarm_email" {
  type    = string
  default = null
}
variable "acm_cloudfront_cert_arn" {
  type    = string
  default = null
}
variable "acm_alb_cert_arn" {
  type    = string
  default = null
}
