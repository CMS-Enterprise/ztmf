// Run from infrastructure/modules/falcon-sensor:
//   terraform init -backend=false && terraform test
// Pure module, so no providers or credentials are involved. The fixture
// mirrors the pr-env task: three containers that all carry secrets, api
// waiting on postgres HEALTHY and ui on api START.

variables {
  falconctl_opts_arn = "arn:aws:ssm:us-east-1:000000000000:parameter/crowdstrike/falconctl-opts/test"
  container_definitions = jsonencode([
    {
      name        = "postgres"
      image       = "postgres:16-alpine"
      essential   = true
      environment = [{ name = "POSTGRES_DB", value = "ztmf" }]
      secrets     = [{ name = "POSTGRES_PASSWORD", valueFrom = "arn:pw" }]
      healthCheck = { command = ["CMD-SHELL", "pg_isready"], interval = 5, retries = 10 }
      logConfiguration = {
        logDriver = "awslogs"
        options   = { "awslogs-group" = "/test", "awslogs-stream-prefix" = "postgres" }
      }
    },
    {
      name             = "api"
      image            = "api:1"
      essential        = true
      command          = ["/usr/local/bin/ztmfapi"]
      workingDirectory = "/api"
      dependsOn        = [{ containerName = "postgres", condition = "HEALTHY" }]
      secrets          = [{ name = "DB_PASS", valueFrom = "arn:pw" }, { name = "AUTH_HS256_SECRET", valueFrom = "arn:hs" }]
      linuxParameters = {
        capabilities = { add = ["SYS_PTRACE", "NET_ADMIN"], drop = ["MKNOD"] }
      }
    },
    {
      name         = "ui"
      image        = "ui:1"
      essential    = true
      dependsOn    = [{ containerName = "api", condition = "START" }]
      portMappings = [{ containerPort = 443, protocol = "tcp" }]
      secrets      = [{ name = "AUTH_HS256_SECRET", valueFrom = "arn:hs" }]
      mountPoints  = [{ sourceVolume = "cache", containerPath = "/var/cache/nginx", readOnly = false }]
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
      slice(c.entryPoint, 0, 5) == [
        "/tmp/CrowdStrike/rootfs/lib64/ld-linux-x86-64.so.2",
        "--library-path",
        "/tmp/CrowdStrike/rootfs/lib64",
        "/tmp/CrowdStrike/rootfs/bin/bash",
        "/tmp/CrowdStrike/rootfs/entrypoint-ecs.sh",
      ]
      && length(c.entryPoint) == 6
      && contains(c.linuxParameters.capabilities.add, "SYS_PTRACE")
      && contains([for m in c.mountPoints : m.containerPath], "/tmp/CrowdStrike")
      && contains([for d in c.dependsOn : "${d.containerName}:${d.condition}"], "crowdstrike-falcon-init-container:SUCCESS")
      && contains([for s in c.secrets : "${s.name}=${s.valueFrom}"], "FALCONCTL_OPTS=${var.falconctl_opts_arn}")
      && !strcontains(jsonencode(try(c.environment, [])), "FALCONCTL_OPTS")
    ])
    error_message = "Every container needs the wrapper, SYS_PTRACE, the sensor mount, the init dependency and FALCONCTL_OPTS from secrets only."
  }

  assert {
    condition     = jsondecode(output.container_definitions)[1].entryPoint[5] == "/usr/local/bin/docker-entrypoint.sh" && jsondecode(output.container_definitions)[1].command == ["postgres"]
    error_message = "postgres keeps its image entrypoint and command."
  }

  assert {
    condition     = jsondecode(output.container_definitions)[2].entryPoint[5] == "/usr/local/bin/ztmfapi" && !can(jsondecode(output.container_definitions)[2].command)
    error_message = "A binary-only container runs the binary as the wrapped entrypoint with no command."
  }

  assert {
    condition     = jsondecode(output.container_definitions)[3].entryPoint[5] == "/docker-entrypoint.sh" && jsondecode(output.container_definitions)[3].command == ["nginx", "-g", "daemon off;"]
    error_message = "ui keeps its image entrypoint and its command as its own array."
  }

  assert {
    condition     = tolist(output.volume_names) == tolist(["crowdstrike-falcon-volume"])
    error_message = "Enabled must add the shared sensor volume."
  }
}

run "init_container_contents" {
  command = plan

  variables {
    enabled = true
    image   = "registry.example/falcon-container:1.0.0"
    init_log_configuration = {
      logDriver = "awslogs"
      options   = { "awslogs-group" = "/test", "awslogs-stream-prefix" = "falcon-init" }
    }
  }

  assert {
    condition = (
      jsondecode(output.container_definitions)[0].image == "registry.example/falcon-container:1.0.0"
      && jsondecode(output.container_definitions)[0].essential == false
      && jsondecode(output.container_definitions)[0].user == "0:0"
      && jsondecode(output.container_definitions)[0].memory == 512
      && jsondecode(output.container_definitions)[0].mountPoints == [{ sourceVolume = "crowdstrike-falcon-volume", containerPath = "/tmp/CrowdStrike", readOnly = false }]
      && jsondecode(output.container_definitions)[0].entryPoint[0] == "/bin/bash"
      && strcontains(jsondecode(output.container_definitions)[0].entryPoint[2], "cp -r /bin /etc /lib64 /usr /entrypoint-ecs.sh /tmp/CrowdStrike/rootfs")
    )
    error_message = "Init container must be the sensor image, non-essential, root, 512 MiB, with the sensor volume mounted."
  }

  assert {
    condition     = jsondecode(output.container_definitions)[0].logConfiguration.options["awslogs-stream-prefix"] == "falcon-init"
    error_message = "Init container must carry the given log configuration."
  }
}

run "init_container_defaults" {
  command = plan

  variables {
    enabled = true
  }

  assert {
    condition     = endswith(jsondecode(output.container_definitions)[0].image, "falcon-container@sha256:685810bb17b9f6c50ebdb7c56ce1108207180b647c104644c7a17ff88ac40705") && !can(jsondecode(output.container_definitions)[0].logConfiguration)
    error_message = "Without overrides the init container uses the module's pinned image and no log configuration."
  }
}

run "untouched_keys_are_preserved" {
  command = plan

  variables {
    enabled = true
  }

  assert {
    condition = (
      jsondecode(output.container_definitions)[1].environment == [{ name = "POSTGRES_DB", value = "ztmf" }]
      && jsondecode(output.container_definitions)[1].healthCheck == { command = ["CMD-SHELL", "pg_isready"], interval = 5, retries = 10 }
      && jsondecode(output.container_definitions)[1].logConfiguration.options["awslogs-stream-prefix"] == "postgres"
      && jsondecode(output.container_definitions)[2].workingDirectory == "/api"
      && jsondecode(output.container_definitions)[3].portMappings == [{ containerPort = 443, protocol = "tcp" }]
      && jsondecode(output.container_definitions)[3].image == "ui:1"
    )
    error_message = "Keys the wrapper does not own must pass through unchanged."
  }
}

run "existing_lists_are_merged" {
  command = plan

  variables {
    enabled = true
  }

  assert {
    condition = (
      [for d in jsondecode(output.container_definitions)[2].dependsOn : "${d.containerName}:${d.condition}"] == ["postgres:HEALTHY", "crowdstrike-falcon-init-container:SUCCESS"]
      && [for d in jsondecode(output.container_definitions)[3].dependsOn : "${d.containerName}:${d.condition}"] == ["api:START", "crowdstrike-falcon-init-container:SUCCESS"]
      && [for s in jsondecode(output.container_definitions)[1].secrets : s.name] == ["POSTGRES_PASSWORD", "FALCONCTL_OPTS"]
      && [for s in jsondecode(output.container_definitions)[2].secrets : s.name] == ["DB_PASS", "AUTH_HS256_SECRET", "FALCONCTL_OPTS"]
      && [for s in jsondecode(output.container_definitions)[3].secrets : s.name] == ["AUTH_HS256_SECRET", "FALCONCTL_OPTS"]
      && [for m in jsondecode(output.container_definitions)[3].mountPoints : m.containerPath] == ["/var/cache/nginx", "/tmp/CrowdStrike"]
    )
    error_message = "Existing dependsOn, secrets and mountPoints must be kept, with the sensor's entries appended."
  }

  assert {
    condition = (
      jsondecode(output.container_definitions)[2].linuxParameters.capabilities.add == ["SYS_PTRACE", "NET_ADMIN"]
      && jsondecode(output.container_definitions)[2].linuxParameters.capabilities.drop == ["MKNOD"]
      && jsondecode(output.container_definitions)[1].linuxParameters.capabilities.add == ["SYS_PTRACE"]
    )
    error_message = "linuxParameters must keep existing capabilities and add SYS_PTRACE once."
  }
}

run "reserved_init_name_is_rejected" {
  command = plan

  variables {
    enabled               = true
    container_definitions = jsonencode([{ name = "crowdstrike-falcon-init-container", image = "x" }])
    entrypoints           = { crowdstrike-falcon-init-container = { entry_point = ["/x"] } }
  }

  expect_failures = [var.container_definitions]
}

run "multi_word_entry_point_is_rejected" {
  command = plan

  variables {
    enabled = true
    entrypoints = {
      postgres = { entry_point = ["/usr/local/bin/docker-entrypoint.sh", "postgres"] }
      api      = { entry_point = ["/usr/local/bin/ztmfapi"] }
      ui       = { entry_point = ["/docker-entrypoint.sh"], command = ["nginx", "-g", "daemon off;"] }
    }
  }

  expect_failures = [var.entrypoints]
}

// The name set matches in the next two runs, so the only precondition that
// can fail is the drift check.
run "command_drift_is_rejected" {
  command = plan

  variables {
    enabled = true
    container_definitions = jsonencode([
      { name = "api", image = "api:1", command = ["/usr/local/bin/ztmfapi", "--debug"] },
    ])
    entrypoints = { api = { entry_point = ["/usr/local/bin/ztmfapi"] } }
  }

  expect_failures = [output.container_definitions]
}

run "entrypoint_drift_is_rejected" {
  command = plan

  variables {
    enabled = true
    container_definitions = jsonencode([
      { name = "ui", image = "ui:1", entryPoint = ["/custom-entrypoint.sh"], command = ["nginx", "-g", "daemon off;"] },
    ])
    entrypoints = { ui = { entry_point = ["/docker-entrypoint.sh"], command = ["nginx", "-g", "daemon off;"] } }
  }

  expect_failures = [output.container_definitions]
}

run "matching_own_entrypoint_and_command_pass" {
  command = plan

  variables {
    enabled = true
    container_definitions = jsonencode([
      { name = "ui", image = "ui:1", entryPoint = ["/docker-entrypoint.sh"], command = ["nginx", "-g", "daemon off;"] },
      { name = "pg", image = "pg:1", command = ["postgres"] },
    ])
    entrypoints = {
      ui = { entry_point = ["/docker-entrypoint.sh"], command = ["nginx", "-g", "daemon off;"] }
      pg = { entry_point = ["/usr/local/bin/docker-entrypoint.sh"], command = ["postgres"] }
    }
  }

  assert {
    condition     = length(jsondecode(output.container_definitions)) == 3
    error_message = "Definitions that agree with the entrypoints map must pass."
  }
}

// Missing entry, drift check skips it, so only the name-set check can fail.
run "missing_entrypoint_is_rejected" {
  command = plan

  variables {
    enabled = true
    entrypoints = {
      postgres = { entry_point = ["/usr/local/bin/docker-entrypoint.sh"], command = ["postgres"] }
    }
  }

  expect_failures = [output.container_definitions]
}
