# SPDX-License-Identifier: Apache-2.0
# State backend — configured only when AWS exists (M6); validate uses
# -backend=false. Account rovo-nonprod.
#
# terraform {
#   backend "s3" {
#     bucket       = "rovo-nonprod-tfstate"
#     key          = "staging/terraform.tfstate"
#     region       = "ap-south-1"
#     encrypt      = true
#     use_lockfile = true
#   }
# }
