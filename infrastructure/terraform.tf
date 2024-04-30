terraform {
  required_providers {
    aws = {
      source = "hashicorp/aws"
    }
  }

  backend "s3" {}
}

provider "aws" {
  region  = "us-east-1"
}
