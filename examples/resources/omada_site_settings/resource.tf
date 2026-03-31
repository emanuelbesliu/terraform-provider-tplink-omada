# Site settings is a singleton resource -- each site has exactly one.
# Creating this resource adopts the existing settings.
# Destroying it removes the resource from Terraform state without
# changing settings on the controller.
resource "omada_site_settings" "example" {
  site_id = omada_site.example.id

  led_enable          = true
  mesh_enable         = true
  fast_roaming_enable = true
  lldp_enable         = true

  band_steering_enable               = true
  band_steering_connection_threshold = 30
  band_steering_difference_threshold = 4
  band_steering_max_failures         = 5

  device_account_username = "admin"
  device_account_password = "device-password"
}
