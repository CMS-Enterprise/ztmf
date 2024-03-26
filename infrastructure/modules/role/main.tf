data "aws_caller_identity" "current" {}

resource "aws_iam_role" "role" {
  name                = var.name
  managed_policy_arns = var.managed_policy_arns
  assume_role_policy = jsonencode({
    Version = "2012-10-17",
    Statement = [
      {
        Effect = "Allow",
        Principal = var.principal
        Action = "sts:AssumeRole",
      }
    ]
  })
  # CMS requires all roles to include permissions boundary and path
  permissions_boundary = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:policy/cms-cloud-admin/ct-ado-poweruser-permissions-boundary-policy"
  path = var.principal.Service == "" ? "/delegatedadmin/developer/" : "/delegatedadmin/adodeveloper/service-role/"
}

output "role_arn" {
  value = aws_iam_role.role.arn 
}

output "role_id" {
  value = aws_iam_role.role.id
}

output "role_name" {
  value = aws_iam_role.role.name
}
