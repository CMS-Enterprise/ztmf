locals {
  // adding reference here to make other references shorter to type "local.account_id" :)
  account_id = data.aws_caller_identity.current.account_id
  
  // join because terraform is stupid when it comes to sets, lists, and tuples
  db_cred_secret = join("",data.aws_secretsmanager_secrets.rds.arns)

  domain_name = "${var.domain_name_prefix}ztmf.cms.gov"
}
