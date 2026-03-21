data "omada_site_settings" "current" {
  site_id = omada_site.example.id
}

output "mesh_enabled" {
  value = data.omada_site_settings.current.mesh_enable
}

output "fast_roaming_enabled" {
  value = data.omada_site_settings.current.fast_roaming_enable
}
