package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/gopher-opsx/platform-lab-cli/internal/state"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the active Lab scenario",

	RunE: func(cmd *cobra.Command, args []string) error {
		root, client, err := platformClient()
		if err != nil {
			return err
		}

		out := cmd.OutOrStdout()

		if !state.Exists(root) {
			fmt.Fprintln(out, "No Lab scenario is active.")
			return nil
		}

		session, err := state.Load(root)
		if err != nil {
			return err
		}

		fmt.Fprintf(
			out,
			"Active scenario: %s\n",
			session.Scenario,
		)

		fmt.Fprintf(
			out,
			"Started: %s\n",
			session.StartedAt.Format("2006-01-02 15:04:05"),
		)

		for _, change := range session.Changes {
			if change.Type != "service_state" {
				continue
			}

			running, err := client.IsRunning(change.Target)
			if err != nil {
				return err
			}

			current := "stopped"

			if running {
				current = "running"
			}

			fmt.Fprintf(
				out,
				"%s: %s\n",
				change.Target,
				current,
			)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
