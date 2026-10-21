# SPDX-License-Identifier: Apache-2.0
# RDS PostgreSQL 17 + PostGIS. PostGIS is enabled by a migration
# (CREATE EXTENSION postgis; R22), so no option group is needed. multi_az,
# instance class and storage come from the profile (R32). The master secret is
# managed by RDS in Secrets Manager with rotation.
variable "name_prefix" { type = string }
variable "tags" {
  type    = map(string)
  default = {}
}
variable "db_subnet_ids" { type = list(string) }
variable "vpc_id" { type = string }
variable "ingress_security_group_ids" {
  description = "SGs allowed to reach 5432 (api, worker, migrate, tools)."
  type        = list(string)
}
variable "kms_key_arn" { type = string }
variable "engine_version" {
  type    = string
  default = "17.5"
}
variable "instance_class" {
  type    = string
  default = "db.t4g.small" # closed-pilot
}
variable "multi_az" {
  type    = bool
  default = false # closed-pilot Single-AZ (R32); true before Gate B
}
variable "allocated_storage" {
  type    = number
  default = 20
}
variable "max_allocated_storage" {
  type    = number
  default = 100
}
variable "backup_retention_days" {
  type    = number
  default = 14 # PITR window (doc 23 L1)
}
variable "db_name" {
  type    = string
  default = "rovo"
}
variable "master_username" {
  type    = string
  default = "rovo_root"
}

resource "aws_db_subnet_group" "this" {
  name       = "${var.name_prefix}-db"
  subnet_ids = var.db_subnet_ids
  tags       = var.tags
}

resource "aws_security_group" "db" {
  name_prefix = "${var.name_prefix}-rds-"
  description = "RDS: 5432 from app/worker/migrate/tools only; no egress"
  vpc_id      = var.vpc_id
  dynamic "ingress" {
    for_each = toset(var.ingress_security_group_ids)
    content {
      description     = "PostgreSQL from app tier"
      from_port       = 5432
      to_port         = 5432
      protocol        = "tcp"
      security_groups = [ingress.value]
    }
  }
  tags = merge(var.tags, { Name = "${var.name_prefix}-rds" })
  lifecycle { create_before_destroy = true }
}

resource "aws_db_parameter_group" "this" {
  name_prefix = "${var.name_prefix}-pg17-"
  family      = "postgres17"
  # Business deadlines use the app clock (R19); these are safety/observability.
  parameter {
    name  = "rds.force_ssl"
    value = "1"
  }
  parameter {
    name         = "shared_preload_libraries"
    value        = "pg_stat_statements,auto_explain"
    apply_method = "pending-reboot"
  }
  parameter {
    name  = "log_min_duration_statement"
    value = "500"
  }
  parameter {
    name  = "log_connections"
    value = "1"
  }
  parameter {
    name  = "idle_in_transaction_session_timeout"
    value = "60000"
  }
  lifecycle { create_before_destroy = true }
  tags = var.tags
}

resource "aws_db_instance" "this" {
  identifier     = "${var.name_prefix}-pg"
  engine         = "postgres"
  engine_version = var.engine_version
  instance_class = var.instance_class

  db_name  = var.db_name
  username = var.master_username
  # RDS creates and rotates the master password in Secrets Manager (doc 22 §6.1).
  manage_master_user_password   = true
  master_user_secret_kms_key_id = var.kms_key_arn

  allocated_storage     = var.allocated_storage
  max_allocated_storage = var.max_allocated_storage
  storage_type          = "gp3"
  storage_encrypted     = true
  kms_key_id            = var.kms_key_arn

  multi_az               = var.multi_az
  db_subnet_group_name   = aws_db_subnet_group.this.name
  vpc_security_group_ids = [aws_security_group.db.id]
  publicly_accessible    = false
  parameter_group_name   = aws_db_parameter_group.this.name

  backup_retention_period         = var.backup_retention_days
  backup_window                   = "21:30-22:00"
  maintenance_window              = "sun:22:00-sun:23:00"
  copy_tags_to_snapshot           = true
  deletion_protection             = true
  skip_final_snapshot             = false
  final_snapshot_identifier       = "${var.name_prefix}-pg-final"
  auto_minor_version_upgrade      = true
  performance_insights_enabled    = true
  enabled_cloudwatch_logs_exports = ["postgresql", "upgrade"]

  tags = merge(var.tags, { Name = "${var.name_prefix}-pg" })
}
