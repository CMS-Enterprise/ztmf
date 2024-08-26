# Zero Trust Maturity Framework (ZTMF) Scoring


The ZTMF Scoring Application allows ADOs to view their Zero Trust Maturity score online. An upcoming release will allow new ADOs to answer the questionnaire from scratch, and existing ADOs to update their answers, all within a web-based interface. The interface and the API are protected by AWS Verified Access which requires authentication via IDM (Okta).

This monorepo contains the following major components:
- `backend/` includes a GraphQL API and an ETL process both written in Go
- `infrastructure/` includes all AWS resources as IaC managed by Terraform
- `.github/workflows` contains workflows for Github Actions to test, build, and deploy to AWS

## Architecture

The ZTMF Scoring Application is comprised of a React-based Single-Page Application (SPA) that retrieves data from a REST API. The web assets for the SPA are hosted in an S3 bucket, and the API is hosted as an ECS service with containers deployed via Fargate.

Both the API ECS service, and the S3 bucket are configured as origins behind a CloudFront distribution.

Data delivered by the API is stored in an RDS Aurora serverless PostgreSQL server.

## User Interface 

ZTMF UI has now been moved to a [separate repo](https://github.com/cms-enterprise/ztmf-ui).
