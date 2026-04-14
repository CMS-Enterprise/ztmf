# ECR repo and scanning config are account-level resources.
# When multiple environments share an account (dev + impl), only the VPC owner creates these.
# Impl reuses dev's ECR repo — just pushes images with different tags.
resource "aws_ecr_repository" "ztmf_api" {
  count                = local.is_vpc_owner ? 1 : 0
  name                 = "ztmf/api"
  image_tag_mutability = "IMMUTABLE"
}

resource "aws_ecr_lifecycle_policy" "ztmf_api" {
  count      = local.is_vpc_owner ? 1 : 0
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
  count     = local.is_vpc_owner ? 1 : 0
  scan_type = "ENHANCED"

  rule {
    scan_frequency = "SCAN_ON_PUSH"
    repository_filter {
      filter      = "*"
      filter_type = "WILDCARD"
    }
  }
}

# Data source to look up the shared ECR repo when this environment doesn't own it
data "aws_ecr_repository" "ztmf_api" {
  count = local.is_vpc_owner ? 0 : 1
  name  = "ztmf/api"
}

locals {
  ecr_repository_url = local.is_vpc_owner ? aws_ecr_repository.ztmf_api[0].repository_url : data.aws_ecr_repository.ztmf_api[0].repository_url
}
