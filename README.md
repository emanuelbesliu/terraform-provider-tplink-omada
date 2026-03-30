# Terraform Provider for TP-Link Omada

A Terraform provider for managing [TP-Link Omada Software Controller](https://www.tp-link.com/us/omada-sdn/) 6.x resources as infrastructure-as-code.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.23 (to build the provider)
- TP-Link Omada Software Controller 6.x

## Resources

| Resource | Description |
|---|---|
| `omada_site` | Manages sites on the controller |
| `omada_network` | Manages LAN networks (VLANs) |
| `omada_wireless_network` | Manages wireless SSIDs |
| `omada_wlan_group` | Manages WLAN groups |
| `omada_port_profile` | Manages switch port profiles |
| `omada_site_settings` | Manages site-level settings |
| `omada_device_ap` | Manages access point device configurations |
| `omada_device_switch` | Manages managed switch device configurations |

## Data Sources

| Data Source | Description |
|---|---|
| `omada_sites` | Lists all sites |
| `omada_networks` | Lists networks for a site |
| `omada_wireless_networks` | Lists wireless SSIDs for a site |
| `omada_port_profiles` | Lists port profiles for a site |
| `omada_site_settings` | Reads site settings |
| `omada_devices` | Lists devices for a site |

## Installation

### From the Terraform Registry

```hcl
terraform {
  required_providers {
    omada = {
      source = "emanuelbesliu/tplink-omada"
    }
  }
}
```

### Building from Source

```sh
git clone https://github.com/emanuelbesliu/terraform-provider-tplink-omada.git
cd terraform-provider-tplink-omada
make build
```

## Provider Configuration

```hcl
provider "omada" {
  url             = "https://192.168.1.1:8043"
  username        = "admin"
  password        = var.omada_password
  site            = "Default"
  skip_tls_verify = true
}
```

All attributes can also be set via environment variables:

| Attribute | Environment Variable |
|---|---|
| `url` | `OMADA_URL` |
| `username` | `OMADA_USERNAME` |
| `password` | `OMADA_PASSWORD` |
| `site` | `OMADA_SITE` |
| `skip_tls_verify` | `OMADA_SKIP_TLS_VERIFY` |

## Usage Example

```hcl
resource "omada_site" "home" {
  name = "Home"
}

resource "omada_network" "iot" {
  name        = "IoT"
  purpose     = "VLAN"
  vlan_id     = 100
  igmp_snoop_enable = false
}

resource "omada_wireless_network" "iot_wifi" {
  name           = "IoT-WiFi"
  wlan_group_id  = omada_wlan_group.default.id
  band           = "2g5g"
  security_mode  = "wpa2"
  psk_passphrase = var.iot_passphrase
  vlan_enable    = true
  vlan_id        = 100
}
```

See the [`docs/`](docs/) directory and the [Terraform Registry documentation](https://registry.terraform.io/providers/emanuelbesliu/tplink-omada/latest/docs) for full resource and data source schemas.

## Development

### Prerequisites

- Go >= 1.23
- A running Omada Software Controller 6.x for testing

### Build and Test

```sh
make build      # Build the provider binary
make test       # Run unit tests
make fmt        # Format Go source files
make fmtcheck   # Check formatting without modifying files
make vet        # Run go vet
make dev        # Build and print dev_overrides path
make clean      # Remove built binary
```

### Local Development with dev_overrides

Add the following to `~/.terraformrc` to use a locally built binary:

```hcl
provider_installation {
  dev_overrides {
    "emanuelbesliu/tplink-omada" = "/path/to/terraform-provider-tplink-omada"
  }
  direct {}
}
```

Then build with `make dev` and run Terraform as usual. Note that `terraform init` will fail with dev_overrides enabled -- this is expected. Run `terraform plan` and `terraform apply` directly.

### Commit Conventions

This project uses [Conventional Commits](https://www.conventionalcommits.org/) for automated releases via [release-please](https://github.com/googleapis/release-please):

- `fix:` -- patch release (bug fix)
- `feat:` -- minor release (new feature)
- `feat!:` or `BREAKING CHANGE:` -- major release

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feat/my-feature`)
3. Commit using conventional commits (`git commit -m 'feat: add my feature'`)
4. Push to your branch (`git push origin feat/my-feature`)
5. Open a Pull Request

## License

See [LICENSE](LICENSE) for details.
