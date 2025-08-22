output "app_server_url" {
  description = "アプリケーションサーバーのURL"
  # value       = module.app_server.service_url  # 一時的にコメントアウト
  value       = "Cloud Run app server not deployed yet"
}

output "firestore_service_account_email" {
  description = "Firestoreアクセス用サービスアカウントのメールアドレス"
  value       = module.firestore_service_account.service_account_email
}

output "firestore_key_file_path" {
  description = "Firestoreサービスアカウントキーファイルのパス"
  value       = module.firestore_service_account.key_file_path
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

# Database Configuration (一時的にコメントアウト、設定は残す)
# variable "database_url" {
#   description = "Supabaseデータベース接続URL"
#   type        = string
#   sensitive   = true
# }

# API Keys (一時的にコメントアウト、設定は残す)
# variable "google_maps_api_key" {
#   description = "Google Maps API キー"
#   type        = string
#   sensitive   = true
# }

# variable "gemini_api_key" {
#   description = "Gemini API キー"
#   type        = string
#   sensitive   = true
# }

# Container Images (一時的にコメントアウト、設定は残す)
# variable "app_server_image" {
#   description = "アプリケーションサーバーのコンテナイメージ"
#   type        = string
# }

# variable "story_server_image" {
#   description = "物語生成サーバーのコンテナイメージ"
#   type        = string
# }
