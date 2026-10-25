terraform {
  required_version = ">= 1.6.0"

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
}
