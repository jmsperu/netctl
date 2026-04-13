package unifi

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"time"
)

// Client talks to UniFi Network Controller API.
type Client struct {
	Host     string
	Port     int
	Username string
	Password string
	Site     string
	Insecure bool
	http     *http.Client
	baseURL  string
}

func NewClient(host string, port int, username, password, site string, insecure bool) *Client {
	if port == 0 {
		port = 443
	}
	if site == "" {
		site = "default"
	}
	jar, _ := cookiejar.New(nil)
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure},
	}
	return &Client{
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
		Site:     site,
		Insecure: insecure,
		baseURL:  fmt.Sprintf("https://%s:%d", host, port),
		http:     &http.Client{Transport: transport, Timeout: 30 * time.Second, Jar: jar},
	}
}

type apiResponse struct {
	Meta struct {
		RC  string `json:"rc"`
		Msg string `json:"msg"`
	} `json:"meta"`
	Data json.RawMessage `json:"data"`
}

// Login authenticates with the controller.
func (c *Client) Login() error {
	payload, _ := json.Marshal(map[string]string{
		"username": c.Username,
		"password": c.Password,
	})

	resp, err := c.http.Post(c.baseURL+"/api/login", "application/json", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("UniFi login failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("UniFi login error %d: %s", resp.StatusCode, string(body[:min(len(body), 200)]))
	}

	return nil
}

// Logout ends the session.
func (c *Client) Logout() {
	c.http.Post(c.baseURL+"/api/logout", "application/json", nil)
}

// request makes an authenticated API call.
func (c *Client) request(method, path string, body interface{}) (json.RawMessage, error) {
	var bodyReader io.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(data)
	}

	url := fmt.Sprintf("%s/api/s/%s%s", c.baseURL, c.Site, path)
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var apiResp apiResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("invalid UniFi response")
	}

	if apiResp.Meta.RC != "ok" {
		return nil, fmt.Errorf("UniFi error: %s", apiResp.Meta.Msg)
	}

	return apiResp.Data, nil
}

// ── Data types ──

type Device struct {
	ID          string `json:"_id"`
	MAC         string `json:"mac"`
	Model       string `json:"model"`
	Name        string `json:"name"`
	Type        string `json:"type"` // uap, usw, ugw
	State       int    `json:"state"` // 1=connected, 0=disconnected
	Adopted     bool   `json:"adopted"`
	IP          string `json:"ip"`
	Version     string `json:"version"`
	Uptime      int64  `json:"uptime"`
	NumSTA      int    `json:"num_sta"` // connected clients
	TxBytes     int64  `json:"tx_bytes"`
	RxBytes     int64  `json:"rx_bytes"`
	Satisfaction int   `json:"satisfaction"` // WiFi score 0-100
}

type Client_ struct {
	MAC        string `json:"mac"`
	Hostname   string `json:"hostname"`
	IP         string `json:"ip"`
	ESSID      string `json:"essid"`
	APName     string `json:"ap_name"`
	RSSI       int    `json:"rssi"`
	Signal     int    `json:"signal"`
	Channel    int    `json:"channel"`
	TxRate     int    `json:"tx_rate"`
	RxRate     int    `json:"rx_rate"`
	TxBytes    int64  `json:"tx_bytes"`
	RxBytes    int64  `json:"rx_bytes"`
	Uptime     int64  `json:"uptime"`
	IsWired    bool   `json:"is_wired"`
}

type WLAN struct {
	ID       string `json:"_id"`
	Name     string `json:"name"`
	SSID     string `json:"x_passphrase,omitempty"` // only visible if admin
	Enabled  bool   `json:"enabled"`
	Security string `json:"security"`
	NumSTA   int    `json:"num_sta"`
}

type Network struct {
	ID           string `json:"_id"`
	Name         string `json:"name"`
	Purpose      string `json:"purpose"`
	Subnet       string `json:"ip_subnet"`
	VLAN         int    `json:"vlan"`
	DHCPEnabled  bool   `json:"dhcpd_enabled"`
	DHCPStart    string `json:"dhcpd_start"`
	DHCPStop     string `json:"dhcpd_stop"`
	DomainName   string `json:"domain_name"`
}

type SiteHealth struct {
	Subsystem  string  `json:"subsystem"` // wan, lan, wlan, vpn
	Status     string  `json:"status"`    // ok, warning, error
	NumUser    int     `json:"num_user"`
	NumGuest   int     `json:"num_guest"`
	NumAP      int     `json:"num_ap"`
	NumAdopted int     `json:"num_adopted"`
	TxBytes    float64 `json:"tx_bytes-r"`
	RxBytes    float64 `json:"rx_bytes-r"`
	Latency    int     `json:"latency"`
	Uptime     int64   `json:"uptime"`
	WANName    string  `json:"gw_name"`
	WANIP      string  `json:"wan_ip"`
	ISP        string  `json:"isp_name"`
}

// ── API Methods ──

func (c *Client) GetDevices() ([]Device, error) {
	data, err := c.request("GET", "/stat/device", nil)
	if err != nil {
		return nil, err
	}
	var devices []Device
	json.Unmarshal(data, &devices)
	return devices, nil
}

func (c *Client) GetClients() ([]Client_, error) {
	data, err := c.request("GET", "/stat/sta", nil)
	if err != nil {
		return nil, err
	}
	var clients []Client_
	json.Unmarshal(data, &clients)
	return clients, nil
}

func (c *Client) GetWLANs() ([]WLAN, error) {
	data, err := c.request("GET", "/rest/wlanconf", nil)
	if err != nil {
		return nil, err
	}
	var wlans []WLAN
	json.Unmarshal(data, &wlans)
	return wlans, nil
}

func (c *Client) GetNetworks() ([]Network, error) {
	data, err := c.request("GET", "/rest/networkconf", nil)
	if err != nil {
		return nil, err
	}
	var nets []Network
	json.Unmarshal(data, &nets)
	return nets, nil
}

func (c *Client) GetSiteHealth() ([]SiteHealth, error) {
	data, err := c.request("GET", "/stat/health", nil)
	if err != nil {
		return nil, err
	}
	var health []SiteHealth
	json.Unmarshal(data, &health)
	return health, nil
}

// Device actions
func (c *Client) RestartDevice(mac string) error {
	_, err := c.request("POST", "/cmd/devmgr", map[string]string{
		"cmd": "restart",
		"mac": mac,
	})
	return err
}

func (c *Client) AdoptDevice(mac string) error {
	_, err := c.request("POST", "/cmd/devmgr", map[string]string{
		"cmd": "adopt",
		"mac": mac,
	})
	return err
}

func (c *Client) ForgetDevice(mac string) error {
	_, err := c.request("POST", "/cmd/devmgr", map[string]string{
		"cmd": "forget",
		"mac": mac,
	})
	return err
}

func (c *Client) TestConnection() error {
	if err := c.Login(); err != nil {
		return err
	}
	defer c.Logout()
	_, err := c.GetSiteHealth()
	return err
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
