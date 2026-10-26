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

  encryption {
    key_provider "pbkdf2" "local" {
      passphrase = var.local_state_encryption_passphrase
    }

    method "aes_gcm" "local" {
      keys = key_provider.pbkdf2.local
    }

    state {
      method   = method.aes_gcm.local
      enforced = true
    }

    plan {
      method   = method.aes_gcm.local
      enforced = true
    }
  }
}

provider "github" {
  owner = var.github_owner
  token = var.github_token
}
