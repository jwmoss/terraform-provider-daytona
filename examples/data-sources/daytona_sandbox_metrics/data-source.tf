data "daytona_sandbox_metrics" "example" {
  sandbox_id = "sandbox-id"
  from       = "2026-01-02T03:04:05Z"
  to         = "2026-01-02T04:04:05Z"

  metric_names = ["cpu.usage", "memory.usage"]
}
