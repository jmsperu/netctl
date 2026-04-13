package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var execCmd = &cobra.Command{
	Use:   "exec <device> <command>",
	Short: "Run a command on a device via SSH",
	Long: `Execute any command on a network device via SSH.

Examples:
  netctl exec myrouter "show version"
  netctl exec myswitch "show interfaces"
  netctl exec fw1 "show firewall"
  netctl exec -d myrouter "show ip route"`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		dev, err := loadDevice(args[0])
		if err != nil {
			return err
		}

		command := strings.Join(args[1:], " ")
		ssh := newSSHClient(dev)
		if err := ssh.Connect(); err != nil {
			return err
		}
		defer ssh.Close()

		output, err := ssh.Exec(command)
		if err != nil {
			return fmt.Errorf("command failed: %v", err)
		}
		fmt.Print(output)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(execCmd)
}
