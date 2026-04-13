package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	deviceFlag string
	version    = "dev"
)

var rootCmd = &cobra.Command{
	Use:   "netctl",
	Short: "Unified network device manager — routers, firewalls, switches, APs",
	Long: `netctl — manage all your network devices from one CLI.
Routers, firewalls, switches, and access points. Any vendor.

Supported devices:
  Firewalls:    Sophos XG/XGS, FortiGate, pfSense, OPNsense
  Routers:      VyOS, MikroTik, Cisco IOS, Juniper, Arista
  APs:          UniFi, Aruba, Ruckus
  Any SSH:      Generic SSH for any device

Quick start:
  netctl device add mysophos --type sophos --host 192.168.1.1 --user admin --pass password
  netctl device add myrouter --type vyos   --host 192.168.1.1 --key APIKEY
  netctl device add myap     --type unifi  --host unifi.local  --user admin --pass password
  netctl device add myswitch --type generic --host 192.168.1.2 --user admin --pass password

  netctl status mysophos        # device overview
  netctl interfaces myrouter    # show interfaces
  netctl routes myrouter        # routing table
  netctl firewall mysophos      # firewall rules
  netctl config backup myrouter # backup config
  netctl exec myswitch "show version"  # run any command`,
	Version: version,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&deviceFlag, "device", "d", "", "Device to use (default: active device)")
}
