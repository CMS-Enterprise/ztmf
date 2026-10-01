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

`repo` is `ztmf` or `ui` and picks the URL path and the ALB priority band (10000+n or 20000+n), so ztmf#123 and ui#123 coexist. `api_image_tag` names an image in `ztmf/api-pr`, either the PR's own `pr-ztmf-<n>-<sha>` or the `main-<sha>` test-target build a ui PR runs; every image there carries the empire seed, and the root rejects anything else. `ui_image_tag` is an image in `ztmf/ui` from the ztmf-ui Dockerfile, `pr-ui-<n>-<sha>` or `ui-<sha>`. The environment authenticates at the ALB with the dev Okta client, then the frontend uses a bearer token for `test_user_email` (default Grand Moff Tarkin) minted inside the task from the per-environment HS256 secret.

Prerequisites in the dev root: `pr_env_enabled = true` (the `/pr/*` CloudFront behavior and the `ztmf/api-pr` and `ztmf/ui` ECR repos). Debug a live environment with `aws ecs execute-command --cluster ztmf --task <id> --container api --interactive --command sh` or from the `/ztmf/pr-<repo>-<n>` log group.

## Reaper and fleet alarm

Teardown on PR close is the primary cleanup. `scripts/pr-env-reaper.sh`, run nightly at 07:00 UTC by `.github/workflows/pr-env-reaper.yml` (ztmf-misc#344), is the backstop for both repos: it destroys any `pr/<repo>/<n>` environment whose PR is closed or has had no deploy for 7 days (the TTL; the next push rebuilds it and the PR gets a comment saying so), deletes that PR's images, and retires its `pr-<repo>-<n>` GitHub Environment. It also retires any `pr-*` environment whose PR is closed, and reports what it cannot fix: a destroy that failed twice, a `pr-*` ECS service with no state key, and an open PR's state key with no running service. Findings or a failed run go to one tracking issue in ztmf-misc titled "pr-env reaper: orphaned environments or a failed run".

Preview a run from a workstation, changing nothing:

```shell
AWS_PROFILE=ztmf-dev-ro scripts/pr-env-reaper.sh --dry-run
```

Or dispatch the workflow with `dry_run`, or with `force_failure` to exercise the tracking issue.

Independent of GitHub, the dev root's `ztmf-pr-env-task-count-dev` alarm (`infrastructure/monitoring-pr-env.tf`) pages the ztmf-alarms topic when more than `pr_env_max_environments` (10) pr-env tasks run for 10 minutes.
