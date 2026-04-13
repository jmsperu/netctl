package cmd

import (
	"fmt"

	"github.com/jmsperu/netctl/internal/config"
	"github.com/jmsperu/netctl/internal/generic"
	"github.com/jmsperu/netctl/internal/mikrotik"
	"github.com/jmsperu/netctl/internal/sophos"
	"github.com/jmsperu/netctl/internal/unifi"
	"github.com/jmsperu/netctl/internal/vyos"
)

// DeviceClient is an interface for any network device.
type DeviceClient interface {
	TestConnection() error
}

func loadDevice(name string) (*config.Device, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	if name == "" {
		name = deviceFlag
	}
	return cfg.GetDevice(name)
}

func getClient(dev *config.Device) DeviceClient {
	switch dev.Type {
	case "sophos":
		return sophos.NewClient(dev.Host, dev.EffectivePort(), dev.Username, dev.Password, dev.Insecure)
	case "vyos":
		return vyos.NewClient(dev.Host, dev.EffectivePort(), dev.APIKey, dev.Insecure)
	case "mikrotik":
		return mikrotik.NewClient(dev.Host, dev.EffectivePort(), dev.Username, dev.Password, true, dev.Insecure)
	case "unifi":
		return unifi.NewClient(dev.Host, dev.EffectivePort(), dev.Username, dev.Password, "default", dev.Insecure)
	default:
		return generic.NewSSHClient(dev.Host, dev.SSHPort(), dev.Username, dev.Password)
	}
}

func newSSHClient(dev *config.Device) *generic.SSHClient {
	return generic.NewSSHClient(dev.Host, dev.SSHPort(), dev.Username, dev.Password)
}

func formatBytes(b int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)
	switch {
	case b >= TB:
		return fmt.Sprintf("%.1f TB", float64(b)/float64(TB))
	case b >= GB:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(GB))
	case b >= MB:
		return fmt.Sprintf("%.0f MB", float64(b)/float64(MB))
	default:
		return fmt.Sprintf("%.0f KB", float64(b)/float64(KB))
	}
}
