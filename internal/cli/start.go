package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/gopher-opsx/platform-lab-cli/internal/compose"
	"github.com/gopher-opsx/platform-lab-cli/internal/platform"
	"github.com/gopher-opsx/platform-lab-cli/internal/scenario"
	"github.com/gopher-opsx/platform-lab-cli/internal/state"
)

const restartLoopOverride = `services:
  catalog-service:
    restart: always
    environment:
      DATABASE_URL: "not-a-valid-postgres-url"
`

const runningButDeadOverride = `services:
  catalog-service:
    entrypoint:
      - /bin/sh
      - -c
      - |
        echo "Lab scenario active: Catalog application is not started"
        exec sleep 3600
`

var startCmd = &cobra.Command{
	Use:   "start <scenario>",
	Short: "Start a controlled incident scenario",
	Args:  cobra.ExactArgs(1),

	RunE: func(cmd *cobra.Command, args []string) error {
		scenarioID := args[0]

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

		fmt.Fprintf(
			out,
			"Starting scenario: %s\n\n",
			s.Name,
		)

		switch s.ID {

		case "restart-loop":
			return startRestartLoop(
				out,
				root,
				client,
				s.ID,
			)
		case "running-but-dead":
			return startRunningButDead(
				out,
				root,
				client,
				s.ID,
			)

		case "redis-down":
			return startRedisDown(
				out,
				root,
				client,
				s.ID,
			)

		default:
			return fmt.Errorf(
				"scenario %q is defined but not implemented yet",
				s.ID,
			)
		}
	},
}

func startRestartLoop(
	out interface {
		Write([]byte) (int, error)
	},
	root string,
	client compose.Client,
	scenarioID string,
) error {

	running, err := client.IsRunning("catalog-service")
	if err != nil {
		return fmt.Errorf(
			"inspect Catalog state: %w",
			err,
		)
	}

	if !running {
		return fmt.Errorf(
			"catalog-service is not running; start Platform Lab before starting this scenario",
		)
	}

	fmt.Fprintln(out, "✓ Catalog is running")

	session := state.Session{
		Scenario:  scenarioID,
		StartedAt: time.Now(),
		Changes: []state.Change{
			{
				Type:          "compose_override",
				Target:        "catalog-service",
				OriginalState: "running",
				File:          ".lab/compose.override.yaml",
			},
		},
	}

	/*
		Save recovery information before changing
		the running environment.
	*/
	if err := state.Save(root, session); err != nil {
		return err
	}

	fmt.Fprintln(out, "✓ Recovery state recorded")

	if err := compose.WriteOverride(
		root,
		restartLoopOverride,
	); err != nil {

		_ = state.Clear(root)

		return err
	}

	fmt.Fprintln(out, "✓ Failure configuration prepared")

	if err := client.RecreateWithOverride(
		"catalog-service",
	); err != nil {

		_ = compose.RemoveOverride(root)
		_ = state.Clear(root)

		return err
	}

	fmt.Fprintln(out, "✓ Catalog failure injected")

	fmt.Fprintln(
		out,
		"Waiting for restart behavior...",
	)

	if err := client.WaitForRestart(
		"catalog-service",
		15*time.Second,
	); err != nil {

		/*
			The injection did not produce the incident we promised.

			Try to return the environment to baseline rather
			than leaving the student with a partial scenario.
		*/
		_ = compose.RemoveOverride(root)
		_ = client.RecreateBaseline("catalog-service")

		/*
			Only clear state if our cleanup succeeded enough
			to return Catalog to health.
		*/
		if healthErr := client.WaitForHealthy(
			"catalog-service",
			30*time.Second,
		); healthErr == nil {
			_ = state.Clear(root)
		}

		return fmt.Errorf(
			"restart-loop verification failed: %w",
			err,
		)
	}

	fmt.Fprintln(
		out,
		"✓ Restart loop verified",
	)

	fmt.Fprintln(out)
	fmt.Fprintln(
		out,
		"Incident active. Begin troubleshooting from the customer symptom.",
	)

	return nil
}

func startRedisDown(
	out interface {
		Write([]byte) (int, error)
	},
	root string,
	client compose.Client,
	scenarioID string,
) error {

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
		Scenario:  scenarioID,
		StartedAt: time.Now(),
		Changes: []state.Change{
			{
				Type:          "service_state",
				Target:        "redis",
				OriginalState: "running",
			},
		},
	}

	if err := state.Save(root, session); err != nil {
		return err
	}

	fmt.Fprintln(out, "✓ Recovery state recorded")

	if err := client.Stop("redis"); err != nil {
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
}

func startRunningButDead(
	out interface {
		Write([]byte) (int, error)
	},
	root string,
	client compose.Client,
	scenarioID string,
) error {

	running, err := client.IsRunning("catalog-service")
	if err != nil {
		return fmt.Errorf(
			"inspect Catalog state: %w",
			err,
		)
	}

	if !running {
		return fmt.Errorf(
			"catalog-service is not running; start Platform Lab before starting this scenario",
		)
	}

	health, err := client.HealthStatus("catalog-service")
	if err != nil {
		return fmt.Errorf(
			"inspect Catalog health: %w",
			err,
		)
	}

	if health != "healthy" {
		return fmt.Errorf(
			"catalog-service is not healthy before scenario start; current health: %s",
			health,
		)
	}

	fmt.Fprintln(
		out,
		"✓ Catalog baseline is running and healthy",
	)

	session := state.Session{
		Scenario:  scenarioID,
		StartedAt: time.Now(),
		Changes: []state.Change{
			{
				Type:          "compose_override",
				Target:        "catalog-service",
				OriginalState: "running",
				File:          ".lab/compose.override.yaml",
			},
		},
	}

	if err := state.Save(root, session); err != nil {
		return err
	}

	fmt.Fprintln(
		out,
		"✓ Recovery state recorded",
	)

	if err := compose.WriteOverride(
		root,
		runningButDeadOverride,
	); err != nil {

		_ = state.Clear(root)

		return err
	}

	fmt.Fprintln(
		out,
		"✓ Failure configuration prepared",
	)

	if err := client.RecreateWithOverride(
		"catalog-service",
	); err != nil {

		_ = compose.RemoveOverride(root)
		_ = state.Clear(root)

		return err
	}

	fmt.Fprintln(
		out,
		"✓ Controlled failure injected",
	)

	/*
		The defining property of this incident is:

		Container = running
		Application = unavailable

		First prove Docker still considers the
		container running.
	*/
	running, err = client.IsRunning("catalog-service")
	if err != nil {
		return err
	}

	if !running {
		return fmt.Errorf(
			"scenario verification failed: catalog-service is not running",
		)
	}

	fmt.Fprintln(
		out,
		"✓ Catalog container remains running",
	)

	fmt.Fprintln(
		out,
		"Waiting for Catalog application to become unavailable...",
	)

	/*
		Lesson 32 is about the difference between container state
		and application availability. Verify the application boundary
		directly instead of waiting for Docker's health-check state
		to transition from starting to unhealthy.
	*/
	if err := platform.WaitForHTTPUnavailable(
		"http://localhost:8081/healthz",
		10*time.Second,
	); err != nil {

		/*
			Scenario did not reach the state promised
			by Lesson 32. Attempt automatic rollback.
		*/
		_ = compose.RemoveOverride(root)
		_ = client.RecreateBaseline("catalog-service")

		if healthErr := client.WaitForHealthy(
			"catalog-service",
			30*time.Second,
		); healthErr == nil {
			_ = state.Clear(root)
		}

		return fmt.Errorf(
			"running-but-dead verification failed: %w",
			err,
		)
	}

	fmt.Fprintln(
		out,
		"✓ Catalog application is unavailable",
	)

	fmt.Fprintln(out)

	fmt.Fprintln(
		out,
		"Incident active. Begin troubleshooting from the customer symptom.",
	)

	return nil
}

func init() {
	rootCmd.AddCommand(startCmd)
}
