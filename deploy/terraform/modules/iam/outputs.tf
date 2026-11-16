# SPDX-License-Identifier: Apache-2.0
output "deploy_role_arn" { value = aws_iam_role.deploy.arn }
output "ecr_push_role_arn" { value = try(aws_iam_role.ecr_push[0].arn, null) }
output "oidc_provider_arn" { value = local.oidc_arn }
