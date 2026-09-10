# Per-PR environment root

One instance of this root per open PR (ztmf-misc#341, epic ztmf-misc#321). It reads dev's cluster, ALB, listener, VPC, and Okta client by name and creates only per-PR resources: a three-container Fargate task (API, Postgres sidecar, nginx frontend), a one-task service, a target group, one ALB listener rule with CMS login, a 7-day log group, two SSM SecureStrings, and two roles. State is keyed per repo and PR in the dev state bucket, so it never touches the dev root's state. The workflows in ztmf-misc#342 and #343 drive it; the commands below are the manual equivalent.

```shell
cd infrastructure/pr-env
terraform init -reconfigure \
  -backend-config=bucket=ztmf-terraform-state-use1-dev \
  -backend-config=key=pr/ztmf/123 \
  -backend-config=region=us-east-1
terraform apply -auto-approve \
  -var repo=ztmf -var pr_number=123 \
  -var api_image_tag=pr-ztmf-123-abc1234 -var ui_image_tag=ui-def5678
terraform output url        # https://dev.ztmf.cms.gov/pr/ztmf/123/
terraform destroy -auto-approve -var repo=ztmf -var pr_number=123 \
  -var api_image_tag=pr-ztmf-123-abc1234 -var ui_image_tag=ui-def5678
```

`repo` is `ztmf` or `ui` and picks the URL path and the ALB priority band (10000+n or 20000+n), so ztmf#123 and ui#123 coexist. `api_image_tag` must be a `test`-target build of the API image (it carries the empire seed); `ui_image_tag` is an image from the ztmf-ui Dockerfile. The environment authenticates at the ALB with the dev Okta client, then the frontend uses a bearer token for `test_user_email` (default Grand Moff Tarkin) minted inside the task from the per-environment HS256 secret.

Prerequisites in the dev root: `pr_env_enabled = true` (the `/pr/*` CloudFront behavior and the `ztmf/ui` ECR repo). Debug a live environment with `aws ecs execute-command --cluster ztmf --task <id> --container api --interactive --command sh` or from the `/ztmf/pr-<repo>-<n>` log group.
