data "daytona_sandbox_logs" "example" {
  sandbox_id = "sandbox-id"
  from       = "2026-01-02T03:04:05Z"
  to         = "2026-01-02T04:04:05Z"
  limit      = 50

  severities = ["ERROR", "WARN"]
}
