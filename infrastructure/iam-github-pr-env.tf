// GitHub Actions role for the per-PR environment workflows and the reaper, in
// the dev account only. It does what the pr-env root and those workflows need
// and nothing else: push and delete the PR images, keep pr/ state in the dev
// state bucket, apply and destroy one pr-<repo>-<n> environment (its ECS
// service, listener rule, target group, security group, Cognito pool, log
// group, SSM parameters and two task roles), and record the main test image
// tags. The workflows reach it through the repo-level PR_ENV_ROLEARN secret.
//
// Every pr-env resource is named or tagged for its environment: names start
// with pr-, ztmf-pr- or ztmf_pr-, parameters sit under /ztmf/pr/, and the
// pr-env root's default tags put ztmf-pr-env on everything it creates.
locals {
  github_pr_env_subjects = [
    "repo:CMS-Enterprise/ztmf:environment:pr-ztmf-*",
    "repo:CMS-Enterprise/ztmf:pull_request",
    "repo:CMS-Enterprise/ztmf:ref:refs/heads/main",
    "repo:CMS-Enterprise/ztmf-ui:environment:pr-ui-*",
    "repo:CMS-Enterprise/ztmf-ui:pull_request",
  ]

  pr_env_region       = data.aws_region.current.region
  pr_env_state_bucket = "ztmf-terraform-state-use1-dev"
  pr_env_ecr_repos = [
    "arn:aws:ecr:${local.pr_env_region}:${local.account_id}:repository/ztmf/api-pr",
    "arn:aws:ecr:${local.pr_env_region}:${local.account_id}:repository/ztmf/ui",
  ]
  pr_env_task_roles    = "arn:aws:iam::${local.account_id}:role/delegatedadmin/adodeveloper/service-role/ztmf_pr-*"
  pr_env_role_boundary = "arn:aws:iam::${local.account_id}:policy/cms-cloud-admin/ct-ado-poweruser-permissions-boundary-policy"
}

module "github_actions_pr_env" {
  count     = var.environment == "dev" ? 1 : 0
  name      = "ztmf_github_actions_pr_env"
  source    = "./modules/role"
  principal = { Federated = aws_iam_openid_connect_provider.github_actions[0].arn }
  managed_policy_arns = [
    "arn:aws:iam::${local.account_id}:policy/CMSApprovedAWSServices",
    "arn:aws:iam::${local.account_id}:policy/ADO-Restriction-Policy",
    "arn:aws:iam::${local.account_id}:policy/ct-iamCreateUserRestrictionPolicy",
    "arn:aws:iam::${local.account_id}:policy/CMSCloudApprovedRegions",
  ]
  condition = {
    "StringEquals" = {
      "token.actions.githubusercontent.com:aud" = ["sts.amazonaws.com"]
    }
    "StringLike" = {
      "token.actions.githubusercontent.com:sub" = local.github_pr_env_subjects
    }
  }
}

resource "aws_iam_role_policy" "github_actions_pr_env" {
  count = var.environment == "dev" ? 1 : 0
  name  = "prEnvDeploy"
  role  = module.github_actions_pr_env[0].role_id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      // Docker login, then push, pull and delete in the two repos PR images use.
      {
        Sid      = "EcrLogin"
        Effect   = "Allow"
        Action   = ["ecr:GetAuthorizationToken"]
        Resource = ["*"]
      },
      {
        Sid    = "EcrPrImages"
        Effect = "Allow"
        Action = [
          "ecr:BatchCheckLayerAvailability",
          "ecr:BatchDeleteImage",
          "ecr:BatchGetImage",
          "ecr:CompleteLayerUpload",
          "ecr:DescribeImages",
          "ecr:DescribeRepositories",
          "ecr:GetDownloadUrlForLayer",
          "ecr:InitiateLayerUpload",
          "ecr:ListImages",
          "ecr:ListTagsForResource",
          "ecr:PutImage",
          "ecr:UploadLayerPart",
        ]
        Resource = local.pr_env_ecr_repos
      },
      // pr/<repo>/<n> state keys. Listing the bucket shows key names only.
      {
        Sid      = "StateList"
        Effect   = "Allow"
        Action   = ["s3:ListBucket"]
        Resource = ["arn:aws:s3:::${local.pr_env_state_bucket}"]
      },
      {
        Sid      = "StateObjects"
        Effect   = "Allow"
        Action   = ["s3:GetObject", "s3:PutObject", "s3:DeleteObject"]
        Resource = ["arn:aws:s3:::${local.pr_env_state_bucket}/pr/*"]
      },
      // The image tags publish-main records and deploys read, and each
      // environment's own parameters.
      {
        Sid    = "SsmImageTags"
        Effect = "Allow"
        Action = ["ssm:GetParameter", "ssm:PutParameter"]
        Resource = [
          "arn:aws:ssm:${local.pr_env_region}:${local.account_id}:parameter/ztmf_api_test_tag",
          "arn:aws:ssm:${local.pr_env_region}:${local.account_id}:parameter/ztmf_ui_tag",
        ]
      },
      {
        Sid    = "SsmEnvParameters"
        Effect = "Allow"
        Action = [
          "ssm:AddTagsToResource",
          "ssm:DeleteParameter",
          "ssm:GetParameter",
          "ssm:GetParameters",
          "ssm:ListTagsForResource",
          "ssm:PutParameter",
          "ssm:RemoveTagsFromResource",
        ]
        Resource = ["arn:aws:ssm:${local.pr_env_region}:${local.account_id}:parameter/ztmf/pr/*"]
      },
      {
        Sid      = "SsmDescribe"
        Effect   = "Allow"
        Action   = ["ssm:DescribeParameters"]
        Resource = ["*"]
      },
      // The dev Okta client the listener rule shares with dev's login rules.
      // Secrets Manager appends six characters to a secret's name in its ARN;
      // a bare * would also match impl's ztmf_va_trust_provider_impl.
      {
        Sid      = "OktaClientSecret"
        Effect   = "Allow"
        Action   = ["secretsmanager:DescribeSecret", "secretsmanager:GetSecretValue"]
        Resource = ["arn:aws:secretsmanager:${local.pr_env_region}:${local.account_id}:secret:ztmf_va_trust_provider-??????"]
      },
      // Key lookups for the aws/ssm alias, and encryption only when SSM,
      // Secrets Manager or S3 calls KMS on this role's behalf.
      {
        Sid      = "KmsLookup"
        Effect   = "Allow"
        Action   = ["kms:DescribeKey", "kms:ListAliases"]
        Resource = ["*"]
      },
      {
        Sid      = "KmsViaServices"
        Effect   = "Allow"
        Action   = ["kms:Decrypt", "kms:Encrypt", "kms:GenerateDataKey"]
        Resource = ["*"]
        Condition = {
          StringEquals = {
            "kms:ViaService" = [
              "ssm.${local.pr_env_region}.amazonaws.com",
              "secretsmanager.${local.pr_env_region}.amazonaws.com",
              "s3.${local.pr_env_region}.amazonaws.com",
            ]
          }
        }
      },
      // ECS: deregister, describe and list take no resource-level
      // permission; registering, services and tags are limited to pr-*.
      {
        Sid    = "EcsTaskDefinitions"
        Effect = "Allow"
        Action = [
          "ecs:DeregisterTaskDefinition",
          "ecs:DescribeTaskDefinition",
          "ecs:ListTaskDefinitions",
        ]
        Resource = ["*"]
      },
      {
        Sid    = "EcsPrServices"
        Effect = "Allow"
        Action = [
          "ecs:CreateService",
          "ecs:DeleteService",
          "ecs:DescribeServices",
          "ecs:RegisterTaskDefinition",
          "ecs:TagResource",
          "ecs:UntagResource",
          "ecs:UpdateService",
        ]
        Resource = [
          "arn:aws:ecs:${local.pr_env_region}:${local.account_id}:service/ztmf/pr-*",
          "arn:aws:ecs:${local.pr_env_region}:${local.account_id}:task-definition/pr-*:*",
        ]
      },
      {
        Sid    = "EcsRead"
        Effect = "Allow"
        Action = [
          "ecs:DescribeClusters",
          "ecs:DescribeTasks",
          "ecs:ListServices",
          "ecs:ListTasks",
        ]
        Resource = ["*"]
      },
      // The task roles the pr-env root creates and hands to ECS, by name, with
      // the CMS boundary every role needs and the one managed policy it uses.
      // No condition key limits what PutRolePolicy writes, so a workflow on
      // this role could give a ztmf_pr-* task role anything the poweruser
      // boundary allows and run a task with it. Fixed task roles in the dev
      // root would close that.
      {
        Sid    = "TaskRoles"
        Effect = "Allow"
        Action = [
          "iam:DeleteRole",
          "iam:DeleteRolePolicy",
          "iam:GetRole",
          "iam:GetRolePolicy",
          "iam:ListAttachedRolePolicies",
          "iam:ListInstanceProfilesForRole",
          "iam:ListRolePolicies",
          "iam:PutRolePolicy",
          "iam:TagRole",
          "iam:UntagRole",
          "iam:UpdateAssumeRolePolicy",
        ]
        Resource = [local.pr_env_task_roles]
      },
      {
        Sid      = "TaskRoleCreate"
        Effect   = "Allow"
        Action   = ["iam:CreateRole", "iam:PutRolePermissionsBoundary"]
        Resource = [local.pr_env_task_roles]
        Condition = {
          StringEquals = { "iam:PermissionsBoundary" = local.pr_env_role_boundary }
        }
      },
      {
        Sid      = "TaskRolePolicyAttach"
        Effect   = "Allow"
        Action   = ["iam:AttachRolePolicy", "iam:DetachRolePolicy"]
        Resource = [local.pr_env_task_roles]
        Condition = {
          ArnEquals = { "iam:PolicyARN" = "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy" }
        }
      },
      {
        Sid      = "TaskRolePass"
        Effect   = "Allow"
        Action   = ["iam:PassRole"]
        Resource = [local.pr_env_task_roles]
        Condition = {
          StringEquals = { "iam:PassedToService" = "ecs-tasks.amazonaws.com" }
        }
      },
      // A target group and a rule on dev's shared HTTPS listener per
      // environment. Describe calls take no resource-level permission.
      {
        Sid      = "ElbRead"
        Effect   = "Allow"
        Action   = ["elasticloadbalancing:Describe*"]
        Resource = ["*"]
      },
      {
        Sid    = "ElbPrTargetGroups"
        Effect = "Allow"
        Action = [
          "elasticloadbalancing:AddTags",
          "elasticloadbalancing:CreateTargetGroup",
          "elasticloadbalancing:DeleteTargetGroup",
          "elasticloadbalancing:ModifyTargetGroup",
          "elasticloadbalancing:ModifyTargetGroupAttributes",
          "elasticloadbalancing:RemoveTags",
        ]
        Resource = ["arn:aws:elasticloadbalancing:${local.pr_env_region}:${local.account_id}:targetgroup/ztmf-pr-*/*"]
      },
      // New rules go on dev's shared listener, tagged at creation. After that,
      // only rules carrying the ztmf-pr-env tag can change, so dev's own
      // login and /api/* rules are out of reach.
      {
        Sid      = "ElbCreatePrRule"
        Effect   = "Allow"
        Action   = ["elasticloadbalancing:CreateRule"]
        Resource = ["arn:aws:elasticloadbalancing:${local.pr_env_region}:${local.account_id}:listener/app/ztmf-api/*"]
      },
      {
        Sid      = "ElbTagPrRuleOnCreate"
        Effect   = "Allow"
        Action   = ["elasticloadbalancing:AddTags"]
        Resource = ["arn:aws:elasticloadbalancing:${local.pr_env_region}:${local.account_id}:listener-rule/app/ztmf-api/*"]
        Condition = {
          StringEquals = { "elasticloadbalancing:CreateAction" = "CreateRule" }
        }
      },
      {
        Sid    = "ElbPrListenerRules"
        Effect = "Allow"
        Action = [
          "elasticloadbalancing:AddTags",
          "elasticloadbalancing:DeleteRule",
          "elasticloadbalancing:ModifyRule",
          "elasticloadbalancing:RemoveTags",
          "elasticloadbalancing:SetRulePriorities",
        ]
        Resource = ["arn:aws:elasticloadbalancing:${local.pr_env_region}:${local.account_id}:listener-rule/app/ztmf-api/*"]
        Condition = {
          StringLike = { "aws:ResourceTag/ztmf-pr-env" = "pr-*" }
        }
      },
      // One security group per environment, created with the ztmf-pr-env tag
      // and changed or deleted only while it carries it.
      {
        Sid      = "Ec2Read"
        Effect   = "Allow"
        Action   = ["ec2:Describe*"]
        Resource = ["*"]
      },
      {
        Sid      = "SecurityGroupCreateInVpc"
        Effect   = "Allow"
        Action   = ["ec2:CreateSecurityGroup"]
        Resource = ["arn:aws:ec2:${local.pr_env_region}:${local.account_id}:vpc/*"]
      },
      {
        Sid      = "SecurityGroupCreateTagged"
        Effect   = "Allow"
        Action   = ["ec2:CreateSecurityGroup"]
        Resource = ["arn:aws:ec2:${local.pr_env_region}:${local.account_id}:security-group/*"]
        Condition = {
          StringLike = { "aws:RequestTag/ztmf-pr-env" = "pr-*" }
        }
      },
      {
        Sid      = "SecurityGroupTagOnCreate"
        Effect   = "Allow"
        Action   = ["ec2:CreateTags"]
        Resource = ["arn:aws:ec2:${local.pr_env_region}:${local.account_id}:security-group/*"]
        Condition = {
          StringEquals = { "ec2:CreateAction" = "CreateSecurityGroup" }
        }
      },
      {
        Sid    = "SecurityGroupChange"
        Effect = "Allow"
        Action = [
          "ec2:AuthorizeSecurityGroupEgress",
          "ec2:AuthorizeSecurityGroupIngress",
          "ec2:CreateTags",
          "ec2:DeleteSecurityGroup",
          "ec2:DeleteTags",
          "ec2:RevokeSecurityGroupEgress",
          "ec2:RevokeSecurityGroupIngress",
        ]
        Resource = ["arn:aws:ec2:${local.pr_env_region}:${local.account_id}:security-group/*"]
        Condition = {
          StringLike = { "aws:ResourceTag/ztmf-pr-env" = "pr-*" }
        }
      },
      {
        Sid    = "SecurityGroupRules"
        Effect = "Allow"
        Action = [
          "ec2:AuthorizeSecurityGroupEgress",
          "ec2:AuthorizeSecurityGroupIngress",
          "ec2:RevokeSecurityGroupEgress",
          "ec2:RevokeSecurityGroupIngress",
        ]
        Resource = ["arn:aws:ec2:${local.pr_env_region}:${local.account_id}:security-group-rule/*"]
      },
      // Cognito mode: one user pool per environment, with its client, domain
      // and seeded users. CreateUserPool and DescribeUserPoolDomain take no
      // resource-level permission; everything else needs the pool's
      // ztmf-pr-env tag.
      {
        Sid      = "CognitoNoResource"
        Effect   = "Allow"
        Action   = ["cognito-idp:CreateUserPool", "cognito-idp:DescribeUserPoolDomain"]
        Resource = ["*"]
      },
      {
        Sid    = "CognitoPools"
        Effect = "Allow"
        Action = [
          "cognito-idp:AdminCreateUser",
          "cognito-idp:AdminDeleteUser",
          "cognito-idp:AdminDisableUser",
          "cognito-idp:AdminEnableUser",
          "cognito-idp:AdminGetUser",
          "cognito-idp:AdminSetUserPassword",
          "cognito-idp:AdminUpdateUserAttributes",
          "cognito-idp:CreateUserPoolClient",
          "cognito-idp:CreateUserPoolDomain",
          "cognito-idp:DeleteUserPool",
          "cognito-idp:DeleteUserPoolClient",
          "cognito-idp:DeleteUserPoolDomain",
          "cognito-idp:DescribeUserPool",
          "cognito-idp:DescribeUserPoolClient",
          "cognito-idp:GetUserPoolMfaConfig",
          "cognito-idp:ListTagsForResource",
          "cognito-idp:SetUserPoolMfaConfig",
          "cognito-idp:TagResource",
          "cognito-idp:UntagResource",
          "cognito-idp:UpdateUserPool",
          "cognito-idp:UpdateUserPoolClient",
        ]
        Resource = ["arn:aws:cognito-idp:${local.pr_env_region}:${local.account_id}:userpool/*"]
        Condition = {
          StringLike = { "aws:ResourceTag/ztmf-pr-env" = "pr-*" }
        }
      },
      {
        Sid    = "LogGroups"
        Effect = "Allow"
        Action = [
          "logs:CreateLogGroup",
          "logs:DeleteLogGroup",
          "logs:DeleteRetentionPolicy",
          "logs:ListTagsForResource",
          "logs:ListTagsLogGroup",
          "logs:PutRetentionPolicy",
          "logs:TagLogGroup",
          "logs:TagResource",
          "logs:UntagResource",
        ]
        Resource = [
          "arn:aws:logs:${local.pr_env_region}:${local.account_id}:log-group:/ztmf/pr-*",
          "arn:aws:logs:${local.pr_env_region}:${local.account_id}:log-group:/ztmf/pr-*:*",
        ]
      },
      {
        Sid      = "LogGroupsDescribe"
        Effect   = "Allow"
        Action   = ["logs:DescribeLogGroups"]
        Resource = ["*"]
      },
    ]
  })
}
