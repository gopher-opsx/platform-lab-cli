package cli

import (
	"fmt"
	"net/http"
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

const badEnvironmentOverride = `services:
  catalog-service:
    environment:
      DATABASE_URL: "postgres://platform:platform@wrong-postgres:5432/catalog_db?sslmode=disable"
`

const dnsFailureOverride = `services:
  catalog-service:
    entrypoint:
      - /bin/sh
      - -c
      - |
        printf 'nameserver 192.0.2.1\\noptions ndots:0\\n' > /etc/resolv.conf
        exec /service
`

const portFailureOverride = `services:
  web-bff:
    ports: !override
      - "18080:8080"
`

const dependencyNotReadyOverride = `services:
  postgres:
    entrypoint:
      - /bin/sh
      - -c
      - |
        echo "Lab: PostgreSQL dependency is temporarily not ready"
        sleep 20
        exec docker-entrypoint.sh postgres
`

const diskGrowthOverride = `services:
  catalog-service:
    environment:
      PLATFORM_LAB_DISK_GROWTH: "true"
`

const cpuPressureOverride = `services:
  catalog-service:
    environment:
      PLATFORM_LAB_CPU_PRESSURE: "true"
`

const memoryGrowthOverride = `services:
  catalog-service:
    environment:
      PLATFORM_LAB_MEMORY_GROWTH: "true"
`

const oomKillOverride = `services:
  catalog-service:
    environment:
      PLATFORM_LAB_OOM_PRESSURE: "true"
    mem_limit: 64m
    restart: "no"
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

		case "bad-environment":
			return startBadEnvironment(
				out,
				root,
				client,
				s.ID,
			)

		case "dns-failure":
			return startDNSFailure(
				out,
				root,
				client,
				s.ID,
			)

		case "port-failure":
			return startPortFailure(
				out,
				root,
				client,
				s.ID,
			)

		case "dependency-not-ready":
			return startDependencyNotReady(
				out,
				root,
				client,
				s.ID,
			)

		case "lost-persistence":
			return startLostPersistence(
				out,
				root,
				client,
				s.ID,
			)

		case "disk-growth":
			return startDiskGrowth(
				out,
				root,
				client,
				s.ID,
			)

		case "cpu-pressure":
			return startCPUPressure(
				out,
				root,
				client,
				s.ID,
			)

		case "memory-growth":
			return startMemoryGrowth(
				out,
				root,
				client,
				s.ID,
			)

		case "oom-kill":
			return startOOMKill(
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

func startBadEnvironment(
	out interface {
		Write([]byte) (int, error)
	},
	root string,
	client compose.Client,
	scenarioID string,
) error {

	/*
		Lesson 33 begins from a healthy Catalog.

		The incident must not be confused with:
		- Lesson 31: Catalog process exits and restarts.
		- Lesson 32: container runs but Catalog process is absent.

		Here Catalog must remain alive while receiving incorrect
		runtime dependency configuration.
	*/
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

	postgresRunning, err := client.IsRunning("postgres")
	if err != nil {
		return fmt.Errorf(
			"inspect PostgreSQL state: %w",
			err,
		)
	}

	if !postgresRunning {
		return fmt.Errorf(
			"postgres is not running; start Platform Lab before starting this scenario",
		)
	}

	fmt.Fprintln(
		out,
		"✓ Catalog and PostgreSQL baseline are running",
	)

	/*
		Record recovery information before touching the
		running Platform Lab environment.
	*/
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

	/*
		Inject a syntactically valid DATABASE_URL.

		The URL itself is valid, so pgxpool can be created and
		Catalog can start its HTTP server.

		The dependency hostname is intentionally incorrect:
		    wrong-postgres

		This creates a runtime configuration incident rather
		than a malformed-configuration startup failure.
	*/
	if err := compose.WriteOverride(
		root,
		badEnvironmentOverride,
	); err != nil {

		_ = state.Clear(root)

		return err
	}

	fmt.Fprintln(
		out,
		"✓ Incorrect runtime configuration prepared",
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
		"✓ Catalog recreated with controlled configuration failure",
	)

	/*
		The first defining property of Lesson 33 is that the
		Catalog container still runs.

		If it exits, we have accidentally created another
		Lesson 31-style startup failure.
	*/
	running, err = client.IsRunning("catalog-service")
	if err != nil {
		return err
	}

	if !running {
		_ = compose.RemoveOverride(root)
		_ = client.RecreateBaseline("catalog-service")

		if healthErr := client.WaitForHealthy(
			"catalog-service",
			30*time.Second,
		); healthErr == nil {
			_ = state.Clear(root)
		}

		return fmt.Errorf(
			"bad-environment verification failed: catalog-service did not remain running",
		)
	}

	fmt.Fprintln(
		out,
		"✓ Catalog container remains running",
	)

	/*
		The second defining property is application liveness.

		/healthz represents the Catalog process itself, not
		PostgreSQL readiness.

		WaitForHTTPUnavailable is therefore not appropriate
		for this scenario: /healthz should remain available.
	*/
	fmt.Fprintln(
		out,
		"Waiting for Catalog application to become live...",
	)

	if err := platform.WaitForHTTPStatus(
		"http://localhost:8081/healthz",
		http.StatusOK,
		15*time.Second,
	); err != nil {

		_ = compose.RemoveOverride(root)
		_ = client.RecreateBaseline("catalog-service")

		if healthErr := client.WaitForHealthy(
			"catalog-service",
			30*time.Second,
		); healthErr == nil {
			_ = state.Clear(root)
		}

		return fmt.Errorf(
			"bad-environment liveness verification failed: %w",
			err,
		)
	}

	fmt.Fprintln(
		out,
		"✓ Catalog application remains alive",
	)

	fmt.Fprintln(
		out,
		"Waiting for PostgreSQL readiness to fail...",
	)

	/*
		A readiness failure does not have to return a specific HTTP
		status. With an unreachable database target, /readyz may
		return non-2xx or the request may time out.

		Either result proves that readiness is not successful.
	*/
	if err := platform.WaitForHTTPFailure(
		"http://localhost:8081/readyz",
		15*time.Second,
	); err != nil {
		_ = compose.RemoveOverride(root)
		_ = client.RecreateBaseline("catalog-service")

		if healthErr := client.WaitForHealthy(
			"catalog-service",
			30*time.Second,
		); healthErr == nil {
			_ = state.Clear(root)
		}

		return fmt.Errorf(
			"bad-environment readiness verification failed: %w",
			err,
		)
	}

	fmt.Fprintln(
		out,
		"✓ PostgreSQL readiness failure verified",
	)

	running, err = client.IsRunning("catalog-service")
	if err != nil {
		return err
	}

	if !running {
		return fmt.Errorf(
			"bad-environment verification failed: catalog-service stopped",
		)
	}

	fmt.Fprintln(
		out,
		"✓ Catalog remains running while dependency readiness fails",
	)

	fmt.Fprintln(out)

	fmt.Fprintln(
		out,
		"Incident active. Begin troubleshooting from the customer symptom.",
	)

	return nil
}

func startDNSFailure(
	out interface {
		Write([]byte) (int, error)
	},
	root string,
	client compose.Client,
	scenarioID string,
) error {
	running, err := client.IsRunning("catalog-service")
	if err != nil {
		return fmt.Errorf("inspect Catalog state: %w", err)
	}
	if !running {
		return fmt.Errorf("catalog-service is not running; start Platform Lab before starting this scenario")
	}

	health, err := client.HealthStatus("catalog-service")
	if err != nil {
		return fmt.Errorf("inspect Catalog health: %w", err)
	}
	if health != "healthy" {
		return fmt.Errorf("catalog-service is not healthy before scenario start; current health: %s", health)
	}

	postgresRunning, err := client.IsRunning("postgres")
	if err != nil {
		return fmt.Errorf("inspect PostgreSQL state: %w", err)
	}
	if !postgresRunning {
		return fmt.Errorf("postgres is not running; start Platform Lab before starting this scenario")
	}

	fmt.Fprintln(out, "✓ Catalog and PostgreSQL baseline are running")

	session := state.Session{
		Scenario:  scenarioID,
		StartedAt: time.Now(),
		Changes: []state.Change{
			{
				Type:   "compose_override",
				Target: "catalog-service",
				File:   compose.OverridePath(root),
			},
		},
	}
	if err := state.Save(root, session); err != nil {
		return err
	}
	fmt.Fprintln(out, "✓ Recovery state recorded")

	rollback := func() {
		_ = compose.RemoveOverride(root)
		_ = client.RecreateBaseline("catalog-service")
		if err := client.WaitForHealthy("catalog-service", 30*time.Second); err == nil {
			_ = state.Clear(root)
		}
	}

	if err := compose.WriteOverride(root, dnsFailureOverride); err != nil {
		_ = state.Clear(root)
		return err
	}
	fmt.Fprintln(out, "✓ Controlled DNS failure prepared")

	if err := client.RecreateWithOverride("catalog-service"); err != nil {
		rollback()
		return err
	}
	fmt.Fprintln(out, "✓ Catalog recreated with DNS failure active")

	running, err = client.IsRunning("catalog-service")
	if err != nil {
		rollback()
		return err
	}
	if !running {
		rollback()
		return fmt.Errorf("dns-failure verification failed: catalog-service stopped")
	}
	fmt.Fprintln(out, "✓ Catalog container remains running")

	fmt.Fprintln(out, "Waiting for Catalog application to become live...")
	if err := platform.WaitForHTTPStatus(
		"http://localhost:8081/healthz",
		http.StatusOK,
		15*time.Second,
	); err != nil {
		rollback()
		return fmt.Errorf("dns-failure liveness verification failed: %w", err)
	}
	fmt.Fprintln(out, "✓ Catalog application remains alive")

	fmt.Fprintln(out, "Waiting for PostgreSQL readiness to fail...")
	if err := platform.WaitForHTTPFailure(
		"http://localhost:8081/readyz",
		15*time.Second,
	); err != nil {
		rollback()
		return fmt.Errorf("dns-failure readiness verification failed: %w", err)
	}

	fmt.Fprintln(out, "✓ PostgreSQL readiness failure verified")
	fmt.Fprintln(out, "✓ DATABASE_URL remains configured for postgres:5432")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Incident active. Begin troubleshooting from the customer symptom.")
	return nil
}

func startPortFailure(
	out interface {
		Write([]byte) (int, error)
	},
	root string,
	client compose.Client,
	scenarioID string,
) error {
	running, err := client.IsRunning("web-bff")
	if err != nil {
		return fmt.Errorf("inspect Web BFF state: %w", err)
	}
	if !running {
		return fmt.Errorf("web-bff is not running; start Platform Lab before starting this scenario")
	}

	health, err := client.HealthStatus("web-bff")
	if err != nil {
		return fmt.Errorf("inspect Web BFF health: %w", err)
	}
	if health != "healthy" {
		return fmt.Errorf("web-bff is not healthy before scenario start; current health: %s", health)
	}

	if err := platform.WaitForHTTPStatus(
		"http://localhost:8080/healthz",
		http.StatusOK,
		10*time.Second,
	); err != nil {
		return fmt.Errorf("web-bff host baseline verification failed: %w", err)
	}

	fmt.Fprintln(out, "✓ Web BFF baseline is running, healthy, and reachable on localhost:8080")

	session := state.Session{
		Scenario:  scenarioID,
		StartedAt: time.Now(),
		Changes: []state.Change{
			{
				Type:          "compose_override",
				Target:        "web-bff",
				OriginalState: "running",
				File:          compose.OverridePath(root),
			},
		},
	}
	if err := state.Save(root, session); err != nil {
		return err
	}
	fmt.Fprintln(out, "✓ Recovery state recorded")

	rollback := func() {
		_ = compose.RemoveOverride(root)
		_ = client.RecreateBaseline("web-bff")
		if err := client.WaitForHealthy("web-bff", 30*time.Second); err == nil {
			_ = state.Clear(root)
		}
	}

	if err := compose.WriteOverride(root, portFailureOverride); err != nil {
		_ = state.Clear(root)
		return err
	}
	fmt.Fprintln(out, "✓ Controlled host port failure prepared")

	if err := client.RecreateWithOverride("web-bff"); err != nil {
		rollback()
		return err
	}
	fmt.Fprintln(out, "✓ Web BFF recreated with incorrect host port exposure")

	running, err = client.IsRunning("web-bff")
	if err != nil {
		rollback()
		return err
	}
	if !running {
		rollback()
		return fmt.Errorf("port-failure verification failed: web-bff stopped")
	}

	if err := client.WaitForHealthy("web-bff", 30*time.Second); err != nil {
		rollback()
		return fmt.Errorf("port-failure internal health verification failed: %w", err)
	}
	fmt.Fprintln(out, "✓ Web BFF remains healthy inside the container")

	if err := platform.WaitForHTTPUnavailable(
		"http://localhost:8080/healthz",
		10*time.Second,
	); err != nil {
		rollback()
		return fmt.Errorf("port-failure expected-port verification failed: %w", err)
	}
	fmt.Fprintln(out, "✓ Expected host port localhost:8080 is unavailable")

	if err := platform.WaitForHTTPStatus(
		"http://localhost:18080/healthz",
		http.StatusOK,
		10*time.Second,
	); err != nil {
		rollback()
		return fmt.Errorf("port-failure alternate-port verification failed: %w", err)
	}
	fmt.Fprintln(out, "✓ Web BFF is reachable through the incorrectly published host port 18080")

	fmt.Fprintln(out)
	fmt.Fprintln(out, "Incident active. Begin troubleshooting from the customer symptom.")
	return nil
}

func startDependencyNotReady(
	out interface {
		Write([]byte) (int, error)
	},
	root string,
	client compose.Client,
	scenarioID string,
) error {
	postgresRunning, err := client.IsRunning("postgres")
	if err != nil {
		return fmt.Errorf("inspect PostgreSQL state: %w", err)
	}
	if !postgresRunning {
		return fmt.Errorf("postgres is not running; start Platform Lab before starting this scenario")
	}

	postgresHealth, err := client.HealthStatus("postgres")
	if err != nil {
		return fmt.Errorf("inspect PostgreSQL health: %w", err)
	}
	if postgresHealth != "healthy" {
		return fmt.Errorf("postgres is not healthy before scenario start; current health: %s", postgresHealth)
	}

	catalogRunning, err := client.IsRunning("catalog-service")
	if err != nil {
		return fmt.Errorf("inspect Catalog state: %w", err)
	}
	if !catalogRunning {
		return fmt.Errorf("catalog-service is not running; start Platform Lab before starting this scenario")
	}

	if err := platform.WaitForHTTPStatus(
		"http://localhost:8081/readyz",
		http.StatusOK,
		10*time.Second,
	); err != nil {
		return fmt.Errorf("Catalog readiness baseline verification failed: %w", err)
	}

	fmt.Fprintln(out, "✓ Catalog and PostgreSQL baseline are healthy and ready")

	session := state.Session{
		Scenario:  scenarioID,
		StartedAt: time.Now(),
		Changes: []state.Change{
			{
				Type:          "compose_override",
				Target:        "postgres",
				OriginalState: "running",
				File:          compose.OverridePath(root),
			},
		},
	}
	if err := state.Save(root, session); err != nil {
		return err
	}
	fmt.Fprintln(out, "✓ Recovery state recorded")

	rollback := func() {
		_ = compose.RemoveOverride(root)
		_ = client.RecreateBaseline("postgres")
		if err := client.WaitForHealthy("postgres", 30*time.Second); err == nil {
			_ = state.Clear(root)
		}
	}

	if err := compose.WriteOverride(root, dependencyNotReadyOverride); err != nil {
		_ = state.Clear(root)
		return err
	}
	fmt.Fprintln(out, "✓ Controlled PostgreSQL readiness delay prepared")

	if err := client.RecreateWithOverride("postgres"); err != nil {
		rollback()
		return err
	}
	fmt.Fprintln(out, "✓ PostgreSQL recreated with delayed application startup")

	postgresRunning, err = client.IsRunning("postgres")
	if err != nil {
		rollback()
		return err
	}
	if !postgresRunning {
		rollback()
		return fmt.Errorf("dependency-not-ready verification failed: postgres container stopped")
	}
	fmt.Fprintln(out, "✓ PostgreSQL container is running")

	if err := platform.WaitForHTTPFailure(
		"http://localhost:8081/readyz",
		10*time.Second,
	); err != nil {
		rollback()
		return fmt.Errorf("dependency-not-ready verification failed: Catalog did not become not-ready: %w", err)
	}
	fmt.Fprintln(out, "✓ Catalog readiness fails while PostgreSQL is not ready")

	if err := platform.WaitForHTTPStatus(
		"http://localhost:8081/healthz",
		http.StatusOK,
		5*time.Second,
	); err != nil {
		rollback()
		return fmt.Errorf("dependency-not-ready verification failed: Catalog application is not alive: %w", err)
	}
	fmt.Fprintln(out, "✓ Catalog application remains alive")

	fmt.Fprintln(out, "Waiting for PostgreSQL to become ready...")
	if err := client.WaitForHealthy("postgres", 40*time.Second); err != nil {
		rollback()
		return fmt.Errorf("PostgreSQL did not recover from the controlled readiness delay: %w", err)
	}
	fmt.Fprintln(out, "✓ PostgreSQL became healthy")

	if err := platform.WaitForHTTPStatus(
		"http://localhost:8081/readyz",
		http.StatusOK,
		15*time.Second,
	); err != nil {
		rollback()
		return fmt.Errorf("Catalog readiness did not recover after PostgreSQL became ready: %w", err)
	}
	fmt.Fprintln(out, "✓ Catalog readiness recovered without changing Catalog configuration")

	fmt.Fprintln(out)
	fmt.Fprintln(out, "Incident observed and dependency recovery verified. Run lab reset to restore the baseline configuration.")
	return nil
}

func startDiskGrowth(
	out interface {
		Write([]byte) (int, error)
	},
	root string,
	client compose.Client,
	scenarioID string,
) error {
	running, err := client.IsRunning("catalog-service")
	if err != nil {
		return fmt.Errorf("inspect Catalog state: %w", err)
	}
	if !running {
		return fmt.Errorf("catalog-service is not running; start Platform Lab before starting this scenario")
	}

	health, err := client.HealthStatus("catalog-service")
	if err != nil {
		return fmt.Errorf("inspect Catalog health: %w", err)
	}
	if health != "healthy" {
		return fmt.Errorf("catalog-service is not healthy before scenario start; current health: %s", health)
	}

	if err := platform.WaitForHTTPStatus(
		"http://localhost:8081/readyz",
		http.StatusOK,
		10*time.Second,
	); err != nil {
		return fmt.Errorf("Catalog readiness baseline verification failed: %w", err)
	}

	fmt.Fprintln(out, "✓ Catalog baseline is running, healthy, and ready")

	session := state.Session{
		Scenario:  scenarioID,
		StartedAt: time.Now(),
		Changes: []state.Change{
			{
				Type:          "compose_override",
				Target:        "catalog-service",
				OriginalState: "running",
				File:          compose.OverridePath(root),
			},
		},
	}
	if err := state.Save(root, session); err != nil {
		return err
	}
	fmt.Fprintln(out, "✓ Recovery state recorded")

	rollback := func() {
		_ = compose.RemoveOverride(root)
		_ = client.RecreateBaseline("catalog-service")
		if err := client.WaitForHealthy("catalog-service", 30*time.Second); err == nil {
			_ = state.Clear(root)
		}
	}

	if err := compose.WriteOverride(root, diskGrowthOverride); err != nil {
		_ = state.Clear(root)
		return err
	}
	fmt.Fprintln(out, "✓ Controlled disk-growth behavior prepared")

	if err := client.RecreateWithOverride("catalog-service"); err != nil {
		rollback()
		return err
	}
	if err := client.WaitForHealthy("catalog-service", 30*time.Second); err != nil {
		rollback()
		return fmt.Errorf("disk-growth Catalog startup failed: %w", err)
	}
	fmt.Fprintln(out, "✓ Catalog recreated with disk-growth incident armed")

	if err := platform.WaitForHTTPStatus(
		"http://localhost:8081/readyz",
		http.StatusOK,
		10*time.Second,
	); err != nil {
		rollback()
		return fmt.Errorf("disk-growth readiness verification failed: %w", err)
	}
	fmt.Fprintln(out, "✓ Catalog remains healthy and ready")

	// Verify that the scenario flag reached the container, but deliberately
	// do not generate workload here. Lesson 38 begins with a clean storage
	// baseline; the student must correlate later growth with controlled
	// application requests.
	if err := client.Exec(
		"catalog-service",
		"sh",
		"-c",
		`test "$PLATFORM_LAB_DISK_GROWTH" = "true"`,
	); err != nil {
		rollback()
		return fmt.Errorf("disk-growth verification failed: controlled behavior was not armed: %w", err)
	}
	fmt.Fprintln(out, "✓ Controlled workload trigger is armed")

	fmt.Fprintln(out)
	fmt.Fprintln(out, "Incident active. Capture a storage baseline, then generate controlled Catalog workload.")
	fmt.Fprintln(out, "Run lab reset when the investigation is complete.")
	return nil
}

func startCPUPressure(
	out interface {
		Write([]byte) (int, error)
	},
	root string,
	client compose.Client,
	scenarioID string,
) error {
	running, err := client.IsRunning("catalog-service")
	if err != nil {
		return fmt.Errorf("inspect Catalog state: %w", err)
	}
	if !running {
		return fmt.Errorf("catalog-service is not running; start Platform Lab before starting this scenario")
	}

	health, err := client.HealthStatus("catalog-service")
	if err != nil {
		return fmt.Errorf("inspect Catalog health: %w", err)
	}
	if health != "healthy" {
		return fmt.Errorf("catalog-service is not healthy before scenario start; current health: %s", health)
	}

	if err := platform.WaitForHTTPStatus(
		"http://localhost:8081/readyz",
		http.StatusOK,
		10*time.Second,
	); err != nil {
		return fmt.Errorf("Catalog readiness baseline verification failed: %w", err)
	}
	fmt.Fprintln(out, "✓ Catalog baseline is running, healthy, and ready")

	session := state.Session{
		Scenario:  scenarioID,
		StartedAt: time.Now(),
		Changes: []state.Change{
			{
				Type:          "compose_override",
				Target:        "catalog-service",
				OriginalState: "running",
				File:          compose.OverridePath(root),
			},
		},
	}
	if err := state.Save(root, session); err != nil {
		return err
	}
	fmt.Fprintln(out, "✓ Recovery state recorded")

	rollback := func() {
		_ = compose.RemoveOverride(root)
		_ = client.RecreateBaseline("catalog-service")
		if err := client.WaitForHealthy("catalog-service", 30*time.Second); err == nil {
			_ = state.Clear(root)
		}
	}

	if err := compose.WriteOverride(root, cpuPressureOverride); err != nil {
		_ = state.Clear(root)
		return err
	}
	fmt.Fprintln(out, "✓ Controlled CPU-pressure behavior prepared")

	if err := client.RecreateWithOverride("catalog-service"); err != nil {
		rollback()
		return err
	}
	if err := client.WaitForHealthy("catalog-service", 30*time.Second); err != nil {
		rollback()
		return fmt.Errorf("cpu-pressure Catalog startup failed: %w", err)
	}
	fmt.Fprintln(out, "✓ Catalog recreated with CPU-pressure incident armed")

	if err := platform.WaitForHTTPStatus(
		"http://localhost:8081/readyz",
		http.StatusOK,
		10*time.Second,
	); err != nil {
		rollback()
		return fmt.Errorf("cpu-pressure readiness verification failed: %w", err)
	}
	fmt.Fprintln(out, "✓ Catalog remains healthy and ready")

	// Verify only that the hook is armed. Do not generate workload here:
	// Lesson 39 must begin with a clean pre-load CPU baseline.
	if err := client.Exec(
		"catalog-service",
		"sh",
		"-c",
		`test "$PLATFORM_LAB_CPU_PRESSURE" = "true"`,
	); err != nil {
		rollback()
		return fmt.Errorf("cpu-pressure verification failed: controlled behavior was not armed: %w", err)
	}
	fmt.Fprintln(out, "✓ Controlled workload trigger is armed")

	fmt.Fprintln(out)
	fmt.Fprintln(out, "Incident active. Capture a CPU baseline, then generate controlled Catalog workload.")
	fmt.Fprintln(out, "Keep docker stats visible while the workload runs.")
	fmt.Fprintln(out, "Run lab reset when the investigation is complete.")
	return nil
}

func startMemoryGrowth(
	out interface {
		Write([]byte) (int, error)
	},
	root string,
	client compose.Client,
	scenarioID string,
) error {
	running, err := client.IsRunning("catalog-service")
	if err != nil {
		return fmt.Errorf("inspect Catalog state: %w", err)
	}
	if !running {
		return fmt.Errorf("catalog-service is not running; start Platform Lab before starting this scenario")
	}

	health, err := client.HealthStatus("catalog-service")
	if err != nil {
		return fmt.Errorf("inspect Catalog health: %w", err)
	}
	if health != "healthy" {
		return fmt.Errorf(
			"catalog-service is not healthy before scenario start; current health: %s",
			health,
		)
	}

	if err := platform.WaitForHTTPStatus(
		"http://localhost:8081/readyz",
		http.StatusOK,
		10*time.Second,
	); err != nil {
		return fmt.Errorf(
			"Catalog readiness baseline verification failed: %w",
			err,
		)
	}

	fmt.Fprintln(
		out,
		"✓ Catalog baseline is running, healthy, and ready",
	)

	session := state.Session{
		Scenario:  scenarioID,
		StartedAt: time.Now(),
		Changes: []state.Change{
			{
				Type:          "compose_override",
				Target:        "catalog-service",
				OriginalState: "running",
				File:          compose.OverridePath(root),
			},
		},
	}

	if err := state.Save(root, session); err != nil {
		return err
	}

	fmt.Fprintln(out, "✓ Recovery state recorded")

	rollback := func() {
		_ = compose.RemoveOverride(root)
		_ = client.RecreateBaseline("catalog-service")

		if err := client.WaitForHealthy(
			"catalog-service",
			30*time.Second,
		); err == nil {
			_ = state.Clear(root)
		}
	}

	if err := compose.WriteOverride(
		root,
		memoryGrowthOverride,
	); err != nil {
		_ = state.Clear(root)
		return err
	}

	fmt.Fprintln(
		out,
		"✓ Controlled memory-growth behavior prepared",
	)

	if err := client.RecreateWithOverride(
		"catalog-service",
	); err != nil {
		rollback()
		return err
	}

	if err := client.WaitForHealthy(
		"catalog-service",
		30*time.Second,
	); err != nil {
		rollback()

		return fmt.Errorf(
			"memory-growth Catalog startup failed: %w",
			err,
		)
	}

	fmt.Fprintln(
		out,
		"✓ Catalog recreated with memory-growth incident armed",
	)

	if err := platform.WaitForHTTPStatus(
		"http://localhost:8081/readyz",
		http.StatusOK,
		10*time.Second,
	); err != nil {
		rollback()

		return fmt.Errorf(
			"memory-growth readiness verification failed: %w",
			err,
		)
	}

	fmt.Fprintln(
		out,
		"✓ Catalog remains healthy and ready",
	)

	/*
		Verify only that the controlled memory-growth hook
		reached the container.

		Do not generate workload here.

		Lesson 40 must begin with a clean memory baseline so
		the student can correlate later memory growth with
		controlled application requests.
	*/
	if err := client.Exec(
		"catalog-service",
		"sh",
		"-c",
		`test "$PLATFORM_LAB_MEMORY_GROWTH" = "true"`,
	); err != nil {
		rollback()

		return fmt.Errorf(
			"memory-growth verification failed: controlled behavior was not armed: %w",
			err,
		)
	}

	fmt.Fprintln(
		out,
		"✓ Controlled workload trigger is armed",
	)

	fmt.Fprintln(out)
	fmt.Fprintln(
		out,
		"Incident active. Capture a memory baseline, then generate controlled Catalog workload.",
	)
	fmt.Fprintln(
		out,
		"Repeat the workload and compare memory usage between runs.",
	)
	fmt.Fprintln(
		out,
		"Run lab reset when the investigation is complete.",
	)

	return nil
}

func lostPersistenceOverride(volumeName string) string {
	return fmt.Sprintf(`services:
  postgres:
    volumes: !override
      - type: volume
        source: %s
        target: /var/lib/postgresql/data
volumes:
  %s:
    external: true
    name: %s
`, volumeName, volumeName, volumeName)
}

func startOOMKill(
	out interface {
		Write([]byte) (int, error)
	},
	root string,
	client compose.Client,
	scenarioID string,
) error {
	running, err := client.IsRunning("catalog-service")
	if err != nil {
		return fmt.Errorf("inspect Catalog state: %w", err)
	}
	if !running {
		return fmt.Errorf(
			"catalog-service is not running; start Platform Lab before starting this scenario",
		)
	}

	health, err := client.HealthStatus("catalog-service")
	if err != nil {
		return fmt.Errorf("inspect Catalog health: %w", err)
	}
	if health != "healthy" {
		return fmt.Errorf(
			"catalog-service is not healthy before scenario start; current health: %s",
			health,
		)
	}

	if err := platform.WaitForHTTPStatus(
		"http://localhost:8081/readyz",
		http.StatusOK,
		10*time.Second,
	); err != nil {
		return fmt.Errorf(
			"Catalog readiness baseline verification failed: %w",
			err,
		)
	}

	fmt.Fprintln(
		out,
		"✓ Catalog baseline is running, healthy, and ready",
	)

	session := state.Session{
		Scenario:  scenarioID,
		StartedAt: time.Now(),
		Changes: []state.Change{
			{
				Type:          "compose_override",
				Target:        "catalog-service",
				OriginalState: "running",
				File:          compose.OverridePath(root),
			},
		},
	}

	if err := state.Save(root, session); err != nil {
		return err
	}

	fmt.Fprintln(out, "✓ Recovery state recorded")

	rollback := func() {
		_ = compose.RemoveOverride(root)
		_ = client.RecreateBaseline("catalog-service")

		if err := client.WaitForHealthy(
			"catalog-service",
			30*time.Second,
		); err == nil {
			_ = state.Clear(root)
		}
	}

	if err := compose.WriteOverride(
		root,
		oomKillOverride,
	); err != nil {
		_ = state.Clear(root)
		return err
	}

	fmt.Fprintln(
		out,
		"✓ Controlled memory limit prepared",
	)

	if err := client.RecreateWithOverride(
		"catalog-service",
	); err != nil {
		rollback()
		return err
	}

	if err := client.WaitForHealthy(
		"catalog-service",
		30*time.Second,
	); err != nil {
		rollback()

		return fmt.Errorf(
			"OOM-kill Catalog startup failed: %w",
			err,
		)
	}

	fmt.Fprintln(
		out,
		"✓ Catalog recreated with OOM incident armed",
	)

	if err := platform.WaitForHTTPStatus(
		"http://localhost:8081/readyz",
		http.StatusOK,
		10*time.Second,
	); err != nil {
		rollback()

		return fmt.Errorf(
			"OOM-kill readiness verification failed: %w",
			err,
		)
	}

	if err := client.Exec(
		"catalog-service",
		"sh",
		"-c",
		`test "$PLATFORM_LAB_OOM_PRESSURE" = "true"`,
	); err != nil {
		rollback()

		return fmt.Errorf(
			"OOM-kill verification failed: OOM-pressure behavior was not armed: %w",
			err,
		)
	}

	fmt.Fprintln(out, "✓ Controlled OOM-pressure trigger is armed")
	fmt.Fprintln(out, "✓ Catalog remains healthy before workload")
	fmt.Fprintln(out)
	fmt.Fprintln(
		out,
		"Incident active. Observe the memory limit, then generate controlled Catalog workload.",
	)
	fmt.Fprintln(
		out,
		"Investigate the container state after Catalog stops.",
	)
	fmt.Fprintln(
		out,
		"Run lab reset when the investigation is complete.",
	)

	return nil
}

func startLostPersistence(
	out interface {
		Write([]byte) (int, error)
	},
	root string,
	client compose.Client,
	scenarioID string,
) error {
	const volumeA = "platform-lab-lab-postgres-a"
	const volumeB = "platform-lab-lab-postgres-b"

	postgresRunning, err := client.IsRunning("postgres")
	if err != nil {
		return fmt.Errorf("inspect PostgreSQL state: %w", err)
	}
	if !postgresRunning {
		return fmt.Errorf("postgres is not running; start Platform Lab before starting this scenario")
	}

	postgresHealth, err := client.HealthStatus("postgres")
	if err != nil {
		return fmt.Errorf("inspect PostgreSQL health: %w", err)
	}
	if postgresHealth != "healthy" {
		return fmt.Errorf("postgres is not healthy before scenario start; current health: %s", postgresHealth)
	}

	fmt.Fprintln(out, "✓ PostgreSQL baseline is running and healthy")
	fmt.Fprintln(out, "✓ Real PostgreSQL volume docker_postgres-data will remain untouched")

	// Remove leftovers only from the two hard-coded Lab-owned volumes.
	_ = client.RemoveLabVolume(volumeA)
	_ = client.RemoveLabVolume(volumeB)

	if err := client.CreateLabVolume(volumeA); err != nil {
		return err
	}
	if err := client.CreateLabVolume(volumeB); err != nil {
		_ = client.RemoveLabVolume(volumeA)
		return err
	}
	fmt.Fprintln(out, "✓ Isolated Lab-owned PostgreSQL volumes created")

	session := state.Session{
		Scenario:  scenarioID,
		StartedAt: time.Now(),
		Changes: []state.Change{
			{
				Type:          "compose_override",
				Target:        "postgres",
				OriginalState: "running",
				File:          compose.OverridePath(root),
			},
		},
	}
	if err := state.Save(root, session); err != nil {
		_ = client.RemoveLabVolume(volumeA)
		_ = client.RemoveLabVolume(volumeB)
		return err
	}
	fmt.Fprintln(out, "✓ Recovery state recorded")

	rollback := func() {
		_ = compose.RemoveOverride(root)
		_ = client.RecreateBaseline("postgres")
		if err := client.WaitForHealthy("postgres", 30*time.Second); err == nil {
			_ = client.RemoveLabVolume(volumeA)
			_ = client.RemoveLabVolume(volumeB)
			_ = state.Clear(root)
		}
	}

	// Phase A: PostgreSQL uses Lab volume A.
	if err := compose.WriteOverride(root, lostPersistenceOverride(volumeA)); err != nil {
		rollback()
		return err
	}
	if err := client.RecreateWithOverride("postgres"); err != nil {
		rollback()
		return err
	}
	if err := client.WaitForHealthy("postgres", 30*time.Second); err != nil {
		rollback()
		return fmt.Errorf("lost-persistence volume A startup failed: %w", err)
	}
	fmt.Fprintln(out, "✓ PostgreSQL is running on Lab-owned volume A")

	if err := client.Exec(
		"postgres",
		"psql",
		"-U", "platform",
		"-d", "postgres",
		"-v", "ON_ERROR_STOP=1",
		"-c", "CREATE TABLE lab_persistence_marker (value text NOT NULL); INSERT INTO lab_persistence_marker(value) VALUES ('lesson-37');",
	); err != nil {
		rollback()
		return fmt.Errorf("create persistence marker: %w", err)
	}

	if err := client.Exec(
		"postgres",
		"psql",
		"-U", "platform",
		"-d", "postgres",
		"-tAc", "SELECT value FROM lab_persistence_marker WHERE value='lesson-37';",
	); err != nil {
		rollback()
		return fmt.Errorf("verify persistence marker on volume A: %w", err)
	}
	fmt.Fprintln(out, "✓ Test data exists on volume A")

	// Phase B: switch to a different fresh Lab-owned volume.
	if err := compose.WriteOverride(root, lostPersistenceOverride(volumeB)); err != nil {
		rollback()
		return err
	}
	fmt.Fprintln(out, "Switching PostgreSQL to fresh Lab-owned volume B...")

	if err := client.RecreateWithOverride("postgres"); err != nil {
		rollback()
		return err
	}
	if err := client.WaitForHealthy("postgres", 30*time.Second); err != nil {
		rollback()
		return fmt.Errorf("lost-persistence volume B startup failed: %w", err)
	}
	fmt.Fprintln(out, "✓ PostgreSQL recreated on fresh volume B")

	err = client.Exec(
		"postgres",
		"psql",
		"-U", "platform",
		"-d", "postgres",
		"-tAc", "SELECT value FROM lab_persistence_marker WHERE value='lesson-37';",
	)
	if err == nil {
		rollback()
		return fmt.Errorf("lost-persistence verification failed: marker unexpectedly exists on fresh volume B")
	}

	fmt.Fprintln(out, "✓ Test data from volume A is absent on fresh volume B")
	fmt.Fprintln(out, "✓ Lost persistence verified")
	fmt.Fprintln(out, "✓ Real docker_postgres-data volume was never mounted by the incident")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Incident active. Run lab reset to restore the real persistent PostgreSQL volume.")
	return nil
}

func init() {
	rootCmd.AddCommand(startCmd)
}
