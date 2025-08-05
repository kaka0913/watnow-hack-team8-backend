resource "google_cloud_tasks_queue" "queue" {
  name     = var.queue_name
  location = var.location
  project  = var.project_id

  rate_limits {
    max_dispatches_per_second = 10
    max_concurrent_dispatches = 100
  }

  retry_config {
    max_attempts = 3
  }
}

output "queue_name" {
  description = "Cloud Tasks キュー名"
  value       = google_cloud_tasks_queue.queue.name
}

variable "queue_name" {
  description = "Cloud Tasks キュー名"
  type        = string
}

variable "location" {
  description = "Cloud Tasks キューのリージョン"
  type        = string
}

variable "project_id" {
  description = "GCPプロジェクトID"
  type        = string
}
