package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	DefaultDevice string              `yaml:"default_device"`
	Devices       map[string]*Device  `yaml:"devices"`
	Groups        map[string][]string `yaml:"groups,omitempty"` // named groups of devices
}

type Device struct {
	Name     string            `yaml:"name"`
	Type     string            `yaml:"type"` // sophos, vyos, mikrotik, unifi, cisco, fortigate, pfsense, arista, generic
	Host     string            `yaml:"host"`
	Port     int               `yaml:"port,omitempty"`
	Username string            `yaml:"username,omitempty"`
	Password string            `yaml:"password,omitempty"`
	APIKey   string            `yaml:"api_key,omitempty"`
	APIToken string            `yaml:"api_token,omitempty"`
	Insecure bool              `yaml:"insecure,omitempty"`
	Tags     []string          `yaml:"tags,omitempty"`
	Extra    map[string]string `yaml:"extra,omitempty"`
}

func configPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".netctl.yaml")
}

func Load() (*Config, error) {
	cfg := &Config{
		Devices: make(map[string]*Device),
		Groups:  make(map[string][]string),
	}

	data, err := os.ReadFile(configPath())
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %v", err)
	}

	if cfg.Devices == nil {
		cfg.Devices = make(map[string]*Device)
	}
	if cfg.Groups == nil {
		cfg.Groups = make(map[string][]string)
	}

	return cfg, nil
}

func (c *Config) Save() error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(configPath(), data, 0600)
}

func (c *Config) GetDevice(name string) (*Device, error) {
	if name == "" {
		name = c.DefaultDevice
	}
	if name == "" {
		return nil, fmt.Errorf("no device specified and no default set\nRun: netctl device add <name>")
	}
	d, ok := c.Devices[name]
	if !ok {
		return nil, fmt.Errorf("device %q not found\nRun: netctl device list", name)
	}
	return d, nil
}

func (c *Config) GetGroup(name string) ([]*Device, error) {
	names, ok := c.Groups[name]
	if !ok {
		return nil, fmt.Errorf("group %q not found", name)
	}
	var devices []*Device
	for _, n := range names {
		if d, ok := c.Devices[n]; ok {
			devices = append(devices, d)
		}
	}
	return devices, nil
}

// DevicesByTag returns all devices matching a tag.
func (c *Config) DevicesByTag(tag string) []*Device {
	var result []*Device
	for _, d := range c.Devices {
		for _, t := range d.Tags {
			if t == tag {
				result = append(result, d)
				break
			}
		}
	}
	return result
}

// DevicesByType returns all devices of a given type.
func (c *Config) DevicesByType(dtype string) []*Device {
	var result []*Device
	for _, d := range c.Devices {
		if d.Type == dtype {
			result = append(result, d)
		}
	}
	return result
}

// EffectivePort returns the port or a sensible default.
func (d *Device) EffectivePort() int {
	if d.Port > 0 {
		return d.Port
	}
	switch d.Type {
	case "sophos":
		return 4444 // Sophos XG web admin API port
	case "vyos":
		return 443
	case "mikrotik":
		return 8728 // RouterOS API
	case "unifi":
		return 443
	case "fortigate":
		return 443
	case "pfsense", "opnsense":
		return 443
	case "arista":
		return 443
	default:
		return 22 // SSH
	}
}

// SSHPort returns the SSH port.
func (d *Device) SSHPort() int {
	if d.Port > 0 && d.Type == "generic" {
		return d.Port
	}
	return 22
}
