package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/gopher-opsx/platform-lab-cli/internal/state"
)

var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Restore changes made by the active Lab scenario",

	RunE: func(cmd *cobra.Command, args []string) error {
		root, client, err := platformClient()
		if err != nil {
			return err
		}

		out := cmd.OutOrStdout()

		if !state.Exists(root) {
			fmt.Fprintln(
				out,
				"No active Lab scenario. Nothing to reset.",
			)

			return nil
		}

		session, err := state.Load(root)
		if err != nil {
			return err
		}

		fmt.Fprintf(
			out,
			"Resetting scenario: %s\n",
			session.Scenario,
		)

		fmt.Fprintln(out)

		// Restore changes in reverse order.
		for i := len(session.Changes) - 1; i >= 0; i-- {
			change := session.Changes[i]

			switch change.Type {

			case "service_state":

				if change.OriginalState == "running" {
					fmt.Fprintf(
						out,
						"Restoring %s...\n",
						change.Target,
					)

					if err := client.Start(change.Target); err != nil {
						return err
					}

					running, err := client.IsRunning(change.Target)
					if err != nil {
						return err
					}

					if !running {
						return fmt.Errorf(
							"failed to restore %s",
							change.Target,
						)
					}

					fmt.Fprintf(
						out,
						"✓ %s restored\n",
						change.Target,
					)
				}
			}
		}

		// Only remove recovery information after restoration succeeds.
		if err := state.Clear(root); err != nil {
			return err
		}

		fmt.Fprintln(out)
		fmt.Fprintln(
			out,
			"✓ Platform Lab restored.",
		)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(resetCmd)
}
