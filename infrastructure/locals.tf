locals {
  // adding reference here to make other references shorter to type "local.account_id" :)
  account_id = data.aws_caller_identity.current.account_id
  
  // join because terraform is stupid when it comes to sets, lists, and tuples
  db_cred_secret = join("",data.aws_secretsmanager_secrets.rds.arns)

  domain_name = "${var.domain_name_prefix}ztmf.cms.gov"

  // simplify referencing of json object fields for aws_verifiedaccess_trust_provider.ztmf_idmokta.oidc_options
  // technically only one of the fields was a true secret (client_secret), but since we have the space here
  //  we can use it to simplify code instead of placing all the other fields in TF vars
  oidc_options = jsondecode(data.aws_secretsmanager_secret_version.ztmf_va_trust_provider_current.secret_string)
}
