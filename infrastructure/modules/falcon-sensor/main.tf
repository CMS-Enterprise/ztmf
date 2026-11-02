// CrowdStrike Falcon container sensor for Fargate, done the way CrowdStrike's
// terraform-aws-ecs-fargate module does it, but for every container in a task
// instead of one. An init container copies the sensor into a shared volume,
// then each container waits for it, gets SYS_PTRACE, and starts through the
// sensor's entrypoint wrapper with its original entrypoint as the last
// argument and its original command unchanged.
//
// No providers or resources: this only rewrites JSON, so both the root and the
// pr-env root can call it whatever AWS provider version they pin.

terraform {
  required_version = ">= 1.10.0"
}

locals {
  init_container_name = "crowdstrike-falcon-init-container" // name reserved by the sensor
  volume_name         = "crowdstrike-falcon-volume"
  sensor_path         = "/tmp/CrowdStrike"

  // x86_64 loader. All ztmf tasks run on the Fargate default (X86_64).
  wrapper = [
    "${local.sensor_path}/rootfs/lib64/ld-linux-x86-64.so.2",
    "--library-path",
    "${local.sensor_path}/rootfs/lib64",
    "${local.sensor_path}/rootfs/bin/bash",
    "${local.sensor_path}/rootfs/entrypoint-ecs.sh",
  ]

  sensor_mount = {
    sourceVolume  = local.volume_name
    containerPath = local.sensor_path
    readOnly      = false
  }

  init_container = {
    name      = local.init_container_name
    image     = var.image
    user      = "0:0"
    essential = false
    memory    = var.init_memory
    entryPoint = [
      "/bin/bash",
      "-c",
      join(" && ", [
        "chmod u+rwx ${local.sensor_path}",
        "mkdir ${local.sensor_path}/rootfs",
        "cp -r /bin /etc /lib64 /usr /entrypoint-ecs.sh ${local.sensor_path}/rootfs",
        "chmod -R a=rX ${local.sensor_path}",
      ]),
    ]
    mountPoints = [local.sensor_mount]
  }

  containers = jsondecode(var.container_definitions)

  // A missing entry falls back to an empty one so the output precondition, not
  // an index error, reports it.
  entrypoint_for = {
    for c in local.containers : c.name => lookup(var.entrypoints, c.name, { entry_point = [], command = null })
  }

  wrapped = [
    for c in local.containers : merge(
      { for k, v in c : k => v if k != "command" },
      {
        entryPoint  = concat(local.wrapper, [join(" ", local.entrypoint_for[c.name].entry_point)])
        mountPoints = concat(try(c.mountPoints, []), [local.sensor_mount])
        dependsOn   = concat(try(c.dependsOn, []), [{ containerName = local.init_container_name, condition = "COMPLETE" }])
        secrets     = concat(try(c.secrets, []), [{ name = "FALCONCTL_OPTS", valueFrom = var.falconctl_opts_arn }])
        linuxParameters = merge(try(c.linuxParameters, {}), {
          capabilities = merge(try(c.linuxParameters.capabilities, {}), {
            add = distinct(concat(try(c.linuxParameters.capabilities.add, []), ["SYS_PTRACE"]))
          })
        })
      },
      local.entrypoint_for[c.name].command == null ? {} : { command = local.entrypoint_for[c.name].command },
    ) if var.enabled
  ]
}
