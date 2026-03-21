# APs are physical devices adopted through the controller UI.
# Import an adopted AP into Terraform state:
#   terraform import omada_device_ap.example <siteID>/<mac>
# Destroying the resource removes it from Terraform state only.
resource "omada_device_ap" "example" {
  name          = "Office-AP"
  wlan_group_id = omada_wlan_group.default.id

  # Radio settings
  radio_2g_enable         = true
  radio_2g_channel_width  = "20"
  radio_2g_channel        = "0" # Auto
  radio_2g_tx_power_level = 4

  radio_5g_enable         = true
  radio_5g_channel_width  = "80"
  radio_5g_channel        = "0" # Auto
  radio_5g_tx_power_level = 4

  # General settings
  led_setting     = 2 # Site settings
  ip_setting_mode = "dhcp"
  lldp_enable     = 1

  # Management VLAN
  management_vlan_enable     = true
  management_vlan_network_id = omada_network.management.id

  # OFDMA
  ofdma_enable_2g = true
  ofdma_enable_5g = true
}
