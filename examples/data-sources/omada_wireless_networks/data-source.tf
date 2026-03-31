data "omada_wireless_networks" "example" {
  site_id = omada_site.example.id
}

# Or filter by a specific WLAN group
data "omada_wireless_networks" "default_group" {
  site_id       = omada_site.example.id
  wlan_group_id = "696a40fd49039e1d13a9c412"
}

output "wireless_networks" {
  value = data.omada_wireless_networks.example.wireless_networks
}
