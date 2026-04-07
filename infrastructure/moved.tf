# Terraform moved blocks to handle refactoring without destroying resources.
# These tell Terraform that resources have been renamed/re-indexed, not deleted.
# Safe to remove these blocks after both dev and prod have been applied once.

# Snowflake secret was conditional per-environment, now unified.
# Terraform does not allow two moved blocks targeting the same destination,
# even if only one source exists in state. We use the dev block here since
# dev deploys first (on PR). For the first prod apply, run:
#   terraform state mv 'aws_secretsmanager_secret.ztmf_snowflake_prod[0]' aws_secretsmanager_secret.ztmf_snowflake
moved {
  from = aws_secretsmanager_secret.ztmf_snowflake_dev[0]
  to   = aws_secretsmanager_secret.ztmf_snowflake
}

# Bastion resources moved to count-based for VPC sharing support
moved {
  from = module.ec2_bastion
  to   = module.ec2_bastion[0]
}

moved {
  from = aws_iam_instance_profile.ec2_bastion
  to   = aws_iam_instance_profile.ec2_bastion[0]
}

moved {
  from = aws_instance.bastion
  to   = aws_instance.bastion[0]
}

moved {
  from = aws_security_group.ztmf_bastion
  to   = aws_security_group.ztmf_bastion[0]
}

# VPC endpoint resources moved to count-based for VPC sharing support
moved {
  from = aws_security_group.ztmf_vpc_endpoints
  to   = aws_security_group.ztmf_vpc_endpoints[0]
}

# GitHub OIDC resources moved to count-based for account sharing support
moved {
  from = aws_iam_openid_connect_provider.github_actions
  to   = aws_iam_openid_connect_provider.github_actions[0]
}

moved {
  from = module.github_actions
  to   = module.github_actions[0]
}

# ECR resources moved to count-based for account sharing support
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
