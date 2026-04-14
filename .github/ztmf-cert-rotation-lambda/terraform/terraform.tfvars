aws_region           = "us-east-1"
name                 = "ztmf-cert-rotation-dev"
cert_bucket_name     = "ztmf-cert-rotation-dev"
lambda_zip_path      = "../lambda.zip"
env_prefix           = "dev"
domain               = "dev.ztmf.cms.gov"

acm_certificate_arn  = "arn:aws:acm:us-east-1:123456789012:certificate/..."
backup_secret_arn    = "arn:aws:secretsmanager:us-east-1:123456789012:secret:..."
slack_webhook_secret_arn = "arn:aws:secretsmanager:us-east-1:123456789012:secret:..."
dry_run              = false