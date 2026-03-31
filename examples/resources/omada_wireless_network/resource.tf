resource "omada_wireless_network" "example" {
  site_id       = omada_site.example.id
  name          = "HomeWiFi"
  wlan_group_id = omada_wlan_group.default.id
  band          = 3 # Both 2.4GHz and 5GHz
  security      = 3 # WPA2/WPA3-Personal
  passphrase    = var.wifi_password
  broadcast     = true
  vlan_id       = 50
  enable_11r    = true # Fast BSS Transition (802.11r)
  pmf_mode      = 2    # Optional PMF
}

variable "wifi_password" {
  type      = string
  sensitive = true
}
