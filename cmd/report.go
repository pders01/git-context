package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var reportCmd = &cobra.Command{
	Use:   "report <template>",
	Short: "Generate pre-defined reports",
	Long: `Generate formatted reports using pre-defined templates.

Available templates:
  daily   - Today's snapshots grouped by tag with summary stats

Examples:
  context report daily`,
	Args: cobra.ExactArgs(1),
	RunE: runReport,
}

func init() {
	rootCmd.AddCommand(reportCmd)
}

func runReport(cmd *cobra.Command, args []string) error {
	template := args[0]

	switch template {
	case "daily":
		return generateDailyReport()
	default:
		return fmt.Errorf("unknown report template: %s (available: daily)", template)
	}
}

func generateDailyReport() error {
	fmt.Println("Daily Snapshot Report")
	fmt.Println("═════════════════════")
	fmt.Println()

	// Show stats
	fmt.Println("Summary")
	fmt.Println("───────")
	if err := runStats(&cobra.Command{}, []string{}); err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("Today's Snapshots by Tag")
	fmt.Println("────────────────────────")

	// Temporarily set list flags
	oldToday := listToday
	oldGroupBy := listGroupBy
	oldJSON := listJSON
	oldToon := listToon

	listToday = true
	listGroupBy = "tag"
	listJSON = false
	listToon = false

	err := runList(&cobra.Command{}, []string{})

	// Restore flags
	listToday = oldToday
	listGroupBy = oldGroupBy
	listJSON = oldJSON
	listToon = oldToon

	return err
}
