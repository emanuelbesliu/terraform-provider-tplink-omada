# Switches cannot be created or deleted via the API. This resource manages
# the configuration of an already-adopted switch. Destroying the resource
# removes it from Terraform state only.
resource "omada_device_switch" "example" {
  mac  = "10-27-F5-AA-BB-CC"
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
  ports = [
    {
      port       = 1
      name       = "Uplink"
      profile_id = omada_port_profile.trunk.id
    },
    {
      port       = 2
      name       = "AP-Office"
      profile_id = omada_port_profile.ap_access.id
    },
  ]
}
