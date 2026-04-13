package generic

import (
	"fmt"
	"time"

	"golang.org/x/crypto/ssh"
)

// SSHClient provides generic SSH access to any network device.
type SSHClient struct {
	Host     string
	Port     int
	Username string
	Password string
	client   *ssh.Client
}

func NewSSHClient(host string, port int, username, password string) *SSHClient {
	if port == 0 {
		port = 22
	}
	return &SSHClient{
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
	}
}

func (c *SSHClient) Connect() error {
	config := &ssh.ClientConfig{
		User: c.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(c.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", c.Host, c.Port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return fmt.Errorf("SSH connection failed: %v", err)
	}
	c.client = client
	return nil
}

func (c *SSHClient) Close() {
	if c.client != nil {
		c.client.Close()
	}
}

// Exec runs a command and returns the output.
func (c *SSHClient) Exec(cmd string) (string, error) {
	if c.client == nil {
		if err := c.Connect(); err != nil {
			return "", err
		}
	}

	session, err := c.client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	output, err := session.CombinedOutput(cmd)
	return string(output), err
}

// GetConfig retrieves the running config (tries common commands).
func (c *SSHClient) GetConfig() (string, error) {
	// Try common config commands in order
	commands := []string{
		"show running-config",  // Cisco IOS, Arista
		"show configuration",   // Juniper, VyOS
		"display current-configuration", // Huawei
		"/export",              // MikroTik
		"cat /config/config.xml", // pfSense
	}

	for _, cmd := range commands {
		output, err := c.Exec(cmd)
		if err == nil && len(output) > 50 {
			return output, nil
		}
	}
	return "", fmt.Errorf("could not retrieve config — device type unknown")
}

// GetVersion tries to identify the device.
func (c *SSHClient) GetVersion() (string, error) {
	commands := []string{
		"show version",          // Cisco, Arista, VyOS
		"display version",       // Huawei
		"/system/resource/print", // MikroTik
		"uname -a",             // Linux-based
	}

	for _, cmd := range commands {
		output, err := c.Exec(cmd)
		if err == nil && len(output) > 10 {
			return output, nil
		}
	}
	return "", fmt.Errorf("could not get version info")
}

// GetInterfaces tries to list interfaces.
func (c *SSHClient) GetInterfaces() (string, error) {
	commands := []string{
		"show ip interface brief",    // Cisco
		"show interfaces terse",      // Juniper
		"show interfaces",            // VyOS, Arista
		"display interface brief",    // Huawei
		"/interface/print",           // MikroTik
		"ip -br addr",               // Linux
	}

	for _, cmd := range commands {
		output, err := c.Exec(cmd)
		if err == nil && len(output) > 10 {
			return output, nil
		}
	}
	return "", fmt.Errorf("could not get interface info")
}

// GetRoutes tries to show routing table.
func (c *SSHClient) GetRoutes() (string, error) {
	commands := []string{
		"show ip route",         // Cisco, Arista
		"show route",            // Juniper
		"show ip route",         // VyOS
		"display ip routing-table", // Huawei
		"/ip/route/print",       // MikroTik
		"ip route show",         // Linux
	}

	for _, cmd := range commands {
		output, err := c.Exec(cmd)
		if err == nil && len(output) > 10 {
			return output, nil
		}
	}
	return "", fmt.Errorf("could not get routes")
}

// TestConnection verifies SSH access.
func (c *SSHClient) TestConnection() error {
	if err := c.Connect(); err != nil {
		return err
	}
	defer c.Close()
	_, err := c.Exec("echo ok")
	return err
}
