package cmd

import (
	"fmt"

	"github.com/jmsperu/netctl/internal/mikrotik"
	"github.com/jmsperu/netctl/internal/sophos"
	"github.com/jmsperu/netctl/internal/table"
	"github.com/spf13/cobra"
)

var firewallCmd = &cobra.Command{
	Use:     "firewall [device]",
	Short:   "Show firewall rules",
	Aliases: []string{"fw"},
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
			rules, err := client.GetFirewallRules()
			if err != nil {
				return err
			}
			if len(rules) == 0 {
				fmt.Println("No firewall rules (or XML parsing pending)")
				return nil
			}
			t := table.New("NAME", "ACTION", "STATUS", "SRC ZONE", "DST ZONE", "SOURCE", "DEST", "SERVICE")
			for _, r := range rules {
				t.AddRow(r.Name, r.Action, r.Status, r.SourceZone, r.DestZone, r.Source, r.Dest, r.Service)
			}
			t.Render()

		case "mikrotik":
			client := mikrotik.NewClient(dev.Host, dev.EffectivePort(), dev.Username, dev.Password, true, dev.Insecure)
			rules, err := client.GetFirewallFilter()
			if err != nil {
				return err
			}
			t := table.New("CHAIN", "ACTION", "PROTOCOL", "SRC", "DST", "PORT", "COMMENT", "BYTES")
			for _, r := range rules {
				if r.Disabled {
					continue
				}
				t.AddRow(r.Chain, r.Action, r.Protocol, r.SrcAddr, r.DstAddr, r.DstPort, r.Comment, formatBytes(r.Bytes))
			}
			t.Render()

		default:
			ssh := newSSHClient(dev)
			if err := ssh.Connect(); err != nil {
				return err
			}
			defer ssh.Close()
			// Try common firewall commands
			for _, cmd := range []string{
				"show firewall",
				"show access-lists",
				"show ip firewall filter print",
				"iptables -L -n --line-numbers",
				"nft list ruleset",
			} {
				output, err := ssh.Exec(cmd)
				if err == nil && len(output) > 20 {
					fmt.Println(output)
					return nil
				}
			}
			fmt.Println("Could not retrieve firewall rules via SSH")
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(firewallCmd)
}
