action "daytona_suspend_organization" "example" {
  config {
    organization_id                       = "organization-id"
    reason                                = "billing"
    until                                 = "2026-12-31T23:59:59Z"
    suspension_cleanup_grace_period_hours = 24
  }
}
