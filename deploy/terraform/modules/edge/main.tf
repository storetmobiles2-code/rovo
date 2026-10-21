# SPDX-License-Identifier: Apache-2.0
# One CloudFront distribution for all five public hostnames (doc 22 §3.1):
# app./restaurant./rider./admin. serve their SPA prefix with /api/* and /media/*
# behaviours; api. carries only /api/v1/webhooks/*. A viewer-request function
# selects the bucket prefix from Host and does SPA fallback on the default
# behaviour only; /api/* and /media/* are never rewritten.
locals {
  app_hosts       = ["app", "restaurant", "rider", "admin"]
  aliases         = [for h in concat(local.app_hosts, ["api"]) : "${h}.${var.domain}"]
  web_origin_id   = "web-bucket"
  media_origin_id = "media-bucket"
  alb_origin_id   = "api-alb"
}

resource "aws_cloudfront_origin_access_control" "web" {
  name                              = "${var.name_prefix}-web-oac"
  origin_access_control_origin_type = "s3"
  signing_behavior                  = "always"
  signing_protocol                  = "sigv4"
}
resource "aws_cloudfront_origin_access_control" "media" {
  name                              = "${var.name_prefix}-media-oac"
  origin_access_control_origin_type = "s3"
  signing_behavior                  = "always"
  signing_protocol                  = "sigv4"
}

# Host -> bucket prefix selection + SPA fallback (viewer request).
resource "aws_cloudfront_function" "router" {
  name    = "${var.name_prefix}-router"
  runtime = "cloudfront-js-2.0"
  publish = true
  code    = <<-JS
    function handler(event) {
      var req = event.request;
      var host = (req.headers.host && req.headers.host.value || "").toLowerCase();
      var app = host.split(".")[0];
      var valid = { app: "customer", restaurant: "restaurant", rider: "rider", admin: "admin" };
      var prefix = valid[app] || "customer";
      var uri = req.uri;
      // Never rewrite API or media paths.
      if (uri.indexOf("/api/") === 0 || uri.indexOf("/media/") === 0) { return req; }
      // SPA fallback: extensionless path -> that app's index.html.
      if (uri === "/" || uri.lastIndexOf(".") < uri.lastIndexOf("/")) {
        req.uri = "/" + prefix + "/index.html";
      } else {
        req.uri = "/" + prefix + uri;
      }
      return req;
    }
  JS
}

# Security + caching response headers (doc 17 §15.6). Per-app CSP is set by the
# origin (SPA served with headers) / here as a baseline for the static default.
resource "aws_cloudfront_response_headers_policy" "security" {
  name = "${var.name_prefix}-security-headers"
  security_headers_config {
    content_type_options { override = true }
    frame_options {
      frame_option = "DENY"
      override     = true
    }
    referrer_policy {
      referrer_policy = "strict-origin-when-cross-origin"
      override        = true
    }
    strict_transport_security {
      access_control_max_age_sec = 63072000
      include_subdomains         = true
      preload                    = true
      override                   = true
    }
  }
}

resource "aws_cloudfront_cache_policy" "static" {
  name        = "${var.name_prefix}-static"
  default_ttl = 86400
  max_ttl     = 31536000
  min_ttl     = 0
  parameters_in_cache_key_and_forwarded_to_origin {
    cookies_config { cookie_behavior = "none" }
    headers_config { header_behavior = "none" }
    query_strings_config { query_string_behavior = "none" }
    enable_accept_encoding_gzip   = true
    enable_accept_encoding_brotli = true
  }
}

# /api/* : caching disabled, forward Host + all headers/cookies/query (doc 22).
resource "aws_cloudfront_cache_policy" "api" {
  name        = "${var.name_prefix}-api-nocache"
  default_ttl = 0
  max_ttl     = 0
  min_ttl     = 0
  parameters_in_cache_key_and_forwarded_to_origin {
    cookies_config { cookie_behavior = "all" }
    headers_config {
      header_behavior = "whitelist"
      headers { items = ["Host", "Authorization", "Accept", "Content-Type", "Origin"] }
    }
    query_strings_config { query_string_behavior = "all" }
    enable_accept_encoding_gzip   = false
    enable_accept_encoding_brotli = false
  }
}
resource "aws_cloudfront_origin_request_policy" "api" {
  name = "${var.name_prefix}-api-origin"
  cookies_config { cookie_behavior = "all" }
  headers_config {
    header_behavior = "allViewerAndWhitelistCloudFront"
    headers { items = ["CloudFront-Viewer-Country"] }
  }
  query_strings_config { query_string_behavior = "all" }
}

resource "aws_cloudfront_distribution" "this" {
  enabled         = true
  is_ipv6_enabled = true
  comment         = "rovo ${var.name_prefix} edge"
  price_class     = "PriceClass_100" # India/US/EU edge locations
  aliases         = var.acm_certificate_arn == null ? [] : local.aliases
  web_acl_id      = var.web_acl_arn

  origin {
    origin_id                = local.web_origin_id
    domain_name              = var.web_bucket_regional_domain
    origin_access_control_id = aws_cloudfront_origin_access_control.web.id
  }
  origin {
    origin_id                = local.media_origin_id
    domain_name              = var.media_bucket_regional_domain
    origin_access_control_id = aws_cloudfront_origin_access_control.media.id
  }
  origin {
    origin_id   = local.alb_origin_id
    domain_name = var.alb_domain
    custom_origin_config {
      http_port                = 80
      https_port               = 443
      origin_protocol_policy   = "https-only"
      origin_ssl_protocols     = ["TLSv1.2"]
      origin_read_timeout      = 120 # SSE (R10)
      origin_keepalive_timeout = 60
    }
    custom_header {
      name  = "X-Rovo-Origin-Verify"
      value = var.origin_verify_secret_value
    }
  }

  # Default: SPA static from the web bucket, host->prefix via the function.
  default_cache_behavior {
    target_origin_id           = local.web_origin_id
    viewer_protocol_policy     = "redirect-to-https"
    allowed_methods            = ["GET", "HEAD", "OPTIONS"]
    cached_methods             = ["GET", "HEAD"]
    cache_policy_id            = aws_cloudfront_cache_policy.static.id
    response_headers_policy_id = aws_cloudfront_response_headers_policy.security.id
    compress                   = true
    function_association {
      event_type   = "viewer-request"
      function_arn = aws_cloudfront_function.router.arn
    }
  }

  ordered_cache_behavior {
    path_pattern             = "/api/*"
    target_origin_id         = local.alb_origin_id
    viewer_protocol_policy   = "https-only"
    allowed_methods          = ["GET", "HEAD", "OPTIONS", "PUT", "POST", "PATCH", "DELETE"]
    cached_methods           = ["GET", "HEAD"]
    cache_policy_id          = aws_cloudfront_cache_policy.api.id
    origin_request_policy_id = aws_cloudfront_origin_request_policy.api.id
    compress                 = false
  }

  ordered_cache_behavior {
    path_pattern           = "/media/*"
    target_origin_id       = local.media_origin_id
    viewer_protocol_policy = "redirect-to-https"
    allowed_methods        = ["GET", "HEAD"]
    cached_methods         = ["GET", "HEAD"]
    cache_policy_id        = aws_cloudfront_cache_policy.static.id
    compress               = true
  }

  restrictions {
    geo_restriction { restriction_type = "none" }
  }

  viewer_certificate {
    cloudfront_default_certificate = var.acm_certificate_arn == null
    acm_certificate_arn            = var.acm_certificate_arn
    ssl_support_method             = var.acm_certificate_arn == null ? null : "sni-only"
    minimum_protocol_version       = var.acm_certificate_arn == null ? "TLSv1" : "TLSv1.2_2021"
  }

  tags = var.tags
}
