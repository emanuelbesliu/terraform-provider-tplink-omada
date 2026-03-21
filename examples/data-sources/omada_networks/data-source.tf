data "omada_networks" "example" {
  site_id = omada_site.example.id
}

output "networks" {
  value = data.omada_networks.example.networks
}
