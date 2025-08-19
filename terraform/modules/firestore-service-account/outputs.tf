output "service_account_email" {
  description = "作成されたサービスアカウントのメールアドレス"
  value       = google_service_account.firestore_service_account.email
}

output "service_account_unique_id" {
  description = "作成されたサービスアカウントの一意ID"
  value       = google_service_account.firestore_service_account.unique_id
}

output "private_key_data" {
  description = "サービスアカウントの秘密鍵（Base64エンコード済み）"
  value       = google_service_account_key.firestore_key.private_key
  sensitive   = true
}

output "key_file_path" {
  description = "サービスアカウントキーファイルの保存先パス"
  value       = local_file.firestore_key_file.filename
}

output "project_id" {
  description = "GCPプロジェクトID"
  value       = var.project_id
}
