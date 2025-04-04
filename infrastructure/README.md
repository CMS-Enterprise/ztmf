# ZTMF Infrastructure

This directory contains the Terraform configuration for the ZTMF (Zero Trust Maturity Framework) application infrastructure on AWS. The infrastructure is deployed in the `us-east-1` region with separate environments for `dev` and `prod`, each in its own AWS account.

## Architecture Overview

The ZTMF application follows a modern cloud-native architecture with the following key components:

```mermaid
---
title: ZTMF Architecture
---
C4Context
    Person(user, "User", "CMS employee accessing the ZTMF application")
    
    Boundary(aws, "AWS Cloud", "us-east-1 region") {
        System(cloudfront, "CloudFront", "Content delivery network with WAF protection")
        
        Boundary(vpc, "VPC", "Private network") {
            System_Ext(alb, "Internal ALB", "Application Load Balancer with OIDC authentication")
            
            Boundary(ecs, "ECS Cluster", "Container orchestration") {
                Container(api, "API Container", "Go API running in Fargate")
            }
            
            SystemDb(rds, "Aurora PostgreSQL", "Serverless v2 database")
            
            System(bastion, "Bastion Host", "EC2 instance for database access")
            
            System(endpoints, "VPC Endpoints", "Private access to AWS services")
        }
        
        SystemQueue(s3, "S3 Buckets", "Static web assets and logs storage")
        System(ecr, "ECR", "Container registry for API images")
        System(secrets, "Secrets Manager", "Stores credentials and certificates")
    }
    
    Rel(user, cloudfront, "HTTPS")
    Rel(cloudfront, s3, "Fetch static assets")
    Rel(cloudfront, alb, "API requests")
    Rel(alb, api, "Forward authenticated requests")
    Rel(api, rds, "Query database")
    Rel(api, secrets, "Retrieve secrets")
    Rel(bastion, rds, "Database administration")
    Rel_Back(api, endpoints, "Access AWS services")
```

## Key Components

### Frontend
- **CloudFront Distribution**: Serves static web assets and proxies API requests
- **S3 Bucket**: Stores static web assets (HTML, CSS, JS)
- **WAF**: Web Application Firewall provided by CMS Cloud

### Backend
- **Internal ALB**: Application Load Balancer with OIDC authentication (via IDM Okta)
- **ECS Cluster**: Runs the API container using Fargate (serverless)
- **ECR Repository**: Stores the API container images
- **Aurora PostgreSQL**: Serverless v2 database for application data
- **Secrets Manager**: Stores credentials, certificates, and other sensitive information

### Networking
- **VPC**: Provided by CMS Cloud
- **Private Subnets**: Host all application components
- **VPC Endpoints**: Allow private access to AWS services
- **Security Groups**: Control traffic between components

### Access & Authentication
- **OIDC Authentication**: Integrated with the ALB for user authentication
- **IAM Roles**: Defined with appropriate permissions for each component
- **GitHub Actions OIDC**: Enables CI/CD workflows

## State Management

State is stored in S3 buckets, with each environment having its own bucket and state store.
To switch environments, initialize Terraform with the appropriate backend config:

```bash
terraform init -backend-config="config/backend-<env>.tf" -reconfigure
```

Where `<env>` is one of `dev` or `prod`. See files in `infrastructure/config/`.

## Variables

Input variables are defined in `variables.tf` and environment-specific values are in `tfvars/<env>.tfvars`.
To apply with the correct variables:

```bash
terraform <plan|apply> -var-file="tfvars/<env>.tfvars"
```

## Custom Modules

### IAM Role Module

A custom module is used as a factory for IAM roles. CMS requires that all IAM roles include a `path` and `permissions_boundary`. These are expressed in `modules/role/main.tf` and all roles created for use by the application are created by calling the module:

```hcl
module <identifier> {
  name                = <name>
  source              = "./modules/role"
  principal           = { Service = "ecs-tasks.amazonaws.com" } // example
  ...
}
```

## Security Features

- **HTTPS Only**: All traffic is encrypted in transit
- **Private Networking**: All application components run in private subnets
- **WAF Protection**: CloudFront distribution is protected by CMS Cloud WAF
- **OIDC Authentication**: Users are authenticated via OIDC
- **Secrets Management**: Sensitive information is stored in AWS Secrets Manager
- **Geo Restriction**: CloudFront distribution is restricted to US locations only
- **Content Security Policy**: Strict CSP headers are applied to all responses

## Deployment

The infrastructure is deployed using Terraform and GitHub Actions. The GitHub Actions workflow is configured to use OIDC for authentication with AWS.

## Database Access

A bastion host is provided for database administration. The bastion host is accessible via AWS Systems Manager Session Manager and has access to the database.
