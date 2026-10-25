terraform {
  required_version = ">= 1.10.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "= 5.73.0"
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

provider "aws" {
  region = var.aws_region
}
