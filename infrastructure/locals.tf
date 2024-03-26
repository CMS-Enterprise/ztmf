locals {
  # created by Elizabeth through Digicert and imported manually by Richard Jones
  account_id              = data.aws_caller_identity.current.account_id
  certificate_arn         = "arn:aws:acm:us-east-1:${local.account_id}:certificate/9e5b1cf5-50cd-4bc4-bbe4-ca39b9be7c0d"
  domain_name             = "dev.ztmf.cms.gov"
  zscaler_prefix_list_id  = "pl-021a6f9575cc1b686"
  private_route_table_ids = ["rtb-0ca100f90fdf13a28", "rtb-04f14b8ad2a7b3378", "rtb-00ad382390709fd2f"]
  db_cred_secret          = join("",data.aws_secretsmanager_secrets.rds.arns) // join because terraform is stupid when it comes to sets, lists, and tuples
}
