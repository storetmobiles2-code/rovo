# SPDX-License-Identifier: Apache-2.0
# CloudFront-scoped WAF: AWS managed rule sets, rate rules for OTP/login/order,
# an admin-host geo=IN rule (R37), webhooks exempt from bot rules (doc 22 §3.1).
# Must be created in us-east-1 (CLOUDFRONT scope) — the env passes aws.us_east_1.
resource "aws_wafv2_web_acl" "this" {
  name        = "${var.name_prefix}-cf"
  scope       = "CLOUDFRONT"
  description = "rovo edge WAF"
  default_action {
    allow {}
  }
  visibility_config {
    cloudwatch_metrics_enabled = true
    metric_name                = "${var.name_prefix}-cf"
    sampled_requests_enabled   = true
  }

  rule {
    name     = "common"
    priority = 1
    override_action {
      none {}
    }
    statement {
      managed_rule_group_statement {
        name        = "AWSManagedRulesCommonRuleSet"
        vendor_name = "AWS"
      }
    }
    visibility_config {
      cloudwatch_metrics_enabled = true
      metric_name                = "common"
      sampled_requests_enabled   = true
    }
  }

  rule {
    name     = "known-bad-inputs"
    priority = 2
    override_action {
      none {}
    }
    statement {
      managed_rule_group_statement {
        name        = "AWSManagedRulesKnownBadInputsRuleSet"
        vendor_name = "AWS"
      }
    }
    visibility_config {
      cloudwatch_metrics_enabled = true
      metric_name                = "known-bad-inputs"
      sampled_requests_enabled   = true
    }
  }

  rule {
    name     = "ip-reputation"
    priority = 3
    override_action {
      none {}
    }
    statement {
      managed_rule_group_statement {
        name        = "AWSManagedRulesAmazonIpReputationList"
        vendor_name = "AWS"
      }
    }
    visibility_config {
      cloudwatch_metrics_enabled = true
      metric_name                = "ip-reputation"
      sampled_requests_enabled   = true
    }
  }

  # Admin host: only India (R37). Block everything else.
  rule {
    name     = "admin-geo-in"
    priority = 10
    action {
      block {}
    }
    statement {
      and_statement {
        statement {
          byte_match_statement {
            field_to_match {
              single_header {
                name = "host"
              }
            }
            positional_constraint = "STARTS_WITH"
            search_string         = "admin."
            text_transformation {
              priority = 0
              type     = "LOWERCASE"
            }
          }
        }
        statement {
          not_statement {
            statement {
              geo_match_statement { country_codes = ["IN"] }
            }
          }
        }
      }
    }
    visibility_config {
      cloudwatch_metrics_enabled = true
      metric_name                = "admin-geo-in"
      sampled_requests_enabled   = true
    }
  }

  # Rate rule on auth/OTP/order-create paths (CGNAT-aware threshold).
  rule {
    name     = "rate-sensitive-paths"
    priority = 20
    action {
      block {}
    }
    statement {
      rate_based_statement {
        limit              = 600
        aggregate_key_type = "IP"
        scope_down_statement {
          or_statement {
            statement {
              byte_match_statement {
                field_to_match {
                  uri_path {}
                }
                positional_constraint = "STARTS_WITH"
                search_string         = "/api/v1/auth"
                text_transformation {
                  priority = 0
                  type     = "LOWERCASE"
                }
              }
            }
            statement {
              byte_match_statement {
                field_to_match {
                  uri_path {}
                }
                positional_constraint = "STARTS_WITH"
                search_string         = "/api/v1/orders"
                text_transformation {
                  priority = 0
                  type     = "LOWERCASE"
                }
              }
            }
          }
        }
      }
    }
    visibility_config {
      cloudwatch_metrics_enabled = true
      metric_name                = "rate-sensitive-paths"
      sampled_requests_enabled   = true
    }
  }

  tags = var.tags
}
