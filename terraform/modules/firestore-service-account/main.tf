# Firestoreアクセス用サービスアカウントモジュール

# サービスアカウント作成
resource "google_service_account" "firestore_service_account" {
  account_id   = var.service_account_id
  display_name = var.display_name
  description  = "Firestore Database access service account for ${var.environment}"
  project      = var.project_id
}

# Firestore関連の権限を付与
resource "google_project_iam_member" "firestore_user" {
  project = var.project_id
  role    = "roles/datastore.user"
  member  = "serviceAccount:${google_service_account.firestore_service_account.email}"
}

# Cloud Firestore Import Export Admin (バックアップ・復元用)
resource "google_project_iam_member" "firestore_import_export" {
  count   = var.enable_import_export ? 1 : 0
  project = var.project_id
  role    = "roles/datastore.importExportAdmin"
  member  = "serviceAccount:${google_service_account.firestore_service_account.email}"
}

# Firebase Admin SDK Admin Service Agent (Firebase Admin SDK使用時)
resource "google_project_iam_member" "firebase_admin" {
  count   = var.enable_firebase_admin ? 1 : 0
  project = var.project_id
  role    = "roles/firebase.adminsdk.serviceAgent"
  member  = "serviceAccount:${google_service_account.firestore_service_account.email}"
}

# サービスアカウントキー作成
resource "google_service_account_key" "firestore_key" {
  service_account_id = google_service_account.firestore_service_account.name
  public_key_type    = "TYPE_X509_PEM_FILE"
}

# ローカルファイルとしてJSONキーを保存
resource "local_file" "firestore_key_file" {
  content  = base64decode(google_service_account_key.firestore_key.private_key)
  filename = var.key_file_path
  
  # ファイル権限を600に設定（所有者のみ読み書き可能）
  file_permission = "0600"
}
