# SPDX-License-Identifier: Apache-2.0
# Production composition (account rovo-prod). Profile chosen with
# -var-file=../../profiles/<closed-pilot|public-launch>.tfvars (R32).
locals {
  name_prefix = "rovo-${var.env}"
  tags        = { app = "rovo", env = var.env }
  # Host -> audience mapping the API derives from the Host header (doc 12).
  app_hosts = join(",", [
    "app.${var.domain}=customer",
    "restaurant.${var.domain}=restaurant",
    "rider.${var.domain}=rider",
    "admin.${var.domain}=admin",
    "api.${var.domain}=webhook",
  ])
  app_environment = {
    ROVO_ENV                      = var.env
    ROVO_HTTP_ADDR                = ":8080"
    ROVO_INTERNAL_ADDR            = ":9090"
    ROVO_APP_HOSTS                = local.app_hosts
    ROVO_CUSTOMER_BASE_URL        = "https://app.${var.domain}"
    ROVO_DOMAIN                   = var.domain
    ROVO_COOKIE_SECURE            = "true"
    ROVO_S3_REGION                = var.region
    ROVO_S3_BUCKET_MEDIA          = "${local.name_prefix}-media"
    ROVO_S3_BUCKET_KYC            = "${local.name_prefix}-kyc"
    ROVO_S3_PUBLIC_MEDIA_BASE_URL = "/media"
    ROVO_LOG_LEVEL                = "info"
    ROVO_PAYMENT_PROVIDER         = "fake" # real PA adapter enabled at go-live (R25)
    ROVO_SMS_PROVIDER             = "fake"
  }
}

module "secrets" {
  source      = "../../modules/secrets"
  name_prefix = local.name_prefix
  tags        = local.tags
}

module "storage" {
  source              = "../../modules/storage"
  name_prefix         = local.name_prefix
  tags                = local.tags
  kyc_kms_key_arn     = module.secrets.kms_key_arns["kyc"]
  backups_kms_key_arn = module.secrets.kms_key_arns["backups"]
}

module "observability" {
  source      = "../../modules/observability"
  name_prefix = local.name_prefix
  tags        = local.tags
  alarm_email = var.alarm_email
}

module "network" {
  source              = "../../modules/network"
  name_prefix         = local.name_prefix
  region              = var.region
  tags                = local.tags
  azs                 = var.azs
  nat_gateway_count   = var.nat_gateway_count
  flow_log_bucket_arn = module.storage.bucket_arns["logs"]
}

module "compute" {
  source                = "../../modules/compute"
  name_prefix           = local.name_prefix
  region                = var.region
  tags                  = local.tags
  vpc_id                = module.network.vpc_id
  task_subnet_ids       = module.network.task_subnet_ids
  task_assign_public_ip = module.network.task_assign_public_ip
  public_subnet_ids     = module.network.public_subnet_ids
  image                 = var.image
  database_secret_arn   = module.database.master_user_secret_arn
  app_secret_arns       = module.secrets.app_secret_arns
  kms_key_arns          = module.secrets.kms_key_arns
  media_bucket_arn      = module.storage.bucket_arns["media"]
  kyc_bucket_arn        = module.storage.bucket_arns["kyc"]
  log_group_name        = module.observability.app_log_group_name
  app_environment       = local.app_environment
  api_desired_count     = var.api_desired_count
  api_cpu               = var.api_cpu
  api_memory            = var.api_memory
  api_max_count         = var.api_max_count
  worker_desired_count  = var.worker_desired_count
  acm_certificate_arn   = var.acm_alb_cert_arn
}

module "database" {
  source                     = "../../modules/database"
  name_prefix                = local.name_prefix
  tags                       = local.tags
  vpc_id                     = module.network.vpc_id
  db_subnet_ids              = module.network.db_subnet_ids
  ingress_security_group_ids = [module.compute.api_security_group_id, module.compute.worker_security_group_id]
  kms_key_arn                = module.secrets.kms_key_arns["rds"]
  instance_class             = var.db_instance_class
  multi_az                   = var.db_multi_az
  allocated_storage          = var.db_allocated_storage
}

module "edge" {
  source                       = "../../modules/edge"
  providers                    = { aws = aws.us_east_1 }
  name_prefix                  = local.name_prefix
  tags                         = local.tags
  domain                       = var.domain
  web_bucket_regional_domain   = module.storage.bucket_domain_names["web"]
  media_bucket_regional_domain = module.storage.bucket_domain_names["media"]
  alb_domain                   = "origin-api.${var.domain}"
  acm_certificate_arn          = var.acm_cloudfront_cert_arn
}

module "iam" {
  source             = "../../modules/iam"
  name_prefix        = local.name_prefix
  tags               = local.tags
  ecr_push           = false
  deploy_environment = "staging"
}
