resource "omada_port_profile" "example" {
  site_id           = omada_site.example.id
  name              = "IoT-Access"
  native_network_id = omada_network.iot.id
  tag_network_ids   = []

  poe                    = 2 # PoE enabled
  spanning_tree_enable   = true
  loopback_detect_enable = true
  lldp_med_enable        = true
}
