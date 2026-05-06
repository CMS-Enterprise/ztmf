// GitHub OIDC provider is account-scoped (one per token URL). Created in
// dev/prod; impl reuses dev's existing provider implicitly through CI's
// configure-aws-credentials action (which trusts the provider by URL, not
// by terraform-tracked ARN). No data source needed here.
resource "aws_iam_openid_connect_provider" "github_actions" {
  count          = local.manage_account_singletons ? 1 : 0
  url            = "https://token.actions.githubusercontent.com"
  client_id_list = ["sts.amazonaws.com"]
  thumbprint_list = [
    "6938fd4d98bab03faadb97b34396831e3780aea1",
    "1c58a3a8518e8759bf075b76b750d4f2df264fcd"
  ]
}

// GitHub Actions deploy role. Account-scoped IAM role; impl shares dev's role
// (same role assumes via OIDC for impl branch deploys via env-routed secrets).
module "github_actions" {
  count     = local.manage_account_singletons ? 1 : 0
  name      = "ztmf_github_actions"
  source    = "./modules/role"
  principal = { Federated = aws_iam_openid_connect_provider.github_actions[0].arn }
  managed_policy_arns = [
    "arn:aws:iam::${local.account_id}:policy/CMSApprovedAWSServices",
    "arn:aws:iam::${local.account_id}:policy/ADO-Restriction-Policy",
    "arn:aws:iam::${local.account_id}:policy/ct-iamCreateUserRestrictionPolicy",
    "arn:aws:iam::${local.account_id}:policy/CMSCloudApprovedRegions",
    "arn:aws:iam::aws:policy/AdministratorAccess"
  ]
  condition = {
    "ForAllValues:StringEquals" = {
      "token.actions.githubusercontent.com:aud" = ["sts.amazonaws.com"]
    }
    "ForAnyValue:StringLike" = {
      "token.actions.githubusercontent.com:sub" = [
        "repo:CMS-Enterprise/ztmf:*",
        "repo:CMS-Enterprise/ztmf-ui:*"
      ],
    }
  }
}
