package mikrotik

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// Client talks to MikroTik RouterOS via the REST API (v7+) or API port (6.x+).
type Client struct {
	Host     string
	Port     int
	Username string
	Password string
	UseREST  bool // true = REST API (v7+), false = RouterOS API protocol
	Insecure bool
	http     *http.Client
}

func NewClient(host string, port int, username, password string, useREST, insecure bool) *Client {
	if port == 0 {
		if useREST {
			port = 443
		} else {
			port = 8728 // RouterOS API port
		}
	}
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure},
	}
	return &Client{
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
		UseREST:  useREST,
		Insecure: insecure,
		http:     &http.Client{Transport: transport, Timeout: 30 * time.Second},
	}
}

// ── REST API (RouterOS v7+) ──

func (c *Client) restURL(path string) string {
	return fmt.Sprintf("https://%s:%d/rest%s", c.Host, c.Port, path)
}

func (c *Client) RESTGet(path string) (json.RawMessage, error) {
	req, err := http.NewRequest("GET", c.restURL(path), nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.Username, c.Password)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("MikroTik REST request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("MikroTik REST error %d: %s", resp.StatusCode, string(body[:min(len(body), 500)]))
	}

	return json.RawMessage(body), nil
}

// ── Data types ──

type Interface struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Running   bool   `json:"running"`
	Disabled  bool   `json:"disabled"`
	MacAddr   string `json:"mac-address"`
	MTU       int    `json:"actual-mtu"`
	TxBytes   int64  `json:"tx-byte"`
	RxBytes   int64  `json:"rx-byte"`
	Comment   string `json:"comment"`
}

type IPAddress struct {
	Address   string `json:"address"`
	Network   string `json:"network"`
	Interface string `json:"interface"`
	Disabled  bool   `json:"disabled"`
}

type Route struct {
	DstAddress string `json:"dst-address"`
	Gateway    string `json:"gateway"`
	Distance   int    `json:"distance"`
	Active     bool   `json:"active"`
	Dynamic    bool   `json:"dynamic"`
	Disabled   bool   `json:"disabled"`
}

type FirewallFilter struct {
	Chain    string `json:"chain"`
	Action   string `json:"action"`
	SrcAddr  string `json:"src-address"`
	DstAddr  string `json:"dst-address"`
	Protocol string `json:"protocol"`
	DstPort  string `json:"dst-port"`
	Disabled bool   `json:"disabled"`
	Comment  string `json:"comment"`
	Bytes    int64  `json:"bytes"`
	Packets  int64  `json:"packets"`
}

type DHCPLease struct {
	Address    string `json:"address"`
	MacAddr    string `json:"mac-address"`
	HostName   string `json:"host-name"`
	Status     string `json:"status"`
	Server     string `json:"server"`
	ExpiresAt  string `json:"expires-after"`
}

type Identity struct {
	Name string `json:"name"`
}

type Resource struct {
	Uptime           string `json:"uptime"`
	Version          string `json:"version"`
	BoardName        string `json:"board-name"`
	Architecture     string `json:"architecture-name"`
	CPUCount         int    `json:"cpu-count"`
	CPULoad          int    `json:"cpu-load"`
	FreeMemory       int64  `json:"free-memory"`
	TotalMemory      int64  `json:"total-memory"`
	FreeHDD          int64  `json:"free-hdd-space"`
	TotalHDD         int64  `json:"total-hdd-space"`
}

// ── REST API Methods ──

func (c *Client) GetIdentity() (*Identity, error) {
	data, err := c.RESTGet("/system/identity")
	if err != nil {
		return nil, err
	}
	var id Identity
	json.Unmarshal(data, &id)
	return &id, nil
}

func (c *Client) GetResource() (*Resource, error) {
	data, err := c.RESTGet("/system/resource")
	if err != nil {
		return nil, err
	}
	var res Resource
	json.Unmarshal(data, &res)
	return &res, nil
}

func (c *Client) GetInterfaces() ([]Interface, error) {
	data, err := c.RESTGet("/interface")
	if err != nil {
		return nil, err
	}
	var ifaces []Interface
	json.Unmarshal(data, &ifaces)
	return ifaces, nil
}

func (c *Client) GetIPAddresses() ([]IPAddress, error) {
	data, err := c.RESTGet("/ip/address")
	if err != nil {
		return nil, err
	}
	var addrs []IPAddress
	json.Unmarshal(data, &addrs)
	return addrs, nil
}

func (c *Client) GetRoutes() ([]Route, error) {
	data, err := c.RESTGet("/ip/route")
	if err != nil {
		return nil, err
	}
	var routes []Route
	json.Unmarshal(data, &routes)
	return routes, nil
}

func (c *Client) GetFirewallFilter() ([]FirewallFilter, error) {
	data, err := c.RESTGet("/ip/firewall/filter")
	if err != nil {
		return nil, err
	}
	var rules []FirewallFilter
	json.Unmarshal(data, &rules)
	return rules, nil
}

func (c *Client) GetFirewallNAT() ([]FirewallFilter, error) {
	data, err := c.RESTGet("/ip/firewall/nat")
	if err != nil {
		return nil, err
	}
	var rules []FirewallFilter
	json.Unmarshal(data, &rules)
	return rules, nil
}

func (c *Client) GetDHCPLeases() ([]DHCPLease, error) {
	data, err := c.RESTGet("/ip/dhcp-server/lease")
	if err != nil {
		return nil, err
	}
	var leases []DHCPLease
	json.Unmarshal(data, &leases)
	return leases, nil
}

func (c *Client) TestConnection() error {
	_, err := c.GetIdentity()
	return err
}

// ── RouterOS API Protocol (fallback for v6.x) ──

func (c *Client) apiConnect() (net.Conn, error) {
	addr := fmt.Sprintf("%s:%d", c.Host, c.Port)
	var conn net.Conn
	var err error

	if c.Port == 8729 || c.Insecure {
		conn, err = tls.DialWithDialer(&net.Dialer{Timeout: 10 * time.Second}, "tcp", addr, &tls.Config{InsecureSkipVerify: true})
	} else {
		conn, err = net.DialTimeout("tcp", addr, 10*time.Second)
	}
	return conn, err
}

func (c *Client) APICommand(cmd string) ([]map[string]string, error) {
	conn, err := c.apiConnect()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// Login
	writeWord(conn, "/login")
	writeWord(conn, "=name="+c.Username)
	writeWord(conn, "=password="+c.Password)
	writeWord(conn, "")

	// Read login response
	if err := readResponse(conn); err != nil {
		return nil, fmt.Errorf("login failed: %v", err)
	}

	// Send command
	parts := strings.Split(cmd, " ")
	for _, p := range parts {
		writeWord(conn, p)
	}
	writeWord(conn, "")

	// Read response
	return readRecords(conn)
}

func writeWord(conn net.Conn, word string) {
	length := len(word)
	switch {
	case length < 0x80:
		conn.Write([]byte{byte(length)})
	case length < 0x4000:
		conn.Write([]byte{byte(length>>8 | 0x80), byte(length)})
	case length < 0x200000:
		conn.Write([]byte{byte(length>>16 | 0xC0), byte(length >> 8), byte(length)})
	case length < 0x10000000:
		conn.Write([]byte{byte(length>>24 | 0xE0), byte(length >> 16), byte(length >> 8), byte(length)})
	}
	conn.Write([]byte(word))
}

func readResponse(conn net.Conn) error {
	reader := bufio.NewReader(conn)
	for {
		word, err := readWord(reader)
		if err != nil {
			return err
		}
		if word == "" {
			break
		}
		if strings.HasPrefix(word, "!trap") {
			return fmt.Errorf("RouterOS error: %s", word)
		}
	}
	return nil
}

func readRecords(conn net.Conn) ([]map[string]string, error) {
	reader := bufio.NewReader(conn)
	var records []map[string]string
	current := map[string]string{}

	for {
		word, err := readWord(reader)
		if err != nil {
			return records, nil
		}
		if word == "" {
			if len(current) > 0 {
				records = append(records, current)
				current = map[string]string{}
			}
			continue
		}
		if word == "!done" {
			break
		}
		if word == "!re" {
			if len(current) > 0 {
				records = append(records, current)
			}
			current = map[string]string{}
			continue
		}
		if strings.HasPrefix(word, "=") {
			parts := strings.SplitN(word[1:], "=", 2)
			if len(parts) == 2 {
				current[parts[0]] = parts[1]
			}
		}
	}
	if len(current) > 0 {
		records = append(records, current)
	}
	return records, nil
}

func readWord(reader *bufio.Reader) (string, error) {
	b, err := reader.ReadByte()
	if err != nil {
		return "", err
	}

	var length int
	switch {
	case b < 0x80:
		length = int(b)
	case b < 0xC0:
		b2, _ := reader.ReadByte()
		length = int(b&0x3F)<<8 | int(b2)
	case b < 0xE0:
		b2, _ := reader.ReadByte()
		b3, _ := reader.ReadByte()
		length = int(b&0x1F)<<16 | int(b2)<<8 | int(b3)
	case b < 0xF0:
		b2, _ := reader.ReadByte()
		b3, _ := reader.ReadByte()
		b4, _ := reader.ReadByte()
		length = int(b&0x0F)<<24 | int(b2)<<16 | int(b3)<<8 | int(b4)
	}

	if length == 0 {
		return "", nil
	}

	word := make([]byte, length)
	_, err = io.ReadFull(reader, word)
	return string(word), err
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
