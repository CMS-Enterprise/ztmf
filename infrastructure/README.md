# Infrastructure as Code via Terraform

All resources are managed by Terraform, and deployed into AWS in the `us-east-1` region. There are two environments, `dev` and `prod` each with their own account. Each account houses it's own S3 bucket that serves as a backend state store for Terraform. Both accounts include numerous resources such as VPCs that were created by CMS cloud and cloud security. These are referenced with `data` sources to avoid hardcoding ids.

## State

State is stored in S3 buckets, each environment with it's own bucket and state store.
To switch environments you need to init terraform with a backend config like so:
```bash
terraform init -backend-config="config/backend-<env>.tf" -reconfigure
```
Where `<env>` is one of `dev` or `prod`. See files in `infrastructure/config/`.

## Vars

While there are few input vars, they still need to be passed in during plan and apply
```bash
terraform <plan|apply> -var-file="tfvars/<env>.tfvars"
```
Where `<env>` is one of `dev` or `prod`. See files in `infrastructure/tfvars`.

## Modules

Only one custom module is used as a factory for IAM roles. CMS requires that all IAM roles include a `path` and `permissions_boundary`. These are expressed in `modules/role/main.tf` and all roles created for use by the application are created by calling the module like the following example:

```hcl
module <identifier> {
  name                = <name>
  source              = "./modules/role"
  principal           = { Service = "ecs-tasks.amazonaws.com" } // example
  ...
}
```
