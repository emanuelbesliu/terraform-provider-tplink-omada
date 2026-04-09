# Changelog

## [2.1.0](https://github.com/emanuelbesliu/terraform-provider-tplink-omada/compare/v2.0.2...v2.1.0) (2026-04-09)


### Features

* add omada_controller_certificate resource for TLS certificate management ([2f3f5c7](https://github.com/emanuelbesliu/terraform-provider-tplink-omada/commit/2f3f5c757df5805299b2c4b7f093f4535b775f16))
* **saml:** add SAML IdP and SAML Role resources ([4cfdf80](https://github.com/emanuelbesliu/terraform-provider-tplink-omada/commit/4cfdf80de30a075f36b2bf540e20da163565508d))

## [2.0.1](https://github.com/emanuelbesliu/terraform-provider-tplink-omada/compare/v2.0.0...v2.0.1) (2026-04-01)


### Bug Fixes

* make GPG passphrase optional in release workflows ([f11efba](https://github.com/emanuelbesliu/terraform-provider-tplink-omada/commit/f11efba299408f4ec584f11d2d61a61b0ffa1ba5))

## [2.0.0](https://github.com/emanuelbesliu/terraform-provider-tplink-omada/compare/v1.0.0...v2.0.0) (2026-03-31)


### ⚠ BREAKING CHANGES

* All import ID formats changed to include siteID prefix. The provider 'site' attribute has been removed.

### Features

* add firewall ACL and IP group resources, data sources, and tests ([3a4b9dc](https://github.com/emanuelbesliu/terraform-provider-tplink-omada/commit/3a4b9dc9b5cf62515aa99b5bcde91ef017d8cc95))
* add mDNS reflector resource, data source, and tests ([836b1f1](https://github.com/emanuelbesliu/terraform-provider-tplink-omada/commit/836b1f180afd9fab9f1a01b3e516c76fde6c54d3))
* multi-site support and virtual/physical resource semantics ([ef13a9d](https://github.com/emanuelbesliu/terraform-provider-tplink-omada/commit/ef13a9d899a58993a3ca93286aac45eb2b8bbf5f))

## [1.0.0](https://github.com/emanuelbesliu/terraform-provider-tplink-omada/compare/v0.1.3...v1.0.0) (2026-03-30)


### ⚠ BREAKING CHANGES

* correct provider registry address to emanuelbesliu/tplink-omada

### Features

* add CI workflow, release-please, dependabot, issue/PR templates, Makefile, and README ([ace2169](https://github.com/emanuelbesliu/terraform-provider-tplink-omada/commit/ace21697115699c8fb863f7c12b8ba011cd65e83))


### Bug Fixes

* correct provider registry address to emanuelbesliu/tplink-omada ([f482058](https://github.com/emanuelbesliu/terraform-provider-tplink-omada/commit/f482058b2dec101c854e4aa050d61e2ced68fa50))
