package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Device configuration management",
}

var configBackupCmd = &cobra.Command{
	Use:   "backup <device> [output-dir]",
	Short: "Backup device configuration",
	Long: `Download the running configuration from a device.

The config is saved to a timestamped file. Use this regularly
to track configuration changes over time.

Examples:
  netctl config backup myrouter
  netctl config backup myrouter ./backups/
  netctl config backup fw1 /opt/network-backups/`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		dev, err := loadDevice(args[0])
		if err != nil {
			return err
		}

		outputDir := "."
		if len(args) > 1 {
			outputDir = args[1]
		}

		fmt.Printf("Backing up config from %s (%s)...\n", dev.Name, dev.Host)

		ssh := newSSHClient(dev)
		if err := ssh.Connect(); err != nil {
			return err
		}
		defer ssh.Close()

		config, err := ssh.GetConfig()
		if err != nil {
			return err
		}

		// Create output directory
		os.MkdirAll(outputDir, 0755)

		// Save with timestamp
		timestamp := time.Now().Format("2006-01-02_150405")
		filename := fmt.Sprintf("%s_%s.conf", dev.Name, timestamp)
		filepath := filepath.Join(outputDir, filename)

		if err := os.WriteFile(filepath, []byte(config), 0644); err != nil {
			return err
		}

		fmt.Printf("Config saved to %s (%d bytes)\n", filepath, len(config))
		return nil
	},
}

var configDiffCmd = &cobra.Command{
	Use:   "diff <file1> <file2>",
	Short: "Compare two config backups",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		data1, err := os.ReadFile(args[0])
		if err != nil {
			return fmt.Errorf("cannot read %s: %v", args[0], err)
		}
		data2, err := os.ReadFile(args[1])
		if err != nil {
			return fmt.Errorf("cannot read %s: %v", args[1], err)
		}

		if string(data1) == string(data2) {
			fmt.Println("Configurations are identical")
			return nil
		}

		fmt.Printf("Configurations differ (%s: %d bytes, %s: %d bytes)\n",
			args[0], len(data1), args[1], len(data2))
		// Simple line diff
		lines1 := splitLines(string(data1))
		lines2 := splitLines(string(data2))

		maxLines := len(lines1)
		if len(lines2) > maxLines {
			maxLines = len(lines2)
		}

		for i := 0; i < maxLines; i++ {
			l1 := ""
			l2 := ""
			if i < len(lines1) {
				l1 = lines1[i]
			}
			if i < len(lines2) {
				l2 = lines2[i]
			}
			if l1 != l2 {
				if l1 != "" {
					fmt.Printf("- %s\n", l1)
				}
				if l2 != "" {
					fmt.Printf("+ %s\n", l2)
				}
			}
		}
		return nil
	},
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i, c := range s {
		if c == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func init() {
	configCmd.AddCommand(configBackupCmd, configDiffCmd)
	rootCmd.AddCommand(configCmd)
}
