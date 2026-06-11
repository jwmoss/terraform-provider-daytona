resource "daytona_volume" "workspace" {
  count = var.create_volume ? 1 : 0

  name = coalesce(var.volume_name, "${var.name}-workspace")
}

resource "daytona_sandbox" "this" {
  name                  = var.name
  snapshot              = var.snapshot
  desired_state         = var.desired_state
  target                = var.target
  user                  = var.user
  labels                = var.labels
  env                   = var.env
  cpu                   = var.cpu
  memory                = var.memory
  disk                  = var.disk
  gpu                   = var.gpu
  public                = var.public
  auto_stop_interval    = var.auto_stop_interval
  auto_archive_interval = var.auto_archive_interval
  auto_delete_interval  = var.auto_delete_interval
  network_block_all     = var.network_block_all
  network_allow_list    = var.network_allow_list
  linked_sandbox        = var.linked_sandbox
}
