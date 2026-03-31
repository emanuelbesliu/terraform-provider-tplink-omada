package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// testAccProtoV6ProviderFactories creates the provider factory for acceptance tests.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"omada": providerserver.NewProtocol6WithError(New()),
}

// testAccPreCheck validates required environment variables before running acceptance tests.
func testAccPreCheck(t *testing.T) {
	t.Helper()
	for _, env := range []string{"OMADA_URL", "OMADA_USERNAME", "OMADA_PASSWORD"} {
		if v := os.Getenv(env); v == "" {
			t.Fatalf("environment variable %s must be set for acceptance tests", env)
		}
	}
}

// testSiteID returns the site ID to use for acceptance tests.
// Set via OMADA_TEST_SITE_ID env var.
func testSiteID(t *testing.T) string {
	t.Helper()
	id := os.Getenv("OMADA_TEST_SITE_ID")
	if id == "" {
		t.Fatal("OMADA_TEST_SITE_ID must be set for acceptance tests")
	}
	return id
}

// =============================================================================
// Data Source: omada_sites
// =============================================================================

func TestAccDataSourceSites(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "omada_sites" "test" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.omada_sites.test", "sites.#"),
				),
			},
		},
	})
}

// =============================================================================
// Data Source: omada_networks
// =============================================================================

func TestAccDataSourceNetworks(t *testing.T) {
	siteID := os.Getenv("OMADA_TEST_SITE_ID")
	if siteID == "" {
		siteID = "696a40fd49039e1d13a9c3f9" // fallback: Iasi
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
data "omada_networks" "test" {
  site_id = %q
}
`, siteID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.omada_networks.test", "networks.#"),
				),
			},
		},
	})
}

// =============================================================================
// Data Source: omada_wireless_networks
// =============================================================================

func TestAccDataSourceWirelessNetworks(t *testing.T) {
	siteID := os.Getenv("OMADA_TEST_SITE_ID")
	if siteID == "" {
		siteID = "696a40fd49039e1d13a9c3f9"
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
data "omada_wireless_networks" "test" {
  site_id = %q
}
`, siteID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.omada_wireless_networks.test", "wireless_networks.#"),
				),
			},
		},
	})
}

// =============================================================================
// Data Source: omada_port_profiles
// =============================================================================

func TestAccDataSourcePortProfiles(t *testing.T) {
	siteID := os.Getenv("OMADA_TEST_SITE_ID")
	if siteID == "" {
		siteID = "696a40fd49039e1d13a9c3f9"
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
data "omada_port_profiles" "test" {
  site_id = %q
}
`, siteID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.omada_port_profiles.test", "port_profiles.#"),
				),
			},
		},
	})
}

// =============================================================================
// Data Source: omada_devices
// =============================================================================

func TestAccDataSourceDevices(t *testing.T) {
	siteID := os.Getenv("OMADA_TEST_SITE_ID")
	if siteID == "" {
		siteID = "696a40fd49039e1d13a9c3f9"
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
data "omada_devices" "test" {
  site_id = %q
}
`, siteID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.omada_devices.test", "devices.#"),
				),
			},
		},
	})
}

// =============================================================================
// Data Source: omada_site_settings
// =============================================================================

func TestAccDataSourceSiteSettings(t *testing.T) {
	siteID := os.Getenv("OMADA_TEST_SITE_ID")
	if siteID == "" {
		siteID = "696a40fd49039e1d13a9c3f9"
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
data "omada_site_settings" "test" {
  site_id = %q
}
`, siteID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.omada_site_settings.test", "site_name"),
					resource.TestCheckResourceAttrSet("data.omada_site_settings.test", "timezone"),
				),
			},
		},
	})
}

// =============================================================================
// Resource: omada_network (CRUD lifecycle)
// Creates a temporary VLAN network, validates, updates, and auto-destroys.
// =============================================================================

func TestAccResourceNetwork_CRUD(t *testing.T) {
	siteID := os.Getenv("OMADA_TEST_SITE_ID")
	if siteID == "" {
		siteID = "696a40fd49039e1d13a9c3f9"
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create
			{
				Config: fmt.Sprintf(`
resource "omada_network" "test" {
  site_id = %q
  name    = "TF_ACC_TEST_NET"
  purpose = "vlan"
  vlan_id = 199
}
`, siteID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("omada_network.test", "id"),
					resource.TestCheckResourceAttr("omada_network.test", "name", "TF_ACC_TEST_NET"),
					resource.TestCheckResourceAttr("omada_network.test", "purpose", "vlan"),
					resource.TestCheckResourceAttr("omada_network.test", "vlan_id", "199"),
				),
			},
			// Step 2: Update name
			{
				Config: fmt.Sprintf(`
resource "omada_network" "test" {
  site_id = %q
  name    = "TF_ACC_TEST_NET_UPDATED"
  purpose = "vlan"
  vlan_id = 199
}
`, siteID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("omada_network.test", "name", "TF_ACC_TEST_NET_UPDATED"),
					resource.TestCheckResourceAttr("omada_network.test", "vlan_id", "199"),
				),
			},
			// Step 3: Import
			{
				ResourceName:      "omada_network.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources["omada_network.test"]
					if rs == nil {
						return "", fmt.Errorf("resource not found in state")
					}
					return fmt.Sprintf("%s/%s", rs.Primary.Attributes["site_id"], rs.Primary.ID), nil
				},
			},
		},
	})
}

// =============================================================================
// Resource: omada_port_profile (CRUD lifecycle)
// =============================================================================

func TestAccResourcePortProfile_CRUD(t *testing.T) {
	siteID := os.Getenv("OMADA_TEST_SITE_ID")
	if siteID == "" {
		siteID = "696a40fd49039e1d13a9c3f9"
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create network and port profile
			{
				Config: fmt.Sprintf(`
resource "omada_network" "test_pp" {
  site_id = %[1]q
  name    = "TF_ACC_PP_NET"
  purpose = "vlan"
  vlan_id = 198
}

resource "omada_port_profile" "test" {
  site_id           = %[1]q
  name              = "TF_ACC_TEST_PP"
  native_network_id = omada_network.test_pp.id
}
`, siteID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("omada_port_profile.test", "id"),
					resource.TestCheckResourceAttr("omada_port_profile.test", "name", "TF_ACC_TEST_PP"),
				),
			},
			// Step 2: Update
			{
				Config: fmt.Sprintf(`
resource "omada_network" "test_pp" {
  site_id = %[1]q
  name    = "TF_ACC_PP_NET"
  purpose = "vlan"
  vlan_id = 198
}

resource "omada_port_profile" "test" {
  site_id           = %[1]q
  name              = "TF_ACC_TEST_PP_UPDATED"
  native_network_id = omada_network.test_pp.id
}
`, siteID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("omada_port_profile.test", "name", "TF_ACC_TEST_PP_UPDATED"),
				),
			},
		},
	})
}
