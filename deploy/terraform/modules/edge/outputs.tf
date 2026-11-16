# SPDX-License-Identifier: Apache-2.0
output "distribution_id" { value = aws_cloudfront_distribution.this.id }
output "distribution_arn" { value = aws_cloudfront_distribution.this.arn }
output "distribution_domain_name" { value = aws_cloudfront_distribution.this.domain_name }
output "web_acl_arn" { value = try(aws_wafv2_web_acl.this.arn, null) }
