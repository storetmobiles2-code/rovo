# SPDX-License-Identifier: Apache-2.0
# State backend — configured only when AWS exists (M6). During Phase-2
# validation we run `tofu init -backend=false` (scripts/tf-validate.sh), so this
# is intentionally commented out. One versioned, SSE-KMS state bucket per
# workload account with native S3 locking (doc 22 §12).
#
# terraform {
#   backend "s3" {
#     bucket       = "rovo-prod-tfstate"
#     key          = "prod/terraform.tfstate"
#     region       = "ap-south-1"
#     encrypt      = true
#     use_lockfile = true
#   }
# }
