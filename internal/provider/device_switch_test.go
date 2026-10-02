package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// This acceptance test uses a local mock controller, never a real device.
func TestAccDeviceSwitchOperationalSpeed(t *testing.T) {
	var speed atomic.Int64
	speed.Store(1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var result any
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/info":
			result = map[string]any{"omadacId": "controller"}
		case r.Method == http.MethodPost && r.URL.Path == "/controller/api/v2/login":
			result = map[string]any{"token": "test-token"}
		case r.Method == http.MethodGet && r.URL.Path == "/controller/api/v2/sites/site/switches/02-00-00-00-00-01":
			// A port configured for auto negotiation (linkSpeed=0) can report
			// different operational speeds without any configuration change.
			linkStatus := 1
			if speed.Load() == 0 {
				linkStatus = 0
			}
			result = map[string]any{
				"mac": "02-00-00-00-00-01", "name": "Test-Switch", "ledSetting": 2,
				"ipSetting": map[string]any{"mode": "dhcp"},
				"ports": []map[string]any{{
					"port": 7, "name": "Test-Port", "speed": speed.Load(), "linkSpeed": 0,
					"portStatus": map[string]any{"linkStatus": linkStatus, "linkSpeed": speed.Load()},
				}},
			}
		default:
			// In particular, no configuration writes should be attempted.
			t.Errorf("unexpected controller request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if err := json.NewEncoder(w).Encode(map[string]any{"errorCode": 0, "result": result}); err != nil {
			t.Errorf("encoding controller response: %v", err)
		}
	}))
	defer server.Close()

	config := fmt.Sprintf(`
provider "omada" {
  url      = %q
  username = "test"
  password = "test"
}

resource "omada_device_switch" "test" {
  name = "Test-Switch"
  ports = [{ port = 7, name = "Test-Port" }]
}
`, server.URL)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config + `
import {
  to = omada_device_switch.test
  id = "site/02-00-00-00-00-01"
}
`,
				Check: resource.TestCheckResourceAttr("omada_device_switch.test", "ports.0.speed", "1"),
			},
			{
				PreConfig: func() { speed.Store(3) },
				Config:    config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.TestCheckResourceAttr("omada_device_switch.test", "ports.0.speed", "3"),
			},
			{
				PreConfig: func() { speed.Store(0) },
				Config:    config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.TestCheckResourceAttr("omada_device_switch.test", "ports.0.speed", "0"),
			},
		},
	})
}

func TestAccDeviceSwitchRejectConfiguredSpeed(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: `
resource "omada_device_switch" "test" {
  ports = [{ port = 7, speed = 3 }]
}
`,
			PlanOnly:    true,
			ExpectError: regexp.MustCompile(`Invalid Configuration for Read-Only Attribute`),
		}},
	})
}
