# SPDX-License-Identifier: Apache-2.0
variable "name_prefix" { type = string }
variable "tags" {
  type    = map(string)
  default = {}
}
variable "vpc_cidr" {
  type    = string
  default = "10.20.0.0/16"
}
variable "azs" {
  description = "Availability zones (two for the pilot; a third is reserved in the CIDR plan)."
  type        = list(string)
}
variable "nat_gateway_count" {
  description = "0 for closed-pilot (tasks in public subnets, compensating controls R28); 1 before Gate B; one per AZ at 10x."
  type        = number
  default     = 0
  validation {
    condition     = var.nat_gateway_count >= 0 && var.nat_gateway_count <= length(var.azs)
    error_message = "nat_gateway_count must be between 0 and the number of AZs."
  }
}
variable "flow_log_retention_days" {
  type    = number
  default = 400 # security events, CERT-In (R36)
}
