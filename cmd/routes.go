package cmd

import (
	"fmt"

	"github.com/jmsperu/netctl/internal/mikrotik"
	"github.com/jmsperu/netctl/internal/table"
	"github.com/spf13/cobra"
)

var routesCmd = &cobra.Command{
	Use:     "routes [device]",
	Short:   "Show routing table",
	Aliases: []string{"route", "rt"},
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
		case "mikrotik":
			client := mikrotik.NewClient(dev.Host, dev.EffectivePort(), dev.Username, dev.Password, true, dev.Insecure)
			routes, err := client.GetRoutes()
			if err != nil {
				return err
			}
			t := table.New("DESTINATION", "GATEWAY", "DISTANCE", "ACTIVE")
			for _, r := range routes {
				active := "yes"
				if !r.Active {
					active = "no"
				}
				t.AddRow(r.DstAddress, r.Gateway, fmt.Sprint(r.Distance), active)
			}
			t.Render()

		default:
			ssh := newSSHClient(dev)
			if err := ssh.Connect(); err != nil {
				return err
			}
			defer ssh.Close()
			output, err := ssh.GetRoutes()
			if err != nil {
				return err
			}
			fmt.Println(output)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(routesCmd)
}
