# SPDX-License-Identifier: Apache-2.0
# Shared provider pins. Symlinked into each env via the module graph; env roots
# re-declare the provider block with their region. OpenTofu >= 1.8 (R23).
terraform {
  required_version = ">= 1.8.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.60"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.6"
    }
  }
}
