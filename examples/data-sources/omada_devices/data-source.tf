data "omada_devices" "example" {
  site_id = omada_site.example.id
}

output "devices" {
  value = data.omada_devices.example.devices
}
