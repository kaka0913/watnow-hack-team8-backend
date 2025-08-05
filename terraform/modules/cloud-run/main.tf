resource "google_cloud_run_v2_service" "service" {
  name     = var.service_name
  location = var.location
  project  = var.project_id

  template {
    containers {
      image = var.image

      ports {
        container_port = 8080
      }

      resources {
        limits = {
          cpu    = "1000m"
          memory = "512Mi"
        }
      }

      dynamic "env" {
        for_each = var.environment_variables
        content {
          name  = env.key
          value = env.value
        }
      }
    }
  }

  traffic {
    percent = 100
    type    = "TRAFFIC_TARGET_ALLOCATION_TYPE_LATEST"
  }
}

# Cloud Run サービスへのパブリックアクセスを許可
resource "google_cloud_run_service_iam_member" "public_access" {
  location = google_cloud_run_v2_service.service.location
  project  = google_cloud_run_v2_service.service.project
  service  = google_cloud_run_v2_service.service.name
  role     = "roles/run.invoker"
  member   = "allUsers"
}

variable "service_name" {
  description = "Cloud Run サービス名"
  type        = string
}

variable "location" {
  description = "Cloud Run サービスのリージョン"
  type        = string
}

variable "project_id" {
  description = "GCPプロジェクトID"
  type        = string
}

variable "image" {
  description = "コンテナイメージ"
  type        = string
}

variable "environment_variables" {
  description = "環境変数"
  type        = map(string)
  default     = {}
}

output "service_url" {
  description = "Cloud Run サービスのURL"
  value       = google_cloud_run_v2_service.service.uri
}