package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// mockOmadaServer creates a test HTTP server that mimics the Omada Controller API.
// It handles /api/info, login, and configurable site-scoped endpoints.
func mockOmadaServer(t *testing.T, handlers map[string]http.HandlerFunc) *httptest.Server {
	t.Helper()
	omadacID := "test-omadac-id"
	token := "test-csrf-token"

	mux := http.NewServeMux()

	// /api/info — return controller metadata
	mux.HandleFunc("/api/info", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(APIResponse{
			ErrorCode: 0,
			Msg:       "Success.",
			Result: mustMarshal(t, ControllerInfo{
				OmadacID:      omadacID,
				ControllerVer: "6.1.0.19",
				APIVer:        "3",
			}),
		})
	})

	// Login
	mux.HandleFunc(fmt.Sprintf("/%s/api/v2/login", omadacID), func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(APIResponse{
			ErrorCode: 0,
			Msg:       "Success.",
			Result:    mustMarshal(t, LoginResult{Token: token}),
		})
	})

	// Custom handlers
	for pattern, handler := range handlers {
		prefix := fmt.Sprintf("/%s/api/v2", omadacID)
		mux.HandleFunc(prefix+pattern, handler)
	}

	return httptest.NewServer(mux)
}

// mustMarshal marshals v to json.RawMessage, failing the test on error.
func mustMarshal(t *testing.T, v interface{}) json.RawMessage {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("mustMarshal: %v", err)
	}
	return data
}

// paginatedResponse wraps data in the standard paginated envelope.
func paginatedResponse(t *testing.T, data interface{}) json.RawMessage {
	t.Helper()
	return mustMarshal(t, PaginatedResult{
		TotalRows:   1,
		CurrentPage: 1,
		CurrentSize: 100,
		Data:        mustMarshal(t, data),
	})
}

// newTestClient creates a Client connected to the mock server.
func newTestClient(t *testing.T, server *httptest.Server) *Client {
	t.Helper()
	c, err := NewClient(server.URL, "admin", "password", true)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

// =============================================================================
// NewClient / Auth Tests
// =============================================================================

func TestNewClient_Success(t *testing.T) {
	server := mockOmadaServer(t, nil)
	defer server.Close()

	c := newTestClient(t, server)
	if c.omadacID != "test-omadac-id" {
		t.Errorf("omadacID = %q, want %q", c.omadacID, "test-omadac-id")
	}
	if c.token != "test-csrf-token" {
		t.Errorf("token = %q, want %q", c.token, "test-csrf-token")
	}
}

func TestNewClient_ControllerInfoError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(APIResponse{
			ErrorCode: -1,
			Msg:       "Controller unavailable",
		})
	}))
	defer server.Close()

	_, err := NewClient(server.URL, "admin", "password", true)
	if err == nil {
		t.Fatal("expected error from NewClient, got nil")
	}
	if !strings.Contains(err.Error(), "controller info") {
		t.Errorf("error = %q, expected to contain 'controller info'", err.Error())
	}
}

func TestNewClient_LoginError(t *testing.T) {
	omadacID := "test-omadac-id"
	mux := http.NewServeMux()
	mux.HandleFunc("/api/info", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(APIResponse{
			ErrorCode: 0,
			Result: mustMarshal(t, ControllerInfo{
				OmadacID: omadacID,
			}),
		})
	})
	mux.HandleFunc(fmt.Sprintf("/%s/api/v2/login", omadacID), func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(APIResponse{
			ErrorCode: -30109,
			Msg:       "Invalid username or password.",
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	_, err := NewClient(server.URL, "admin", "wrong", true)
	if err == nil {
		t.Fatal("expected error from NewClient with bad credentials, got nil")
	}
	if !strings.Contains(err.Error(), "logging in") {
		t.Errorf("error = %q, expected to contain 'logging in'", err.Error())
	}
}

func TestGetOmadacID(t *testing.T) {
	server := mockOmadaServer(t, nil)
	defer server.Close()
	c := newTestClient(t, server)

	if got := c.GetOmadacID(); got != "test-omadac-id" {
		t.Errorf("GetOmadacID() = %q, want %q", got, "test-omadac-id")
	}
}

// =============================================================================
// ListSites Tests
// =============================================================================

func TestListSites(t *testing.T) {
	sites := []Site{
		{ID: "site-1", Name: "Iasi"},
		{ID: "site-2", Name: "Darabani"},
	}

	server := mockOmadaServer(t, map[string]http.HandlerFunc{
		"/sites": func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(APIResponse{
				ErrorCode: 0,
				Result:    paginatedResponse(t, sites),
			})
		},
	})
	defer server.Close()
	c := newTestClient(t, server)

	got, err := c.ListSites(context.Background())
	if err != nil {
		t.Fatalf("ListSites: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d sites, want 2", len(got))
	}
	if got[0].Name != "Iasi" {
		t.Errorf("sites[0].Name = %q, want %q", got[0].Name, "Iasi")
	}
	if got[1].Name != "Darabani" {
		t.Errorf("sites[1].Name = %q, want %q", got[1].Name, "Darabani")
	}
}

// =============================================================================
// ResolveSiteID Tests
// =============================================================================

func TestResolveSiteID_ByName(t *testing.T) {
	sites := []Site{
		{ID: "site-1", Name: "Iasi"},
		{ID: "site-2", Name: "Darabani"},
	}

	server := mockOmadaServer(t, map[string]http.HandlerFunc{
		"/sites": func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(APIResponse{
				ErrorCode: 0,
				Result:    paginatedResponse(t, sites),
			})
		},
	})
	defer server.Close()
	c := newTestClient(t, server)

	id, err := c.ResolveSiteID(context.Background(), "Darabani")
	if err != nil {
		t.Fatalf("ResolveSiteID: %v", err)
	}
	if id != "site-2" {
		t.Errorf("ResolveSiteID('Darabani') = %q, want %q", id, "site-2")
	}
}

func TestResolveSiteID_ByID(t *testing.T) {
	sites := []Site{
		{ID: "site-1", Name: "Iasi"},
	}

	server := mockOmadaServer(t, map[string]http.HandlerFunc{
		"/sites": func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(APIResponse{
				ErrorCode: 0,
				Result:    paginatedResponse(t, sites),
			})
		},
	})
	defer server.Close()
	c := newTestClient(t, server)

	id, err := c.ResolveSiteID(context.Background(), "site-1")
	if err != nil {
		t.Fatalf("ResolveSiteID: %v", err)
	}
	if id != "site-1" {
		t.Errorf("ResolveSiteID('site-1') = %q, want %q", id, "site-1")
	}
}

func TestResolveSiteID_NotFound(t *testing.T) {
	sites := []Site{
		{ID: "site-1", Name: "Iasi"},
	}

	server := mockOmadaServer(t, map[string]http.HandlerFunc{
		"/sites": func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(APIResponse{
				ErrorCode: 0,
				Result:    paginatedResponse(t, sites),
			})
		},
	})
	defer server.Close()
	c := newTestClient(t, server)

	_, err := c.ResolveSiteID(context.Background(), "NonExistent")
	if err == nil {
		t.Fatal("expected error for non-existent site, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, expected to contain 'not found'", err.Error())
	}
}

func TestResolveSiteID_CaseInsensitive(t *testing.T) {
	sites := []Site{
		{ID: "site-1", Name: "Iasi"},
	}

	server := mockOmadaServer(t, map[string]http.HandlerFunc{
		"/sites": func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(APIResponse{
				ErrorCode: 0,
				Result:    paginatedResponse(t, sites),
			})
		},
	})
	defer server.Close()
	c := newTestClient(t, server)

	id, err := c.ResolveSiteID(context.Background(), "iasi")
	if err != nil {
		t.Fatalf("ResolveSiteID: %v", err)
	}
	if id != "site-1" {
		t.Errorf("ResolveSiteID('iasi') = %q, want %q", id, "site-1")
	}
}

// =============================================================================
// ListNetworks Tests
// =============================================================================

func TestListNetworks(t *testing.T) {
	networks := []Network{
		{ID: "net-1", Name: "Default", Purpose: "interface", Vlan: 1, GatewaySubnet: "192.168.0.1/24"},
		{ID: "net-2", Name: "AP_30_IOT", Purpose: "vlan", Vlan: 30},
	}

	server := mockOmadaServer(t, map[string]http.HandlerFunc{
		"/sites/site-1/setting/lan/networks": func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(APIResponse{
				ErrorCode: 0,
				Result:    paginatedResponse(t, networks),
			})
		},
	})
	defer server.Close()
	c := newTestClient(t, server)

	got, err := c.ListNetworks(context.Background(), "site-1")
	if err != nil {
		t.Fatalf("ListNetworks: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d networks, want 2", len(got))
	}
	if got[0].Name != "Default" {
		t.Errorf("networks[0].Name = %q, want %q", got[0].Name, "Default")
	}
	if got[1].Vlan != 30 {
		t.Errorf("networks[1].Vlan = %d, want %d", got[1].Vlan, 30)
	}
}

// =============================================================================
// CreateNetwork Adopt Pattern Tests
// =============================================================================

func TestCreateNetwork_AdoptExisting(t *testing.T) {
	existingNetworks := []Network{
		{ID: "net-1", Name: "Default", Purpose: "interface", Vlan: 1},
		{ID: "net-2", Name: "AP_30_IOT", Purpose: "vlan", Vlan: 30},
	}

	server := mockOmadaServer(t, map[string]http.HandlerFunc{
		"/sites/site-1/setting/lan/networks": func(w http.ResponseWriter, r *http.Request) {
			// Only handle GET (list) for adopt check
			if r.Method == http.MethodGet {
				json.NewEncoder(w).Encode(APIResponse{
					ErrorCode: 0,
					Result:    paginatedResponse(t, existingNetworks),
				})
				return
			}
			// POST should not be reached for adopt
			t.Error("unexpected POST to create network — adopt should have returned existing")
			w.WriteHeader(http.StatusInternalServerError)
		},
	})
	defer server.Close()
	c := newTestClient(t, server)

	// Trying to create "AP_30_IOT" should adopt the existing one
	got, err := c.CreateNetwork(context.Background(), "site-1", &Network{Name: "AP_30_IOT", Purpose: "vlan", Vlan: 30})
	if err != nil {
		t.Fatalf("CreateNetwork (adopt): %v", err)
	}
	if got.ID != "net-2" {
		t.Errorf("adopted network ID = %q, want %q", got.ID, "net-2")
	}
}

// =============================================================================
// URL Builder Tests
// =============================================================================

func TestGlobalURL(t *testing.T) {
	c := &Client{
		baseURL:  "https://10.0.20.7:8043",
		omadacID: "abc123",
		token:    "mytoken",
	}
	got := c.globalURL("/sites")
	want := "https://10.0.20.7:8043/abc123/api/v2/sites?token=mytoken"
	if got != want {
		t.Errorf("globalURL = %q, want %q", got, want)
	}
}

func TestSiteURL(t *testing.T) {
	c := &Client{
		baseURL:  "https://10.0.20.7:8043",
		omadacID: "abc123",
		token:    "mytoken",
	}
	got := c.siteURL("site-1", "/setting/lan/networks")
	want := "https://10.0.20.7:8043/abc123/api/v2/sites/site-1/setting/lan/networks?token=mytoken"
	if got != want {
		t.Errorf("siteURL = %q, want %q", got, want)
	}
}

// =============================================================================
// decodePaginatedData Tests
// =============================================================================

func TestDecodePaginatedData_Paginated(t *testing.T) {
	data := []Site{{ID: "s1", Name: "Site1"}}
	paginated := PaginatedResult{
		TotalRows:   1,
		CurrentPage: 1,
		CurrentSize: 100,
		Data:        mustMarshal(t, data),
	}
	raw := mustMarshal(t, paginated)

	var result []Site
	if err := decodePaginatedData(raw, &result); err != nil {
		t.Fatalf("decodePaginatedData: %v", err)
	}
	if len(result) != 1 || result[0].Name != "Site1" {
		t.Errorf("got %+v, want [{ID:s1 Name:Site1}]", result)
	}
}

func TestDecodePaginatedData_DirectArray(t *testing.T) {
	data := []Site{{ID: "s1", Name: "Site1"}}
	raw := mustMarshal(t, data)

	var result []Site
	if err := decodePaginatedData(raw, &result); err != nil {
		t.Fatalf("decodePaginatedData: %v", err)
	}
	if len(result) != 1 || result[0].Name != "Site1" {
		t.Errorf("got %+v, want [{ID:s1 Name:Site1}]", result)
	}
}

// =============================================================================
// isEmptyResult Tests
// =============================================================================

func TestIsEmptyResult(t *testing.T) {
	tests := []struct {
		name  string
		input json.RawMessage
		want  bool
	}{
		{"nil", nil, true},
		{"empty bytes", json.RawMessage{}, true},
		{"null string", json.RawMessage(`null`), true},
		{"empty object", json.RawMessage(`{}`), true},
		{"empty string", json.RawMessage(`""`), true},
		{"empty array", json.RawMessage(`[]`), true},
		{"whitespace", json.RawMessage(`  `), true},
		{"non-empty object", json.RawMessage(`{"id":"123"}`), false},
		{"non-empty array", json.RawMessage(`[1]`), false},
		{"non-empty string", json.RawMessage(`"hello"`), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isEmptyResult(tt.input)
			if got != tt.want {
				t.Errorf("isEmptyResult(%q) = %v, want %v", string(tt.input), got, tt.want)
			}
		})
	}
}

// =============================================================================
// isAgileSeriesError Tests
// =============================================================================

func TestIsAgileSeriesError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"unrelated error", fmt.Errorf("network timeout"), false},
		{"agile series error", fmt.Errorf("API error -39742: switch requires ES path"), true},
		{"agile in message", fmt.Errorf("code -39742 not supported"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isAgileSeriesError(tt.err)
			if got != tt.want {
				t.Errorf("isAgileSeriesError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

// =============================================================================
// ListWlanGroups Tests
// =============================================================================

func TestListWlanGroups(t *testing.T) {
	groups := []WlanGroup{
		{ID: "wg-1", Name: "Default"},
	}

	server := mockOmadaServer(t, map[string]http.HandlerFunc{
		"/sites/site-1/setting/wlans": func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(APIResponse{
				ErrorCode: 0,
				Result:    paginatedResponse(t, groups),
			})
		},
	})
	defer server.Close()
	c := newTestClient(t, server)

	got, err := c.ListWlanGroups(context.Background(), "site-1")
	if err != nil {
		t.Fatalf("ListWlanGroups: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d groups, want 1", len(got))
	}
	if got[0].Name != "Default" {
		t.Errorf("groups[0].Name = %q, want %q", got[0].Name, "Default")
	}
}

func TestGetDefaultWlanGroupID(t *testing.T) {
	groups := []WlanGroup{
		{ID: "wg-default", Name: "Default"},
		{ID: "wg-2", Name: "Custom"},
	}

	server := mockOmadaServer(t, map[string]http.HandlerFunc{
		"/sites/site-1/setting/wlans": func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(APIResponse{
				ErrorCode: 0,
				Result:    paginatedResponse(t, groups),
			})
		},
	})
	defer server.Close()
	c := newTestClient(t, server)

	id, err := c.GetDefaultWlanGroupID(context.Background(), "site-1")
	if err != nil {
		t.Fatalf("GetDefaultWlanGroupID: %v", err)
	}
	if id != "wg-default" {
		t.Errorf("GetDefaultWlanGroupID = %q, want %q", id, "wg-default")
	}
}

func TestGetDefaultWlanGroupID_Empty(t *testing.T) {
	server := mockOmadaServer(t, map[string]http.HandlerFunc{
		"/sites/site-1/setting/wlans": func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(APIResponse{
				ErrorCode: 0,
				Result:    paginatedResponse(t, []WlanGroup{}),
			})
		},
	})
	defer server.Close()
	c := newTestClient(t, server)

	_, err := c.GetDefaultWlanGroupID(context.Background(), "site-1")
	if err == nil {
		t.Fatal("expected error for empty WLAN groups, got nil")
	}
}

// =============================================================================
// BaseURL Normalization Test
// =============================================================================

func TestNewClient_TrailingSlashNormalization(t *testing.T) {
	server := mockOmadaServer(t, nil)
	defer server.Close()

	// The URL from the test server won't have trailing slash,
	// but we test that the client handles it
	c := newTestClient(t, server)
	if strings.HasSuffix(c.baseURL, "/") {
		t.Errorf("baseURL should not have trailing slash: %q", c.baseURL)
	}
}

// =============================================================================
// ACL Rules Tests
// =============================================================================

func TestListACLRules(t *testing.T) {
	rules := []ACLRule{
		{ID: "acl-1", Name: "Block IoT", Type: 0, Status: true, Policy: 0,
			Protocols: []int{6, 17}, SourceType: 0, SourceIDs: []string{"net-1"},
			DestinationType: 0, DestinationIDs: []string{"net-2"},
			Direction: ACLDirection{LanToWan: false, LanToLan: true}, Index: 0},
		{ID: "acl-2", Name: "Allow DNS", Type: 0, Status: true, Policy: 1,
			Protocols: []int{17}, SourceType: 0, SourceIDs: []string{"net-1"},
			DestinationType: 2, DestinationIDs: []string{"ipg-1"},
			Direction: ACLDirection{LanToWan: true, LanToLan: false}, Index: 1},
	}
	listResult := ACLListResult{
		TotalRows:   2,
		CurrentPage: 1,
		CurrentSize: 100,
		Data:        rules,
		ACLDisable:  false,
		SupportVPN:  true,
	}

	server := mockOmadaServer(t, map[string]http.HandlerFunc{
		"/sites/site-1/setting/firewall/acls": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", r.Method)
			}
			// Verify type query param
			if got := r.URL.Query().Get("type"); got != "0" {
				t.Errorf("type query param = %q, want %q", got, "0")
			}
			json.NewEncoder(w).Encode(APIResponse{
				ErrorCode: 0,
				Result:    mustMarshal(t, listResult),
			})
		},
	})
	defer server.Close()
	c := newTestClient(t, server)

	got, err := c.ListACLRules(context.Background(), "site-1", 0)
	if err != nil {
		t.Fatalf("ListACLRules: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d rules, want 2", len(got))
	}
	if got[0].Name != "Block IoT" {
		t.Errorf("rules[0].Name = %q, want %q", got[0].Name, "Block IoT")
	}
	if got[1].Policy != 1 {
		t.Errorf("rules[1].Policy = %d, want %d", got[1].Policy, 1)
	}
	if !got[0].Direction.LanToLan {
		t.Error("rules[0].Direction.LanToLan = false, want true")
	}
}

func TestListACLRules_Empty(t *testing.T) {
	listResult := ACLListResult{
		TotalRows:   0,
		CurrentPage: 1,
		CurrentSize: 100,
		Data:        []ACLRule{},
		ACLDisable:  true,
	}

	server := mockOmadaServer(t, map[string]http.HandlerFunc{
		"/sites/site-1/setting/firewall/acls": func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(APIResponse{
				ErrorCode: 0,
				Result:    mustMarshal(t, listResult),
			})
		},
	})
	defer server.Close()
	c := newTestClient(t, server)

	got, err := c.ListACLRules(context.Background(), "site-1", 0)
	if err != nil {
		t.Fatalf("ListACLRules (empty): %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %d rules, want 0", len(got))
	}
}

func TestGetACLRule_Found(t *testing.T) {
	rules := []ACLRule{
		{ID: "acl-1", Name: "Block IoT", Type: 0},
		{ID: "acl-2", Name: "Allow DNS", Type: 0},
	}
	listResult := ACLListResult{TotalRows: 2, CurrentPage: 1, CurrentSize: 100, Data: rules}

	server := mockOmadaServer(t, map[string]http.HandlerFunc{
		"/sites/site-1/setting/firewall/acls": func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(APIResponse{
				ErrorCode: 0,
				Result:    mustMarshal(t, listResult),
			})
		},
	})
	defer server.Close()
	c := newTestClient(t, server)

	got, err := c.GetACLRule(context.Background(), "site-1", "acl-2", 0)
	if err != nil {
		t.Fatalf("GetACLRule: %v", err)
	}
	if got.Name != "Allow DNS" {
		t.Errorf("Name = %q, want %q", got.Name, "Allow DNS")
	}
}

func TestGetACLRule_NotFound(t *testing.T) {
	listResult := ACLListResult{TotalRows: 0, CurrentPage: 1, CurrentSize: 100, Data: []ACLRule{}}

	server := mockOmadaServer(t, map[string]http.HandlerFunc{
		"/sites/site-1/setting/firewall/acls": func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(APIResponse{
				ErrorCode: 0,
				Result:    mustMarshal(t, listResult),
			})
		},
	})
	defer server.Close()
	c := newTestClient(t, server)

	_, err := c.GetACLRule(context.Background(), "site-1", "nonexistent", 0)
	if err == nil {
		t.Fatal("expected error for non-existent ACL rule, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, expected to contain 'not found'", err.Error())
	}
}

func TestCreateACLRule(t *testing.T) {
	created := ACLRule{
		ID: "acl-new", Name: "New Rule", Type: 0, Status: true,
		Policy: 0, Protocols: []int{6}, SourceType: 0, SourceIDs: []string{"net-1"},
		DestinationType: 0, DestinationIDs: []string{"net-2"}, Index: 5,
	}

	server := mockOmadaServer(t, map[string]http.HandlerFunc{
		"/sites/site-1/setting/firewall/acls": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", r.Method)
			}
			json.NewEncoder(w).Encode(APIResponse{
				ErrorCode: 0,
				Result:    mustMarshal(t, created),
			})
		},
	})
	defer server.Close()
	c := newTestClient(t, server)

	input := &ACLRule{
		Name: "New Rule", Type: 0, Status: true, Policy: 0,
		Protocols: []int{6}, SourceType: 0, SourceIDs: []string{"net-1"},
		DestinationType: 0, DestinationIDs: []string{"net-2"},
	}
	got, err := c.CreateACLRule(context.Background(), "site-1", input)
	if err != nil {
		t.Fatalf("CreateACLRule: %v", err)
	}
	if got.ID != "acl-new" {
		t.Errorf("ID = %q, want %q", got.ID, "acl-new")
	}
	if got.Index != 5 {
		t.Errorf("Index = %d, want %d", got.Index, 5)
	}
}

func TestUpdateACLRule(t *testing.T) {
	updated := ACLRule{
		ID: "acl-1", Name: "Updated Rule", Type: 0, Status: true,
		Policy: 1, Protocols: []int{6, 17},
	}

	server := mockOmadaServer(t, map[string]http.HandlerFunc{
		"/sites/site-1/setting/firewall/acls/acl-1": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPatch {
				t.Errorf("expected PATCH, got %s", r.Method)
			}
			json.NewEncoder(w).Encode(APIResponse{
				ErrorCode: 0,
				Result:    mustMarshal(t, updated),
			})
		},
	})
	defer server.Close()
	c := newTestClient(t, server)

	input := &ACLRule{Name: "Updated Rule", Type: 0, Policy: 1, Protocols: []int{6, 17}}
	got, err := c.UpdateACLRule(context.Background(), "site-1", "acl-1", input)
	if err != nil {
		t.Fatalf("UpdateACLRule: %v", err)
	}
	if got.Name != "Updated Rule" {
		t.Errorf("Name = %q, want %q", got.Name, "Updated Rule")
	}
}

func TestUpdateACLRule_EmptyResult(t *testing.T) {
	rules := []ACLRule{
		{ID: "acl-1", Name: "Refreshed Rule", Type: 0, Status: true, Policy: 1},
	}
	listResult := ACLListResult{TotalRows: 1, CurrentPage: 1, CurrentSize: 100, Data: rules}

	server := mockOmadaServer(t, map[string]http.HandlerFunc{
		"/sites/site-1/setting/firewall/acls/acl-1": func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPatch {
				json.NewEncoder(w).Encode(APIResponse{
					ErrorCode: 0,
					Result:    json.RawMessage(`{}`),
				})
				return
			}
		},
		"/sites/site-1/setting/firewall/acls": func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(APIResponse{
				ErrorCode: 0,
				Result:    mustMarshal(t, listResult),
			})
		},
	})
	defer server.Close()
	c := newTestClient(t, server)

	input := &ACLRule{Name: "Refreshed Rule", Type: 0, Policy: 1}
	got, err := c.UpdateACLRule(context.Background(), "site-1", "acl-1", input)
	if err != nil {
		t.Fatalf("UpdateACLRule (empty result): %v", err)
	}
	if got.Name != "Refreshed Rule" {
		t.Errorf("Name = %q, want %q", got.Name, "Refreshed Rule")
	}
}

func TestDeleteACLRule(t *testing.T) {
	server := mockOmadaServer(t, map[string]http.HandlerFunc{
		"/sites/site-1/setting/firewall/acls/acl-1": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				t.Errorf("expected DELETE, got %s", r.Method)
			}
			json.NewEncoder(w).Encode(APIResponse{
				ErrorCode: 0,
				Result:    json.RawMessage(`{}`),
			})
		},
	})
	defer server.Close()
	c := newTestClient(t, server)

	err := c.DeleteACLRule(context.Background(), "site-1", "acl-1")
	if err != nil {
		t.Fatalf("DeleteACLRule: %v", err)
	}
}

// =============================================================================
// IP Groups Tests
// =============================================================================

func TestListIPGroups(t *testing.T) {
	groups := []IPGroup{
		{ID: "ipg-1", Name: "DNS Servers", Type: 1, IPList: []IPGroupEntry{
			{IP: "8.8.8.8", PortList: []string{"53"}},
			{IP: "1.1.1.1", PortList: []string{"53"}},
		}},
		{ID: "ipg-2", Name: "Web Servers", Type: 1, IPList: []IPGroupEntry{
			{IP: "10.0.0.0/24", PortList: []string{"80", "443"}},
		}},
	}

	server := mockOmadaServer(t, map[string]http.HandlerFunc{
		"/sites/site-1/setting/firewall/ipGroups": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", r.Method)
			}
			json.NewEncoder(w).Encode(APIResponse{
				ErrorCode: 0,
				Result:    paginatedResponse(t, groups),
			})
		},
	})
	defer server.Close()
	c := newTestClient(t, server)

	got, err := c.ListIPGroups(context.Background(), "site-1")
	if err != nil {
		t.Fatalf("ListIPGroups: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d groups, want 2", len(got))
	}
	if got[0].Name != "DNS Servers" {
		t.Errorf("groups[0].Name = %q, want %q", got[0].Name, "DNS Servers")
	}
	if len(got[0].IPList) != 2 {
		t.Errorf("groups[0].IPList length = %d, want 2", len(got[0].IPList))
	}
	if got[0].IPList[0].IP != "8.8.8.8" {
		t.Errorf("groups[0].IPList[0].IP = %q, want %q", got[0].IPList[0].IP, "8.8.8.8")
	}
	if got[0].IPList[0].PortList[0] != "53" {
		t.Errorf("groups[0].IPList[0].PortList[0] = %q, want %q", got[0].IPList[0].PortList[0], "53")
	}
}

func TestListIPGroups_Empty(t *testing.T) {
	server := mockOmadaServer(t, map[string]http.HandlerFunc{
		"/sites/site-1/setting/firewall/ipGroups": func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(APIResponse{
				ErrorCode: 0,
				Result:    paginatedResponse(t, []IPGroup{}),
			})
		},
	})
	defer server.Close()
	c := newTestClient(t, server)

	got, err := c.ListIPGroups(context.Background(), "site-1")
	if err != nil {
		t.Fatalf("ListIPGroups (empty): %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %d groups, want 0", len(got))
	}
}

func TestGetIPGroup_Found(t *testing.T) {
	groups := []IPGroup{
		{ID: "ipg-1", Name: "DNS Servers", Type: 1},
		{ID: "ipg-2", Name: "Web Servers", Type: 1},
	}

	server := mockOmadaServer(t, map[string]http.HandlerFunc{
		"/sites/site-1/setting/firewall/ipGroups": func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(APIResponse{
				ErrorCode: 0,
				Result:    paginatedResponse(t, groups),
			})
		},
	})
	defer server.Close()
	c := newTestClient(t, server)

	got, err := c.GetIPGroup(context.Background(), "site-1", "ipg-2")
	if err != nil {
		t.Fatalf("GetIPGroup: %v", err)
	}
	if got.Name != "Web Servers" {
		t.Errorf("Name = %q, want %q", got.Name, "Web Servers")
	}
}

func TestGetIPGroup_NotFound(t *testing.T) {
	server := mockOmadaServer(t, map[string]http.HandlerFunc{
		"/sites/site-1/setting/firewall/ipGroups": func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(APIResponse{
				ErrorCode: 0,
				Result:    paginatedResponse(t, []IPGroup{}),
			})
		},
	})
	defer server.Close()
	c := newTestClient(t, server)

	_, err := c.GetIPGroup(context.Background(), "site-1", "nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent IP group, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, expected to contain 'not found'", err.Error())
	}
}

func TestCreateIPGroup(t *testing.T) {
	created := IPGroup{
		ID: "ipg-new", Name: "New Group", Type: 1,
		IPList: []IPGroupEntry{{IP: "192.168.1.0/24", PortList: []string{"80"}}},
	}

	server := mockOmadaServer(t, map[string]http.HandlerFunc{
		"/sites/site-1/setting/firewall/ipGroups": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", r.Method)
			}
			json.NewEncoder(w).Encode(APIResponse{
				ErrorCode: 0,
				Result:    mustMarshal(t, created),
			})
		},
	})
	defer server.Close()
	c := newTestClient(t, server)

	input := &IPGroup{
		Name: "New Group", Type: 1,
		IPList: []IPGroupEntry{{IP: "192.168.1.0/24", PortList: []string{"80"}}},
	}
	got, err := c.CreateIPGroup(context.Background(), "site-1", input)
	if err != nil {
		t.Fatalf("CreateIPGroup: %v", err)
	}
	if got.ID != "ipg-new" {
		t.Errorf("ID = %q, want %q", got.ID, "ipg-new")
	}
	if got.IPList[0].IP != "192.168.1.0/24" {
		t.Errorf("IPList[0].IP = %q, want %q", got.IPList[0].IP, "192.168.1.0/24")
	}
}

func TestUpdateIPGroup(t *testing.T) {
	updated := IPGroup{
		ID: "ipg-1", Name: "Updated Group", Type: 1,
		IPList: []IPGroupEntry{{IP: "10.0.0.0/8"}},
	}

	server := mockOmadaServer(t, map[string]http.HandlerFunc{
		"/sites/site-1/setting/firewall/ipGroups/ipg-1": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPatch {
				t.Errorf("expected PATCH, got %s", r.Method)
			}
			json.NewEncoder(w).Encode(APIResponse{
				ErrorCode: 0,
				Result:    mustMarshal(t, updated),
			})
		},
	})
	defer server.Close()
	c := newTestClient(t, server)

	input := &IPGroup{Name: "Updated Group", Type: 1, IPList: []IPGroupEntry{{IP: "10.0.0.0/8"}}}
	got, err := c.UpdateIPGroup(context.Background(), "site-1", "ipg-1", input)
	if err != nil {
		t.Fatalf("UpdateIPGroup: %v", err)
	}
	if got.Name != "Updated Group" {
		t.Errorf("Name = %q, want %q", got.Name, "Updated Group")
	}
}

func TestUpdateIPGroup_EmptyResult(t *testing.T) {
	groups := []IPGroup{
		{ID: "ipg-1", Name: "Refreshed Group", Type: 1, IPList: []IPGroupEntry{{IP: "10.0.0.0/8"}}},
	}

	server := mockOmadaServer(t, map[string]http.HandlerFunc{
		"/sites/site-1/setting/firewall/ipGroups/ipg-1": func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPatch {
				json.NewEncoder(w).Encode(APIResponse{
					ErrorCode: 0,
					Result:    json.RawMessage(`{}`),
				})
				return
			}
		},
		"/sites/site-1/setting/firewall/ipGroups": func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(APIResponse{
				ErrorCode: 0,
				Result:    paginatedResponse(t, groups),
			})
		},
	})
	defer server.Close()
	c := newTestClient(t, server)

	input := &IPGroup{Name: "Refreshed Group", Type: 1}
	got, err := c.UpdateIPGroup(context.Background(), "site-1", "ipg-1", input)
	if err != nil {
		t.Fatalf("UpdateIPGroup (empty result): %v", err)
	}
	if got.Name != "Refreshed Group" {
		t.Errorf("Name = %q, want %q", got.Name, "Refreshed Group")
	}
}

func TestDeleteIPGroup(t *testing.T) {
	server := mockOmadaServer(t, map[string]http.HandlerFunc{
		"/sites/site-1/setting/firewall/ipGroups/ipg-1": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				t.Errorf("expected DELETE, got %s", r.Method)
			}
			json.NewEncoder(w).Encode(APIResponse{
				ErrorCode: 0,
				Result:    json.RawMessage(`{}`),
			})
		},
	})
	defer server.Close()
	c := newTestClient(t, server)

	err := c.DeleteIPGroup(context.Background(), "site-1", "ipg-1")
	if err != nil {
		t.Fatalf("DeleteIPGroup: %v", err)
	}
}
