// Cognito sign-in for demo environments (auth_provider = "cognito"). The pool
// replaces dev's Okta client at this environment's ALB rule, so people without
// CMS accounts can sign in as seeded test users. Nothing exists in okta mode.
//
// The ALB forwards the signed-in user to the API in x-amzn-oidc-data, signed
// with the ALB's regional ES256 key, the same header and validation path dev
// uses for Okta. The API resolves the user by the email claim, so each Cognito
// user lands with the role its seeded users row carries.
resource "aws_cognito_user_pool" "env" {
  count = local.cognito ? 1 : 0
  name  = "ztmf-${local.env_name}"

  username_attributes = ["email"]
  username_configuration {
    case_sensitive = false
  }

  // Test users only: nobody signs themselves up, and nothing emails them.
  admin_create_user_config {
    allow_admin_create_user_only = true
  }

  account_recovery_setting {
    recovery_mechanism {
      name     = "admin_only"
      priority = 1
    }
  }

  password_policy {
    minimum_length                   = 16
    require_lowercase                = true
    require_uppercase                = true
    require_numbers                  = true
    require_symbols                  = true
    temporary_password_validity_days = 7
  }

  deletion_protection = "INACTIVE"
}

// The domain prefix is global across AWS, so a random suffix keeps a rebuilt
// environment from colliding with a prefix still held by a deleted pool.
resource "random_id" "cognito_domain" {
  count       = local.cognito ? 1 : 0
  byte_length = 3
}

resource "aws_cognito_user_pool_domain" "env" {
  count        = local.cognito ? 1 : 0
  domain       = "ztmf-${local.env_name}-${random_id.cognito_domain[0].hex}"
  user_pool_id = aws_cognito_user_pool.env[0].id
}

// The ALB is the OAuth client: it runs the code flow against the hosted UI and
// returns to the dev host's /oauth2/idpresponse, the same redirect the Okta
// rule uses, so it must hold a client secret.
resource "aws_cognito_user_pool_client" "env" {
  count        = local.cognito ? 1 : 0
  name         = "ztmf-${local.env_name}-alb"
  user_pool_id = aws_cognito_user_pool.env[0].id

  generate_secret                      = true
  allowed_oauth_flows_user_pool_client = true
  allowed_oauth_flows                  = ["code"]
  allowed_oauth_scopes                 = ["openid", "email", "profile"]
  supported_identity_providers         = ["COGNITO"]
  callback_urls                        = [local.idp_response_url]
  explicit_auth_flows                  = ["ALLOW_REFRESH_TOKEN_AUTH"]
}

// One shared password for the environment's test users: the cognito_password
// variable when set (the workflow passes a repo secret), else generated here.
// Either way it is readable from SSM. The users hold synthetic data only.
resource "random_password" "cognito_users" {
  count            = local.cognito && var.cognito_password == "" ? 1 : 0
  length           = 24
  special          = true
  override_special = "!#%*-_=+"
  min_lower        = 1
  min_upper        = 1
  min_numeric      = 1
  min_special      = 1
}

resource "aws_ssm_parameter" "cognito_password" {
  count = local.cognito ? 1 : 0
  name  = "/ztmf/pr/${var.repo}/${var.pr_number}/cognito-password"
  type  = "SecureString"
  value = local.cognito_password
}

resource "aws_cognito_user" "demo" {
  for_each     = local.cognito ? toset(var.cognito_users) : toset([])
  user_pool_id = aws_cognito_user_pool.env[0].id
  username     = lower(each.value)

  attributes = {
    email          = lower(each.value)
    email_verified = "true"
  }

  // A permanent password, so the first sign-in is not a forced reset.
  password       = local.cognito_password
  message_action = "SUPPRESS"
}

// The UI image always writes an HS256 bearer token for TEST_USER_EMAIL into
// config.js. In cognito mode the API reads x-amzn-oidc-data instead, but test
// mode still accepts HS256 there, so the UI signs its token with a key the API
// does not hold: the token in the page can never authenticate.
resource "random_password" "ui_hs256" {
  count   = local.cognito ? 1 : 0
  length  = 48
  special = false
}

resource "aws_ssm_parameter" "ui_hs256" {
  count = local.cognito ? 1 : 0
  name  = "/ztmf/pr/${var.repo}/${var.pr_number}/ui-hs256-secret"
  type  = "SecureString"
  value = random_password.ui_hs256[0].result
}
