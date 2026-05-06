// Address shifts caused by adding `count` to account-singleton resources so
// impl can reuse dev's existing copies via data sources. Without these blocks,
// `terraform plan` against dev/prod state would propose destroy + recreate.
// With them, the plan is a state-only "moved" no-op.

moved {
  from = aws_ecr_repository.ztmf_api
  to   = aws_ecr_repository.ztmf_api[0]
}

moved {
  from = aws_ecr_lifecycle_policy.ztmf_api
  to   = aws_ecr_lifecycle_policy.ztmf_api[0]
}

moved {
  from = aws_ecr_registry_scanning_configuration.ztmf_api
  to   = aws_ecr_registry_scanning_configuration.ztmf_api[0]
}

moved {
  from = aws_ecr_repository.ztmf_ops
  to   = aws_ecr_repository.ztmf_ops[0]
}

moved {
  from = aws_ecr_lifecycle_policy.ztmf_ops
  to   = aws_ecr_lifecycle_policy.ztmf_ops[0]
}

moved {
  from = aws_iam_openid_connect_provider.github_actions
  to   = aws_iam_openid_connect_provider.github_actions[0]
}

moved {
  from = module.github_actions
  to   = module.github_actions[0]
}

moved {
  from = aws_secretsmanager_secret.ztmf_smtp
  to   = aws_secretsmanager_secret.ztmf_smtp[0]
}

moved {
  from = aws_secretsmanager_secret.ztmf_smtp_ca_root
  to   = aws_secretsmanager_secret.ztmf_smtp_ca_root[0]
}

moved {
  from = aws_secretsmanager_secret.ztmf_smtp_intermediate
  to   = aws_secretsmanager_secret.ztmf_smtp_intermediate[0]
}

moved {
  from = aws_secretsmanager_secret.ztmf_tls_cert
  to   = aws_secretsmanager_secret.ztmf_tls_cert[0]
}

moved {
  from = aws_secretsmanager_secret.ztmf_tls_key
  to   = aws_secretsmanager_secret.ztmf_tls_key[0]
}

moved {
  from = aws_security_group.ztmf_vpc_endpoints
  to   = aws_security_group.ztmf_vpc_endpoints[0]
}

moved {
  from = aws_s3_bucket.ztmf_logs
  to   = aws_s3_bucket.ztmf_logs[0]
}

moved {
  from = aws_s3_bucket_server_side_encryption_configuration.ztmf_logs
  to   = aws_s3_bucket_server_side_encryption_configuration.ztmf_logs[0]
}

moved {
  from = aws_s3_bucket_policy.ztmf_logs_access
  to   = aws_s3_bucket_policy.ztmf_logs_access[0]
}
