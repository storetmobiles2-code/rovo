# SPDX-License-Identifier: Apache-2.0
output "app_log_group_name" { value = aws_cloudwatch_log_group.app.name }
output "security_log_group_name" { value = aws_cloudwatch_log_group.security.name }
output "alerts_topic_arn" { value = aws_sns_topic.alerts.arn }
