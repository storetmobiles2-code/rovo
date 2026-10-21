# SPDX-License-Identifier: Apache-2.0
# ECS Fargate (ARM64) cluster with api + worker services, a migrate one-off task
# definition, an internet-facing ALB reachable only from CloudFront, and least-
# privilege task roles (doc 22 §5). Health via /readyz on the internal port.
data "aws_caller_identity" "current" {}

locals {
  internal_port = 9090
  http_port     = 8080
  app_env_list  = [for k, v in var.app_environment : { name = k, value = v }]
  # Secret env: DB url parts come from the RDS-managed secret's JSON keys.
  db_secrets = [
    { name = "ROVO_DB_HOST", valueFrom = "${var.database_secret_arn}:host::" },
    { name = "ROVO_DB_PORT", valueFrom = "${var.database_secret_arn}:port::" },
    { name = "ROVO_DB_USER", valueFrom = "${var.database_secret_arn}:username::" },
    { name = "ROVO_DB_PASSWORD", valueFrom = "${var.database_secret_arn}:password::" },
  ]
  app_secrets = [for name, arn in var.app_secret_arns : {
    name      = "ROVO_${upper(replace(name, "-", "_"))}"
    valueFrom = arn
  }]
}

resource "aws_ecs_cluster" "this" {
  name = "${var.name_prefix}-cluster"
  setting {
    name  = "containerInsights"
    value = "enabled"
  }
  tags = var.tags
}

resource "aws_ecs_cluster_capacity_providers" "this" {
  cluster_name       = aws_ecs_cluster.this.name
  capacity_providers = ["FARGATE", "FARGATE_SPOT"]
  default_capacity_provider_strategy {
    capacity_provider = "FARGATE"
    weight            = 1
    base              = 1
  }
}

# --- security groups ---
resource "aws_security_group" "alb" {
  name_prefix = "${var.name_prefix}-alb-"
  description = "ALB: 443 only from CloudFront origin-facing prefix list"
  vpc_id      = var.vpc_id
  tags        = merge(var.tags, { Name = "${var.name_prefix}-alb" })
  lifecycle { create_before_destroy = true }
}

data "aws_ec2_managed_prefix_list" "cloudfront" {
  name = "com.amazonaws.global.cloudfront.origin-facing"
}

resource "aws_vpc_security_group_ingress_rule" "alb_from_cloudfront" {
  security_group_id = aws_security_group.alb.id
  from_port         = 443
  to_port           = 443
  ip_protocol       = "tcp"
  prefix_list_id    = data.aws_ec2_managed_prefix_list.cloudfront.id
  description       = "HTTPS from CloudFront only"
}

resource "aws_security_group" "api" {
  name_prefix = "${var.name_prefix}-api-"
  description = "api tasks: 8080 from ALB; egress 443 + 5432"
  vpc_id      = var.vpc_id
  tags        = merge(var.tags, { Name = "${var.name_prefix}-api" })
  lifecycle { create_before_destroy = true }
}
resource "aws_vpc_security_group_ingress_rule" "api_from_alb" {
  security_group_id            = aws_security_group.api.id
  from_port                    = local.http_port
  to_port                      = local.http_port
  ip_protocol                  = "tcp"
  referenced_security_group_id = aws_security_group.alb.id
}
resource "aws_vpc_security_group_egress_rule" "api_https" {
  security_group_id = aws_security_group.api.id
  from_port         = 443
  to_port           = 443
  ip_protocol       = "tcp"
  cidr_ipv4         = "0.0.0.0/0" # app-level host allow-list enforces egress (R28)
  description       = "Providers, Secrets/ECR/Logs endpoints, Grafana OTLP"
}

resource "aws_vpc_security_group_egress_rule" "alb_to_api" {
  security_group_id            = aws_security_group.alb.id
  from_port                    = local.http_port
  to_port                      = local.http_port
  ip_protocol                  = "tcp"
  referenced_security_group_id = aws_security_group.api.id
}

resource "aws_security_group" "worker" {
  name_prefix = "${var.name_prefix}-worker-"
  description = "worker tasks: no inbound; egress 443 + 5432"
  vpc_id      = var.vpc_id
  tags        = merge(var.tags, { Name = "${var.name_prefix}-worker" })
  lifecycle { create_before_destroy = true }
}
resource "aws_vpc_security_group_egress_rule" "worker_https" {
  security_group_id = aws_security_group.worker.id
  from_port         = 443
  to_port           = 443
  ip_protocol       = "tcp"
  cidr_ipv4         = "0.0.0.0/0"
}

# --- IAM roles ---
data "aws_iam_policy_document" "task_assume" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["ecs-tasks.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "execution" {
  name               = "${var.name_prefix}-task-exec"
  assume_role_policy = data.aws_iam_policy_document.task_assume.json
  tags               = var.tags
}
resource "aws_iam_role_policy_attachment" "execution" {
  role       = aws_iam_role.execution.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"
}
# Execution role also reads the secrets it injects + decrypts them.
data "aws_iam_policy_document" "execution_secrets" {
  statement {
    actions   = ["secretsmanager:GetSecretValue"]
    resources = concat([var.database_secret_arn], values(var.app_secret_arns))
  }
  dynamic "statement" {
    for_each = length(var.kms_key_arns) > 0 ? [1] : []
    content {
      actions   = ["kms:Decrypt"]
      resources = values(var.kms_key_arns)
    }
  }
}
resource "aws_iam_role_policy" "execution_secrets" {
  name   = "secrets"
  role   = aws_iam_role.execution.id
  policy = data.aws_iam_policy_document.execution_secrets.json
}

resource "aws_iam_role" "task" {
  name               = "${var.name_prefix}-task"
  assume_role_policy = data.aws_iam_policy_document.task_assume.json
  tags               = var.tags
}
data "aws_iam_policy_document" "task" {
  statement {
    sid       = "MediaReadWrite"
    actions   = ["s3:GetObject", "s3:PutObject", "s3:DeleteObject"]
    resources = ["${var.media_bucket_arn}/*"]
  }
  statement {
    sid       = "KycReadWrite"
    actions   = ["s3:GetObject", "s3:PutObject"]
    resources = ["${var.kyc_bucket_arn}/*"]
  }
  statement {
    sid       = "ListMediaKyc"
    actions   = ["s3:ListBucket"]
    resources = [var.media_bucket_arn, var.kyc_bucket_arn]
  }
  dynamic "statement" {
    for_each = contains(keys(var.kms_key_arns), "kyc") ? [1] : []
    content {
      sid       = "KycKms"
      actions   = ["kms:Decrypt", "kms:GenerateDataKey"]
      resources = [var.kms_key_arns["kyc"]]
    }
  }
}
resource "aws_iam_role_policy" "task" {
  name   = "app"
  role   = aws_iam_role.task.id
  policy = data.aws_iam_policy_document.task.json
}

# --- ALB (reachable only from CloudFront) ---
resource "aws_lb" "this" {
  name                       = "${var.name_prefix}-alb"
  internal                   = false
  load_balancer_type         = "application"
  security_groups            = [aws_security_group.alb.id]
  subnets                    = var.public_subnet_ids
  idle_timeout               = 120 # SSE (R10)
  drop_invalid_header_fields = true
  tags                       = var.tags
}

resource "aws_lb_target_group" "api" {
  name                 = "${var.name_prefix}-api"
  port                 = local.http_port
  protocol             = "HTTP"
  vpc_id               = var.vpc_id
  target_type          = "ip"
  deregistration_delay = 30
  health_check {
    path                = "/readyz"
    port                = tostring(local.internal_port)
    healthy_threshold   = 2
    unhealthy_threshold = 3
    timeout             = 5
    interval            = 15
    matcher             = "200"
  }
  tags = var.tags
}

# Default action is a fixed 403; a rule forwards only when the origin-verify
# header matches (RV-025). Without a cert (validate), expose HTTP only.
resource "aws_lb_listener" "https" {
  count             = var.acm_certificate_arn == null ? 0 : 1
  load_balancer_arn = aws_lb.this.arn
  port              = 443
  protocol          = "HTTPS"
  ssl_policy        = "ELBSecurityPolicy-TLS13-1-2-2021-06"
  certificate_arn   = var.acm_certificate_arn
  default_action {
    type = "fixed-response"
    fixed_response {
      content_type = "text/plain"
      message_body = "forbidden"
      status_code  = "403"
    }
  }
  tags = var.tags
}

# When no origin-verify secret ARN is given yet (validate), skip the listener
# rule; otherwise forward to the api target group only on a header match.
data "aws_secretsmanager_secret_version" "origin_verify" {
  count     = var.origin_verify_secret_arn == null ? 0 : 1
  secret_id = var.origin_verify_secret_arn
}
resource "aws_lb_listener_rule" "api_origin_verify" {
  count        = (var.acm_certificate_arn == null || var.origin_verify_secret_arn == null) ? 0 : 1
  listener_arn = aws_lb_listener.https[0].arn
  priority     = 10
  action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.api.arn
  }
  condition {
    http_header {
      http_header_name = "X-Rovo-Origin-Verify"
      values           = [data.aws_secretsmanager_secret_version.origin_verify[0].secret_string]
    }
  }
}

# --- container definition shared by api/worker/migrate ---
locals {
  log_config = {
    logDriver = "awslogs"
    options = {
      "awslogs-group"         = var.log_group_name
      "awslogs-region"        = var.region
      "awslogs-stream-prefix" = var.name_prefix
    }
  }
  common_container = {
    image                  = var.image
    essential              = true
    environment            = local.app_env_list
    secrets                = concat(local.db_secrets, local.app_secrets)
    logConfiguration       = local.log_config
    readonlyRootFilesystem = true
    linuxParameters        = { initProcessEnabled = true }
  }
}

resource "aws_ecs_task_definition" "api" {
  family                   = "${var.name_prefix}-api"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = var.api_cpu
  memory                   = var.api_memory
  execution_role_arn       = aws_iam_role.execution.arn
  task_role_arn            = aws_iam_role.task.arn
  runtime_platform {
    cpu_architecture        = "ARM64"
    operating_system_family = "LINUX"
  }
  container_definitions = jsonencode([merge(local.common_container, {
    name    = "api"
    command = ["api"]
    portMappings = [
      { containerPort = local.http_port, protocol = "tcp" },
      { containerPort = local.internal_port, protocol = "tcp" },
    ]
    healthCheck = {
      command     = ["CMD", "/usr/local/bin/healthcheck", "http://127.0.0.1:${local.internal_port}/healthz"]
      interval    = 15
      timeout     = 4
      retries     = 3
      startPeriod = 20
    }
  })])
  tags = var.tags
}

resource "aws_ecs_task_definition" "worker" {
  family                   = "${var.name_prefix}-worker"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = var.worker_cpu
  memory                   = var.worker_memory
  execution_role_arn       = aws_iam_role.execution.arn
  task_role_arn            = aws_iam_role.task.arn
  runtime_platform {
    cpu_architecture        = "ARM64"
    operating_system_family = "LINUX"
  }
  container_definitions = jsonencode([merge(local.common_container, {
    name        = "worker"
    command     = ["worker"]
    stopTimeout = 120
    healthCheck = {
      command     = ["CMD", "/usr/local/bin/healthcheck", "http://127.0.0.1:${local.internal_port}/healthz"]
      interval    = 15
      timeout     = 4
      retries     = 3
      startPeriod = 20
    }
  })])
  tags = var.tags
}

# One-off migrate task (CI RunTask before api/worker; doc 21 §7).
resource "aws_ecs_task_definition" "migrate" {
  family                   = "${var.name_prefix}-migrate"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = 256
  memory                   = 512
  execution_role_arn       = aws_iam_role.execution.arn
  task_role_arn            = aws_iam_role.task.arn
  runtime_platform {
    cpu_architecture        = "ARM64"
    operating_system_family = "LINUX"
  }
  container_definitions = jsonencode([merge(local.common_container, {
    name    = "migrate"
    command = ["migrate", "up"]
  })])
  tags = var.tags
}

resource "aws_ecs_service" "api" {
  name            = "${var.name_prefix}-api"
  cluster         = aws_ecs_cluster.this.id
  task_definition = aws_ecs_task_definition.api.arn
  desired_count   = var.api_desired_count
  launch_type     = "FARGATE"
  propagate_tags  = "SERVICE"

  network_configuration {
    subnets          = var.task_subnet_ids
    security_groups  = [aws_security_group.api.id]
    assign_public_ip = var.task_assign_public_ip
  }
  dynamic "load_balancer" {
    for_each = var.acm_certificate_arn == null ? [] : [1]
    content {
      target_group_arn = aws_lb_target_group.api.arn
      container_name   = "api"
      container_port   = local.http_port
    }
  }
  deployment_circuit_breaker {
    enable   = true
    rollback = true
  }
  deployment_minimum_healthy_percent = 100
  deployment_maximum_percent         = 200
  # CI swaps the image digest; ignore drift on task_definition between deploys.
  lifecycle { ignore_changes = [task_definition, desired_count] }
  tags = var.tags
}

resource "aws_ecs_service" "worker" {
  name            = "${var.name_prefix}-worker"
  cluster         = aws_ecs_cluster.this.id
  task_definition = aws_ecs_task_definition.worker.arn
  desired_count   = var.worker_desired_count
  launch_type     = "FARGATE"
  propagate_tags  = "SERVICE"
  network_configuration {
    subnets          = var.task_subnet_ids
    security_groups  = [aws_security_group.worker.id]
    assign_public_ip = var.task_assign_public_ip
  }
  lifecycle { ignore_changes = [task_definition] }
  tags = var.tags
}

# --- autoscaling (api) ---
resource "aws_appautoscaling_target" "api" {
  max_capacity       = var.api_max_count
  min_capacity       = var.api_desired_count
  resource_id        = "service/${aws_ecs_cluster.this.name}/${aws_ecs_service.api.name}"
  scalable_dimension = "ecs:service:DesiredCount"
  service_namespace  = "ecs"
}
resource "aws_appautoscaling_policy" "api_cpu" {
  name               = "${var.name_prefix}-api-cpu"
  policy_type        = "TargetTrackingScaling"
  resource_id        = aws_appautoscaling_target.api.resource_id
  scalable_dimension = aws_appautoscaling_target.api.scalable_dimension
  service_namespace  = aws_appautoscaling_target.api.service_namespace
  target_tracking_scaling_policy_configuration {
    predefined_metric_specification {
      predefined_metric_type = "ECSServiceAverageCPUUtilization"
    }
    target_value       = 60
    scale_in_cooldown  = 120
    scale_out_cooldown = 60
  }
}
