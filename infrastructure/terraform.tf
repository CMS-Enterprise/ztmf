terraform {
  required_version = "~> 1.5.7"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.33.0"
    }
  }

  backend "s3" {}
}

provider "aws" {
  region = "us-east-1"
}
