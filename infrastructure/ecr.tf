resource "aws_ecr_repository" "ztmf_api" {
  name                 = "ztmf/api"
  image_tag_mutability = "IMMUTABLE"
}

resource "aws_ecr_lifecycle_policy" "ztmf_api" {
  repository = aws_ecr_repository.ztmf_api.name

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
  scan_type = "ENHANCED"

  rule {
    scan_frequency = "SCAN_ON_PUSH"
    repository_filter {
      filter      = "*"
      filter_type = "WILDCARD"
    }
  }
}


output "ztmf_api_repository_url" {
  value = aws_ecr_repository.ztmf_api.repository_url
}
