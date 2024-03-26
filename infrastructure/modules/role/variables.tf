variable "name" {
  type = string
}

variable "principal" {
  type = map(any)
}

variable "managed_policy_arns" {
  type    = list(string)
  default = null
}
