package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/gopher-opsx/platform-lab-cli/internal/scenario"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available incident scenarios",

	RunE: func(cmd *cobra.Command, args []string) error {
		scenarios, err := scenario.LoadAll()
		if err != nil {
			return err
		}

		if len(scenarios) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No scenarios available.")
			return nil
		}

		fmt.Fprintln(
			cmd.OutOrStdout(),
			"AVAILABLE SCENARIOS",
		)

		fmt.Fprintln(cmd.OutOrStdout())

		for _, s := range scenarios {
			fmt.Fprintf(
				cmd.OutOrStdout(),
				"%-22s Lesson %-3d %s\n",
				s.ID,
				s.Lesson,
				s.Name,
			)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
