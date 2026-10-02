package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/gopher-opsx/platform-lab-cli/internal/scenario"
	"github.com/gopher-opsx/platform-lab-cli/internal/state"
)

var startCmd = &cobra.Command{
	Use:   "start <scenario>",
	Short: "Start a controlled incident scenario",
	Args:  cobra.ExactArgs(1),

	RunE: func(cmd *cobra.Command, args []string) error {
		scenarioID := args[0]

		// Make sure the scenario exists.
		s, err := scenario.Load(scenarioID)
		if err != nil {
			return err
		}

		root, client, err := platformClient()
		if err != nil {
			return err
		}

		if err := ensureNoActiveScenario(root); err != nil {
			return err
		}

		out := cmd.OutOrStdout()

		fmt.Fprintf(out, "Starting scenario: %s\n", s.Name)
		fmt.Fprintln(out)

		switch s.ID {

		case "redis-down":

			running, err := client.IsRunning("redis")
			if err != nil {
				return fmt.Errorf(
					"inspect Redis state: %w",
					err,
				)
			}

			if !running {
				return fmt.Errorf(
					"Redis is not running; scenario was not started",
				)
			}

			fmt.Fprintln(out, "✓ Redis is running")

			session := state.Session{
				Scenario:  s.ID,
				StartedAt: time.Now(),
				Changes: []state.Change{
					{
						Type:          "service_state",
						Target:        "redis",
						OriginalState: "running",
					},
				},
			}

			// Save recovery information BEFORE making the change.
			if err := state.Save(root, session); err != nil {
				return err
			}

			fmt.Fprintln(out, "✓ Recovery state recorded")

			if err := client.Stop("redis"); err != nil {
				// Injection failed, so remove the unused state file.
				_ = state.Clear(root)

				return err
			}

			fmt.Fprintln(out, "✓ Scenario activated")

			running, err = client.IsRunning("redis")
			if err != nil {
				return err
			}

			if running {
				return fmt.Errorf(
					"scenario verification failed: Redis is still running",
				)
			}

			fmt.Fprintln(out, "✓ Incident verified")

			fmt.Fprintln(out)
			fmt.Fprintln(
				out,
				"Begin troubleshooting from the customer symptom.",
			)

			return nil

		default:
			return fmt.Errorf(
				"scenario %q is defined but not implemented yet",
				s.ID,
			)
		}
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}
