package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "unknown"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show Lab CLI version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("lab %s\n", version)
			fmt.Printf("commit: %s\n", commit)
		},
	}
}

func init() {
	rootCmd.AddCommand(newVersionCmd())
}
