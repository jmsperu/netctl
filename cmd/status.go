package cmd

import (
	"fmt"
	"strings"

	"github.com/jmsperu/netctl/internal/sophos"
	"github.com/jmsperu/netctl/internal/mikrotik"
	"github.com/jmsperu/netctl/internal/unifi"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status [device]",
	Short: "Show device status and system info",
	Aliases: []string{"info"},
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

		fmt.Printf("Device: %s (%s @ %s)\n", dev.Name, dev.Type, dev.Host)
		fmt.Println(strings.Repeat("═", 50))

		switch dev.Type {
		case "sophos":
			client := sophos.NewClient(dev.Host, dev.EffectivePort(), dev.Username, dev.Password, dev.Insecure)
			info, err := client.GetSystemInfo()
			if err != nil {
				return err
			}
			fmt.Printf("  Hostname:  %s\n", info.Hostname)
			fmt.Printf("  Model:     %s\n", info.Model)
			fmt.Printf("  Firmware:  %s\n", info.Firmware)
			fmt.Printf("  Serial:    %s\n", info.Serial)
			fmt.Printf("  Uptime:    %s\n", info.Uptime)

		case "mikrotik":
			client := mikrotik.NewClient(dev.Host, dev.EffectivePort(), dev.Username, dev.Password, true, dev.Insecure)
			id, _ := client.GetIdentity()
			res, _ := client.GetResource()
			if id != nil {
				fmt.Printf("  Identity:  %s\n", id.Name)
			}
			if res != nil {
				fmt.Printf("  Board:     %s\n", res.BoardName)
				fmt.Printf("  Version:   %s\n", res.Version)
				fmt.Printf("  Arch:      %s\n", res.Architecture)
				fmt.Printf("  CPUs:      %d (load: %d%%)\n", res.CPUCount, res.CPULoad)
				fmt.Printf("  Memory:    %s / %s\n", formatBytes(res.FreeMemory), formatBytes(res.TotalMemory))
				fmt.Printf("  Disk:      %s / %s\n", formatBytes(res.FreeHDD), formatBytes(res.TotalHDD))
				fmt.Printf("  Uptime:    %s\n", res.Uptime)
			}

		case "unifi":
			client := unifi.NewClient(dev.Host, dev.EffectivePort(), dev.Username, dev.Password, "default", dev.Insecure)
			if err := client.Login(); err != nil {
				return err
			}
			defer client.Logout()
			health, _ := client.GetSiteHealth()
			for _, h := range health {
				fmt.Printf("  %s: %s", strings.ToUpper(h.Subsystem), h.Status)
				if h.NumUser > 0 {
					fmt.Printf(" (%d users)", h.NumUser)
				}
				if h.WANIP != "" {
					fmt.Printf(" — %s (%s)", h.WANIP, h.ISP)
				}
				fmt.Println()
			}
			devices, _ := client.GetDevices()
			fmt.Printf("\n  Devices: %d\n", len(devices))
			for _, d := range devices {
				state := "connected"
				if d.State != 1 {
					state = "disconnected"
				}
				fmt.Printf("    %s (%s) — %s, %d clients, %s\n", d.Name, d.Model, state, d.NumSTA, d.IP)
			}

		case "vyos":
			ssh := newSSHClient(dev)
			if err := ssh.Connect(); err != nil {
				return err
			}
			defer ssh.Close()
			output, _ := ssh.Exec("show version")
			fmt.Println(output)

		default:
			// Generic SSH
			ssh := newSSHClient(dev)
			if err := ssh.Connect(); err != nil {
				return err
			}
			defer ssh.Close()
			output, _ := ssh.GetVersion()
			fmt.Println(output)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
