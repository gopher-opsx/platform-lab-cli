package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/gopher-opsx/platform-lab-cli/internal/compose"
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
			"Resetting scenario: %s\n\n",
			session.Scenario,
		)

		for i := len(session.Changes) - 1; i >= 0; i-- {
			change := session.Changes[i]

			switch change.Type {

			case "service_state":

				if change.OriginalState != "running" {
					continue
				}

				fmt.Fprintf(
					out,
					"Restoring %s...\n",
					change.Target,
				)

				if err := client.Start(
					change.Target,
				); err != nil {
					return err
				}

				running, err := client.IsRunning(
					change.Target,
				)
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

			case "compose_override":

				if err := resetComposeOverride(
					out,
					root,
					client,
					change.Target,
				); err != nil {
					return err
				}

			}
		}

		/*
			State is removed only after all rollback
			operations succeed.
		*/
		if err := state.Clear(root); err != nil {
			return err
		}

		fmt.Fprintln(out)
		fmt.Fprintln(
			out,
			"✓ Lab changes removed.",
		)

		return nil
	},
}

func resetComposeOverride(
	out interface {
		Write([]byte) (int, error)
	},
	root string,
	client compose.Client,
	service string,
) error {

	fmt.Fprintf(
		out,
		"Removing injected configuration from %s...\n",
		service,
	)

	/*
		First remove the Lab-owned override.

		Then recreate the service using only the normal
		Platform Lab Compose configuration.
	*/
	if err := compose.RemoveOverride(root); err != nil {
		return err
	}

	if err := client.RecreateBaseline(service); err != nil {
		return err
	}

	fmt.Fprintf(
		out,
		"Waiting for %s to become healthy...\n",
		service,
	)

	if err := client.WaitForHealthy(
		service,
		30*time.Second,
	); err != nil {
		return fmt.Errorf(
			"baseline configuration was restored but recovery verification failed: %w",
			err,
		)
	}

	fmt.Fprintf(
		out,
		"✓ %s baseline configuration restored\n",
		service,
	)

	fmt.Fprintf(
		out,
		"✓ %s recovery verified\n",
		service,
	)

	return nil
}

func init() {
	rootCmd.AddCommand(resetCmd)
}
