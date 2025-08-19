terraform {
  required_version = ">= 1.0"
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}

module "apis" {
  source     = "../../modules/apis"
  project_id = var.project_id
}

# Cloud Runサービス（一時的にコメントアウト）
# module "app_server" {
#   source       = "../../modules/cloud-run"
#   service_name = "dev-app-server"
#   location     = var.region
#   project_id   = var.project_id
#   image        = var.app_server_image
#   
#   environment_variables = {
#     ENVIRONMENT         = "development"
#     DATABASE_URL        = var.database_url
#     GOOGLE_MAPS_API_KEY = var.google_maps_api_key
#     GEMINI_API_KEY      = var.gemini_api_key
#     FIRESTORE_PROJECT_ID = var.project_id
#     GOOGLE_APPLICATION_CREDENTIALS = "/app/keys/befree-firestore-key.json"
#   }
#   
#   depends_on = [module.apis, module.firestore_service_account]
# }

# module "story_server" {
#   source       = "../../modules/cloud-run"
#   service_name = "dev-story-server"
#   location     = var.region
#   project_id   = var.project_id
#   image        = var.story_server_image
#   
#   environment_variables = {
#     ENVIRONMENT    = "development"
#     DATABASE_URL   = var.database_url
#     GEMINI_API_KEY = var.gemini_api_key
#     FIRESTORE_PROJECT_ID = var.project_id
#     GOOGLE_APPLICATION_CREDENTIALS = "/app/keys/befree-firestore-key.json"
#   }
#   
#   depends_on = [module.apis, module.firestore_service_account]
# }

# module "story_queue" {
#   source     = "../../modules/cloud-tasks"
#   queue_name = "dev-story-queue"
#   location   = var.region
#   project_id = var.project_id
#   
#   depends_on = [module.apis]
# }

module "firestore_service_account" {
  source             = "../../modules/firestore-service-account"
  project_id         = var.project_id
  service_account_id = "befree-firestore-access"
  display_name       = "Befree Firestore Access Service Account"
  environment        = "development"
  key_file_path      = "./keys/befree-firestore-key.json"
  
  # Firebase Admin SDK権限は一時的に無効化
  enable_firebase_admin = false
  
  # バックアップ・復元機能が必要な場合は有効化
  enable_import_export = false
}
