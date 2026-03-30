# Changelog

## [2.0.0](https://github.com/emanuelbesliu/terraform-provider-tplink-omada/compare/v1.0.0...v2.0.0) (2026-03-30)


### ⚠ BREAKING CHANGES

* correct provider registry address to emanuelbesliu/tplink-omada

### Features

* add CI workflow, release-please, dependabot, issue/PR templates, Makefile, and README ([ace2169](https://github.com/emanuelbesliu/terraform-provider-tplink-omada/commit/ace21697115699c8fb863f7c12b8ba011cd65e83))
* Terraform provider for TP-Link Omada Controller v6.x ([1b67f47](https://github.com/emanuelbesliu/terraform-provider-tplink-omada/commit/1b67f47db291d508e06469ff278a234eeb6d33f5))


### Bug Fixes

* chain GoReleaser workflow from release-please to build release binaries ([501b7d3](https://github.com/emanuelbesliu/terraform-provider-tplink-omada/commit/501b7d3770ddc1eb83b4f621939a1244f69ae4dc))
* correct provider registry address to emanuelbesliu/tplink-omada ([f482058](https://github.com/emanuelbesliu/terraform-provider-tplink-omada/commit/f482058b2dec101c854e4aa050d61e2ced68fa50))
* handle empty PATCH responses and deviceAccount password leak ([54b12f0](https://github.com/emanuelbesliu/terraform-provider-tplink-omada/commit/54b12f045a4940d4e52e0fbbf825a2ddac8f5b8f))
* handle VLAN-only network fields and add igmp_snoop_enable support ([ac513b1](https://github.com/emanuelbesliu/terraform-provider-tplink-omada/commit/ac513b1d68d0baa52d3fed86fa92d4424cf8c722))
* handle VLAN-only network fields and add igmp_snoop_enable support ([b9878fd](https://github.com/emanuelbesliu/terraform-provider-tplink-omada/commit/b9878fde047d2ed2c73c6b55d6215348673af426))
* pin GoReleaser to v2.14.x to avoid SHA256SUMS signature regression in v2.15.0 ([0256d3c](https://github.com/emanuelbesliu/terraform-provider-tplink-omada/commit/0256d3cee959750bb86d42496a35886b5fa80036))
* resolve API create errors for networks, SSIDs, port profiles, and WLAN groups ([7642ca0](https://github.com/emanuelbesliu/terraform-provider-tplink-omada/commit/7642ca07b532307c36b93f3171fafc589be389ac))

## [1.0.0](https://github.com/emanuelbesliu/terraform-provider-tplink-omada/compare/v0.1.3...v1.0.0) (2026-03-30)


### ⚠ BREAKING CHANGES

* correct provider registry address to emanuelbesliu/tplink-omada

### Features

* add CI workflow, release-please, dependabot, issue/PR templates, Makefile, and README ([ace2169](https://github.com/emanuelbesliu/terraform-provider-tplink-omada/commit/ace21697115699c8fb863f7c12b8ba011cd65e83))


### Bug Fixes

* correct provider registry address to emanuelbesliu/tplink-omada ([f482058](https://github.com/emanuelbesliu/terraform-provider-tplink-omada/commit/f482058b2dec101c854e4aa050d61e2ced68fa50))
