terraform {
  required_version = ">= 1.10.0"

  required_providers {
    github = {
      source  = "integrations/github"
      version = "= 6.3.1"
    }
  }

  backend "local" {
    path = "terraform.tfstate"
  }
}

provider "github" {
  owner = var.github_owner
  token = var.github_token
}
