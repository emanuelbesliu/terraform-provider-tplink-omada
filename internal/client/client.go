package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"sync"
	"time"
)

// Client is the Omada Controller API client.
type Client struct {
	baseURL    string
	username   string
	password   string
	omadacID   string
	token      string
	httpClient *http.Client
	mu         sync.Mutex
	readOnly   bool
}

// ErrReadOnly is returned when a write operation is attempted in read-only mode.
var ErrReadOnly = fmt.Errorf("operation blocked: provider is in read_only mode — only data sources and imports are allowed")

// APIResponse is the standard response envelope from the Omada API.
type APIResponse struct {
	ErrorCode int             `json:"errorCode"`
	Msg       string          `json:"msg"`
	Result    json.RawMessage `json:"result"`
}

// PaginatedResult wraps paginated list responses.
type PaginatedResult struct {
	TotalRows   int             `json:"totalRows"`
	CurrentPage int             `json:"currentPage"`
	CurrentSize int             `json:"currentSize"`
	Data        json.RawMessage `json:"data"`
}

// ControllerInfo holds the controller metadata returned by /api/info.
type ControllerInfo struct {
	OmadacID      string `json:"omadacId"`
	ControllerVer string `json:"controllerVer"`
	APIVer        string `json:"apiVer"`
	Type          int    `json:"type"`
}

// LoginResult holds the login response.
type LoginResult struct {
	Token string `json:"token"`
}

// Site represents an Omada site (full details from GET /api/v2/sites/{id}).
type Site struct {
	ID       string `json:"id"`
	Key      string `json:"key,omitempty"`
	Name     string `json:"name"`
	Type     int    `json:"type,omitempty"`
	Region   string `json:"region,omitempty"`
	TimeZone string `json:"timeZone,omitempty"`
	Scenario string `json:"scenario,omitempty"`
}

// SiteCreateRequest is the payload for POST /api/v2/sites.
type SiteCreateRequest struct {
	Name                 string              `json:"name"`
	Region               string              `json:"region"`
	TimeZone             string              `json:"timeZone"`
	Scenario             string              `json:"scenario"`
	Type                 int                 `json:"type"`
	DeviceAccountSetting *DeviceAccountInput `json:"deviceAccountSetting,omitempty"`
}

// DeviceAccountInput is the device account payload for site creation.
type DeviceAccountInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// SiteCreateResult is the response from POST /api/v2/sites.
type SiteCreateResult struct {
	SiteID string `json:"siteId"`
}

// SiteSettingUpdate is the payload for PATCH /sites/{id}/setting to update site-level fields.
type SiteSettingUpdate struct {
	Site *SiteSettingFields `json:"site"`
}

// SiteSettingFields holds the updatable site fields sent inside the "site" key.
type SiteSettingFields struct {
	Name     string `json:"name,omitempty"`
	Region   string `json:"region,omitempty"`
	TimeZone string `json:"timeZone,omitempty"`
	Scenario string `json:"scenario,omitempty"`
}

// DHCPSettings holds DHCP server configuration for a network.
type DHCPSettings struct {
	Enable      bool   `json:"enable"`
	IPAddrStart string `json:"ipaddrStart,omitempty"`
	IPAddrEnd   string `json:"ipaddrEnd,omitempty"`
	LeaseTime   int    `json:"leasetime,omitempty"`
}

// Network represents a LAN network / VLAN configuration.
type Network struct {
	ID              string        `json:"id,omitempty"`
	Name            string        `json:"name"`
	Purpose         string        `json:"purpose,omitempty"`
	Vlan            int           `json:"vlan"`
	GatewaySubnet   string        `json:"gatewaySubnet,omitempty"`
	DHCPSettings    *DHCPSettings `json:"dhcpSettings,omitempty"`
	Isolation       bool          `json:"isolation"`
	IGMPSnoopEnable bool          `json:"igmpSnoopEnable"`
}

// WlanGroup represents a wireless LAN group.
type WlanGroup struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Clone   bool   `json:"clone"`
	Primary bool   `json:"primary"`
	Site    string `json:"site,omitempty"`
}

// WlanGroupCreateRequest is the payload for POST /setting/wlans.
type WlanGroupCreateRequest struct {
	Name  string `json:"name"`
	Clone bool   `json:"clone"`
}

// WlanGroupCreateResult is the response from POST /setting/wlans.
type WlanGroupCreateResult struct {
	WlanID string `json:"wlanId"`
}

// WlanGroupUpdateRequest is the payload for PATCH /setting/wlans/{id}.
type WlanGroupUpdateRequest struct {
	Name string `json:"name"`
}

// WirelessNetwork represents an SSID/WLAN configuration.
type WirelessNetwork struct {
	ID                 string       `json:"id,omitempty"`
	Name               string       `json:"name"`
	WlanID             string       `json:"wlanId,omitempty"`
	Band               int          `json:"band"`
	GuestNetEnable     bool         `json:"guestNetEnable"`
	Security           int          `json:"security"`
	Broadcast          bool         `json:"broadcast"`
	PSKSetting         *PSKSetting  `json:"pskSetting,omitempty"`
	VlanSetting        *VlanSetting `json:"vlanSetting,omitempty"`
	Enable11r          bool         `json:"enable11r"`
	PmfMode            int          `json:"pmfMode"`
	WlanScheduleEnable bool         `json:"wlanScheduleEnable"`
	MacFilterEnable    bool         `json:"macFilterEnable"`
	RateLimit          *RateLimit   `json:"rateLimit"`

	// Additional fields required by the API for create
	RateAndBeaconCtrl *RateAndBeaconCtrl `json:"rateAndBeaconCtrl,omitempty"`
	MultiCastSetting  *MultiCastSetting  `json:"multiCastSetting,omitempty"`
	SSIDRateLimit     *RateLimit         `json:"ssidRateLimit,omitempty"`
	MloEnable         bool               `json:"mloEnable"`
	ProhibitWifiShare bool               `json:"prohibitWifiShare"`

	// Store raw JSON for PATCH operations (full object required)
	RawJSON map[string]interface{} `json:"-"`
}

// PSKSetting holds WPA pre-shared key settings.
type PSKSetting struct {
	VersionPsk        int    `json:"versionPsk"`
	EncryptionPsk     int    `json:"encryptionPsk"`
	GikRekeyPskEnable bool   `json:"gikRekeyPskEnable"`
	SecurityKey       string `json:"securityKey"`
}

// VlanSetting holds VLAN configuration for an SSID.
type VlanSetting struct {
	Mode           int           `json:"mode"`
	CustomConfig   *CustomConfig `json:"customConfig,omitempty"`
	CurrentVlanId  int           `json:"currentVlanId,omitempty"`
	CurrentVlanIds string        `json:"currentVlanIds,omitempty"`
}

// CustomConfig holds custom VLAN configuration for an SSID.
type CustomConfig struct {
	CustomMode        int              `json:"customMode"`
	LanNetworkID      string           `json:"lanNetworkId,omitempty"`
	LanNetworkVlanIds map[string][]int `json:"lanNetworkVlanIds,omitempty"`
	BridgeVlan        int              `json:"bridgeVlan,omitempty"`
}

// RateLimit holds rate limiting configuration.
type RateLimit struct {
	RateLimitID     string `json:"rateLimitId,omitempty"`
	DownLimitEnable bool   `json:"downLimitEnable"`
	UpLimitEnable   bool   `json:"upLimitEnable"`
}

// RateAndBeaconCtrl holds rate and beacon control settings for an SSID.
type RateAndBeaconCtrl struct {
	Rate2gCtrlEnable          bool `json:"rate2gCtrlEnable"`
	Rate5gCtrlEnable          bool `json:"rate5gCtrlEnable"`
	ManageRateControl2gEnable bool `json:"manageRateControl2gEnable"`
	ManageRateControl5gEnable bool `json:"manageRateControl5gEnable"`
	Rate6gCtrlEnable          bool `json:"rate6gCtrlEnable"`
}

// MultiCastSetting holds multicast configuration for an SSID.
type MultiCastSetting struct {
	MultiCastEnable bool `json:"multiCastEnable"`
	ChannelUtil     int  `json:"channelUtil"`
	ArpCastEnable   bool `json:"arpCastEnable"`
	Ipv6CastEnable  bool `json:"ipv6CastEnable"`
	FilterEnable    bool `json:"filterEnable"`
}

// PortProfile represents a switch port profile.
type PortProfile struct {
	ID                            string               `json:"id,omitempty"`
	Name                          string               `json:"name"`
	NativeNetworkID               string               `json:"nativeNetworkId,omitempty"`
	TagNetworkIDs                 []string             `json:"tagNetworkIds"`
	UntagNetworkIDs               []string             `json:"untagNetworkIds,omitempty"`
	POE                           int                  `json:"poe"`
	Dot1x                         int                  `json:"dot1x"`
	PortIsolationEnable           bool                 `json:"portIsolationEnable"`
	LLDPMedEnable                 bool                 `json:"lldpMedEnable"`
	TopoNotifyEnable              bool                 `json:"topoNotifyEnable"`
	SpanningTreeEnable            bool                 `json:"spanningTreeEnable"`
	LoopbackDetectEnable          bool                 `json:"loopbackDetectEnable"`
	Type                          int                  `json:"type,omitempty"`
	BandWidthCtrlType             int                  `json:"bandWidthCtrlType"`
	EeeEnable                     bool                 `json:"eeeEnable"`
	FlowControlEnable             bool                 `json:"flowControlEnable"`
	FastLeaveEnable               bool                 `json:"fastLeaveEnable"`
	LoopbackDetectVlanBasedEnable bool                 `json:"loopbackDetectVlanBasedEnable"`
	IgmpFastLeaveEnable           bool                 `json:"igmpFastLeaveEnable"`
	MldFastLeaveEnable            bool                 `json:"mldFastLeaveEnable"`
	Dot1pPriority                 int                  `json:"dot1pPriority"`
	TrustMode                     int                  `json:"trustMode"`
	SpanningTreeSetting           *SpanningTreeSetting `json:"spanningTreeSetting"`
	DhcpL2RelaySettings           *DhcpL2RelaySettings `json:"dhcpL2RelaySettings"`
}

// SpanningTreeSetting holds STP settings for a port profile.
type SpanningTreeSetting struct {
	Priority    int  `json:"priority"`
	ExtPathCost int  `json:"extPathCost"`
	IntPathCost int  `json:"intPathCost"`
	EdgePort    bool `json:"edgePort"`
	P2pLink     int  `json:"p2pLink"`
	Mcheck      bool `json:"mcheck"`
	LoopProtect bool `json:"loopProtect"`
	RootProtect bool `json:"rootProtect"`
	TcGuard     bool `json:"tcGuard"`
	BpduProtect bool `json:"bpduProtect"`
	BpduFilter  bool `json:"bpduFilter"`
	BpduForward bool `json:"bpduForward"`
}

// DhcpL2RelaySettings holds DHCP L2 relay settings for a port profile.
type DhcpL2RelaySettings struct {
	Enable bool `json:"enable"`
}

// NewClient creates a new Omada API client.
func NewClient(baseURL, username, password string, skipTLSVerify bool) (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("creating cookie jar: %w", err)
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: skipTLSVerify,
		},
	}

	httpClient := &http.Client{
		Jar:       jar,
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	// Normalize base URL
	baseURL = strings.TrimRight(baseURL, "/")

	c := &Client{
		baseURL:    baseURL,
		username:   username,
		password:   password,
		httpClient: httpClient,
	}

	// Step 1: Get controller ID
	if err := c.getControllerInfo(context.Background()); err != nil {
		return nil, fmt.Errorf("getting controller info: %w", err)
	}

	// Step 2: Login
	if err := c.login(context.Background()); err != nil {
		return nil, fmt.Errorf("logging in: %w", err)
	}

	return c, nil
}

// getControllerInfo fetches the controller ID from /api/info.
func (c *Client) getControllerInfo(ctx context.Context) error {
	url := fmt.Sprintf("%s/api/info", c.baseURL)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	if apiResp.ErrorCode != 0 {
		return fmt.Errorf("API error %d: %s", apiResp.ErrorCode, apiResp.Msg)
	}

	var info ControllerInfo
	if err := json.Unmarshal(apiResp.Result, &info); err != nil {
		return fmt.Errorf("decoding controller info: %w", err)
	}

	c.omadacID = info.OmadacID
	return nil
}

// login authenticates with the controller and stores the CSRF token.
func (c *Client) login(ctx context.Context) error {
	url := fmt.Sprintf("%s/%s/api/v2/login", c.baseURL, c.omadacID)

	body := map[string]string{
		"username": c.username,
		"password": c.password,
	}
	jsonBody, _ := json.Marshal(body)

	resp, err := c.httpClient.Post(url, "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("POST %s: %w", url, err)
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	if apiResp.ErrorCode != 0 {
		return fmt.Errorf("login failed (code %d): %s", apiResp.ErrorCode, apiResp.Msg)
	}

	var loginResult LoginResult
	if err := json.Unmarshal(apiResp.Result, &loginResult); err != nil {
		return fmt.Errorf("decoding login result: %w", err)
	}

	c.token = loginResult.Token
	return nil
}

// ensureAuth re-authenticates if the session has expired.
func (c *Client) ensureAuth(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token == "" {
		if err := c.login(ctx); err != nil {
			return err
		}
	}
	return nil
}

// reAuth forces re-authentication.
func (c *Client) reAuth(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.token = ""
	return c.login(ctx)
}

// globalURL builds a URL for non-site-scoped endpoints.
func (c *Client) globalURL(path string) string {
	return fmt.Sprintf("%s/%s/api/v2%s?token=%s", c.baseURL, c.omadacID, path, c.token)
}

// siteURL builds a URL for site-scoped endpoints.
func (c *Client) siteURL(siteID, path string) string {
	return fmt.Sprintf("%s/%s/api/v2/sites/%s%s?token=%s", c.baseURL, c.omadacID, siteID, path, c.token)
}

// doSiteRequest performs a site-scoped API request.
func (c *Client) doSiteRequest(ctx context.Context, siteID, method, path string, body interface{}) (*APIResponse, error) {
	url := c.siteURL(siteID, path)
	return c.doRequest(ctx, method, url, body)
}

// doSiteRequestWithParams is like doSiteRequest but appends extra query params.
func (c *Client) doSiteRequestWithParams(ctx context.Context, siteID, method, path, extraParams string, body interface{}) (*APIResponse, error) {
	url := c.siteURL(siteID, path) + extraParams
	return c.doRequest(ctx, method, url, body)
}

// doRequest performs an HTTP request with authentication headers and retry on session expiry.
func (c *Client) doRequest(ctx context.Context, method, url string, body interface{}) (*APIResponse, error) {
	return c.doRequestWithRetry(ctx, method, url, body, true)
}

func (c *Client) doRequestWithRetry(ctx context.Context, method, url string, body interface{}, retry bool) (*APIResponse, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Csrf-Token", c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w", method, url, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("decoding response (status %d, body: %s): %w", resp.StatusCode, string(respBody), err)
	}

	// Session expired — re-auth and retry once
	if apiResp.ErrorCode == -1 && retry {
		if err := c.reAuth(ctx); err != nil {
			return nil, fmt.Errorf("re-authentication failed: %w", err)
		}
		// Rebuild URL with new token
		url = strings.Replace(url, "&token="+c.token, "&token="+c.token, 1)
		return c.doRequestWithRetry(ctx, method, url, body, false)
	}

	if apiResp.ErrorCode != 0 {
		return &apiResp, fmt.Errorf("API error %d: %s", apiResp.ErrorCode, apiResp.Msg)
	}

	return &apiResp, nil
}

// decodePaginatedData decodes paginated list data from an API response.
func decodePaginatedData(result json.RawMessage, target interface{}) error {
	var paginated PaginatedResult
	if err := json.Unmarshal(result, &paginated); err != nil {
		// Try direct array decode (some endpoints don't paginate)
		return json.Unmarshal(result, target)
	}
	if paginated.Data == nil {
		return json.Unmarshal(result, target)
	}
	return json.Unmarshal(paginated.Data, target)
}

// isEmptyResult returns true if the API response result is empty, null, or
// contains only whitespace. The Omada 6.x API sometimes returns an empty
// result body on successful PATCH operations.
func isEmptyResult(result json.RawMessage) bool {
	if len(result) == 0 {
		return true
	}
	trimmed := strings.TrimSpace(string(result))
	return trimmed == "" || trimmed == "null" || trimmed == "{}" || trimmed == "\"\"" || trimmed == "[]"
}

// isAgileSeriesError returns true if the API error indicates the switch requires
// the Agile Series (/es/) path (error code -39742).
func isAgileSeriesError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "-39742")
}

// GetOmadacID returns the controller ID.
func (c *Client) GetOmadacID() string { return c.omadacID }

// ResolveSiteID looks up a site ID by name. Returns the ID if the input
// already matches a site ID directly.
func (c *Client) ResolveSiteID(ctx context.Context, nameOrID string) (string, error) {
	sites, err := c.ListSites(ctx)
	if err != nil {
		return "", err
	}
	for _, s := range sites {
		if strings.EqualFold(s.Name, nameOrID) || s.ID == nameOrID {
			return s.ID, nil
		}
	}
	return "", fmt.Errorf("site %q not found", nameOrID)
}

// --- Sites ---

// ListSites returns all sites from the controller.
func (c *Client) ListSites(ctx context.Context) ([]Site, error) {
	url := c.globalURL("/sites") + "&currentPage=1&currentPageSize=100"
	resp, err := c.doRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	var sites []Site
	if err := decodePaginatedData(resp.Result, &sites); err != nil {
		return nil, fmt.Errorf("decoding sites: %w", err)
	}
	return sites, nil
}

// GetSite returns a single site by ID via GET /api/v2/sites/{siteId}.
func (c *Client) GetSite(ctx context.Context, siteID string) (*Site, error) {
	url := c.globalURL(fmt.Sprintf("/sites/%s", siteID))
	resp, err := c.doRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	var site Site
	if err := json.Unmarshal(resp.Result, &site); err != nil {
		return nil, fmt.Errorf("decoding site: %w", err)
	}
	return &site, nil
}

// CreateSite creates a new site via POST /api/v2/sites.
func (c *Client) CreateSite(ctx context.Context, req *SiteCreateRequest) (string, error) {
	url := c.globalURL("/sites")
	resp, err := c.doRequest(ctx, http.MethodPost, url, req)
	if err != nil {
		return "", err
	}
	var result SiteCreateResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return "", fmt.Errorf("decoding create site result: %w", err)
	}
	return result.SiteID, nil
}

// UpdateSite updates a site's name, region, timezone, and scenario via PATCH /sites/{id}/setting.
func (c *Client) UpdateSite(ctx context.Context, siteID string, fields *SiteSettingFields) error {
	url := fmt.Sprintf("%s/%s/api/v2/sites/%s/setting?token=%s", c.baseURL, c.omadacID, siteID, c.token)
	payload := &SiteSettingUpdate{Site: fields}
	_, err := c.doRequest(ctx, http.MethodPatch, url, payload)
	return err
}

// DeleteSite deletes a site via DELETE /api/v2/sites/{siteId}.
func (c *Client) DeleteSite(ctx context.Context, siteID string) error {
	url := c.globalURL(fmt.Sprintf("/sites/%s", siteID))
	_, err := c.doRequest(ctx, http.MethodDelete, url, nil)
	return err
}

// --- Networks ---

// ListNetworks returns all LAN networks for the given site.
func (c *Client) ListNetworks(ctx context.Context, siteID string) ([]Network, error) {
	resp, err := c.doSiteRequestWithParams(ctx, siteID, http.MethodGet, "/setting/lan/networks", "&currentPage=1&currentPageSize=100", nil)
	if err != nil {
		return nil, err
	}
	var networks []Network
	if err := decodePaginatedData(resp.Result, &networks); err != nil {
		return nil, fmt.Errorf("decoding networks: %w", err)
	}
	return networks, nil
}

// GetNetwork returns a network by ID.
func (c *Client) GetNetwork(ctx context.Context, siteID, networkID string) (*Network, error) {
	networks, err := c.ListNetworks(ctx, siteID)
	if err != nil {
		return nil, err
	}
	for _, n := range networks {
		if n.ID == networkID {
			return &n, nil
		}
	}
	return nil, fmt.Errorf("network %q not found", networkID)
}

// CreateNetwork creates a new LAN network, or adopts an existing one with the
// same name (the controller auto-creates a "Default" network on site creation).
func (c *Client) CreateNetwork(ctx context.Context, siteID string, network *Network) (*Network, error) {
	// Check for an existing network with the same name (adopt pattern).
	existing, err := c.ListNetworks(ctx, siteID)
	if err != nil {
		return nil, fmt.Errorf("listing networks for adopt check: %w", err)
	}
	for _, n := range existing {
		if n.Name == network.Name {
			// Adopt: return the existing network instead of creating a duplicate.
			return &n, nil
		}
	}

	resp, err := c.doSiteRequest(ctx, siteID, http.MethodPost, "/setting/lan/networks", network)
	if err != nil {
		return nil, err
	}

	// The API may return a full Network object or just a string ID.
	// Try to unmarshal as a string first (VLAN-only networks return the new ID).
	var networkID string
	if err := json.Unmarshal(resp.Result, &networkID); err == nil && networkID != "" {
		// Got a string ID — do a follow-up GET to retrieve the full object.
		return c.GetNetwork(ctx, siteID, networkID)
	}

	// Otherwise try to unmarshal as a Network object.
	var created Network
	if err := json.Unmarshal(resp.Result, &created); err != nil {
		return nil, fmt.Errorf("decoding created network (raw: %s): %w", string(resp.Result), err)
	}
	if created.ID != "" {
		return &created, nil
	}
	return nil, fmt.Errorf("network created but no ID in response: %s", string(resp.Result))
}

// UpdateNetwork updates an existing LAN network.
func (c *Client) UpdateNetwork(ctx context.Context, siteID, networkID string, network *Network) (*Network, error) {
	resp, err := c.doSiteRequest(ctx, siteID, http.MethodPatch, fmt.Sprintf("/setting/lan/networks/%s", networkID), network)
	if err != nil {
		return nil, err
	}
	if isEmptyResult(resp.Result) {
		return c.GetNetwork(ctx, siteID, networkID)
	}
	var updated Network
	if err := json.Unmarshal(resp.Result, &updated); err != nil {
		return nil, fmt.Errorf("decoding updated network: %w", err)
	}
	return &updated, nil
}

// DeleteNetwork deletes a LAN network.
func (c *Client) DeleteNetwork(ctx context.Context, siteID, networkID string) error {
	_, err := c.doSiteRequest(ctx, siteID, http.MethodDelete, fmt.Sprintf("/setting/lan/networks/%s", networkID), nil)
	return err
}

// --- Wireless Networks (SSIDs) ---

// ListWlanGroups returns all WLAN groups.
func (c *Client) ListWlanGroups(ctx context.Context, siteID string) ([]WlanGroup, error) {
	resp, err := c.doSiteRequestWithParams(ctx, siteID, http.MethodGet, "/setting/wlans", "&currentPage=1&currentPageSize=100", nil)
	if err != nil {
		return nil, err
	}
	var groups []WlanGroup
	if err := decodePaginatedData(resp.Result, &groups); err != nil {
		return nil, fmt.Errorf("decoding wlan groups: %w", err)
	}
	return groups, nil
}

// GetDefaultWlanGroupID returns the first WLAN group's ID (usually "Default").
func (c *Client) GetDefaultWlanGroupID(ctx context.Context, siteID string) (string, error) {
	groups, err := c.ListWlanGroups(ctx, siteID)
	if err != nil {
		return "", err
	}
	if len(groups) == 0 {
		return "", fmt.Errorf("no WLAN groups found")
	}
	return groups[0].ID, nil
}

// GetWlanGroup returns a WLAN group by ID (fetches from list since individual GET is not supported).
func (c *Client) GetWlanGroup(ctx context.Context, siteID, wlanGroupID string) (*WlanGroup, error) {
	groups, err := c.ListWlanGroups(ctx, siteID)
	if err != nil {
		return nil, err
	}
	for _, g := range groups {
		if g.ID == wlanGroupID {
			return &g, nil
		}
	}
	return nil, fmt.Errorf("WLAN group %q not found", wlanGroupID)
}

// CreateWlanGroup creates a new WLAN group.
func (c *Client) CreateWlanGroup(ctx context.Context, siteID, name string, clone bool) (string, error) {
	req := &WlanGroupCreateRequest{
		Name:  name,
		Clone: clone,
	}
	resp, err := c.doSiteRequest(ctx, siteID, http.MethodPost, "/setting/wlans", req)
	if err != nil {
		return "", err
	}
	var result WlanGroupCreateResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return "", fmt.Errorf("decoding create wlan group result: %w", err)
	}
	return result.WlanID, nil
}

// UpdateWlanGroup renames a WLAN group.
func (c *Client) UpdateWlanGroup(ctx context.Context, siteID, wlanGroupID, name string) error {
	req := &WlanGroupUpdateRequest{Name: name}
	_, err := c.doSiteRequest(ctx, siteID, http.MethodPatch, fmt.Sprintf("/setting/wlans/%s", wlanGroupID), req)
	return err
}

// DeleteWlanGroup deletes a WLAN group.
func (c *Client) DeleteWlanGroup(ctx context.Context, siteID, wlanGroupID string) error {
	_, err := c.doSiteRequest(ctx, siteID, http.MethodDelete, fmt.Sprintf("/setting/wlans/%s", wlanGroupID), nil)
	return err
}

// ListWirelessNetworks returns all SSIDs in a WLAN group.
func (c *Client) ListWirelessNetworks(ctx context.Context, siteID, wlanGroupID string) ([]WirelessNetwork, error) {
	resp, err := c.doSiteRequestWithParams(ctx, siteID, http.MethodGet, fmt.Sprintf("/setting/wlans/%s/ssids", wlanGroupID), "&currentPage=1&currentPageSize=100", nil)
	if err != nil {
		return nil, err
	}
	var ssids []WirelessNetwork
	if err := decodePaginatedData(resp.Result, &ssids); err != nil {
		return nil, fmt.Errorf("decoding SSIDs: %w", err)
	}
	return ssids, nil
}

// GetWirelessNetwork returns a specific SSID.
func (c *Client) GetWirelessNetwork(ctx context.Context, siteID, wlanGroupID, ssidID string) (*WirelessNetwork, error) {
	ssids, err := c.ListWirelessNetworks(ctx, siteID, wlanGroupID)
	if err != nil {
		return nil, err
	}
	for _, s := range ssids {
		if s.ID == ssidID {
			return &s, nil
		}
	}
	return nil, fmt.Errorf("SSID %q not found in WLAN group %q", ssidID, wlanGroupID)
}

// GetWirelessNetworkRaw returns the raw JSON for a specific SSID (needed for PATCH).
func (c *Client) GetWirelessNetworkRaw(ctx context.Context, siteID, wlanGroupID, ssidID string) (map[string]interface{}, error) {
	resp, err := c.doSiteRequestWithParams(ctx, siteID, http.MethodGet, fmt.Sprintf("/setting/wlans/%s/ssids", wlanGroupID), "&currentPage=1&currentPageSize=100", nil)
	if err != nil {
		return nil, err
	}

	var paginated PaginatedResult
	if err := json.Unmarshal(resp.Result, &paginated); err != nil {
		return nil, err
	}

	var ssids []map[string]interface{}
	if err := json.Unmarshal(paginated.Data, &ssids); err != nil {
		return nil, err
	}

	for _, s := range ssids {
		if id, ok := s["id"].(string); ok && id == ssidID {
			return s, nil
		}
	}
	return nil, fmt.Errorf("SSID %q not found", ssidID)
}

// CreateWirelessNetwork creates a new SSID.
func (c *Client) CreateWirelessNetwork(ctx context.Context, siteID, wlanGroupID string, ssid *WirelessNetwork) (*WirelessNetwork, error) {
	resp, err := c.doSiteRequest(ctx, siteID, http.MethodPost, fmt.Sprintf("/setting/wlans/%s/ssids", wlanGroupID), ssid)
	if err != nil {
		return nil, err
	}

	// The API returns {"ssidId": "<id>"}, not a full SSID object.
	var createResult struct {
		SsidID string `json:"ssidId"`
	}
	if err := json.Unmarshal(resp.Result, &createResult); err == nil && createResult.SsidID != "" {
		return c.GetWirelessNetwork(ctx, siteID, wlanGroupID, createResult.SsidID)
	}

	// Fallback: try to unmarshal as a full WirelessNetwork.
	var created WirelessNetwork
	if err := json.Unmarshal(resp.Result, &created); err != nil {
		return nil, fmt.Errorf("decoding created SSID (raw: %s): %w", string(resp.Result), err)
	}
	return &created, nil
}

// UpdateWirelessNetwork updates an existing SSID (requires full object).
func (c *Client) UpdateWirelessNetwork(ctx context.Context, siteID, wlanGroupID, ssidID string, ssid map[string]interface{}) (*WirelessNetwork, error) {
	// Remove read-only fields that must not be in PATCH
	readOnlyFields := []string{"id", "idInt", "index", "site", "resource", "vlanEnable", "portalEnable", "accessEnable"}
	for _, f := range readOnlyFields {
		delete(ssid, f)
	}

	resp, err := c.doSiteRequest(ctx, siteID, http.MethodPatch, fmt.Sprintf("/setting/wlans/%s/ssids/%s", wlanGroupID, ssidID), ssid)
	if err != nil {
		return nil, err
	}
	if isEmptyResult(resp.Result) {
		return c.GetWirelessNetwork(ctx, siteID, wlanGroupID, ssidID)
	}
	var updated WirelessNetwork
	if err := json.Unmarshal(resp.Result, &updated); err != nil {
		return nil, fmt.Errorf("decoding updated SSID: %w", err)
	}
	return &updated, nil
}

// DeleteWirelessNetwork deletes an SSID.
func (c *Client) DeleteWirelessNetwork(ctx context.Context, siteID, wlanGroupID, ssidID string) error {
	_, err := c.doSiteRequest(ctx, siteID, http.MethodDelete, fmt.Sprintf("/setting/wlans/%s/ssids/%s", wlanGroupID, ssidID), nil)
	return err
}

// --- Port Profiles ---

// ListPortProfiles returns all LAN port profiles.
func (c *Client) ListPortProfiles(ctx context.Context, siteID string) ([]PortProfile, error) {
	resp, err := c.doSiteRequestWithParams(ctx, siteID, http.MethodGet, "/setting/lan/profiles", "&currentPage=1&currentPageSize=100", nil)
	if err != nil {
		return nil, err
	}
	var profiles []PortProfile
	if err := decodePaginatedData(resp.Result, &profiles); err != nil {
		return nil, fmt.Errorf("decoding port profiles: %w", err)
	}
	return profiles, nil
}

// GetPortProfile returns a port profile by ID.
func (c *Client) GetPortProfile(ctx context.Context, siteID, profileID string) (*PortProfile, error) {
	profiles, err := c.ListPortProfiles(ctx, siteID)
	if err != nil {
		return nil, err
	}
	for _, p := range profiles {
		if p.ID == profileID {
			return &p, nil
		}
	}
	return nil, fmt.Errorf("port profile %q not found", profileID)
}

// CreatePortProfile creates a new port profile, or adopts an existing one with the same name.
func (c *Client) CreatePortProfile(ctx context.Context, siteID string, profile *PortProfile) (*PortProfile, error) {
	// Check if a profile with this name already exists (adopt pattern).
	existing, err := c.ListPortProfiles(ctx, siteID)
	if err == nil {
		for _, p := range existing {
			if p.Name == profile.Name {
				// Adopt the existing profile — update it to match desired state.
				updated, err := c.UpdatePortProfile(ctx, siteID, p.ID, profile)
				if err != nil {
					return nil, fmt.Errorf("adopting existing port profile %q (%s): %w", p.Name, p.ID, err)
				}
				return updated, nil
			}
		}
	}

	resp, err := c.doSiteRequest(ctx, siteID, http.MethodPost, "/setting/lan/profiles", profile)
	if err != nil {
		return nil, err
	}
	var created PortProfile
	if err := json.Unmarshal(resp.Result, &created); err != nil {
		return nil, fmt.Errorf("decoding created port profile: %w", err)
	}
	return &created, nil
}

// UpdatePortProfile updates a port profile.
func (c *Client) UpdatePortProfile(ctx context.Context, siteID, profileID string, profile *PortProfile) (*PortProfile, error) {
	resp, err := c.doSiteRequest(ctx, siteID, http.MethodPatch, fmt.Sprintf("/setting/lan/profiles/%s", profileID), profile)
	if err != nil {
		return nil, err
	}
	if isEmptyResult(resp.Result) {
		return c.GetPortProfile(ctx, siteID, profileID)
	}
	var updated PortProfile
	if err := json.Unmarshal(resp.Result, &updated); err != nil {
		return nil, fmt.Errorf("decoding updated port profile: %w", err)
	}
	return &updated, nil
}

// DeletePortProfile deletes a port profile.
func (c *Client) DeletePortProfile(ctx context.Context, siteID, profileID string) error {
	_, err := c.doSiteRequest(ctx, siteID, http.MethodDelete, fmt.Sprintf("/setting/lan/profiles/%s", profileID), nil)
	return err
}

// --- Site Settings ---

// SiteSettings represents the full site settings object from GET /setting.
type SiteSettings struct {
	Site                     *SiteSettingsSite         `json:"site,omitempty"`
	AutoUpgrade              *AutoUpgrade              `json:"autoUpgrade,omitempty"`
	Mesh                     *MeshSettings             `json:"mesh,omitempty"`
	SpeedTest                *SpeedTest                `json:"speedTest,omitempty"`
	Alert                    *AlertSettings            `json:"alert,omitempty"`
	RemoteLog                *RemoteLog                `json:"remoteLog,omitempty"`
	AdvancedFeature          *AdvancedFeature          `json:"advancedFeature,omitempty"`
	LLDP                     *LLDPSettings             `json:"lldp,omitempty"`
	BeaconControl            *BeaconControl            `json:"beaconControl,omitempty"`
	BandSteering             *BandSteering             `json:"bandSteering,omitempty"`
	BandSteeringForMultiBand *BandSteeringForMultiBand `json:"bandSteeringForMultiBand,omitempty"`
	AirtimeFairness          *AirtimeFairness          `json:"airtimeFairness,omitempty"`
	LED                      *LEDSettings              `json:"led,omitempty"`
	DeviceAccount            *DeviceAccount            `json:"deviceAccount,omitempty"`
	Roaming                  *RoamingSettings          `json:"roaming,omitempty"`
	RememberDevice           *RememberDevice           `json:"rememberDevice,omitempty"`
}

// SiteSettingsSite holds the core site identity fields within settings.
type SiteSettingsSite struct {
	Key      string `json:"key,omitempty"`
	Name     string `json:"name,omitempty"`
	Region   string `json:"region,omitempty"`
	TimeZone string `json:"timeZone,omitempty"`
	Scenario string `json:"scenario,omitempty"`
}

// AutoUpgrade controls automatic firmware upgrade.
type AutoUpgrade struct {
	Enable bool `json:"enable"`
}

// MeshSettings controls mesh networking.
type MeshSettings struct {
	MeshEnable         bool `json:"meshEnable"`
	AutoFailoverEnable bool `json:"autoFailoverEnable"`
	DefGatewayEnable   bool `json:"defGatewayEnable"`
	FullSector         bool `json:"fullSector"`
}

// SpeedTest controls the speed test schedule.
type SpeedTest struct {
	Enable   bool `json:"enable"`
	Interval int  `json:"interval,omitempty"`
}

// AlertSettings controls alert notifications.
type AlertSettings struct {
	Enable      bool `json:"enable"`
	DelayEnable bool `json:"delayEnable"`
	Delay       int  `json:"delay,omitempty"`
}

// RemoteLog controls syslog remote logging.
type RemoteLog struct {
	Enable        bool   `json:"enable"`
	Server        string `json:"server,omitempty"`
	Port          int    `json:"port,omitempty"`
	MoreClientLog bool   `json:"moreClientLog"`
}

// AdvancedFeature controls the advanced features toggle.
type AdvancedFeature struct {
	Enable bool `json:"enable"`
}

// LLDPSettings controls the LLDP protocol toggle.
type LLDPSettings struct {
	Enable bool `json:"enable"`
}

// BeaconControl holds Wi-Fi beacon and DTIM settings per band.
type BeaconControl struct {
	BeaconIntvMode2g         int `json:"beaconIntvMode2g"`
	DtimPeriod2g             int `json:"dtimPeriod2g"`
	RtsThreshold2g           int `json:"rtsThreshold2g"`
	FragmentationThreshold2g int `json:"fragmentationThreshold2g"`
	BeaconIntvMode5g         int `json:"beaconIntvMode5g"`
	DtimPeriod5g             int `json:"dtimPeriod5g"`
	RtsThreshold5g           int `json:"rtsThreshold5g"`
	FragmentationThreshold5g int `json:"fragmentationThreshold5g"`
	BeaconInterval6g         int `json:"beaconInterval6g"`
	BeaconIntvMode6g         int `json:"beaconIntvMode6g"`
	DtimPeriod6g             int `json:"dtimPeriod6g"`
	RtsThreshold6g           int `json:"rtsThreshold6g"`
	FragmentationThreshold6g int `json:"fragmentationThreshold6g"`
}

// BandSteering controls band steering parameters.
type BandSteering struct {
	Enable              bool `json:"enable"`
	ConnectionThreshold int  `json:"connectionThreshold,omitempty"`
	DifferenceThreshold int  `json:"differenceThreshold,omitempty"`
	MaxFailures         int  `json:"maxFailures,omitempty"`
}

// BandSteeringForMultiBand controls multi-band steering mode.
type BandSteeringForMultiBand struct {
	Mode int `json:"mode"`
}

// AirtimeFairness controls airtime fairness per band.
type AirtimeFairness struct {
	Enable2g bool `json:"enable2g"`
	Enable5g bool `json:"enable5g"`
	Enable6g bool `json:"enable6g"`
}

// LEDSettings controls AP LED on/off.
type LEDSettings struct {
	Enable bool `json:"enable"`
}

// DeviceAccount holds device SSH/management credentials.
type DeviceAccount struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

// RoamingSettings controls fast and AI roaming.
type RoamingSettings struct {
	FastRoamingEnable         bool `json:"fastRoamingEnable"`
	AiRoamingEnable           bool `json:"aiRoamingEnable"`
	DualBand11kReportEnable   bool `json:"dualBand11kReportEnable"`
	ForceDisassociationEnable bool `json:"forceDisassociationEnable"`
	NonStickRoamingEnable     bool `json:"nonStickRoamingEnable"`
	NonPingPongRoamingEnable  bool `json:"nonPingPongRoamingEnable"`
}

// RememberDevice controls the remember device toggle.
type RememberDevice struct {
	Enable bool `json:"enable"`
}

// GetSiteSettings returns the full site settings for the given site.
func (c *Client) GetSiteSettings(ctx context.Context, siteID string) (*SiteSettings, error) {
	resp, err := c.doSiteRequest(ctx, siteID, http.MethodGet, "/setting", nil)
	if err != nil {
		return nil, err
	}
	var settings SiteSettings
	if err := json.Unmarshal(resp.Result, &settings); err != nil {
		return nil, fmt.Errorf("decoding site settings: %w", err)
	}
	return &settings, nil
}

// UpdateSiteSettings patches site settings with the provided partial object.
// The Omada API may return an empty result body on success (e.g., when
// deviceAccount is omitted). In that case, we do a follow-up GET.
func (c *Client) UpdateSiteSettings(ctx context.Context, siteID string, settings *SiteSettings) (*SiteSettings, error) {
	resp, err := c.doSiteRequest(ctx, siteID, http.MethodPatch, "/setting", settings)
	if err != nil {
		return nil, err
	}
	if isEmptyResult(resp.Result) {
		return c.GetSiteSettings(ctx, siteID)
	}
	var updated SiteSettings
	if err := json.Unmarshal(resp.Result, &updated); err != nil {
		return nil, fmt.Errorf("decoding updated site settings: %w", err)
	}
	return &updated, nil
}

// --- Devices ---

// Device represents a device in the Omada controller (AP, switch, gateway).
type Device struct {
	Type            string  `json:"type"`
	MAC             string  `json:"mac"`
	Name            string  `json:"name"`
	Model           string  `json:"model"`
	ModelVersion    string  `json:"modelVersion,omitempty"`
	FirmwareVersion string  `json:"firmwareVersion,omitempty"`
	Version         string  `json:"version,omitempty"`
	IP              string  `json:"ip"`
	Status          int     `json:"status"`
	StatusCategory  int     `json:"statusCategory,omitempty"`
	Uptime          string  `json:"uptime,omitempty"`
	UptimeLong      int64   `json:"uptimeLong,omitempty"`
	CPUUtil         float64 `json:"cpuUtil,omitempty"`
	MemUtil         float64 `json:"memUtil,omitempty"`
	ClientNum       int     `json:"clientNum,omitempty"`
}

// APRadioSetting represents radio configuration for 2.4GHz or 5GHz.
type APRadioSetting struct {
	RadioEnable  bool   `json:"radioEnable"`
	ChannelWidth string `json:"channelWidth"`
	Channel      string `json:"channel"`
	TxPower      int    `json:"txPower"`
	TxPowerLevel int    `json:"txPowerLevel"`
	Freq         int    `json:"freq,omitempty"`
	WirelessMode int    `json:"wirelessMode,omitempty"`
}

// APIPSetting holds IP configuration for the AP.
type APIPSetting struct {
	Mode         string `json:"mode"`
	Fallback     bool   `json:"fallback"`
	FallbackIP   string `json:"fallbackIp,omitempty"`
	FallbackMask string `json:"fallbackMask,omitempty"`
	FallbackGate string `json:"fallbackGate,omitempty"`
	UseFixedAddr bool   `json:"useFixedAddr"`
}

// APMVlanSetting holds management VLAN settings.
type APMVlanSetting struct {
	Mode         int    `json:"mode"`
	LanNetworkID string `json:"lanNetworkId,omitempty"`
}

// APLBSetting holds load balancing settings per band.
type APLBSetting struct {
	LBEnable   bool `json:"lbEnable"`
	MaxClients int  `json:"maxClients,omitempty"`
}

// APRSSISetting holds RSSI threshold settings per band.
type APRSSISetting struct {
	RSSIEnable bool `json:"rssiEnable"`
	Threshold  int  `json:"threshold,omitempty"`
}

// APQoSSetting holds QoS/WMM settings per band.
type APQoSSetting struct {
	WmmEnable         bool `json:"wmmEnable"`
	NoAcknowledgement bool `json:"noAcknowledgement"`
	DeliveryEnable    bool `json:"deliveryEnable"`
}

// APL3AccessSetting holds L3 management access settings.
type APL3AccessSetting struct {
	Enable bool `json:"enable"`
}

// APSSIDOverride represents a per-SSID override on an AP.
type APSSIDOverride struct {
	Index        int    `json:"index"`
	GlobalSsid   string `json:"globalSsid,omitempty"`
	SupportBands []int  `json:"supportBands,omitempty"`
	SSIDEnable   bool   `json:"ssidEnable"`
	Enable       bool   `json:"enable"`
	SSID         string `json:"ssid,omitempty"`
	PSK          string `json:"psk,omitempty"`
	VlanEnable   bool   `json:"vlanEnable,omitempty"`
	VlanID       int    `json:"vlanId,omitempty"`
	Security     int    `json:"security,omitempty"`
}

// APLanPortSetting represents per-LAN-port config on an AP.
type APLanPortSetting struct {
	LanPort            interface{} `json:"lanPort"`
	PortType           int         `json:"portType,omitempty"`
	SupportVlan        bool        `json:"supportVlan,omitempty"`
	LocalVlanEnable    bool        `json:"localVlanEnable,omitempty"`
	SupportPoe         bool        `json:"supportPoe,omitempty"`
	PoeOutEnable       bool        `json:"poeOutEnable,omitempty"`
	Dot1xEnable        bool        `json:"dot1xEnable,omitempty"`
	MabEnable          bool        `json:"mabEnable,omitempty"`
	TaggedNetworkIDs   []string    `json:"taggedNetworkId,omitempty"`
	UntaggedNetworkIDs []string    `json:"untaggedNetworkId,omitempty"`
	Status             int         `json:"status,omitempty"`
	Name               string      `json:"name,omitempty"`
}

// APConfig represents the full configurable AP object from GET /eaps/{mac}.
// Fields that may be absent on certain AP models use pointer types so we can
// distinguish absent from zero value.
type APConfig struct {
	Type            string          `json:"type,omitempty"`
	MAC             string          `json:"mac,omitempty"`
	Name            string          `json:"name"`
	Model           string          `json:"model,omitempty"`
	IP              string          `json:"ip,omitempty"`
	Status          int             `json:"status,omitempty"`
	FirmwareVersion string          `json:"firmwareVersion,omitempty"`
	WlanID          string          `json:"wlanId,omitempty"`
	RadioSetting2g  *APRadioSetting `json:"radioSetting2g,omitempty"`
	RadioSetting5g  *APRadioSetting `json:"radioSetting5g,omitempty"`
	IPSetting       *APIPSetting    `json:"ipSetting,omitempty"`
	LEDSetting      int             `json:"ledSetting"`

	// Pointer fields — absent on some AP models (nil = unsupported by hardware)
	LLDPEnable           *int  `json:"lldpEnable,omitempty"`
	OFDMAEnable2g        *bool `json:"ofdmaEnable2g,omitempty"`
	OFDMAEnable5g        *bool `json:"ofdmaEnable5g,omitempty"`
	LoopbackDetectEnable *bool `json:"loopbackDetectEnable,omitempty"`

	MVlanEnable     bool               `json:"mvlanEnable"`
	MVlanSetting    *APMVlanSetting    `json:"mvlanSetting,omitempty"`
	L3AccessSetting *APL3AccessSetting `json:"l3AccessSetting,omitempty"`
	LBSetting2g     *APLBSetting       `json:"lbSetting2g,omitempty"`
	LBSetting5g     *APLBSetting       `json:"lbSetting5g,omitempty"`
	RSSISetting2g   *APRSSISetting     `json:"rssiSetting2g,omitempty"`
	RSSISetting5g   *APRSSISetting     `json:"rssiSetting5g,omitempty"`
	QoSSetting2g    *APQoSSetting      `json:"qosSetting2g,omitempty"`
	QoSSetting5g    *APQoSSetting      `json:"qosSetting5g,omitempty"`
	AnyPoeEnable    bool               `json:"anyPoeEnable,omitempty"`
	IPv6Enable      bool               `json:"ipv6Enable,omitempty"`

	// Complex nested fields stored as raw JSON — parsed separately when needed
	SSIDOverrides   json.RawMessage `json:"ssidOverrides,omitempty"`
	LanPortSettings json.RawMessage `json:"lanPortSettings,omitempty"`
}

// ListDevices returns all devices in the given site.
// The devices endpoint returns a plain JSON array (not paginated).
func (c *Client) ListDevices(ctx context.Context, siteID string) ([]Device, error) {
	resp, err := c.doSiteRequest(ctx, siteID, http.MethodGet, "/devices", nil)
	if err != nil {
		return nil, err
	}
	var devices []Device
	if err := json.Unmarshal(resp.Result, &devices); err != nil {
		return nil, fmt.Errorf("decoding devices: %w", err)
	}
	return devices, nil
}

// GetAPConfig returns the full configuration for an AP by MAC address.
func (c *Client) GetAPConfig(ctx context.Context, siteID, mac string) (*APConfig, error) {
	resp, err := c.doSiteRequest(ctx, siteID, http.MethodGet, fmt.Sprintf("/eaps/%s", mac), nil)
	if err != nil {
		return nil, err
	}
	var config APConfig
	if err := json.Unmarshal(resp.Result, &config); err != nil {
		return nil, fmt.Errorf("decoding AP config: %w", err)
	}
	return &config, nil
}

// GetAPConfigRaw returns the raw JSON for an AP (needed for PATCH).
func (c *Client) GetAPConfigRaw(ctx context.Context, siteID, mac string) (map[string]interface{}, error) {
	resp, err := c.doSiteRequest(ctx, siteID, http.MethodGet, fmt.Sprintf("/eaps/%s", mac), nil)
	if err != nil {
		return nil, err
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(resp.Result, &raw); err != nil {
		return nil, fmt.Errorf("decoding AP config raw: %w", err)
	}
	return raw, nil
}

// UpdateAPConfig updates AP general configuration via PATCH /eaps/{mac}.
// This handles: name, wlanId, ledSetting, ipSetting, mvlanEnable, mvlanSetting,
// loopbackDetectEnable. Radio, advanced (OFDMA/LB/RSSI), and services (LLDP/L3)
// settings must be updated via their dedicated endpoints.
func (c *Client) UpdateAPConfig(ctx context.Context, siteID, mac string, config map[string]interface{}) (*APConfig, error) {
	// Remove read-only / status fields that must not be in PATCH
	readOnlyFields := []string{
		"type", "mac", "model", "modelVersion", "ip", "status", "statusCategory",
		"firmwareVersion", "version", "uptime", "uptimeLong", "cpuUtil", "memUtil",
		"clientNum", "deviceMisc", "devCap", "wp2g", "wp5g",
		"radioTraffic2g", "radioTraffic5g", "wiredUplink", "lanTraffic",
		"lastSeen", "needUpgrade", "fwDownloadStatus", "adoptFailType",
		"site", "compatible", "showModel", "snmpLocation",
	}
	for _, f := range readOnlyFields {
		delete(config, f)
	}

	resp, err := c.doSiteRequest(ctx, siteID, http.MethodPatch, fmt.Sprintf("/eaps/%s", mac), config)
	if err != nil {
		return nil, err
	}
	if isEmptyResult(resp.Result) {
		return c.GetAPConfig(ctx, siteID, mac)
	}
	var updated APConfig
	if err := json.Unmarshal(resp.Result, &updated); err != nil {
		return nil, fmt.Errorf("decoding updated AP config: %w", err)
	}
	return &updated, nil
}

// APRadioConfig is the payload for PUT /eaps/{mac}/config/radios.
// The Omada API ignores radio settings sent via the main PATCH endpoint;
// they must be sent to this dedicated endpoint.
type APRadioConfig struct {
	RadioSetting2g *APRadioSetting `json:"radioSetting2g,omitempty"`
	RadioSetting5g *APRadioSetting `json:"radioSetting5g,omitempty"`
}

// APAdvancedConfig is the payload for PUT /eaps/{mac}/config/advanced.
// Handles OFDMA, load balancing, RSSI, and QoS settings.
// The Omada API ignores these fields when sent via the main PATCH endpoint.
type APAdvancedConfig struct {
	OFDMAEnable2g *bool          `json:"ofdmaEnable2g,omitempty"`
	OFDMAEnable5g *bool          `json:"ofdmaEnable5g,omitempty"`
	LBSetting2g   *APLBSetting   `json:"lbSetting2g,omitempty"`
	LBSetting5g   *APLBSetting   `json:"lbSetting5g,omitempty"`
	RSSISetting2g *APRSSISetting `json:"rssiSetting2g,omitempty"`
	RSSISetting5g *APRSSISetting `json:"rssiSetting5g,omitempty"`
	QoSSetting2g  *APQoSSetting  `json:"qosSetting2g,omitempty"`
	QoSSetting5g  *APQoSSetting  `json:"qosSetting5g,omitempty"`
}

// APServicesConfig is the payload for PUT /eaps/{mac}/config/services.
// Handles LLDP and L3 access settings.
// The Omada API ignores these fields when sent via the main PATCH endpoint.
type APServicesConfig struct {
	LLDPEnable      *int               `json:"lldpEnable,omitempty"`
	L3AccessSetting *APL3AccessSetting `json:"l3AccessSetting,omitempty"`
	SNMP            *SwitchSNMP        `json:"snmp,omitempty"`
}

// UpdateAPRadioConfig updates AP radio settings via PUT /eaps/{mac}/config/radios.
func (c *Client) UpdateAPRadioConfig(ctx context.Context, siteID, mac string, config *APRadioConfig) error {
	_, err := c.doSiteRequest(ctx, siteID, http.MethodPut, fmt.Sprintf("/eaps/%s/config/radios", mac), config)
	return err
}

// UpdateAPAdvancedConfig updates AP advanced settings via PUT /eaps/{mac}/config/advanced.
func (c *Client) UpdateAPAdvancedConfig(ctx context.Context, siteID, mac string, config *APAdvancedConfig) error {
	_, err := c.doSiteRequest(ctx, siteID, http.MethodPut, fmt.Sprintf("/eaps/%s/config/advanced", mac), config)
	return err
}

// UpdateAPServicesConfig updates AP services settings via PUT /eaps/{mac}/config/services.
func (c *Client) UpdateAPServicesConfig(ctx context.Context, siteID, mac string, config *APServicesConfig) error {
	_, err := c.doSiteRequest(ctx, siteID, http.MethodPut, fmt.Sprintf("/eaps/%s/config/services", mac), config)
	return err
}

// --- Switch Devices ---
//
// The Omada controller uses different API path prefixes for Agile Series (ES)
// switches vs standard switches:
//
//   Standard:      /switches/{mac}/...
//   Agile Series:  /switches/es/{mac}/...
//
// Detection is automatic:
//   - GET always tries /switches/{mac} first. If the controller returns error
//     -39742 ("Agile Series Switch should use the corresponding path"), the
//     request is automatically retried with /switches/es/{mac}.
//   - Write operations use the "es" boolean field present in the GET response
//     to select the correct path.
//
// Port updates (PATCH /switches/{mac}/ports/{port}) work universally across
// all switch series and do not require the /es/ prefix.

// SwitchIPSetting holds IP configuration for a switch.
type SwitchIPSetting struct {
	Mode         string `json:"mode"`
	Fallback     bool   `json:"fallback"`
	FallbackIP   string `json:"fallbackIp,omitempty"`
	FallbackMask string `json:"fallbackMask,omitempty"`
	FallbackGate string `json:"fallbackGate,omitempty"`
}

// SwitchSNMP holds SNMP settings for a switch.
type SwitchSNMP struct {
	Location string `json:"location"`
	Contact  string `json:"contact"`
}

// SwitchPort represents a port configuration on a switch.
type SwitchPort struct {
	ID                    string   `json:"id,omitempty"`
	Port                  int      `json:"port"`
	Name                  string   `json:"name"`
	Disable               bool     `json:"disable"`
	Type                  int      `json:"type"`
	MaxSpeed              int      `json:"maxSpeed,omitempty"`
	NativeNetworkID       string   `json:"nativeNetworkId,omitempty"`
	NetworkTagsSetting    int      `json:"networkTagsSetting"`
	TagNetworkIDs         []string `json:"tagNetworkIds"`
	UntagNetworkIDs       []string `json:"untagNetworkIds"`
	VoiceNetworkEnable    bool     `json:"voiceNetworkEnable"`
	VoiceDscpEnable       bool     `json:"voiceDscpEnable"`
	ProfileID             string   `json:"profileId"`
	ProfileName           string   `json:"profileName,omitempty"`
	ProfileOverrideEnable bool     `json:"profileOverrideEnable"`
	Operation             string   `json:"operation,omitempty"`
	Speed                 int      `json:"speed"`
}

// SwitchServiceConfig is the payload for PUT /switches/{mac}/config/service.
// Handles loopback detection and STP settings.
// The Omada API ignores these fields when sent via the general config endpoint.
type SwitchServiceConfig struct {
	LoopbackDetectEnable bool `json:"loopbackDetectEnable"`
	STP                  *int `json:"stp,omitempty"`
}

// SwitchConfig represents the full configurable switch object from GET /switches/{mac}.
type SwitchConfig struct {
	Type                 string           `json:"type,omitempty"`
	MAC                  string           `json:"mac,omitempty"`
	Name                 string           `json:"name"`
	Model                string           `json:"model,omitempty"`
	IP                   string           `json:"ip,omitempty"`
	Status               int              `json:"status,omitempty"`
	FirmwareVersion      string           `json:"firmwareVersion,omitempty"`
	LEDSetting           int              `json:"ledSetting"`
	MVlanNetworkID       string           `json:"mvlanNetworkId,omitempty"`
	IPSetting            *SwitchIPSetting `json:"ipSetting,omitempty"`
	LoopbackDetectEnable bool             `json:"loopbackDetectEnable"`
	STP                  int              `json:"stp"`
	Priority             int              `json:"priority"`
	HelloTime            int              `json:"helloTime"`
	MaxAge               int              `json:"maxAge"`
	ForwardDelay         int              `json:"forwardDelay"`
	TxHoldCount          int              `json:"txHoldCount"`
	MaxHops              int              `json:"maxHops"`
	SNMP                 *SwitchSNMP      `json:"snmp,omitempty"`
	Jumbo                int              `json:"jumbo"`
	LagHashAlg           int              `json:"lagHashAlg"`
	Ports                []SwitchPort     `json:"ports,omitempty"`
	// Complex fields stored as raw JSON
	Lags json.RawMessage `json:"lags,omitempty"`
}

// getSwitchRaw fetches raw switch config, automatically retrying with the
// Agile Series path (/switches/es/{mac}) if the standard path returns -39742.
func (c *Client) getSwitchRaw(ctx context.Context, siteID, mac string) (map[string]interface{}, error) {
	resp, err := c.doSiteRequest(ctx, siteID, http.MethodGet, fmt.Sprintf("/switches/%s", mac), nil)
	if err != nil {
		if isAgileSeriesError(err) {
			resp, err = c.doSiteRequest(ctx, siteID, http.MethodGet, fmt.Sprintf("/switches/es/%s", mac), nil)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(resp.Result, &raw); err != nil {
		return nil, fmt.Errorf("decoding switch config raw: %w", err)
	}
	return raw, nil
}

// GetSwitchConfig returns the full configuration for a switch by MAC address.
// Automatically handles both standard and Agile Series (ES) switches.
func (c *Client) GetSwitchConfig(ctx context.Context, siteID, mac string) (*SwitchConfig, error) {
	raw, err := c.getSwitchRaw(ctx, siteID, mac)
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("re-marshaling switch config: %w", err)
	}
	var config SwitchConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("decoding switch config: %w", err)
	}
	return &config, nil
}

// GetSwitchConfigRaw returns the raw JSON for a switch.
// Automatically handles both standard and Agile Series (ES) switches.
func (c *Client) GetSwitchConfigRaw(ctx context.Context, siteID, mac string) (map[string]interface{}, error) {
	return c.getSwitchRaw(ctx, siteID, mac)
}

// UpdateSwitchConfig updates switch-level general configuration.
//
// The path prefix is selected based on the "es" field in the raw GET response:
//   - Agile Series (ES): PATCH /switches/es/{mac}/config/general
//   - Standard switches: PATCH /switches/{mac}/config/general
//
// The /config/general path was confirmed via browser capture on an ES205G
// (Agile Series). The standard switch path follows the same convention by
// analogy — community testing on TL/JetStream series is welcome.
//
// Port updates are handled separately by UpdateSwitchPort.
func (c *Client) UpdateSwitchConfig(ctx context.Context, siteID, mac string, config map[string]interface{}) (*SwitchConfig, error) {
	readOnlyFields := []string{
		"type", "mac", "model", "modelVersion", "compoundModel", "showModel",
		"firmwareVersion", "version", "hwVersion", "ip", "publicIp",
		"status", "statusCategory", "site", "siteName", "omadacId",
		"compatible", "category", "sn", "addedInAdvanced", "customId",
		"remember", "rememberDevice", "boundSiteTemplate", "deviceSeriesType",
		"resource", "ecspFirstVersion", "deviceMisc", "devCap",
		"lastSeen", "needUpgrade", "uptime", "uptimeLong", "cpuUtil", "memUtil",
		"poeTotalPower", "poeRemain", "poeRemainPercent", "fanStatus",
		"download", "upload", "supportVlanIf", "speeds", "loop", "loopbackNum",
		"sdm", "terminalPrefix", "supportHealth", "downlinkList",
		"tagIds", "ipv6List", "ports", "lags",
	}
	for _, f := range readOnlyFields {
		delete(config, f)
	}

	// Determine series from the "es" field then remove it before sending
	// Note, the v1 API shows a PATCH for this, but v2 seems to require a PUT
	// THIS WAS ONLY TESTED USING ES SWITCH
	// ITS POSSIBLE ITS PUT FOR ES AND PATCH FOR THE REST (but that would be odd)
	isES, _ := config["es"].(bool)
	delete(config, "es")

	var path string
	if isES {
		path = fmt.Sprintf("/switches/es/%s/config/general", mac)
	} else {
		path = fmt.Sprintf("/switches/%s/config/general", mac)
	}

	resp, err := c.doSiteRequest(ctx, siteID, http.MethodPut, path, config)
	if err != nil {
		return nil, err
	}
	if isEmptyResult(resp.Result) {
		return c.GetSwitchConfig(ctx, siteID, mac)
	}
	var updated SwitchConfig
	if err := json.Unmarshal(resp.Result, &updated); err != nil {
		return nil, fmt.Errorf("decoding updated switch config: %w", err)
	}
	return &updated, nil
}

// UpdateSwitchPort updates a single port on a switch via PATCH /switches/{mac}/ports/{port}.
// This endpoint works universally across all switch series without the /es/ prefix.
func (c *Client) UpdateSwitchPort(ctx context.Context, siteID, mac string, port int, config map[string]interface{}) error {
	_, err := c.doSiteRequest(ctx, siteID, http.MethodPatch, fmt.Sprintf("/switches/%s/ports/%d", mac, port), config)
	return err
}

// UpdateSwitchServiceConfig updates switch service settings via PUT /switches/{mac}/config/service.
// The ES series path is determined from the "es" field in the raw GET response,
// consistent with UpdateSwitchConfig.
func (c *Client) UpdateSwitchServiceConfig(ctx context.Context, siteID, mac string, isES bool, config *SwitchServiceConfig) error {
	var path string
	if isES {
		path = fmt.Sprintf("/switches/es/%s/config/service", mac)
	} else {
		path = fmt.Sprintf("/switches/%s/config/service", mac)
	}
	_, err := c.doSiteRequest(ctx, siteID, http.MethodPut, path, config)
	return err
}

// --- Firewall ACL Rules ---

// ACLDirection specifies which traffic directions an ACL applies to.
type ACLDirection struct {
	LanToWan bool     `json:"lanToWan"`
	LanToLan bool     `json:"lanToLan"`
	WanInIDs []string `json:"wanInIds,omitempty"`
	VpnInIDs []string `json:"vpnInIds,omitempty"`
}

// ACLRule represents a firewall ACL rule.
type ACLRule struct {
	ID              string       `json:"id,omitempty"`
	Name            string       `json:"name"`
	Type            int          `json:"type"`            // 0=gateway, 1=switch, 2=eap
	Index           int          `json:"index,omitempty"` // rule ordering (first-match-wins)
	Status          bool         `json:"status"`          // enabled/disabled
	Policy          int          `json:"policy"`          // 0=deny, 1=permit
	Protocols       []int        `json:"protocols"`       // 6=TCP, 17=UDP, 1=ICMP, etc.
	SourceType      int          `json:"sourceType"`      // 0=network, 2=ip_group
	SourceIDs       []string     `json:"sourceIds"`
	DestinationType int          `json:"destinationType"` // 0=network, 2=ip_group
	DestinationIDs  []string     `json:"destinationIds"`
	Direction       ACLDirection `json:"direction"`
	StateMode       int          `json:"stateMode,omitempty"` // 0=auto (stateful)
	BiDirectional   bool         `json:"biDirectional,omitempty"`
	IPSec           int          `json:"ipSec,omitempty"`
	Syslog          bool         `json:"syslog,omitempty"`
	Resource        int          `json:"resource,omitempty"`
}

// ACLListResult wraps the paginated ACL list response with metadata.
type ACLListResult struct {
	TotalRows          int       `json:"totalRows"`
	CurrentPage        int       `json:"currentPage"`
	CurrentSize        int       `json:"currentSize"`
	Data               []ACLRule `json:"data"`
	ACLDisable         bool      `json:"aclDisable"`
	SupportVPN         bool      `json:"supportVpn"`
	SupportLanToLan    bool      `json:"supportLanToLan"`
	SupportOsgMgtPage  bool      `json:"supportOsgMgtPage"`
	SupportIPv6        bool      `json:"supportIpv6"`
	SupportCountry     bool      `json:"supportCountry"`
	SupportWireless    bool      `json:"supportWireless"`
	SupportDomainGroup bool      `json:"supportDomainGroup"`
	SupportSyslog      bool      `json:"supportSyslog"`
	SupportNot         bool      `json:"supportNot"`
	Resource           int       `json:"resource"`
}

// ListACLRules returns all ACL rules of the given type for a site.
// aclType: 0=gateway, 1=switch, 2=eap
func (c *Client) ListACLRules(ctx context.Context, siteID string, aclType int) ([]ACLRule, error) {
	params := fmt.Sprintf("&type=%d&currentPage=1&currentPageSize=100", aclType)
	resp, err := c.doSiteRequestWithParams(ctx, siteID, http.MethodGet, "/setting/firewall/acls", params, nil)
	if err != nil {
		return nil, err
	}

	var listResult ACLListResult
	if err := json.Unmarshal(resp.Result, &listResult); err != nil {
		return nil, fmt.Errorf("decoding ACL list: %w", err)
	}
	return listResult.Data, nil
}

// GetACLRule returns a single ACL rule by ID (found by listing all rules of the type).
func (c *Client) GetACLRule(ctx context.Context, siteID, ruleID string, aclType int) (*ACLRule, error) {
	rules, err := c.ListACLRules(ctx, siteID, aclType)
	if err != nil {
		return nil, err
	}
	for _, r := range rules {
		if r.ID == ruleID {
			return &r, nil
		}
	}
	return nil, fmt.Errorf("ACL rule %q not found", ruleID)
}

// CreateACLRule creates a new ACL rule.
func (c *Client) CreateACLRule(ctx context.Context, siteID string, rule *ACLRule) (*ACLRule, error) {
	resp, err := c.doSiteRequest(ctx, siteID, http.MethodPost, "/setting/firewall/acls", rule)
	if err != nil {
		return nil, err
	}

	var created ACLRule
	if err := json.Unmarshal(resp.Result, &created); err != nil {
		return nil, fmt.Errorf("decoding created ACL rule: %w", err)
	}
	return &created, nil
}

// UpdateACLRule updates an existing ACL rule.
func (c *Client) UpdateACLRule(ctx context.Context, siteID, ruleID string, rule *ACLRule) (*ACLRule, error) {
	resp, err := c.doSiteRequest(ctx, siteID, http.MethodPatch, fmt.Sprintf("/setting/firewall/acls/%s", ruleID), rule)
	if err != nil {
		return nil, err
	}
	if isEmptyResult(resp.Result) {
		return c.GetACLRule(ctx, siteID, ruleID, rule.Type)
	}
	var updated ACLRule
	if err := json.Unmarshal(resp.Result, &updated); err != nil {
		return nil, fmt.Errorf("decoding updated ACL rule: %w", err)
	}
	return &updated, nil
}

// DeleteACLRule deletes an ACL rule.
func (c *Client) DeleteACLRule(ctx context.Context, siteID, ruleID string) error {
	_, err := c.doSiteRequest(ctx, siteID, http.MethodDelete, fmt.Sprintf("/setting/firewall/acls/%s", ruleID), nil)
	return err
}

// --- IP Groups ---

// IPGroupEntry represents a single IP + port combination within an IP group.
type IPGroupEntry struct {
	IP       string   `json:"ip"`
	PortList []string `json:"portList,omitempty"` // port numbers/ranges as strings (e.g., "80", "7000-7100")
}

// IPGroup represents an IP/Port group used in ACL rules.
type IPGroup struct {
	ID     string         `json:"id,omitempty"`
	Name   string         `json:"name"`
	Type   int            `json:"type"` // 1=IP/Port group
	IPList []IPGroupEntry `json:"ipList"`
}

// ListIPGroups returns all IP groups for a site.
// Note: requires a gateway device adopted into the site.
func (c *Client) ListIPGroups(ctx context.Context, siteID string) ([]IPGroup, error) {
	resp, err := c.doSiteRequestWithParams(ctx, siteID, http.MethodGet, "/setting/firewall/ipGroups", "&currentPage=1&currentPageSize=100", nil)
	if err != nil {
		return nil, err
	}
	var groups []IPGroup
	if err := decodePaginatedData(resp.Result, &groups); err != nil {
		return nil, fmt.Errorf("decoding IP groups: %w", err)
	}
	return groups, nil
}

// GetIPGroup returns a single IP group by ID.
func (c *Client) GetIPGroup(ctx context.Context, siteID, groupID string) (*IPGroup, error) {
	groups, err := c.ListIPGroups(ctx, siteID)
	if err != nil {
		return nil, err
	}
	for _, g := range groups {
		if g.ID == groupID {
			return &g, nil
		}
	}
	return nil, fmt.Errorf("IP group %q not found", groupID)
}

// CreateIPGroup creates a new IP group.
func (c *Client) CreateIPGroup(ctx context.Context, siteID string, group *IPGroup) (*IPGroup, error) {
	resp, err := c.doSiteRequest(ctx, siteID, http.MethodPost, "/setting/firewall/ipGroups", group)
	if err != nil {
		return nil, err
	}

	var created IPGroup
	if err := json.Unmarshal(resp.Result, &created); err != nil {
		return nil, fmt.Errorf("decoding created IP group: %w", err)
	}
	return &created, nil
}

// UpdateIPGroup updates an existing IP group.
func (c *Client) UpdateIPGroup(ctx context.Context, siteID, groupID string, group *IPGroup) (*IPGroup, error) {
	resp, err := c.doSiteRequest(ctx, siteID, http.MethodPatch, fmt.Sprintf("/setting/firewall/ipGroups/%s", groupID), group)
	if err != nil {
		return nil, err
	}
	if isEmptyResult(resp.Result) {
		return c.GetIPGroup(ctx, siteID, groupID)
	}
	var updated IPGroup
	if err := json.Unmarshal(resp.Result, &updated); err != nil {
		return nil, fmt.Errorf("decoding updated IP group: %w", err)
	}
	return &updated, nil
}

// DeleteIPGroup deletes an IP group.
func (c *Client) DeleteIPGroup(ctx context.Context, siteID, groupID string) error {
	_, err := c.doSiteRequest(ctx, siteID, http.MethodDelete, fmt.Sprintf("/setting/firewall/ipGroups/%s", groupID), nil)
	return err
}

// --- mDNS Reflector ---

// MDNSNetworkSetting holds the network references for an mDNS rule.
// For OSG (gateway) rules, the key is "osg"; for AP rules, the key is "ap".
type MDNSNetworkSetting struct {
	ProfileIDs      []string `json:"profileIds"`      // built-in service profiles (e.g. "buildIn-1" = AirPlay)
	ServiceNetworks []string `json:"serviceNetworks"` // network IDs where services are provided
	ClientNetworks  []string `json:"clientNetworks"`  // network IDs where clients discover services
}

// MDNSRule represents an mDNS reflector rule.
type MDNSRule struct {
	ID       string              `json:"id,omitempty"`
	Name     string              `json:"name"`
	Status   bool                `json:"status"`
	Type     int                 `json:"type"`          // 0=AP, 1=OSG (gateway)
	OSG      *MDNSNetworkSetting `json:"osg,omitempty"` // present when type=1
	AP       *MDNSNetworkSetting `json:"ap,omitempty"`  // present when type=0
	Resource int                 `json:"resource,omitempty"`
}

// MDNSListResult wraps the paginated mDNS list response with metadata.
type MDNSListResult struct {
	TotalRows    int        `json:"totalRows"`
	CurrentPage  int        `json:"currentPage"`
	CurrentSize  int        `json:"currentSize"`
	Data         []MDNSRule `json:"data"`
	APRuleNum    int        `json:"apRuleNum"`
	OSGRuleNum   int        `json:"osgRuleNum"`
	APRuleLimit  int        `json:"apRuleLimit"`
	OSGRuleLimit int        `json:"osgRuleLimit"`
}

// ListMDNSRules returns all mDNS reflector rules for a site.
func (c *Client) ListMDNSRules(ctx context.Context, siteID string) ([]MDNSRule, error) {
	resp, err := c.doSiteRequestWithParams(ctx, siteID, http.MethodGet, "/setting/service/mdns", "&currentPage=1&currentPageSize=100", nil)
	if err != nil {
		return nil, err
	}

	var listResult MDNSListResult
	if err := json.Unmarshal(resp.Result, &listResult); err != nil {
		return nil, fmt.Errorf("decoding mDNS list: %w", err)
	}
	return listResult.Data, nil
}

// GetMDNSRule returns a single mDNS rule by ID (found by listing all rules).
// The Omada 6.x API does not support GET by individual mDNS rule ID.
func (c *Client) GetMDNSRule(ctx context.Context, siteID, ruleID string) (*MDNSRule, error) {
	rules, err := c.ListMDNSRules(ctx, siteID)
	if err != nil {
		return nil, err
	}
	for _, r := range rules {
		if r.ID == ruleID {
			return &r, nil
		}
	}
	return nil, fmt.Errorf("mDNS rule %q not found", ruleID)
}

// CreateMDNSRule creates a new mDNS reflector rule.
// The API returns the created rule ID as a plain string, not a full rule object.
func (c *Client) CreateMDNSRule(ctx context.Context, siteID string, rule *MDNSRule) (*MDNSRule, error) {
	resp, err := c.doSiteRequest(ctx, siteID, http.MethodPost, "/setting/service/mdns", rule)
	if err != nil {
		return nil, err
	}

	// The API returns the rule ID as a quoted string, not a JSON object.
	var ruleID string
	if err := json.Unmarshal(resp.Result, &ruleID); err != nil {
		return nil, fmt.Errorf("decoding created mDNS rule ID: %w", err)
	}

	// Fetch the full rule by listing + filtering.
	return c.GetMDNSRule(ctx, siteID, ruleID)
}

// UpdateMDNSRule updates an existing mDNS reflector rule.
// The Omada 6.x API uses PUT (not PATCH) for mDNS rules.
func (c *Client) UpdateMDNSRule(ctx context.Context, siteID, ruleID string, rule *MDNSRule) (*MDNSRule, error) {
	_, err := c.doSiteRequest(ctx, siteID, http.MethodPut, fmt.Sprintf("/setting/service/mdns/%s", ruleID), rule)
	if err != nil {
		return nil, err
	}

	// PUT returns empty success; re-fetch via list.
	return c.GetMDNSRule(ctx, siteID, ruleID)
}

// DeleteMDNSRule deletes an mDNS reflector rule.
func (c *Client) DeleteMDNSRule(ctx context.Context, siteID, ruleID string) error {
	_, err := c.doSiteRequest(ctx, siteID, http.MethodDelete, fmt.Sprintf("/setting/service/mdns/%s", ruleID), nil)
	return err
}

// ====================================================================
// SAML Identity Provider (IdP) Connections
// ====================================================================

// SAMLIdP represents a SAML identity provider connection.
type SAMLIdP struct {
	IdpID       string `json:"idpId"`
	OmadacID    string `json:"omadacId,omitempty"`
	IdpName     string `json:"idpName"`
	Description string `json:"description,omitempty"`
	ConfMethod  int    `json:"confMethod"`
	EntityID    string `json:"entityId"`
	LoginURL    string `json:"loginUrl"`
	X509Cert    string `json:"x509Certificate"`
	EntityURL   string `json:"entityUrl,omitempty"`
	SignOnURL   string `json:"signOnUrl,omitempty"`
}

// SAMLIdPCreateRequest is the payload for creating/updating a SAML IdP.
type SAMLIdPCreateRequest struct {
	IdpName     string `json:"idpName"`
	Description string `json:"description,omitempty"`
	ConfMethod  int    `json:"confMethod"`
	EntityID    string `json:"entityId"`
	LoginURL    string `json:"loginUrl"`
	X509Cert    string `json:"x509Certificate"`
}

// ListSAMLIdPs returns all SAML identity provider connections.
func (c *Client) ListSAMLIdPs(ctx context.Context) ([]SAMLIdP, error) {
	url := c.globalURL("/idps") + "&currentPage=1&currentPageSize=100"
	resp, err := c.doRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	var idps []SAMLIdP
	if err := decodePaginatedData(resp.Result, &idps); err != nil {
		return nil, fmt.Errorf("decoding SAML IdPs: %w", err)
	}
	return idps, nil
}

// GetSAMLIdP returns a single SAML IdP by ID.
// The GET-by-ID endpoint is not supported (-1600), so we list all and filter.
func (c *Client) GetSAMLIdP(ctx context.Context, idpID string) (*SAMLIdP, error) {
	idps, err := c.ListSAMLIdPs(ctx)
	if err != nil {
		return nil, err
	}
	for _, idp := range idps {
		if idp.IdpID == idpID {
			return &idp, nil
		}
	}
	return nil, fmt.Errorf("SAML IdP %q not found", idpID)
}

// CreateSAMLIdP creates a new SAML identity provider connection.
// confMethod is always set to 2 (Manual).
// Returns the created IdP (fetched via list+filter since POST returns minimal data).
func (c *Client) CreateSAMLIdP(ctx context.Context, req *SAMLIdPCreateRequest) (*SAMLIdP, error) {
	req.ConfMethod = 2
	url := c.globalURL("/idps")
	_, err := c.doRequest(ctx, http.MethodPost, url, req)
	if err != nil {
		return nil, err
	}

	// POST doesn't return the new ID reliably; list all and match by name.
	idps, err := c.ListSAMLIdPs(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing SAML IdPs after create: %w", err)
	}
	for _, idp := range idps {
		if idp.IdpName == req.IdpName {
			return &idp, nil
		}
	}
	return nil, fmt.Errorf("SAML IdP %q not found after creation", req.IdpName)
}

// UpdateSAMLIdP updates an existing SAML IdP via PUT (full replace).
func (c *Client) UpdateSAMLIdP(ctx context.Context, idpID string, req *SAMLIdPCreateRequest) (*SAMLIdP, error) {
	req.ConfMethod = 2
	url := c.globalURL(fmt.Sprintf("/idps/%s", idpID))
	_, err := c.doRequest(ctx, http.MethodPut, url, req)
	if err != nil {
		return nil, err
	}

	// Re-fetch via list+filter.
	return c.GetSAMLIdP(ctx, idpID)
}

// DeleteSAMLIdP deletes a SAML identity provider connection.
func (c *Client) DeleteSAMLIdP(ctx context.Context, idpID string) error {
	url := c.globalURL(fmt.Sprintf("/idps/%s", idpID))
	_, err := c.doRequest(ctx, http.MethodDelete, url, nil)
	return err
}

// ====================================================================
// SAML Roles (External User Groups)
// ====================================================================

// SAMLRoleSite represents a site reference in a SAML role.
type SAMLRoleSite struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

// SAMLRoleSitePrivilege represents the site privilege configuration.
type SAMLRoleSitePrivilege struct {
	SiteType    int            `json:"siteType"`
	Sites       []SAMLRoleSite `json:"sites"`
	ServiceType int            `json:"serviceType"`
}

// SAMLRoleSitePrivilegeCreate is used in create/update requests where sites
// are specified as plain string IDs.
type SAMLRoleSitePrivilegeCreate struct {
	SiteType    int      `json:"siteType"`
	Sites       []string `json:"sites"`
	ServiceType int      `json:"serviceType"`
}

// SAMLRole represents a SAML external user group (role mapping).
type SAMLRole struct {
	ID              string                  `json:"id"`
	OmadacID        string                  `json:"omadacId,omitempty"`
	UserGroupName   string                  `json:"userGroupName"`
	RoleID          string                  `json:"roleId"`
	RoleName        string                  `json:"roleName,omitempty"`
	RoleType        int                     `json:"roleType,omitempty"`
	SitePrivileges  []SAMLRoleSitePrivilege `json:"sitePrivileges,omitempty"`
	TemporaryEnable bool                    `json:"temporaryEnable"`
	StartTime       int64                   `json:"startTime"`
	EndTime         int64                   `json:"endTime"`
}

// SAMLRoleCreateRequest is the payload for creating/updating a SAML role.
type SAMLRoleCreateRequest struct {
	UserGroupName   string                        `json:"userGroupName"`
	RoleID          string                        `json:"roleId"`
	TemporaryEnable bool                          `json:"temporaryEnable"`
	StartTime       int64                         `json:"startTime"`
	EndTime         int64                         `json:"endTime"`
	SitePrivileges  []SAMLRoleSitePrivilegeCreate `json:"sitePrivileges"`
}

// ListSAMLRoles returns all SAML external user groups.
func (c *Client) ListSAMLRoles(ctx context.Context) ([]SAMLRole, error) {
	url := c.globalURL("/extendUserGroups") + "&currentPage=1&currentPageSize=100"
	resp, err := c.doRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	var roles []SAMLRole
	if err := decodePaginatedData(resp.Result, &roles); err != nil {
		return nil, fmt.Errorf("decoding SAML roles: %w", err)
	}
	return roles, nil
}

// GetSAMLRole returns a single SAML role by ID.
func (c *Client) GetSAMLRole(ctx context.Context, roleID string) (*SAMLRole, error) {
	url := c.globalURL(fmt.Sprintf("/extendUserGroups/%s", roleID))
	resp, err := c.doRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	var role SAMLRole
	if err := json.Unmarshal(resp.Result, &role); err != nil {
		return nil, fmt.Errorf("decoding SAML role: %w", err)
	}
	return &role, nil
}

// CreateSAMLRole creates a new SAML external user group.
func (c *Client) CreateSAMLRole(ctx context.Context, req *SAMLRoleCreateRequest) (*SAMLRole, error) {
	url := c.globalURL("/extendUserGroups")
	resp, err := c.doRequest(ctx, http.MethodPost, url, req)
	if err != nil {
		return nil, err
	}

	// POST returns the new ID.
	var roleID string
	if err := json.Unmarshal(resp.Result, &roleID); err != nil {
		// Some endpoints return the object directly; try fetching by listing.
		roles, listErr := c.ListSAMLRoles(ctx)
		if listErr != nil {
			return nil, fmt.Errorf("decoding created SAML role ID: %w", err)
		}
		for _, r := range roles {
			if r.UserGroupName == req.UserGroupName {
				return &r, nil
			}
		}
		return nil, fmt.Errorf("SAML role %q not found after creation", req.UserGroupName)
	}

	return c.GetSAMLRole(ctx, roleID)
}

// UpdateSAMLRole updates an existing SAML role via PUT (full replace).
func (c *Client) UpdateSAMLRole(ctx context.Context, roleID string, req *SAMLRoleCreateRequest) (*SAMLRole, error) {
	url := c.globalURL(fmt.Sprintf("/extendUserGroups/%s", roleID))
	_, err := c.doRequest(ctx, http.MethodPut, url, req)
	if err != nil {
		return nil, err
	}

	return c.GetSAMLRole(ctx, roleID)
}

// DeleteSAMLRole deletes a SAML external user group.
func (c *Client) DeleteSAMLRole(ctx context.Context, roleID string) error {
	url := c.globalURL(fmt.Sprintf("/extendUserGroups/%s", roleID))
	_, err := c.doRequest(ctx, http.MethodDelete, url, nil)
	return err
}
