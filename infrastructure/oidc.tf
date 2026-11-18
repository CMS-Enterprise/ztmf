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

// OIDC subjects allowed to assume the deploy role, one list per account. Both
// repos use GitHub's default subject template, so a job that names a GitHub
// Environment presents "repo:<repo>:environment:<name>" (case-sensitive, and
// each repo's environment names differ in case), and a job without one
// presents "repo:<repo>:pull_request" on a PR event or
// "repo:<repo>:ref:<ref>" otherwise.
//
// prod: only the PROD (ztmf) and prod (ztmf-ui) environment jobs.
// dev: the DEV and IMPL environment jobs, plus the per-PR environment
// workflows, which assume this same role through the repo-level
// PR_ENV_ROLEARN secret: their build, teardown, and publish jobs run on
// pull_request with no environment, their deploy job runs in a pr-<repo>-<n>
// environment, and the reaper runs on schedule from main (so a reaper dispatch
// from any other branch is refused).
locals {
  github_actions_subjects_prod = [
    "repo:CMS-Enterprise/ztmf:environment:PROD",
    "repo:CMS-Enterprise/ztmf-ui:environment:prod",
  ]
  github_actions_subjects_dev = [
    "repo:CMS-Enterprise/ztmf:environment:DEV",
    "repo:CMS-Enterprise/ztmf:environment:IMPL",
    "repo:CMS-Enterprise/ztmf:environment:pr-ztmf-*",
    "repo:CMS-Enterprise/ztmf:pull_request",
    "repo:CMS-Enterprise/ztmf:ref:refs/heads/main",
    "repo:CMS-Enterprise/ztmf-ui:environment:dev",
    "repo:CMS-Enterprise/ztmf-ui:environment:impl",
    "repo:CMS-Enterprise/ztmf-ui:environment:pr-ui-*",
    "repo:CMS-Enterprise/ztmf-ui:pull_request",
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
  // aud and sub are single-valued claims, so they take the plain operators.
  // ForAllValues also matches when the key is absent from the token.
  condition = {
    "StringEquals" = {
      "token.actions.githubusercontent.com:aud" = ["sts.amazonaws.com"]
    }
    "StringLike" = {
      "token.actions.githubusercontent.com:sub" = var.environment == "prod" ? local.github_actions_subjects_prod : local.github_actions_subjects_dev
    }
  }
}
