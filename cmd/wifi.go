package cmd

import (
	"fmt"

	"github.com/jmsperu/netctl/internal/table"
	"github.com/jmsperu/netctl/internal/unifi"
	"github.com/spf13/cobra"
)

var wifiCmd = &cobra.Command{
	Use:   "wifi",
	Short: "WiFi and access point management",
	Aliases: []string{"ap"},
}

var wifiDevicesCmd = &cobra.Command{
	Use:   "devices [controller]",
	Short: "List access points and network devices",
	Aliases: []string{"aps"},
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := ""
		if len(args) > 0 {
			name = args[0]
		}
		dev, err := loadDevice(name)
		if err != nil {
			return err
		}

		if dev.Type != "unifi" {
			return fmt.Errorf("wifi commands require a UniFi controller (type: unifi)")
		}

		client := unifi.NewClient(dev.Host, dev.EffectivePort(), dev.Username, dev.Password, "default", dev.Insecure)
		if err := client.Login(); err != nil {
			return err
		}
		defer client.Logout()

		devices, err := client.GetDevices()
		if err != nil {
			return err
		}

		t := table.New("NAME", "MODEL", "IP", "STATUS", "CLIENTS", "VERSION", "UPTIME")
		for _, d := range devices {
			status := "Connected"
			if d.State != 1 {
				status = "Disconnected"
			}
			uptime := fmt.Sprintf("%dd %dh", d.Uptime/86400, (d.Uptime%86400)/3600)
			t.AddRow(d.Name, d.Model, d.IP, status, fmt.Sprint(d.NumSTA), d.Version, uptime)
		}
		t.Render()
		fmt.Printf("\n%d devices\n", len(devices))
		return nil
	},
}

var wifiClientsCmd = &cobra.Command{
	Use:   "clients [controller]",
	Short: "List connected WiFi clients",
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := ""
		if len(args) > 0 {
			name = args[0]
		}
		dev, err := loadDevice(name)
		if err != nil {
			return err
		}

		client := unifi.NewClient(dev.Host, dev.EffectivePort(), dev.Username, dev.Password, "default", dev.Insecure)
		if err := client.Login(); err != nil {
			return err
		}
		defer client.Logout()

		clients, err := client.GetClients()
		if err != nil {
			return err
		}

		t := table.New("HOSTNAME", "IP", "MAC", "SSID", "AP", "SIGNAL", "TX", "RX")
		for _, c := range clients {
			if c.IsWired {
				continue
			}
			t.AddRow(c.Hostname, c.IP, c.MAC, c.ESSID, c.APName,
				fmt.Sprintf("%d dBm", c.Signal),
				formatBytes(c.TxBytes), formatBytes(c.RxBytes))
		}
		t.Render()
		fmt.Printf("\n%d wireless clients\n", len(clients))
		return nil
	},
}

var wifiSSIDsCmd = &cobra.Command{
	Use:   "ssids [controller]",
	Short: "List WiFi networks (SSIDs)",
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := ""
		if len(args) > 0 {
			name = args[0]
		}
		dev, err := loadDevice(name)
		if err != nil {
			return err
		}

		client := unifi.NewClient(dev.Host, dev.EffectivePort(), dev.Username, dev.Password, "default", dev.Insecure)
		if err := client.Login(); err != nil {
			return err
		}
		defer client.Logout()

		wlans, err := client.GetWLANs()
		if err != nil {
			return err
		}

		t := table.New("NAME", "SECURITY", "ENABLED", "CLIENTS")
		for _, w := range wlans {
			enabled := "yes"
			if !w.Enabled {
				enabled = "no"
			}
			t.AddRow(w.Name, w.Security, enabled, fmt.Sprint(w.NumSTA))
		}
		t.Render()
		return nil
	},
}

var wifiRestartCmd = &cobra.Command{
	Use:   "restart <controller> <mac>",
	Short: "Restart an access point",
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		dev, err := loadDevice(args[0])
		if err != nil {
			return err
		}

		client := unifi.NewClient(dev.Host, dev.EffectivePort(), dev.Username, dev.Password, "default", dev.Insecure)
		if err := client.Login(); err != nil {
			return err
		}
		defer client.Logout()

		fmt.Printf("Restarting device %s...\n", args[1])
		if err := client.RestartDevice(args[1]); err != nil {
			return err
		}
		fmt.Println("Restart command sent")
		return nil
	},
}

func init() {
	wifiCmd.AddCommand(wifiDevicesCmd, wifiClientsCmd, wifiSSIDsCmd, wifiRestartCmd)
	rootCmd.AddCommand(wifiCmd)
}
