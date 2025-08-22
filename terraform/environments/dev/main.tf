terraform {
  required_version = ">= 1.0"
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
  }
  
  backend "gcs" {
    bucket = "befree-terraform-state-bucket"
    prefix = "terraform/environments/dev"
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

module "app_server" {
  source       = "../../modules/cloud-run"
  service_name = "dev-app-server"
  location     = var.region
  project_id   = var.project_id
  image        = var.app_server_image
  service_account_email = "github-actions-deploy@befree-468615.iam.gserviceaccount.com"
  
  environment_variables = {
    ENVIRONMENT         = "development"
    SUPABASE_URL        = var.supabase_url
    SUPABASE_ANON_KEY   = var.supabase_anon_key
    SUPABASE_DB_PASSWORD = var.supabase_db_password
    POSTGRES_URL        = var.postgres_url
    GOOGLE_MAPS_API_KEY = var.google_maps_api_key
    GEMINI_API_KEY      = var.gemini_api_key
    FIRESTORE_PROJECT_ID = var.project_id
  }
  
  depends_on = [module.apis]
}

# Firestore Service Account（一時的にコメントアウト）
# module "firestore_service_account" {
#   source             = "../../modules/firestore-service-account"
#   project_id         = var.project_id
#   service_account_id = "befree-firestore-access"
#   display_name       = "Befree Firestore Access Service Account"
#   environment        = "development"
#   key_file_path      = "./keys/befree-firestore-key.json"
#   
#   # Firebase Admin SDK権限は一時的に無効化
#   enable_firebase_admin = false
#   
#   # バックアップ・復元機能が必要な場合は有効化
#   enable_import_export = false
# }
