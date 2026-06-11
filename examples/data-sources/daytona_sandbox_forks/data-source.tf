data "daytona_sandbox_forks" "example" {
  sandbox_id_or_name = "parent-sandbox"
  include_destroyed  = false
}
