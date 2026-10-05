package cli

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/spf13/cobra"

	"github.com/gopher-opsx/platform-lab-cli/internal/compose"
	"github.com/gopher-opsx/platform-lab-cli/internal/platform"
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
		userOut := out

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

		if session.Challenge {
			fmt.Fprintln(userOut, "Resetting troubleshooting challenge")
			fmt.Fprintln(userOut)
			fmt.Fprintln(userOut, "Restoring Platform Lab baseline...")

			// Keep the hidden scenario hidden during challenge cleanup.
			out = io.Discard
		} else {
			fmt.Fprintf(
				out,
				"Resetting scenario: %s\n\n",
				session.Scenario,
			)
		}

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

		if session.Scenario == "redis-down" {
			fmt.Fprintln(out, "Verifying Redis dependency recovery...")

			if err := client.WaitForHealthy("redis", 30*time.Second); err != nil {
				return fmt.Errorf("Redis was restarted but did not become healthy: %w", err)
			}

			if err := platform.WaitForHTTPStatus(
				"http://localhost:8082/readyz",
				http.StatusOK,
				20*time.Second,
			); err != nil {
				return fmt.Errorf("Redis recovered but Cart readiness did not recover: %w", err)
			}

			if err := platform.WaitForHTTPStatus(
				"http://localhost:8080/api/products",
				http.StatusOK,
				20*time.Second,
			); err != nil {
				return fmt.Errorf("Redis recovered but the product path is not healthy: %w", err)
			}

			if err := platform.WaitForHTTPStatusWithHeaders(
				"http://localhost:8080/api/cart",
				map[string]string{"X-Customer-ID": "lab-lesson-45"},
				http.StatusOK,
				20*time.Second,
			); err != nil {
				return fmt.Errorf("Redis recovered but the cart workflow did not recover: %w", err)
			}

			fmt.Fprintln(out, "✓ Redis is healthy")
			fmt.Fprintln(out, "✓ Cart readiness recovered")
			fmt.Fprintln(out, "✓ Product path remains healthy")
			fmt.Fprintln(out, "✓ Cart customer workflow recovered")
		}

		if session.Scenario == "kafka-down" {
			fmt.Fprintln(out, "Verifying Kafka dependency recovery...")

			if err := client.WaitForHealthy("kafka", 45*time.Second); err != nil {
				return fmt.Errorf("Kafka was restarted but did not become healthy: %w", err)
			}

			for _, service := range []string{
				"order-service",
				"inventory-service",
				"payment-service",
				"notification-service",
			} {
				if err := client.WaitForHealthy(service, 20*time.Second); err != nil {
					return fmt.Errorf("Kafka recovered but %s is not healthy: %w", service, err)
				}
			}

			if err := platform.WaitForHTTPStatus(
				"http://localhost:8083/readyz",
				http.StatusOK,
				20*time.Second,
			); err != nil {
				return fmt.Errorf("Kafka recovered but Order readiness is not healthy: %w", err)
			}

			if err := platform.WaitForHTTPStatus(
				"http://localhost:8080/api/products",
				http.StatusOK,
				20*time.Second,
			); err != nil {
				return fmt.Errorf("Kafka recovered but the product path is not healthy: %w", err)
			}

			if err := platform.WaitForHTTPStatusWithHeaders(
				"http://localhost:8080/api/cart",
				map[string]string{"X-Customer-ID": "lab-lesson-46"},
				http.StatusOK,
				20*time.Second,
			); err != nil {
				return fmt.Errorf("Kafka recovered but the cart path is not healthy: %w", err)
			}

			if err := platform.WaitForHTTPStatusWithHeaders(
				"http://localhost:8080/api/orders",
				map[string]string{"X-Customer-ID": "lab-lesson-46"},
				http.StatusOK,
				20*time.Second,
			); err != nil {
				return fmt.Errorf("Kafka recovered but the Order HTTP path is not healthy: %w", err)
			}

			fmt.Fprintln(out, "✓ Kafka is healthy")
			fmt.Fprintln(out, "✓ Order and downstream services are healthy")
			fmt.Fprintln(out, "✓ Product, cart, and order HTTP paths remain healthy")
			fmt.Fprintln(out, "✓ Repeat the same order workflow and verify consumer processing")
		}

		if session.Scenario == "kafka-consumer-stops" {
			fmt.Fprintln(out, "Verifying Kafka consumer recovery...")

			if err := client.WaitForHealthy("inventory-service", 30*time.Second); err != nil {
				return fmt.Errorf("Inventory was restarted but did not become healthy: %w", err)
			}

			if err := client.WaitForHealthy("kafka", 20*time.Second); err != nil {
				return fmt.Errorf("Inventory recovered but Kafka is not healthy: %w", err)
			}

			for _, service := range []string{
				"order-service",
				"payment-service",
				"notification-service",
			} {
				if err := client.WaitForHealthy(service, 20*time.Second); err != nil {
					return fmt.Errorf("Inventory recovered but %s is not healthy: %w", service, err)
				}
			}

			if err := platform.WaitForHTTPStatus(
				"http://localhost:8080/api/products",
				http.StatusOK,
				20*time.Second,
			); err != nil {
				return fmt.Errorf("Inventory recovered but the product path is not healthy: %w", err)
			}

			if err := platform.WaitForHTTPStatusWithHeaders(
				"http://localhost:8080/api/cart",
				map[string]string{"X-Customer-ID": "lab-lesson-47"},
				http.StatusOK,
				20*time.Second,
			); err != nil {
				return fmt.Errorf("Inventory recovered but the cart path is not healthy: %w", err)
			}

			if err := platform.WaitForHTTPStatusWithHeaders(
				"http://localhost:8080/api/orders",
				map[string]string{"X-Customer-ID": "lab-lesson-47"},
				http.StatusOK,
				20*time.Second,
			); err != nil {
				return fmt.Errorf("Inventory recovered but the Order HTTP path is not healthy: %w", err)
			}

			fmt.Fprintln(out, "✓ Inventory consumer is healthy again")
			fmt.Fprintln(out, "✓ Kafka remained healthy")
			fmt.Fprintln(out, "✓ Order and unaffected downstream services are healthy")
			fmt.Fprintln(out, "✓ Customer-facing HTTP paths remain healthy")
			fmt.Fprintln(out, "✓ Check the original pending order: retained Kafka events can now be consumed")
		}

		if session.Scenario == "cascading-incident" {
			fmt.Fprintln(out, "Verifying cascading-incident recovery...")

			if err := client.WaitForHealthy("payment-service", 30*time.Second); err != nil {
				return fmt.Errorf("Payment baseline was restored but did not become healthy: %w", err)
			}

			if err := platform.WaitForHTTPStatus(
				"http://localhost:8085/readyz",
				http.StatusOK,
				20*time.Second,
			); err != nil {
				return fmt.Errorf("Payment baseline was restored but readiness did not recover: %w", err)
			}

			if err := client.WaitForHealthy("kafka", 20*time.Second); err != nil {
				return fmt.Errorf("Payment baseline recovered but Kafka is not healthy: %w", err)
			}

			for _, service := range []string{
				"order-service",
				"inventory-service",
				"notification-service",
			} {
				if err := client.WaitForHealthy(service, 20*time.Second); err != nil {
					return fmt.Errorf("cascading-incident recovery incomplete: %s is not healthy: %w", service, err)
				}
			}

			if err := platform.WaitForHTTPStatus(
				"http://localhost:8080/api/products",
				http.StatusOK,
				20*time.Second,
			); err != nil {
				return fmt.Errorf("cascading-incident recovery failed: product path is not healthy: %w", err)
			}

			if err := platform.WaitForHTTPStatusWithHeaders(
				"http://localhost:8080/api/cart",
				map[string]string{"X-Customer-ID": "lab-lesson-48"},
				http.StatusOK,
				20*time.Second,
			); err != nil {
				return fmt.Errorf("cascading-incident recovery failed: cart path is not healthy: %w", err)
			}

			if err := platform.WaitForHTTPStatusWithHeaders(
				"http://localhost:8080/api/orders",
				map[string]string{"X-Customer-ID": "lab-lesson-48"},
				http.StatusOK,
				20*time.Second,
			); err != nil {
				return fmt.Errorf("cascading-incident recovery failed: Order HTTP path is not healthy: %w", err)
			}

			fmt.Fprintln(out, "✓ Event-path configuration restored")
			fmt.Fprintln(out, "✓ Kafka and downstream services are healthy")
			fmt.Fprintln(out, "✓ Product, cart, and order HTTP paths remain healthy")
			fmt.Fprintln(out, "✓ Recheck the original pending order; retained events can now continue through Payment and Notification")
		}

		if session.Scenario == "postgres-down" {
			fmt.Fprintln(out, "Verifying PostgreSQL dependency recovery...")

			if err := client.WaitForHealthy("postgres", 30*time.Second); err != nil {
				return fmt.Errorf("PostgreSQL was restarted but did not become healthy: %w", err)
			}

			if err := platform.WaitForHTTPStatus(
				"http://localhost:8081/readyz",
				http.StatusOK,
				20*time.Second,
			); err != nil {
				return fmt.Errorf("PostgreSQL recovered but Catalog readiness did not recover: %w", err)
			}

			if err := platform.WaitForHTTPStatus(
				"http://localhost:8080/api/products",
				http.StatusOK,
				20*time.Second,
			); err != nil {
				return fmt.Errorf("PostgreSQL recovered but the product path did not recover: %w", err)
			}

			fmt.Fprintln(out, "✓ PostgreSQL is healthy")
			fmt.Fprintln(out, "✓ Catalog readiness recovered")
			fmt.Fprintln(out, "✓ Product request recovered")
		}

		/*
			Lesson 37 creates two hard-coded Lab-owned PostgreSQL
			volumes. Remove them only after the normal PostgreSQL
			baseline has been restored successfully.

			RemoveLabVolume has its own safety guard and refuses
			to remove volumes outside the platform-lab-lab- namespace.
		*/
		if session.Scenario == "lost-persistence" {
			fmt.Fprintln(out, "Removing Lab-owned temporary PostgreSQL volumes...")

			for _, volume := range []string{
				"platform-lab-lab-postgres-a",
				"platform-lab-lab-postgres-b",
			} {
				if err := client.RemoveLabVolume(volume); err != nil {
					return err
				}
			}

			fmt.Fprintln(out, "✓ Lab-owned temporary PostgreSQL volumes removed")
		}

		/*
			State is removed only after all rollback
			operations succeed.
		*/
		if err := state.Clear(root); err != nil {
			return err
		}

		if session.Challenge {
			fmt.Fprintln(userOut, "✓ Challenge environment restored")
		}

		fmt.Fprintln(userOut)
		fmt.Fprintln(
			userOut,
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
