terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
  required_version = ">= 1.5.0"
}

provider "aws" {
  region = var.aws_region
}

variable "aws_region" {
  description = "AWS region"
  type        = string
  default     = "us-east-1"
}

variable "prism_origin_domain" {
  description = "The Prism server domain"
  type        = string
}

variable "environment" {
  description = "Environment name"
  type        = string
  default     = "staging"
}

# 1. CloudFront Key Group + Public Key for signed URLs
resource "aws_cloudfront_public_key" "prism" {
  encoded_key = file("${path.module}/public_key.pem")
  name        = "prism-public-key-${var.environment}"
  comment     = "Public key for Prism signed URLs"
}

resource "aws_cloudfront_key_group" "prism" {
  name    = "prism-key-group-${var.environment}"
  comment = "Key group for Prism signed URLs"
  items   = [aws_cloudfront_public_key.prism.id]
}

# 2. Cache policies
resource "aws_cloudfront_cache_policy" "manifests" {
  name        = "prism-manifests-${var.environment}"
  default_ttl = 30
  min_ttl     = 0
  max_ttl     = 60

  parameters_in_cache_key_and_forwarded_to_origin {
    query_strings_config {
      query_string_behavior = "whitelist"
      query_strings {
        items = ["codec", "maxBandwidth", "resolution"]
      }
    }
    headers_config {
      header_behavior = "none"
    }
    cookies_config {
      cookie_behavior = "none"
    }
    enable_accept_encoding_gzip   = true
    enable_accept_encoding_brotli = true
  }
}

resource "aws_cloudfront_cache_policy" "licenses_no_cache" {
  name        = "prism-licenses-${var.environment}"
  default_ttl = 0
  min_ttl     = 0
  max_ttl     = 0

  parameters_in_cache_key_and_forwarded_to_origin {
    query_strings_config {
      query_string_behavior = "none"
    }
    headers_config {
      header_behavior = "none"
    }
    cookies_config {
      cookie_behavior = "none"
    }
  }
}

# 3. Origin request policy that forwards DRM specific headers
resource "aws_cloudfront_origin_request_policy" "drm_headers" {
  name    = "prism-drm-headers-${var.environment}"
  comment = "Policy to forward DRM specific headers"

  headers_config {
    header_behavior = "whitelist"
    headers {
      items = ["X-DRM-Token", "X-Asset-ID", "X-FairPlay-SPC"]
    }
  }

  cookies_config {
    cookie_behavior = "none"
  }

  query_strings_config {
    query_string_behavior = "all"
  }
}

# 4. CloudFront distribution
resource "aws_cloudfront_distribution" "prism" {
  enabled = true
  comment = "Prism OTT — ${var.environment}"

  default_cache_behavior {
    target_origin_id       = "prism-origin"
    viewer_protocol_policy = "redirect-to-https"
    allowed_methods        = ["GET", "HEAD", "OPTIONS", "POST", "PUT", "PATCH", "DELETE"]
    cached_methods         = ["GET", "HEAD"]
    cache_policy_id        = aws_cloudfront_cache_policy.manifests.id
  }

  ordered_cache_behavior {
    path_pattern           = "/manifest/*"
    target_origin_id       = "prism-origin"
    cache_policy_id        = aws_cloudfront_cache_policy.manifests.id
    allowed_methods        = ["GET", "HEAD", "OPTIONS"]
    cached_methods         = ["GET", "HEAD"]
    viewer_protocol_policy = "redirect-to-https"
    trusted_key_groups     = [aws_cloudfront_key_group.prism.id]
  }

  ordered_cache_behavior {
    path_pattern             = "/license/*"
    target_origin_id         = "prism-origin"
    cache_policy_id          = aws_cloudfront_cache_policy.licenses_no_cache.id
    origin_request_policy_id = aws_cloudfront_origin_request_policy.drm_headers.id
    allowed_methods          = ["POST", "GET", "HEAD", "OPTIONS"]
    cached_methods           = ["GET", "HEAD"]
    viewer_protocol_policy   = "redirect-to-https"
  }

  origin {
    domain_name = var.prism_origin_domain
    origin_id   = "prism-origin"

    custom_origin_config {
      http_port              = 80
      https_port             = 443
      origin_protocol_policy = "https-only"
      origin_ssl_protocols   = ["TLSv1.2"]
    }
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  viewer_certificate {
    cloudfront_default_certificate = true
  }

  price_class = "PriceClass_100"
}

# 5. Outputs
output "distribution_id" {
  value = aws_cloudfront_distribution.prism.id
}

output "domain_name" {
  value = aws_cloudfront_distribution.prism.domain_name
}

output "key_pair_id" {
  value = aws_cloudfront_public_key.prism.id
}
