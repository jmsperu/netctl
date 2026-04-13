package vyos

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Client talks to VyOS via its HTTP API (available in VyOS 1.3+).
type Client struct {
	Host     string
	Port     int
	APIKey   string
	Insecure bool
	http     *http.Client
}

func NewClient(host string, port int, apiKey string, insecure bool) *Client {
	if port == 0 {
		port = 443
	}
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure},
	}
	return &Client{
		Host:     host,
		Port:     port,
		APIKey:   apiKey,
		Insecure: insecure,
		http:     &http.Client{Transport: transport, Timeout: 30 * time.Second},
	}
}

type apiResponse struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Error   string          `json:"error"`
}

// Request makes a VyOS API call.
func (c *Client) Request(endpoint string, data map[string]interface{}) (json.RawMessage, error) {
	apiURL := fmt.Sprintf("https://%s:%d%s", c.Host, c.Port, endpoint)

	payload := map[string]interface{}{
		"key":  c.APIKey,
		"data": data,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Post(apiURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("VyOS API request failed: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var apiResp apiResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("invalid VyOS response: %s", string(respBody[:min(len(respBody), 200)]))
	}

	if !apiResp.Success {
		return nil, fmt.Errorf("VyOS API error: %s", apiResp.Error)
	}

	return apiResp.Data, nil
}

// Retrieve gets a config path.
func (c *Client) Retrieve(path []string) (json.RawMessage, error) {
	return c.Request("/retrieve", map[string]interface{}{
		"op":   "showConfig",
		"path": path,
	})
}

// Show runs an operational command.
func (c *Client) Show(cmd string) (string, error) {
	apiURL := fmt.Sprintf("https://%s:%d/show", c.Host, c.Port)

	form := url.Values{}
	form.Set("key", c.APIKey)
	form.Set("data", fmt.Sprintf(`{"op": "show", "path": ["%s"]}`, cmd))

	resp, err := c.http.PostForm(apiURL, form)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var apiResp apiResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return string(body), nil
	}

	if !apiResp.Success {
		return "", fmt.Errorf("VyOS error: %s", apiResp.Error)
	}

	var output string
	json.Unmarshal(apiResp.Data, &output)
	return output, nil
}

// Configure sets a config path.
func (c *Client) Configure(op string, path []string) error {
	_, err := c.Request("/configure", map[string]interface{}{
		"op":   op, // "set", "delete"
		"path": path,
	})
	return err
}

// ConfigSet sets a config value.
func (c *Client) ConfigSet(path []string) error {
	return c.Configure("set", path)
}

// ConfigDelete deletes a config path.
func (c *Client) ConfigDelete(path []string) error {
	return c.Configure("delete", path)
}

// SaveConfig saves the running config.
func (c *Client) SaveConfig() error {
	_, err := c.Request("/config-file", map[string]interface{}{
		"op": "save",
	})
	return err
}

// GenerateConfig gets the full config (for backup).
func (c *Client) GenerateConfig() (string, error) {
	result, err := c.Request("/generate", map[string]interface{}{
		"op":   "generate",
		"path": []string{"tech-support", "config"},
	})
	if err != nil {
		// Fallback: show full config
		return c.Show("configuration")
	}
	var output string
	json.Unmarshal(result, &output)
	return output, nil
}

// ── Convenience methods ──

func (c *Client) GetInterfaces() (string, error) {
	return c.Show("interfaces")
}

func (c *Client) GetRoutes() (string, error) {
	return c.Show("ip route")
}

func (c *Client) GetBGPSummary() (string, error) {
	return c.Show("ip bgp summary")
}

func (c *Client) GetFirewallRules() (string, error) {
	return c.Show("firewall")
}

func (c *Client) GetVPNStatus() (string, error) {
	return c.Show("vpn ipsec sa")
}

func (c *Client) GetWireGuardStatus() (string, error) {
	return c.Show("interfaces wireguard")
}

func (c *Client) GetDHCPLeases() (string, error) {
	return c.Show("dhcp server leases")
}

func (c *Client) GetNATRules() (string, error) {
	return c.Show("nat source rules")
}

func (c *Client) GetSystemInfo() (string, error) {
	return c.Show("version")
}

func (c *Client) TestConnection() error {
	_, err := c.GetSystemInfo()
	return err
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
