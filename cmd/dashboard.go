package cmd

import (
	"fmt"
	"strings"
	"sync"

	"github.com/jmsperu/netctl/internal/config"
	"github.com/jmsperu/netctl/internal/table"
	"github.com/spf13/cobra"
)

var dashboardCmd = &cobra.Command{
	Use:     "dashboard",
	Short:   "Show overview of all managed devices",
	Aliases: []string{"dash"},
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		if len(cfg.Devices) == 0 {
			fmt.Println("No devices configured. Run: netctl device add <name>")
			return nil
		}

		fmt.Println("Network Dashboard")
		fmt.Println(strings.Repeat("═", 60))
		fmt.Printf("%d devices managed\n\n", len(cfg.Devices))

		// Group by type
		byType := map[string][]*config.Device{}
		for _, d := range cfg.Devices {
			byType[d.Type] = append(byType[d.Type], d)
		}

		for dtype, devices := range byType {
			fmt.Printf("── %s (%d) ──\n", strings.ToUpper(dtype), len(devices))
			fmt.Println()

			// Test connectivity in parallel
			type result struct {
				dev    *config.Device
				status string
			}
			results := make([]result, len(devices))
			var wg sync.WaitGroup

			for i, dev := range devices {
				wg.Add(1)
				go func(idx int, d *config.Device) {
					defer wg.Done()
					client := getClient(d)
					err := client.TestConnection()
					status := "UP"
					if err != nil {
						status = "DOWN"
					}
					results[idx] = result{dev: d, status: status}
				}(i, dev)
			}
			wg.Wait()

			t := table.New("NAME", "HOST", "STATUS", "TAGS")
			for _, r := range results {
				tags := strings.Join(r.dev.Tags, ", ")
				t.AddRow(r.dev.Name, r.dev.Host, r.status, tags)
			}
			t.Render()
			fmt.Println()
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(dashboardCmd)
}
