# SPDX-License-Identifier: Apache-2.0
# GitHub Actions OIDC federation (no long-lived cloud keys; P16, doc 21 §6).
# One OIDC provider per workload account; roles scoped by the `sub` claim.
variable "name_prefix" { type = string }
variable "tags" {
  type    = map(string)
  default = {}
}
variable "github_org" {
  type    = string
  default = "storetmobiles2-code"
}
variable "github_repo" {
  type    = string
  default = "rovo"
}
variable "create_oidc_provider" {
  description = "One provider per account; set false if it already exists."
  type        = bool
  default     = true
}
variable "ecr_push" {
  description = "Grant ECR push (only in the account that holds ECR, rovo-prod)."
  type        = bool
  default     = false
}
variable "deploy_environment" {
  description = "GitHub Environment gating deploys (staging | production)."
  type        = string
  default     = "staging"
}

data "aws_caller_identity" "current" {}

# GitHub's OIDC thumbprint is no longer validated by AWS, but the attribute is
# still required by the API; this is GitHub's well-known intermediate.
resource "aws_iam_openid_connect_provider" "github" {
  count           = var.create_oidc_provider ? 1 : 0
  url             = "https://token.actions.githubusercontent.com"
  client_id_list  = ["sts.amazonaws.com"]
  thumbprint_list = ["6938fd4d98bab03faadb97b34396831e3780aea1"]
  tags            = var.tags
}

locals {
  oidc_arn = var.create_oidc_provider ? aws_iam_openid_connect_provider.github[0].arn : "arn:aws:iam::${data.aws_caller_identity.current.account_id}:oidc-provider/token.actions.githubusercontent.com"
  repo     = "${var.github_org}/${var.github_repo}"
}

data "aws_iam_policy_document" "deploy_trust" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRoleWithWebIdentity"]
    principals {
      type        = "Federated"
      identifiers = [local.oidc_arn]
    }
    condition {
      test     = "StringEquals"
      variable = "token.actions.githubusercontent.com:aud"
      values   = ["sts.amazonaws.com"]
    }
    condition {
      test     = "StringEquals"
      variable = "token.actions.githubusercontent.com:sub"
      values   = ["repo:${local.repo}:environment:${var.deploy_environment}"]
    }
  }
}

resource "aws_iam_role" "deploy" {
  name                 = "${var.name_prefix}-gha-deploy"
  assume_role_policy   = data.aws_iam_policy_document.deploy_trust.json
  max_session_duration = 3600
  tags                 = var.tags
}

# Deploy permissions: ECS task registration/update, pass only rovo task roles,
# web bucket sync, CloudFront invalidation (doc 21 §6).
data "aws_iam_policy_document" "deploy" {
  statement {
    sid = "EcsDeploy"
    actions = [
      "ecs:RegisterTaskDefinition", "ecs:DeregisterTaskDefinition",
      "ecs:UpdateService", "ecs:RunTask", "ecs:DescribeServices",
      "ecs:DescribeTasks", "ecs:DescribeTaskDefinition", "ecs:ListTasks",
    ]
    resources = ["*"]
  }
  statement {
    sid       = "PassRovoRoles"
    actions   = ["iam:PassRole"]
    resources = ["arn:aws:iam::${data.aws_caller_identity.current.account_id}:role/${var.name_prefix}-*"]
    condition {
      test     = "StringEquals"
      variable = "iam:PassedToService"
      values   = ["ecs-tasks.amazonaws.com"]
    }
  }
  statement {
    sid       = "WebDeploy"
    actions   = ["s3:PutObject", "s3:DeleteObject", "s3:ListBucket"]
    resources = ["arn:aws:s3:::${var.name_prefix}-web", "arn:aws:s3:::${var.name_prefix}-web/*"]
  }
  statement {
    sid       = "Invalidate"
    actions   = ["cloudfront:CreateInvalidation"]
    resources = ["*"]
  }
}

resource "aws_iam_role_policy" "deploy" {
  name   = "deploy"
  role   = aws_iam_role.deploy.id
  policy = data.aws_iam_policy_document.deploy.json
}

# Optional ECR push role (rovo-prod only), trusted for main + release tags.
data "aws_iam_policy_document" "ecr_trust" {
  count = var.ecr_push ? 1 : 0
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRoleWithWebIdentity"]
    principals {
      type        = "Federated"
      identifiers = [local.oidc_arn]
    }
    condition {
      test     = "StringEquals"
      variable = "token.actions.githubusercontent.com:aud"
      values   = ["sts.amazonaws.com"]
    }
    condition {
      test     = "StringLike"
      variable = "token.actions.githubusercontent.com:sub"
      values   = ["repo:${local.repo}:ref:refs/heads/main", "repo:${local.repo}:ref:refs/tags/v*"]
    }
  }
}

resource "aws_iam_role" "ecr_push" {
  count                = var.ecr_push ? 1 : 0
  name                 = "${var.name_prefix}-gha-ecr-push"
  assume_role_policy   = data.aws_iam_policy_document.ecr_trust[0].json
  max_session_duration = 3600
  tags                 = var.tags
}

data "aws_iam_policy_document" "ecr_push" {
  count = var.ecr_push ? 1 : 0
  statement {
    actions   = ["ecr:GetAuthorizationToken"]
    resources = ["*"]
  }
  statement {
    actions = [
      "ecr:BatchCheckLayerAvailability", "ecr:CompleteLayerUpload",
      "ecr:InitiateLayerUpload", "ecr:PutImage", "ecr:UploadLayerPart",
      "ecr:BatchGetImage", "ecr:GetDownloadUrlForLayer",
    ]
    resources = ["arn:aws:ecr:*:${data.aws_caller_identity.current.account_id}:repository/rovo*"]
  }
}

resource "aws_iam_role_policy" "ecr_push" {
  count  = var.ecr_push ? 1 : 0
  name   = "ecr-push"
  role   = aws_iam_role.ecr_push[0].id
  policy = data.aws_iam_policy_document.ecr_push[0].json
}
