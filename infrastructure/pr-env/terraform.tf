// Per-PR environment root (ztmf-misc#341). One instance per PR, state keyed
// pr/<repo>/<n> in the dev state bucket, passed at init:
//   terraform init -backend-config=bucket=ztmf-terraform-state-use1-dev \
//     -backend-config=key=pr/ztmf/123 -backend-config=region=us-east-1
// It never touches the dev root's state; it only reads dev's shared resources.
terraform {
  required_version = ">= 1.10.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.82.2"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.6"
    }
  }

  backend "s3" {}
}

provider "aws" {
  region = "us-east-1"
  default_tags {
    tags = {
      ztmf-pr-env    = local.env_name
      ztmf-pr-repo   = var.repo
      ztmf-pr-number = tostring(var.pr_number)
    }
  }
}
