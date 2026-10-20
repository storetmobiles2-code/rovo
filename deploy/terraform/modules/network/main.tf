# SPDX-License-Identifier: Apache-2.0
# VPC with public / private-app / private-db tiers across the given AZs.
# Closed pilot: ECS tasks run in public subnets (egress-only public IP, R28);
# public-launch: tasks move to private-app behind one NAT gateway.
locals {
  az_count       = length(var.azs)
  public_cidrs   = [for i in range(local.az_count) : cidrsubnet(var.vpc_cidr, 6, i)]      # /22s
  app_cidrs      = [for i in range(local.az_count) : cidrsubnet(var.vpc_cidr, 4, i + 1)]  # /20s
  db_cidrs       = [for i in range(local.az_count) : cidrsubnet(var.vpc_cidr, 8, i + 64)] # /24s
  interface_svcs = ["ecr.api", "ecr.dkr", "secretsmanager", "logs"]
}

resource "aws_vpc" "this" {
  cidr_block           = var.vpc_cidr
  enable_dns_support   = true
  enable_dns_hostnames = true
  tags                 = merge(var.tags, { Name = "${var.name_prefix}-vpc" })
}

resource "aws_internet_gateway" "this" {
  vpc_id = aws_vpc.this.id
  tags   = merge(var.tags, { Name = "${var.name_prefix}-igw" })
}

resource "aws_subnet" "public" {
  count                   = local.az_count
  vpc_id                  = aws_vpc.this.id
  cidr_block              = local.public_cidrs[count.index]
  availability_zone       = var.azs[count.index]
  map_public_ip_on_launch = true
  tags                    = merge(var.tags, { Name = "${var.name_prefix}-public-${var.azs[count.index]}", Tier = "public" })
}

resource "aws_subnet" "app" {
  count             = local.az_count
  vpc_id            = aws_vpc.this.id
  cidr_block        = local.app_cidrs[count.index]
  availability_zone = var.azs[count.index]
  tags              = merge(var.tags, { Name = "${var.name_prefix}-app-${var.azs[count.index]}", Tier = "private-app" })
}

resource "aws_subnet" "db" {
  count             = local.az_count
  vpc_id            = aws_vpc.this.id
  cidr_block        = local.db_cidrs[count.index]
  availability_zone = var.azs[count.index]
  tags              = merge(var.tags, { Name = "${var.name_prefix}-db-${var.azs[count.index]}", Tier = "private-db" })
}

# Public route table -> IGW.
resource "aws_route_table" "public" {
  vpc_id = aws_vpc.this.id
  tags   = merge(var.tags, { Name = "${var.name_prefix}-public" })
}
resource "aws_route" "public_igw" {
  route_table_id         = aws_route_table.public.id
  destination_cidr_block = "0.0.0.0/0"
  gateway_id             = aws_internet_gateway.this.id
}
resource "aws_route_table_association" "public" {
  count          = local.az_count
  subnet_id      = aws_subnet.public[count.index].id
  route_table_id = aws_route_table.public.id
}

# Optional NAT gateways (0/1/per-AZ). Private-app subnets egress through them.
resource "aws_eip" "nat" {
  count  = var.nat_gateway_count
  domain = "vpc"
  tags   = merge(var.tags, { Name = "${var.name_prefix}-nat-${count.index}" })
}
resource "aws_nat_gateway" "this" {
  count         = var.nat_gateway_count
  allocation_id = aws_eip.nat[count.index].id
  subnet_id     = aws_subnet.public[count.index].id
  tags          = merge(var.tags, { Name = "${var.name_prefix}-nat-${count.index}" })
  depends_on    = [aws_internet_gateway.this]
}

resource "aws_route_table" "app" {
  count  = local.az_count
  vpc_id = aws_vpc.this.id
  tags   = merge(var.tags, { Name = "${var.name_prefix}-app-${var.azs[count.index]}" })
}
# Route app subnets to a NAT when one exists; share the single NAT (AZ-a) across AZs.
resource "aws_route" "app_nat" {
  count                  = var.nat_gateway_count > 0 ? local.az_count : 0
  route_table_id         = aws_route_table.app[count.index].id
  destination_cidr_block = "0.0.0.0/0"
  nat_gateway_id         = aws_nat_gateway.this[min(count.index, var.nat_gateway_count - 1)].id
}
resource "aws_route_table_association" "app" {
  count          = local.az_count
  subnet_id      = aws_subnet.app[count.index].id
  route_table_id = aws_route_table.app[count.index].id
}

# DB subnets: no internet route at all.
resource "aws_route_table" "db" {
  vpc_id = aws_vpc.this.id
  tags   = merge(var.tags, { Name = "${var.name_prefix}-db" })
}
resource "aws_route_table_association" "db" {
  count          = local.az_count
  subnet_id      = aws_subnet.db[count.index].id
  route_table_id = aws_route_table.db.id
}

# --- VPC endpoints (R28): S3 gateway + interface endpoints in AZ-a ---
resource "aws_security_group" "vpce" {
  name_prefix = "${var.name_prefix}-vpce-"
  description = "Interface VPC endpoints: 443 from within the VPC"
  vpc_id      = aws_vpc.this.id
  ingress {
    description = "HTTPS from VPC"
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = [var.vpc_cidr]
  }
  egress {
    description = "none"
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["127.0.0.1/32"]
  }
  tags = merge(var.tags, { Name = "${var.name_prefix}-vpce" })
  lifecycle { create_before_destroy = true }
}

resource "aws_vpc_endpoint" "s3" {
  vpc_id            = aws_vpc.this.id
  service_name      = "com.amazonaws.${var.region}.s3"
  vpc_endpoint_type = "Gateway"
  route_table_ids   = concat([aws_route_table.db.id], aws_route_table.app[*].id)
  tags              = merge(var.tags, { Name = "${var.name_prefix}-vpce-s3" })
}

resource "aws_vpc_endpoint" "interface" {
  for_each            = toset(local.interface_svcs)
  vpc_id              = aws_vpc.this.id
  service_name        = "com.amazonaws.${var.region}.${each.value}"
  vpc_endpoint_type   = "Interface"
  subnet_ids          = [aws_subnet.app[0].id] # one AZ (AZ-a), like the DB/NAT
  security_group_ids  = [aws_security_group.vpce.id]
  private_dns_enabled = true
  tags                = merge(var.tags, { Name = "${var.name_prefix}-vpce-${replace(each.value, ".", "-")}" })
}

variable "region" { type = string }

# --- flow logs: all traffic -> S3 archive ---
variable "flow_log_bucket_arn" {
  type    = string
  default = null
}
resource "aws_flow_log" "all" {
  count                = var.flow_log_bucket_arn == null ? 0 : 1
  log_destination      = var.flow_log_bucket_arn
  log_destination_type = "s3"
  traffic_type         = "ALL"
  vpc_id               = aws_vpc.this.id
  tags                 = merge(var.tags, { Name = "${var.name_prefix}-flowlogs" })
}
