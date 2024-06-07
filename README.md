# Zero Trust Maturity Framework (ZTMF) Scoring


The ZTMF Scoring Application allows ADOs to view their Zero Trust Maturity score online. An upcoming release will allow new ADOs to answer the questionnaire from scratch, and existing ADOs to update their answers, all within a web-based interface. The interface and the API are protected by AWS Verified Access which requires authentication via IDM (Okta).

This monorepo contains the following major components:
- `backend/` includes a GraphQL API and an ETL process both written in Go
- `infrastructure/` includes all AWS resources as IaC managed by Terraform
- `ui/` includes a React-based SPA written in Typescript
- `.github/workflows` contains workflows for Github Actions to test, build, and deploy to AWS

## Architecture

The ZTMF Scoring Application is comprised of a React-based Single-Page Application (SPA) that retrieves data from the GraphQL API. The web assets for the SPA are hosted in an S3 bucket, and the API is hosted as an ECS service with containers deployed via Fargate.

Both the API ECS service, and the S3 bucket are configured as targets behind an _internal_ application load balancer (ALB), with S3 connectivity provided by PrivateLink VPC endpoints. The internal ALB is the target for the AWS Verified Access endpoint. The the public domain name points to the Verified Access endpoint which in turn acts as a proxy to the application, allowing access to only known trusted identites. AWS Verified Access is configured to use IDM (Okta) as the user identity trust provider. This allows users with the ZTMF job code to login via IDM and access the application.

Data delivered by the API is stored in an RDS Aurora serverless PostgreSQL server.
