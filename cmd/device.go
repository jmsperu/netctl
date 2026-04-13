package cmd

import (
	"fmt"

	"github.com/jmsperu/netctl/internal/config"
	"github.com/jmsperu/netctl/internal/table"
	"github.com/spf13/cobra"
)

var deviceCmd = &cobra.Command{
	Use:   "device",
	Short: "Manage network devices",
	Aliases: []string{"dev"},
}

var deviceAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a network device",
	Long: `Add a network device to manage.

Types:
  sophos     — Sophos XG/XGS Firewall (XML API, port 4444)
  vyos       — VyOS Router (HTTP API)
  mikrotik   — MikroTik RouterOS (REST API v7+ or RouterOS API)
  unifi      — Ubiquiti UniFi Controller (REST API)
  fortigate  — FortiGate Firewall (REST API)
  pfsense    — pfSense/OPNsense (REST API)
  arista     — Arista EOS (eAPI)
  cisco      — Cisco IOS/IOS-XE (SSH)
  juniper    — Juniper (SSH/NETCONF)
  generic    — Any device via SSH

Examples:
  netctl device add fw1 --type sophos --host 192.168.1.1 --user admin --pass P@ss
  netctl device add rtr1 --type vyos --host 10.0.0.1 --api-key MY_KEY --insecure
  netctl device add sw1 --type mikrotik --host 10.0.0.2 --user admin --pass admin
  netctl device add ap1 --type unifi --host unifi.local --user admin --pass admin
  netctl device add core --type generic --host 10.0.0.3 --user admin --pass admin --tag core,datacenter`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		dtype, _ := cmd.Flags().GetString("type")
		host, _ := cmd.Flags().GetString("host")
		port, _ := cmd.Flags().GetInt("port")
		user, _ := cmd.Flags().GetString("user")
		pass, _ := cmd.Flags().GetString("pass")
		apiKey, _ := cmd.Flags().GetString("api-key")
		insecure, _ := cmd.Flags().GetBool("insecure")
		tags, _ := cmd.Flags().GetStringSlice("tag")

		if dtype == "" {
			return fmt.Errorf("--type is required")
		}
		if host == "" {
			return fmt.Errorf("--host is required")
		}

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		cfg.Devices[name] = &config.Device{
			Name:     name,
			Type:     dtype,
			Host:     host,
			Port:     port,
			Username: user,
			Password: pass,
			APIKey:   apiKey,
			Insecure: insecure,
			Tags:     tags,
		}

		if cfg.DefaultDevice == "" {
			cfg.DefaultDevice = name
		}

		if err := cfg.Save(); err != nil {
			return err
		}

		fmt.Printf("Device %q added (%s @ %s)\n", name, dtype, host)
		return nil
	},
}

var deviceListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List all devices",
	Aliases: []string{"ls"},
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		if len(cfg.Devices) == 0 {
			fmt.Println("No devices configured. Run: netctl device add <name> --type <type> --host <host>")
			return nil
		}

		t := table.New("NAME", "TYPE", "HOST", "PORT", "TAGS", "DEFAULT")
		for name, d := range cfg.Devices {
			def := ""
			if name == cfg.DefaultDevice {
				def = "*"
			}
			tags := ""
			if len(d.Tags) > 0 {
				for i, tag := range d.Tags {
					if i > 0 {
						tags += ","
					}
					tags += tag
				}
			}
			t.AddRow(name, d.Type, d.Host, fmt.Sprint(d.EffectivePort()), tags, def)
		}
		t.Render()
		return nil
	},
}

var deviceRemoveCmd = &cobra.Command{
	Use:     "remove <name>",
	Short:   "Remove a device",
	Aliases: []string{"rm"},
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		if _, ok := cfg.Devices[name]; !ok {
			return fmt.Errorf("device %q not found", name)
		}
		delete(cfg.Devices, name)
		if cfg.DefaultDevice == name {
			cfg.DefaultDevice = ""
			for n := range cfg.Devices {
				cfg.DefaultDevice = n
				break
			}
		}
		if err := cfg.Save(); err != nil {
			return err
		}
		fmt.Printf("Device %q removed\n", name)
		return nil
	},
}

var deviceUseCmd = &cobra.Command{
	Use:   "use <name>",
	Short: "Set default device",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		if _, ok := cfg.Devices[args[0]]; !ok {
			return fmt.Errorf("device %q not found", args[0])
		}
		cfg.DefaultDevice = args[0]
		return cfg.Save()
	},
}

var deviceTestCmd = &cobra.Command{
	Use:   "test [name]",
	Short: "Test connection to a device",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := deviceFlag
		if len(args) > 0 {
			name = args[0]
		}
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		dev, err := cfg.GetDevice(name)
		if err != nil {
			return err
		}

		fmt.Printf("Testing %s connection to %s:%d...\n", dev.Type, dev.Host, dev.EffectivePort())
		client := getClient(dev)
		if err := client.TestConnection(); err != nil {
			fmt.Printf("FAILED: %v\n", err)
			return err
		}
		fmt.Println("SUCCESS")
		return nil
	},
}

func init() {
	deviceAddCmd.Flags().String("type", "", "Device type (sophos, vyos, mikrotik, unifi, fortigate, pfsense, arista, cisco, juniper, generic)")
	deviceAddCmd.Flags().String("host", "", "Device hostname or IP")
	deviceAddCmd.Flags().Int("port", 0, "Port (auto-detected from type)")
	deviceAddCmd.Flags().String("user", "", "Username")
	deviceAddCmd.Flags().String("pass", "", "Password")
	deviceAddCmd.Flags().String("api-key", "", "API key (VyOS, FortiGate)")
	deviceAddCmd.Flags().Bool("insecure", false, "Skip TLS verification")
	deviceAddCmd.Flags().StringSlice("tag", nil, "Tags for grouping (e.g. --tag core,datacenter)")

	deviceCmd.AddCommand(deviceAddCmd, deviceListCmd, deviceRemoveCmd, deviceUseCmd, deviceTestCmd)
	rootCmd.AddCommand(deviceCmd)
}
