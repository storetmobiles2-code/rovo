# SPDX-License-Identifier: Apache-2.0
terraform {
  required_version = ">= 1.8.0"
  required_providers {
    aws = {
      source                = "hashicorp/aws"
      version               = "~> 5.60"
      configuration_aliases = [aws.us_east_1]
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.6"
    }
  }
}

provider "aws" {
  region = var.region
  default_tags {
    tags = {
      app         = "rovo"
      env         = var.env
      managed_by  = "opentofu"
      cost_center = "rovo"
    }
  }
}

# CloudFront WAF + ACM certs must live in us-east-1 (doc 22 §3.2).
provider "aws" {
  alias  = "us_east_1"
  region = "us-east-1"
  default_tags {
    tags = {
      app        = "rovo"
      env        = var.env
      managed_by = "opentofu"
    }
  }
}
