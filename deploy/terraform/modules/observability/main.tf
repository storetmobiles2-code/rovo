# SPDX-License-Identifier: Apache-2.0
# CloudWatch log groups (CERT-In archive, R36) and last-resort alarms -> SNS.
# Grafana Cloud is the working telemetry backend (doc 24); CloudWatch is the
# compliance copy and the fires-even-if-Grafana-is-down alarm path.
variable "name_prefix" { type = string }
variable "tags" {
  type    = map(string)
  default = {}
}
variable "app_log_retention_days" {
  type    = number
  default = 180 # CERT-In app logs
}
variable "security_log_retention_days" {
  type    = number
  default = 400 # security events >= 1 year
}
variable "alarm_email" {
  type    = string
  default = null
}

resource "aws_cloudwatch_log_group" "app" {
  name              = "/rovo/${var.name_prefix}/app"
  retention_in_days = var.app_log_retention_days
  log_group_class   = "INFREQUENT_ACCESS"
  tags              = var.tags
}

resource "aws_cloudwatch_log_group" "security" {
  name              = "/rovo/${var.name_prefix}/security"
  retention_in_days = var.security_log_retention_days
  tags              = merge(var.tags, { Class = "security" })
}

resource "aws_sns_topic" "alerts" {
  name = "${var.name_prefix}-last-resort-alerts"
  tags = var.tags
}

resource "aws_sns_topic_subscription" "email" {
  count     = var.alarm_email == null ? 0 : 1
  topic_arn = aws_sns_topic.alerts.arn
  protocol  = "email"
  endpoint  = var.alarm_email
}

# A24 CERT-In archive delivery: no app log events for 30 min during service hours.
resource "aws_cloudwatch_log_metric_filter" "app_events" {
  name           = "${var.name_prefix}-app-log-events"
  log_group_name = aws_cloudwatch_log_group.app.name
  pattern        = ""
  metric_transformation {
    name          = "AppLogEvents"
    namespace     = "rovo/${var.name_prefix}"
    value         = "1"
    default_value = "0"
  }
}

resource "aws_cloudwatch_metric_alarm" "cert_in_delivery" {
  alarm_name          = "${var.name_prefix}-cert-in-archive-stalled"
  comparison_operator = "LessThanThreshold"
  evaluation_periods  = 2
  metric_name         = "AppLogEvents"
  namespace           = "rovo/${var.name_prefix}"
  period              = 900
  statistic           = "Sum"
  threshold           = 1
  alarm_description   = "No app log delivery to the CERT-In archive (A25)"
  treat_missing_data  = "breaching"
  alarm_actions       = [aws_sns_topic.alerts.arn]
  ok_actions          = [aws_sns_topic.alerts.arn]
  tags                = var.tags
}
