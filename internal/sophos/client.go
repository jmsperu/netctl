package sophos

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client talks to Sophos XG/XGS Firewall API.
// The XG API uses XML by default but v18+ supports JSON on port 4444.
type Client struct {
	Host     string
	Port     int
	Username string
	Password string
	Insecure bool
	http     *http.Client
}

func NewClient(host string, port int, username, password string, insecure bool) *Client {
	if port == 0 {
		port = 4444
	}
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure},
	}
	return &Client{
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
		Insecure: insecure,
		http:     &http.Client{Transport: transport, Timeout: 30 * time.Second},
	}
}

// apiURL builds the Sophos XG API URL.
func (c *Client) apiURL(entity string) string {
	return fmt.Sprintf("https://%s:%d/webconsole/APIController?reqxml=", c.Host, c.Port)
}

// Request makes an XML API call and returns the response body.
func (c *Client) Request(xmlBody string) (string, error) {
	reqURL := fmt.Sprintf("https://%s:%d/webconsole/APIController", c.Host, c.Port)

	form := url.Values{}
	form.Set("reqxml", xmlBody)

	req, err := http.NewRequest("POST", reqURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("Sophos API request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// authXML wraps a request with authentication.
func (c *Client) authXML(operation string) string {
	return fmt.Sprintf(`<Request>
<Login>
<Username>%s</Username>
<Password>%s</Password>
</Login>
%s
</Request>`, c.Username, c.Password, operation)
}

// ── Data types ──

type FirewallRule struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"` // enable/disable
	Action      string `json:"action"` // accept/drop/reject
	SourceZone  string `json:"source_zone"`
	DestZone    string `json:"dest_zone"`
	Source      string `json:"source"`
	Dest        string `json:"dest"`
	Service     string `json:"service"`
	Position    int    `json:"position"`
}

type Interface struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	IPAddress string `json:"ip_address"`
	Netmask   string `json:"netmask"`
	Zone      string `json:"zone"`
	Speed     string `json:"speed"`
	MTU       int    `json:"mtu"`
	Hardware  string `json:"hardware"`
}

type Route struct {
	Destination string `json:"destination"`
	Netmask     string `json:"netmask"`
	Gateway     string `json:"gateway"`
	Interface   string `json:"interface"`
	Metric      int    `json:"metric"`
}

type VPNTunnel struct {
	Name       string `json:"name"`
	Type       string `json:"type"` // ipsec, ssl
	Status     string `json:"status"`
	LocalNet   string `json:"local_network"`
	RemoteNet  string `json:"remote_network"`
	RemoteHost string `json:"remote_host"`
}

type SystemInfo struct {
	Hostname    string `json:"hostname"`
	Model       string `json:"model"`
	Firmware    string `json:"firmware"`
	Serial      string `json:"serial"`
	Uptime      string `json:"uptime"`
	CPUUsage    string `json:"cpu_usage"`
	MemoryUsage string `json:"memory_usage"`
	DiskUsage   string `json:"disk_usage"`
}

type HAStatus struct {
	Mode    string `json:"mode"` // standalone, active-active, active-passive
	State   string `json:"state"`
	PeerIP  string `json:"peer_ip"`
	PeerState string `json:"peer_state"`
}

// ── API Methods ──

func (c *Client) GetSystemInfo() (*SystemInfo, error) {
	xml := c.authXML(`<Get><SystemInfo/></Get>`)
	resp, err := c.Request(xml)
	if err != nil {
		return nil, err
	}
	// Parse XML response — Sophos returns XML
	info := parseSystemInfo(resp)
	return info, nil
}

func (c *Client) GetInterfaces() ([]Interface, error) {
	xml := c.authXML(`<Get><Interface/></Get>`)
	resp, err := c.Request(xml)
	if err != nil {
		return nil, err
	}
	return parseInterfaces(resp), nil
}

func (c *Client) GetFirewallRules() ([]FirewallRule, error) {
	xml := c.authXML(`<Get><FirewallRule/></Get>`)
	resp, err := c.Request(xml)
	if err != nil {
		return nil, err
	}
	return parseFirewallRules(resp), nil
}

func (c *Client) GetRoutes() ([]Route, error) {
	xml := c.authXML(`<Get><Route/></Get>`)
	resp, err := c.Request(xml)
	if err != nil {
		return nil, err
	}
	return parseRoutes(resp), nil
}

func (c *Client) GetVPNTunnels() ([]VPNTunnel, error) {
	xml := c.authXML(`<Get><IPSecConnection/></Get>`)
	resp, err := c.Request(xml)
	if err != nil {
		return nil, err
	}
	return parseVPNTunnels(resp), nil
}

func (c *Client) GetHAStatus() (*HAStatus, error) {
	xml := c.authXML(`<Get><HAStatus/></Get>`)
	resp, err := c.Request(xml)
	if err != nil {
		return nil, err
	}
	return parseHAStatus(resp), nil
}

func (c *Client) BackupConfig() (string, error) {
	xml := c.authXML(`<Get><BackupRestore/></Get>`)
	return c.Request(xml)
}

// TestConnection verifies API access.
func (c *Client) TestConnection() error {
	_, err := c.GetSystemInfo()
	return err
}

// ── Parsers (simplified XML parsing) ──
// Sophos XG API returns XML. We extract values with string matching
// since Go's encoding/xml needs struct tags for the specific schema.

func parseSystemInfo(xmlResp string) *SystemInfo {
	return &SystemInfo{
		Hostname: extractXMLValue(xmlResp, "HostName"),
		Model:    extractXMLValue(xmlResp, "Model"),
		Firmware: extractXMLValue(xmlResp, "FirmwareVersion"),
		Serial:   extractXMLValue(xmlResp, "SerialNumber"),
		Uptime:   extractXMLValue(xmlResp, "UpTime"),
	}
}

func parseInterfaces(xmlResp string) []Interface {
	// Return raw for now — full XML parsing can be added
	_ = xmlResp
	return nil
}

func parseFirewallRules(xmlResp string) []FirewallRule {
	_ = xmlResp
	return nil
}

func parseRoutes(xmlResp string) []Route {
	_ = xmlResp
	return nil
}

func parseVPNTunnels(xmlResp string) []VPNTunnel {
	_ = xmlResp
	return nil
}

func parseHAStatus(xmlResp string) *HAStatus {
	return &HAStatus{
		Mode:  extractXMLValue(xmlResp, "Mode"),
		State: extractXMLValue(xmlResp, "State"),
	}
}

func extractXMLValue(xml, tag string) string {
	start := fmt.Sprintf("<%s>", tag)
	end := fmt.Sprintf("</%s>", tag)
	i := strings.Index(xml, start)
	if i == -1 {
		return ""
	}
	i += len(start)
	j := strings.Index(xml[i:], end)
	if j == -1 {
		return ""
	}
	return xml[i : i+j]
}

// Ensure json is imported (used by struct tags)
var _ = json.Marshal
