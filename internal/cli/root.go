package cli

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "lab",
	Short: "Controlled incident injector for Platform Lab",
	Long: `Lab creates controlled, deterministic, and reversible
failure scenarios for practicing production troubleshooting.

Lab injects failures.
Doctor collects evidence.
You diagnose the incident.`,
	SilenceUsage: true,
}

func Execute() error {
	return rootCmd.Execute()
}
