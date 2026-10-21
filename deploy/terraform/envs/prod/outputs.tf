# SPDX-License-Identifier: Apache-2.0
output "cloudfront_domain" { value = module.edge.distribution_domain_name }
output "alb_dns_name" { value = module.compute.alb_dns_name }
output "rds_endpoint" { value = module.database.endpoint }
output "web_bucket" { value = module.storage.bucket_ids["web"] }
output "deploy_role_arn" { value = module.iam.deploy_role_arn }
output "ecr_push_role_arn" { value = module.iam.ecr_push_role_arn }
