// ECR repos are account-scoped singletons. Created in dev/prod; impl reuses
// dev's via data.aws_ecr_repository.* so the same image can deploy across envs.
resource "aws_ecr_repository" "ztmf_api" {
  count                = local.manage_account_singletons ? 1 : 0
  name                 = "ztmf/api"
  image_tag_mutability = "IMMUTABLE"
}

resource "aws_ecr_lifecycle_policy" "ztmf_api" {
  count      = local.manage_account_singletons ? 1 : 0
  repository = aws_ecr_repository.ztmf_api[0].name

  policy = <<EOF
{
    "rules": [
        {
            "rulePriority": 1,
            "description": "Keep last 4 images",
            "selection": {
                "tagStatus": "any",
                "countType": "imageCountMoreThan",
                "countNumber": 4
            },
            "action": {
                "type": "expire"
            }
        }
    ]
}
EOF
}

resource "aws_ecr_registry_scanning_configuration" "ztmf_api" {
  count     = local.manage_account_singletons ? 1 : 0
  scan_type = "ENHANCED"

  rule {
    scan_frequency = "SCAN_ON_PUSH"
    repository_filter {
      filter      = "*"
      filter_type = "WILDCARD"
    }
  }
}

resource "aws_ecr_repository" "ztmf_ops" {
  count                = local.manage_account_singletons ? 1 : 0
  name                 = "ztmf/ops"
  image_tag_mutability = "IMMUTABLE"
}

resource "aws_ecr_lifecycle_policy" "ztmf_ops" {
  count      = local.manage_account_singletons ? 1 : 0
  repository = aws_ecr_repository.ztmf_ops[0].name

  policy = <<EOF
{
    "rules": [
        {
            "rulePriority": 1,
            "description": "Keep last 2 images",
            "selection": {
                "tagStatus": "any",
                "countType": "imageCountMoreThan",
                "countNumber": 2
            },
            "action": {
                "type": "expire"
            }
        }
    ]
}
EOF
}

// Frontend image for per-PR environments (ztmf-misc#343), dev account only.
// Tags: ui-<sha> from main merges, pr-ui-<n>-<sha> from PRs; the pr-* rule
// mirrors ztmf/api's.
resource "aws_ecr_repository" "ztmf_ui" {
  count                = local.manage_account_singletons && var.pr_env_enabled ? 1 : 0
  name                 = "ztmf/ui"
  image_tag_mutability = "IMMUTABLE"
}

resource "aws_ecr_lifecycle_policy" "ztmf_ui" {
  count      = local.manage_account_singletons && var.pr_env_enabled ? 1 : 0
  repository = aws_ecr_repository.ztmf_ui[0].name

  policy = <<EOF
{
    "rules": [
        {
            "rulePriority": 1,
            "description": "Expire pr-* images 14 days after push",
            "selection": {
                "tagStatus": "tagged",
                "tagPrefixList": ["pr-"],
                "countType": "sinceImagePushed",
                "countUnit": "days",
                "countNumber": 14
            },
            "action": {
                "type": "expire"
            }
        },
        {
            "rulePriority": 10,
            "description": "Keep last 4 images",
            "selection": {
                "tagStatus": "any",
                "countType": "imageCountMoreThan",
                "countNumber": 4
            },
            "action": {
                "type": "expire"
            }
        }
    ]
}
EOF
}
