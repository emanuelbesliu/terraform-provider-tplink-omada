resource "omada_network" "example" {
  site_id        = omada_site.example.id
  name           = "IoT"
  purpose        = "vlan"
  vlan_id        = 30
  gateway_subnet = "192.168.30.1/24"
  dhcp_enabled   = true
  dhcp_start     = "192.168.30.100"
  dhcp_end       = "192.168.30.254"
}
