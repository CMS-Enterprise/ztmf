variable "environment" {
  type = string
}

variable "domain_name_prefix" {
  type    = string
  default = ""
}

variable "ecs_service_task_count" {
  type    = number
  default = 1
}

variable "entra_enabled" {
  description = "Enable the Entra ID identity provider alongside Okta: adds the per-IdP ALB listener rules, flips /api/* off ALB OIDC to backend session validation, and injects the Entra + session env into the API task. Defaults to false so the secrets can be created and seeded (aws secretsmanager put-secret-value) before any auth wiring goes live. Flip to true only after ztmf_entra_oidc and ztmf_session_signing_key hold real values in the target account."
  type        = bool
  default     = false
}

variable "kion_rotate_schedule_enabled" {
  description = "Enable the daily EventBridge schedule for ztmf-kion-key-rotate. Kion NAT allowlist is in place (CMS-Enterprise/ztmf-misc#174) and real rotation was validated end to end on 2026-04-22, so this defaults to true. Set to false only for temporary maintenance windows when rotation must be paused."
  type        = bool
  default     = true
}
