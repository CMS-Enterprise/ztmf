// Run from infrastructure/modules/falcon-sensor: terraform init && terraform test
// Pure module, so no providers or credentials are involved.

variables {
  image              = "registry.example/falcon-container:1.0.0"
  falconctl_opts_arn = "arn:aws:ssm:us-east-1:000000000000:parameter/crowdstrike/falconctl-opts/test"
  container_definitions = jsonencode([
    {
      name      = "postgres"
      image     = "postgres:16-alpine"
      essential = true
      secrets   = [{ name = "POSTGRES_PASSWORD", valueFrom = "arn:pw" }]
    },
    {
      name      = "api"
      image     = "api:1"
      essential = true
      command   = ["/usr/local/bin/ztmfapi"]
      dependsOn = [{ containerName = "postgres", condition = "HEALTHY" }]
    },
    {
      name      = "ui"
      image     = "ui:1"
      essential = true
    },
  ])
  entrypoints = {
    postgres = { entry_point = ["/usr/local/bin/docker-entrypoint.sh"], command = ["postgres"] }
    api      = { entry_point = ["/usr/local/bin/ztmfapi"] }
    ui       = { entry_point = ["/docker-entrypoint.sh"], command = ["nginx", "-g", "daemon off;"] }
  }
}

run "disabled_passes_through_unchanged" {
  command = plan

  variables {
    enabled     = false
    entrypoints = {}
  }

  assert {
    condition     = output.container_definitions == var.container_definitions
    error_message = "Disabled must return the input JSON byte for byte."
  }

  assert {
    condition     = length(output.volume_names) == 0
    error_message = "Disabled must not add a volume."
  }
}

run "enabled_wraps_every_container" {
  command = plan

  variables {
    enabled = true
  }

  assert {
    condition     = [for c in jsondecode(output.container_definitions) : c.name] == ["crowdstrike-falcon-init-container", "postgres", "api", "ui"]
    error_message = "Init container must come first, followed by every original container in order."
  }

  assert {
    condition = alltrue([
      for c in slice(jsondecode(output.container_definitions), 1, 4) :
      c.entryPoint[4] == "/tmp/CrowdStrike/rootfs/entrypoint-ecs.sh"
      && contains(c.linuxParameters.capabilities.add, "SYS_PTRACE")
      && contains([for m in c.mountPoints : m.containerPath], "/tmp/CrowdStrike")
      && contains([for d in c.dependsOn : "${d.containerName}:${d.condition}"], "crowdstrike-falcon-init-container:COMPLETE")
      && contains([for s in c.secrets : "${s.name}=${s.valueFrom}"], "FALCONCTL_OPTS=${var.falconctl_opts_arn}")
      && !strcontains(jsonencode(try(c.environment, [])), "FALCONCTL_OPTS")
    ])
    error_message = "Every container needs the wrapper, SYS_PTRACE, the sensor mount, the init dependency and FALCONCTL_OPTS from secrets only."
  }

  assert {
    condition     = jsondecode(output.container_definitions)[2].entryPoint[5] == "/usr/local/bin/ztmfapi" && !can(jsondecode(output.container_definitions)[2].command)
    error_message = "A binary-only container runs the binary as the wrapped entrypoint with no command."
  }

  assert {
    condition     = jsondecode(output.container_definitions)[3].command == ["nginx", "-g", "daemon off;"]
    error_message = "The original command must be kept as its own array."
  }

  assert {
    condition     = length(jsondecode(output.container_definitions)[2].dependsOn) == 2 && length(jsondecode(output.container_definitions)[1].secrets) == 2
    error_message = "Existing dependsOn and secrets must be kept, not replaced."
  }

  assert {
    condition     = tolist(output.volume_names) == tolist(["crowdstrike-falcon-volume"])
    error_message = "Enabled must add the shared sensor volume."
  }
}

run "enabled_requires_every_entrypoint" {
  command = plan

  variables {
    enabled = true
    entrypoints = {
      postgres = { entry_point = ["/usr/local/bin/docker-entrypoint.sh"], command = ["postgres"] }
    }
  }

  expect_failures = [output.container_definitions]
}
