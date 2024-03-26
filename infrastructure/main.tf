terraform {
  required_providers {
    aws = {
      source = "hashicorp/aws"
    }
  }

  backend "s3" {
    bucket  = "ztmf-terraform-state-use1-dev"
    key     = "tfstate"
    region  = "us-east-1"
    profile = "ztmf-dev"
  }
}

provider "aws" {
  region  = "us-east-1"
  profile = "ztmf-dev"
}
