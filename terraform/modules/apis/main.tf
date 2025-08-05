# Google Cloud APIの有効化
resource "google_project_service" "apis" {
  for_each = toset([
    "run.googleapis.com",
    "cloudtasks.googleapis.com"
  ])
  project  = var.project_id
  service  = each.value

  disable_dependent_services = false
  disable_on_destroy         = false
}

output "enabled_apis" {
  description = "有効化されたAPI一覧"
  value       = [for api in google_project_service.apis : api.service]
}

variable "project_id" {
  description = "GCPプロジェクトID"
  type        = string
}
