# SPDX-License-Identifier: Apache-2.0
output "vpc_id" { value = aws_vpc.this.id }
output "vpc_cidr" { value = aws_vpc.this.cidr_block }
output "public_subnet_ids" { value = aws_subnet.public[*].id }
output "app_subnet_ids" { value = aws_subnet.app[*].id }
output "db_subnet_ids" { value = aws_subnet.db[*].id }
output "task_subnet_ids" {
  description = "Where ECS tasks run: public subnets at closed pilot (no NAT), app subnets once a NAT exists."
  value       = var.nat_gateway_count > 0 ? aws_subnet.app[*].id : aws_subnet.public[*].id
}
output "task_assign_public_ip" { value = var.nat_gateway_count == 0 }
output "vpce_security_group_id" { value = aws_security_group.vpce.id }
