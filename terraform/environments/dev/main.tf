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

module "app_server" {
  source       = "../../modules/cloud-run"
  service_name = "dev-app-server"
  location     = var.region
  project_id   = var.project_id
  image        = var.app_server_image
  
  environment_variables = {
    ENVIRONMENT         = "development"
    DATABASE_URL        = var.database_url
    GOOGLE_MAPS_API_KEY = var.google_maps_api_key
    GEMINI_API_KEY      = var.gemini_api_key
  }
  
  depends_on = [module.apis]
}

module "story_server" {
  source       = "../../modules/cloud-run"
  service_name = "dev-story-server"
  location     = var.region
  project_id   = var.project_id
  image        = var.story_server_image
  
  environment_variables = {
    ENVIRONMENT    = "development"
    DATABASE_URL   = var.database_url
    GEMINI_API_KEY = var.gemini_api_key
  }
  
  depends_on = [module.apis]
}

module "story_queue" {
  source     = "../../modules/cloud-tasks"
  queue_name = "dev-story-queue"
  location   = var.region
  project_id = var.project_id
  
  depends_on = [module.apis]
}
