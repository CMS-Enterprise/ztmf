variable "aws_region" {
  type        = string
  description = "AWS region to deploy into."
  default     = "us-east-1"
}

variable "name" {
  type        = string
  description = "Base name for resources."
  default     = "ztmf-cert-rotation-dev"
}

variable "cert_bucket_name" {
  type        = string
  description = "S3 bucket to receive cert files (e.g. ztmf-cert-rotation-dev)."
}

variable "lambda_zip_path" {
  type        = string
  description = "Path to a pre-built Lambda zip containing `bootstrap` at the root."
}

variable "lambda_handler_runtime" {
  type        = string
  description = "Lambda runtime (provided.al2 for Go custom runtime)."
  default     = "provided.al2"
}

variable "lambda_architectures" {
  type        = list(string)
  description = "Lambda architectures."
  default     = ["x86_64"]
}

variable "domain" {
  type        = string
  description = "Expected domain for this env prefix (phase 1: dev.ztmf.cms.gov)."
  default     = "dev.ztmf.cms.gov"
}

variable "env_prefix" {
  type        = string
  description = "S3 prefix for this environment (without trailing slash)."
  default     = "dev"
}

variable "acm_certificate_arn" {
  type        = string
  description = "ACM certificate ARN to re-import into (fixed ARN)."
}

variable "backup_secret_arn" {
  type        = string
  description = "Secrets Manager secret ARN for backing up the cert bundle JSON."
}

variable "slack_webhook_secret_arn" {
  type        = string
  description = "Secrets Manager secret ARN containing ONLY the Slack webhook URL as SecretString. If null, Terraform will create a placeholder secret."
  default     = null
}

variable "dry_run" {
  type        = bool
  description = "If true, validate + Slack notify only (no ACM import/backup/archive)."
  default     = false
}

