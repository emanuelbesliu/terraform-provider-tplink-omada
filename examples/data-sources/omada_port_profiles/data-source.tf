data "omada_port_profiles" "example" {
  site_id = omada_site.example.id
}

output "port_profiles" {
  value = data.omada_port_profiles.example.port_profiles
}
