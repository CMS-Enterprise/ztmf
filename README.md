[![Go Report Card](https://goreportcard.com/badge/github.com/CMS-Enterprise/ztmf/backend)](https://goreportcard.com/report/github.com/CMS-Enterprise/ztmf/backend) [![Backend](https://github.com/CMS-Enterprise/ztmf/actions/workflows/backend.yml/badge.svg)](https://github.com/CMS-Enterprise/ztmf/actions/workflows/backend.yml) [![Infrastructure](https://github.com/CMS-Enterprise/ztmf/actions/workflows/infrastructure.yml/badge.svg)](https://github.com/CMS-Enterprise/ztmf/actions/workflows/infrastructure.yml)
# Zero Trust Maturity Framework (ZTMF) Scoring

The ZTMF Scoring Application allows ADOs to answer HHS Zero Trust data calls, and view their Zero Trust Maturity score online.

This repo contains the following major components:
- `.github/workflows` contains workflows for Github Actions to test, build, and deploy to AWS
- `backend/` includes a REST API and an ETL process both written in Go
- `infrastructure/` includes all AWS resources as IaC managed by Terraform

## Required Tools

1. [Go](https://go.dev/) at the required version specified in [backend/go.mod](backend/go.mod#L3)
2. [Docker](https://www.docker.com/) for running the development environment
3. [Make](https://www.gnu.org/software/make/) for development workflow automation
4. PostgreSQL management tool of your choice such as [pgAdmin](https://www.pgadmin.org/) (optional for production database access)
5. [AWS CLI](https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html) for establishing SSM tunnels (production only)
6. [Terraform](https://developer.hashicorp.com/terraform/install?product_intent=terraform) for deploying infrastructure changes manually if necessary (though CICD should handle most changes)
7. [Emberfall](https://github.com/aquia-inc/emberfall) for running smoke tests locally (very helpful when adding new routes or parameters)

## Development Environment

The ZTMF project includes a complete containerized development environment using Docker Compose and a Makefile for easy setup.

### Quick Start

```bash
# Clone the repository and navigate to the project root
git checkout feature/pillar-score-breakdown  # or your working branch
make dev-setup
```

This single command will:
- Generate a PostgreSQL database with random passwords
- Create Docker Compose configuration for development
- Start PostgreSQL (port 54321) and Go API (port 3000) containers
- Populate the database with Star Wars-themed test data
- Run database migrations automatically

### Available Commands

```bash
make dev-setup         # Full development environment setup
make dev-up            # Start development services  
make dev-down          # Stop development services
make dev-logs          # Show service logs
make clean             # Clean up generated files

# JWT Token Generation for Testing
make generate-jwt EMAIL=your.email@example.com
make test-empire-data  # Get tokens for all test users
```

### Test Data

The development environment includes anonymized Star Wars Empire-themed test data with:
- 4 test users (1 ADMIN, 3 ISSO roles)
- 3 FISMA systems (Death Star, Executor, Shield Generator)
- Complete Zero Trust questionnaire (18 questions across 6 pillars)
- Sample scores demonstrating different maturity levels
- 2 data calls for testing historical comparisons

### Testing the API

After running `make dev-setup`, test the pillar scores feature:

```bash
# Get test tokens
make test-empire-data

# Test pillar breakdown API (replace TOKEN with output from above)
curl -H "Authorization: TOKEN" \
     "http://localhost:3000/api/v1/scores/aggregate?include_pillars=true"
```

### Development Workflow

1. **Start Environment**: `make dev-setup`
2. **Code Changes**: Edit Go files - the container automatically rebuilds
3. **Database Changes**: Modify migrations, restart with `make dev-down && make dev-up`
4. **Run Tests**: `emberfall ./backend/emberfall_tests.yml`
5. **Clean Up**: `make dev-down` or `make clean`

## Architecture

The ZTMF Scoring Application is comprised of a React-based Single-Page Application (SPA) that retrieves data from a REST API. The web assets for the SPA are hosted in an S3 bucket, and the API is hosted as an ECS service with containers deployed via Fargate behind an application load balancer. CloudFront provides the entrypoint with caching enabled for static assets, and a WAF for geofencing and other security measures. The database is provided by AWS Aurora Serverless V2 PostgreSQL.

### Data Synchronization to Snowflake

An automated data synchronization process exports ZTMF data from PostgreSQL to Snowflake for business intelligence and reporting:

```mermaid
graph LR
    subgraph "ZTMF Application"
        A[React Frontend<br/>S3 + CloudFront]
        B[Go API<br/>ECS Fargate]
        C[PostgreSQL<br/>Aurora Serverless v2]
        A --> B
        B --> C
    end
    
    subgraph "Data Pipeline"
        D[EventBridge<br/>Quarterly Schedule]
        E[Lambda Function<br/>Go Runtime]
        F[Snowflake<br/>Data Warehouse]
    end
    
    subgraph "Monitoring"
        G[CloudWatch Logs]
        H[CloudWatch Alarms]
        I[Dead Letter Queue]
    end
    
    D --> E
    C --> E
    E --> F
    E --> G
    E --> H
    E --> I
    
    style A fill:#e1f5fe
    style B fill:#e8f5e8
    style C fill:#fff3e0
    style E fill:#f3e5f5
    style F fill:#e0f2f1
```

#### Sync Configuration

- **Schedule**: Quarterly in production (1st of every 3rd month at 2 AM UTC)
- **Development**: Weekly dry-runs on Mondays at 9 AM UTC
- **Tables**: All 12 ZTMF tables synchronized with proper ordering
- **Environment**: Dry-run mode in dev, real sync in production
- **Security**: AWS Secrets Manager for Snowflake credentials

## Backend

The backend is a REST API written in Go. See [backend/README](backend/README.md)

## Infrastructure

The infrastructure is managed by Terraform. See [infrastructure/README](infrastructure/README.md)

## CI/CD Workflows

The project uses GitHub Actions for continuous integration and deployment. The workflows are organized into modular components that are orchestrated differently for development and production environments. GitHub Secrets secures sensitive values, and authentication to AWS is provided via OIDC to an IAM IDP.

### Workflow Components

1. **Analysis (`analysis.yml`)**
   - Performs code quality and security checks
   - Lints Go code using staticcheck
   - Lints Terraform code using tflint
   - Runs Snyk security scans for Go code and infrastructure as code

2. **Backend (`backend.yml`)**
   - Builds and tests the backend service
   - Creates a Docker image for the backend
   - Runs security scanning on the Docker image
   - Performs smoke tests using [Emberfall](https://github.com/aquia-inc/emberfall)
   - Pushes the image to ECR
   - Updates SSM parameter with the new image tag

3. **Infrastructure (`infrastructure.yml`)**
   - Deploys AWS infrastructure using Terraform
   - Configures environment-specific settings
   - Applies Terraform changes with auto-approve

### Orchestration

The workflows are orchestrated differently based on the environment:

**Development Environment (`orchestration-dev.yml`)**
- Triggered on pull requests to the main branch
- Runs analysis on all PRs
- For non-draft PRs, checks for changes in the backend code
- If backend changes are detected, runs the backend workflow for DEV
- Finally runs the infrastructure workflow for DEV

**Production Environment (`orchestration-prod.yml`)**
- Triggered when a pull request to main is merged (closed with merge)
- Runs analysis, backend, and infrastructure workflows sequentially for PROD
- Only executes if the PR was actually merged

### Workflow Sequence Diagram

```mermaid
sequenceDiagram
    participant PR as Pull Request or Merge
    participant Orchestration as Orchestration Workflow
    participant Analysis as Analysis Workflow
    participant Backend as Backend Workflow
    participant Infra as Infrastructure Workflow
        
    PR->>Orchestration: Trigger orchestration-<env>.yml
    Orchestration->>Analysis: Trigger analysis.yml
    alt Backend Changes Detected
      Orchestration->>Backend: Trigger backend.yml
    end
    Orchestration->>Infra: Trigger infrastructure.yml
 
```

## User Interface 

ZTMF UI has its own [repository](https://github.com/cms-enterprise/ztmf-ui).
