# Switches are physical devices adopted through the controller UI.
# Import an adopted switch into Terraform state:
#   terraform import omada_device_switch.example <siteID>/<mac>
# Destroying the resource removes it from Terraform state only.
resource "omada_device_switch" "example" {
  name = "Core-Switch"

  led_setting                = 2 # Site settings
  management_vlan_network_id = omada_network.management.id
  ip_setting_mode            = "dhcp"
  loopback_detect_enable     = true

  # Spanning Tree Protocol
  stp               = 2 # RSTP
  stp_priority      = 32768
  stp_hello_time    = 2
  stp_max_age       = 20
  stp_forward_delay = 15

  # Jumbo frames
  jumbo = 1518

  # Port configuration
  ports {
    port       = 1
    name       = "Uplink"
    profile_id = omada_port_profile.trunk.id
  }

  ports {
    port       = 2
    name       = "AP-Office"
    profile_id = omada_port_profile.ap_access.id
  }
}
