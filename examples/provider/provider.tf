terraform {
  required_providers {
    omada = {
      source = "emanuelbesliu/tplink-omada"
    }
  }
}

# Configuration can also be set via environment variables:
#   OMADA_URL, OMADA_USERNAME, OMADA_PASSWORD, OMADA_SITE
provider "omada" {
  url             = "https://192.168.1.1:8043"
  username        = "admin"
  password        = var.omada_password
  site            = "Default"
  skip_tls_verify = true
}

variable "omada_password" {
  type      = string
  sensitive = true
}
