variable "repo" {
  description = "Which repo's PR this environment previews: ztmf or ui. Selects the ALB priority band and the URL path."
  type        = string
  validation {
    condition     = contains(["ztmf", "ui"], var.repo)
    error_message = "repo must be ztmf or ui."
  }
}

variable "pr_number" {
  description = "PR number within the repo. Unique per repo only, so every identifier carries the repo too."
  type        = number
  validation {
    condition     = var.pr_number > 0 && var.pr_number < 10000
    error_message = "pr_number must be between 1 and 9999 so the ALB priority bands (10000+n, 20000+n) do not overlap."
  }
}

variable "api_image_tag" {
  description = "Tag in the ztmf/api-pr ECR repo. Every image there is built from the API Dockerfile's test target so it carries /app/_test_data_empire.sql; the deploy image in ztmf/api has no seed and the API fatals on DB_POPULATE. ztmf PRs pass their own pr-ztmf-<n>-<sha>; ui PRs pass ztmf_api_test_tag from SSM, the main-<sha> build published on ztmf main merges (ztmf-misc#342)."
  type        = string
  validation {
    condition     = can(regex("^(pr-ztmf-[0-9]+-|main-)[0-9a-f]{7,40}$", var.api_image_tag))
    error_message = "api_image_tag must be pr-ztmf-<n>-<sha> or main-<sha>; a bare SHA names the seedless deploy image in ztmf/api."
  }
}

variable "ui_image_tag" {
  description = "Tag in the ztmf/ui ECR repo. The PR's image (pr-ui-<n>-<sha>) for a ui PR, or ztmf_ui_tag from SSM for a ztmf PR."
  type        = string
}

variable "test_user_email" {
  description = "Seeded empire user the frontend's config.js token is minted for."
  type        = string
  default     = "Grand.Moff@DeathStar.Empire"
}

variable "postgres_image" {
  description = "Postgres sidecar image. ECR Public avoids Docker Hub rate limits; the task still needs internet egress to pull it."
  type        = string
  default     = "public.ecr.aws/docker/library/postgres:16-alpine"
}
