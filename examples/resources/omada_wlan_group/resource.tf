resource "omada_wlan_group" "example" {
  site_id = omada_site.example.id
  name    = "Office-WLANs"
}
