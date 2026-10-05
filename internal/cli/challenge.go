package cli

import (
	"crypto/rand"
	"fmt"
	"io"
	"math/big"

	"github.com/spf13/cobra"

	"github.com/gopher-opsx/platform-lab-cli/internal/scenario"
	"github.com/gopher-opsx/platform-lab-cli/internal/state"
)

// challengeEligibleScenarios intentionally excludes scenarios that are
// temporary, resource-intensive, destructive, or awkward for an unknown
// final-course incident. Every entry must be deterministic and reversible.
var challengeEligibleScenarios = []string{
	"restart-loop",
	"running-but-dead",
	"bad-environment",
	"dns-failure",
	"port-failure",
	"postgres-down",
	"kafka-down",
}

var challengeScenarioOverride string

var challengeCmd = &cobra.Command{
	Use:   "challenge",
	Short: "Start an unknown troubleshooting challenge",
	Long: `Challenge mode activates one controlled Course 1 incident without
revealing which scenario was selected.

Start from the customer symptom, follow the system path, collect evidence,
form and test hypotheses, identify the root cause, then recover and verify.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		root, client, err := platformClient()
		if err != nil {
			return err
		}

		if err := ensureNoActiveScenario(root); err != nil {
			return err
		}

		selected, err := selectChallengeScenario(challengeScenarioOverride)
		if err != nil {
			return err
		}

		// Validate that the selected scenario still exists in the embedded
		// Course 1 definitions before touching the running environment.
		if _, err := scenario.Load(selected); err != nil {
			return fmt.Errorf("challenge scenario is unavailable: %w", err)
		}

		// Suppress the scenario-specific start narration. Challenge mode must
		// not reveal the affected component or the underlying failure.
		if err := startScenario(io.Discard, root, client, selected); err != nil {
			return fmt.Errorf("could not activate challenge incident: %w", err)
		}

		session, err := state.Load(root)
		if err != nil {
			return fmt.Errorf("challenge incident started but state could not be read: %w", err)
		}

		session.Challenge = true
		if err := state.Save(root, session); err != nil {
			return fmt.Errorf("challenge incident started but challenge state could not be saved: %w", err)
		}

		out := cmd.OutOrStdout()

		fmt.Fprintln(out, "Platform Lab troubleshooting challenge")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "✓ Healthy baseline verified")
		fmt.Fprintln(out, "✓ Controlled incident activated")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Challenge active.")
		fmt.Fprintln(out, "The root cause is intentionally hidden.")
		fmt.Fprintln(out, "Start with the customer symptom and investigate the system path.")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "When you are finished, run 'lab reset' to restore the baseline.")

		return nil
	},
}

func selectChallengeScenario(override string) (string, error) {
	if override != "" {
		for _, candidate := range challengeEligibleScenarios {
			if candidate == override {
				return candidate, nil
			}
		}

		return "", fmt.Errorf("scenario %q is not eligible for challenge mode", override)
	}

	max := big.NewInt(int64(len(challengeEligibleScenarios)))
	index, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", fmt.Errorf("select challenge incident: %w", err)
	}

	return challengeEligibleScenarios[index.Int64()], nil
}

func init() {
	challengeCmd.Flags().StringVar(
		&challengeScenarioOverride,
		"scenario",
		"",
		"force a challenge scenario (for maintainers and automated tests)",
	)

	// Keep the deterministic override out of normal student help output.
	_ = challengeCmd.Flags().MarkHidden("scenario")

	rootCmd.AddCommand(challengeCmd)
}
