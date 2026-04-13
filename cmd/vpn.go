package cmd

import (
	"fmt"

	"github.com/jmsperu/netctl/internal/sophos"
	"github.com/jmsperu/netctl/internal/table"
	"github.com/spf13/cobra"
)

var vpnCmd = &cobra.Command{
	Use:   "vpn [device]",
	Short: "Show VPN tunnel status",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := ""
		if len(args) > 0 {
			name = args[0]
		}
		dev, err := loadDevice(name)
		if err != nil {
			return err
		}

		switch dev.Type {
		case "sophos":
			client := sophos.NewClient(dev.Host, dev.EffectivePort(), dev.Username, dev.Password, dev.Insecure)
			tunnels, err := client.GetVPNTunnels()
			if err != nil {
				return err
			}
			if len(tunnels) == 0 {
				fmt.Println("No VPN tunnels configured")
				return nil
			}
			t := table.New("NAME", "TYPE", "STATUS", "LOCAL", "REMOTE", "PEER")
			for _, tun := range tunnels {
				t.AddRow(tun.Name, tun.Type, tun.Status, tun.LocalNet, tun.RemoteNet, tun.RemoteHost)
			}
			t.Render()

		default:
			ssh := newSSHClient(dev)
			if err := ssh.Connect(); err != nil {
				return err
			}
			defer ssh.Close()
			for _, c := range []string{
				"show vpn ipsec sa",
				"show interfaces wireguard",
				"show crypto ipsec sa",
				"/ip/ipsec/active-peers/print",
			} {
				output, err := ssh.Exec(c)
				if err == nil && len(output) > 10 {
					fmt.Println(output)
					return nil
				}
			}
			fmt.Println("No VPN info available via SSH")
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(vpnCmd)
}
