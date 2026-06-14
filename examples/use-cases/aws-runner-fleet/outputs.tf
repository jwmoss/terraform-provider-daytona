output "runner_ids" {
  description = "Runner name to Daytona runner ID."
  value       = { for name, runner in daytona_runner.this : name => runner.id }
}

output "fleet" {
  description = "Each runner mapped to its EC2 instance, for inventory and SSH access."
  value = {
    for name, instance in aws_instance.runner : name => {
      runner_id   = daytona_runner.this[name].id
      instance_id = instance.id
      private_ip  = instance.private_ip
      ami         = instance.ami
    }
  }
}
