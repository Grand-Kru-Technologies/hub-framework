terraform {
  required_version = ">= 1.10.0"

  required_providers {
    railway = {
      source  = "terraform-community-providers/railway"
      version = "= 0.4.0"
    }
  }

  backend "s3" {
    bucket       = "qg-hub-staging-tofu-state"
    key          = "hub-framework/staging/terraform.tfstate"
    region       = "us-east-1"
    encrypt      = true
    use_lockfile = true
  }

  encryption {
    key_provider "pbkdf2" "remote" {
      passphrase = var.state_encryption_passphrase
    }

    method "aes_gcm" "remote" {
      keys = key_provider.pbkdf2.remote
    }

    state {
      method   = method.aes_gcm.remote
      enforced = true
    }

    plan {
      method   = method.aes_gcm.remote
      enforced = true
    }
  }
}
