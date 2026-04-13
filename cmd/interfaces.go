package cmd

import (
	"fmt"

	"github.com/jmsperu/netctl/internal/mikrotik"
	"github.com/jmsperu/netctl/internal/sophos"
	"github.com/jmsperu/netctl/internal/table"
	"github.com/spf13/cobra"
)

var interfacesCmd = &cobra.Command{
	Use:     "interfaces [device]",
	Short:   "Show network interfaces",
	Aliases: []string{"iface", "if"},
	Args:    cobra.MaximumNArgs(1),
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
			ifaces, err := client.GetInterfaces()
			if err != nil {
				return err
			}
			t := table.New("NAME", "STATUS", "IP", "ZONE", "SPEED", "MTU")
			for _, i := range ifaces {
				t.AddRow(i.Name, i.Status, i.IPAddress, i.Zone, i.Speed, fmt.Sprint(i.MTU))
			}
			t.Render()

		case "mikrotik":
			client := mikrotik.NewClient(dev.Host, dev.EffectivePort(), dev.Username, dev.Password, true, dev.Insecure)
			ifaces, err := client.GetInterfaces()
			if err != nil {
				return err
			}
			t := table.New("NAME", "TYPE", "RUNNING", "MAC", "TX", "RX")
			for _, i := range ifaces {
				running := "yes"
				if !i.Running {
					running = "no"
				}
				t.AddRow(i.Name, i.Type, running, i.MacAddr, formatBytes(i.TxBytes), formatBytes(i.RxBytes))
			}
			t.Render()

		default:
			// SSH fallback
			ssh := newSSHClient(dev)
			if err := ssh.Connect(); err != nil {
				return err
			}
			defer ssh.Close()
			output, err := ssh.GetInterfaces()
			if err != nil {
				return err
			}
			fmt.Println(output)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(interfacesCmd)
}
