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

  // Rules apply in priority order and an image matched by one rule is not
  // evaluated by later ones, so pr-* images never count toward the keep-last-4.
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

// Per-PR environment repos (ztmf-misc#341), dev account only. PR images never
// share a repo with the bare-SHA deploy images in ztmf/api: a keep-last-N rule
// with tagStatus any counts images a higher-priority rule protects, so PR
// images would push the deployables out of the keep set (lifecycle preview,
// ztmf#584). Here every long-lived tag carries a prefix, so a prefixed
// keep-last-4 sits above an age rule that only ever reaches the PR images.
locals {
  pr_env_lifecycle = <<EOF
{
    "rules": [
        {
            "rulePriority": 1,
            "description": "Keep the last 4 PREFIX images",
            "selection": {
                "tagStatus": "tagged",
                "tagPrefixList": ["PREFIX"],
                "countType": "imageCountMoreThan",
                "countNumber": 4
            },
            "action": {
                "type": "expire"
            }
        },
        {
            "rulePriority": 10,
            "description": "Expire everything else 14 days after push",
            "selection": {
                "tagStatus": "any",
                "countType": "sinceImagePushed",
                "countUnit": "days",
                "countNumber": 14
            },
            "action": {
                "type": "expire"
            }
        }
    ]
}
EOF
}

// API images for PR environments: pr-ztmf-<n>-<sha> from ztmf PRs and
// main-<sha> test-target builds from ztmf main merges (what a ui PR runs).
resource "aws_ecr_repository" "ztmf_api_pr" {
  count                = local.manage_account_singletons && var.pr_env_enabled ? 1 : 0
  name                 = "ztmf/api-pr"
  image_tag_mutability = "IMMUTABLE"
}

resource "aws_ecr_lifecycle_policy" "ztmf_api_pr" {
  count      = local.manage_account_singletons && var.pr_env_enabled ? 1 : 0
  repository = aws_ecr_repository.ztmf_api_pr[0].name
  policy     = replace(local.pr_env_lifecycle, "PREFIX", "main-")
}

// Frontend images (ztmf-misc#343): ui-<sha> from main merges, pr-ui-<n>-<sha>
// from ui PRs. Dev and prod serve the bundle from S3, so this repo exists only
// for PR environments.
resource "aws_ecr_repository" "ztmf_ui" {
  count                = local.manage_account_singletons && var.pr_env_enabled ? 1 : 0
  name                 = "ztmf/ui"
  image_tag_mutability = "IMMUTABLE"
}

resource "aws_ecr_lifecycle_policy" "ztmf_ui" {
  count      = local.manage_account_singletons && var.pr_env_enabled ? 1 : 0
  repository = aws_ecr_repository.ztmf_ui[0].name
  policy     = replace(local.pr_env_lifecycle, "PREFIX", "ui-")
}
