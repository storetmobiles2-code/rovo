# SPDX-License-Identifier: Apache-2.0
output "cluster_name" { value = aws_ecs_cluster.this.name }
output "alb_dns_name" { value = aws_lb.this.dns_name }
output "alb_zone_id" { value = aws_lb.this.zone_id }
output "api_security_group_id" { value = aws_security_group.api.id }
output "worker_security_group_id" { value = aws_security_group.worker.id }
output "task_role_arn" { value = aws_iam_role.task.arn }
output "api_service_name" { value = aws_ecs_service.api.name }
output "worker_service_name" { value = aws_ecs_service.worker.name }
