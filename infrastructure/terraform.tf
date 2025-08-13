terraform {
  required_version = ">= 1.10.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.82.2"
    }
  }

  backend "s3" {}
}

provider "aws" {
  region = "us-east-1"
}
