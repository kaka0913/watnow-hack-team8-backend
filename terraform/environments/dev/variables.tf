output "app_server_url" {
  description = "アプリケーションサーバーのURL"
  value       = module.app_server.service_url
}

variable "project_id" {
  description = "GCPプロジェクトID"
  type        = string
}

variable "region" {
  description = "リージョン"
  type        = string
  default     = "asia-northeast1"
}

# Database Configuration
variable "database_url" {
  description = "Supabaseデータベース接続URL"
  type        = string
  sensitive   = true
}

# API Keys
variable "google_maps_api_key" {
  description = "Google Maps API キー"
  type        = string
  sensitive   = true
}

variable "gemini_api_key" {
  description = "Gemini API キー"
  type        = string
  sensitive   = true
}

# Container Images
variable "app_server_image" {
  description = "アプリケーションサーバーのコンテナイメージ"
  type        = string
}

variable "story_server_image" {
  description = "物語生成サーバーのコンテナイメージ"
  type        = string
}
