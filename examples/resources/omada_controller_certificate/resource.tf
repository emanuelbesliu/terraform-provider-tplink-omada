# Example: Upload and activate a new TLS certificate on the Omada Controller

# Read certificate and key from files (e.g., from Let's Encrypt or self-signed)
resource "omada_controller_certificate" "main" {
  certificate_pem = file("${path.module}/certs/controller.crt")
  private_key_pem = file("${path.module}/certs/controller.key")
}

# Example: Using a certificate from another Terraform module or data source
resource "omada_controller_certificate" "from_acme" {
  certificate_pem = acme_certificate.controller.certificate_pem
  private_key_pem = acme_certificate.controller.private_key_pem
}

# Example: Using output from acme_certificate with the full chain
resource "omada_controller_certificate" "with_chain" {
  certificate_pem = "${acme_certificate.controller.certificate_pem}${acme_certificate.controller.issuer_pem}"
  private_key_pem = acme_certificate.controller.private_key_pem
}

# Output the certificate details after upload
output "controller_cert_id" {
  value       = omada_controller_certificate.main.cert_id
  description = "The certificate ID on the Omada Controller"
}

output "controller_key_id" {
  value       = omada_controller_certificate.main.key_id
  description = "The private key ID on the Omada Controller"
}
