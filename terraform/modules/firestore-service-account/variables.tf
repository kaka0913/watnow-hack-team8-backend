variable "project_id" {
  description = "GCPプロジェクトID"
  type        = string
}

variable "service_account_id" {
  description = "サービスアカウントID"
  type        = string
  default     = "firestore-access"
}

variable "display_name" {
  description = "サービスアカウントの表示名"
  type        = string
  default     = "Firestore Access Service Account"
}

variable "environment" {
  description = "環境名（dev, staging, prod等）"
  type        = string
  default     = "dev"
}

variable "enable_import_export" {
  description = "Firestore Import/Export権限を有効にするか"
  type        = bool
  default     = false
}

variable "enable_firebase_admin" {
  description = "Firebase Admin SDK権限を有効にするか"
  type        = bool
  default     = true
}

variable "key_file_path" {
  description = "サービスアカウントキーの保存先パス"
  type        = string
  default     = "./firestore-service-account-key.json"
}
