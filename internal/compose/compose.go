package compose

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gopher-opsx/platform-lab-cli/internal/runner"
)

type Client struct {
	Root         string
	ComposeFiles []string
}

func (c Client) args(extra ...string) []string {
	args := []string{"compose"}

	for _, file := range c.ComposeFiles {
		args = append(args, "-f", file)
	}

	args = append(args, extra...)

	return args
}

func (c Client) Stop(service string) error {
	_, err := runner.Run(
		c.Root,
		"docker",
		c.args("stop", service)...,
	)

	if err != nil {
		return fmt.Errorf(
			"stop service %s: %w",
			service,
			err,
		)
	}

	return nil
}

func (c Client) Start(service string) error {
	_, err := runner.Run(
		c.Root,
		"docker",
		c.args("start", service)...,
	)

	if err != nil {
		return fmt.Errorf(
			"start service %s: %w",
			service,
			err,
		)
	}

	return nil
}

func (c Client) ServiceID(service string) (string, error) {
	result, err := runner.Run(
		c.Root,
		"docker",
		c.args("ps", "-q", service)...,
	)

	if err != nil {
		return "", err
	}

	return strings.TrimSpace(result.Stdout), nil
}

func (c Client) IsRunning(service string) (bool, error) {
	id, err := c.ServiceID(service)
	if err != nil {
		return false, err
	}

	if id == "" {
		return false, nil
	}

	result, err := runner.Run(
		c.Root,
		"docker",
		"inspect",
		"-f",
		"{{.State.Running}}",
		id,
	)

	if err != nil {
		return false, err
	}

	return strings.TrimSpace(result.Stdout) == "true", nil
}

func (c Client) RestartCount(service string) (int, error) {
	id, err := c.ServiceID(service)
	if err != nil {
		return 0, err
	}

	if id == "" {
		return 0, fmt.Errorf(
			"container for service %s was not found",
			service,
		)
	}

	result, err := runner.Run(
		c.Root,
		"docker",
		"inspect",
		"-f",
		"{{.RestartCount}}",
		id,
	)
	if err != nil {
		return 0, err
	}

	count, err := strconv.Atoi(
		strings.TrimSpace(result.Stdout),
	)
	if err != nil {
		return 0, fmt.Errorf(
			"parse restart count for %s: %w",
			service,
			err,
		)
	}

	return count, nil
}

func (c Client) HealthStatus(service string) (string, error) {
	id, err := c.ServiceID(service)
	if err != nil {
		return "", err
	}

	if id == "" {
		return "", fmt.Errorf(
			"container for service %s was not found",
			service,
		)
	}

	result, err := runner.Run(
		c.Root,
		"docker",
		"inspect",
		"-f",
		"{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}",
		id,
	)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(result.Stdout), nil
}

func (c Client) WaitForRestart(
	service string,
	timeout time.Duration,
) error {

	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		count, err := c.RestartCount(service)
		if err == nil && count > 0 {
			return nil
		}

		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf(
		"%s did not enter a restart loop within %s",
		service,
		timeout,
	)
}

func (c Client) WaitForHealthy(
	service string,
	timeout time.Duration,
) error {

	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		health, err := c.HealthStatus(service)

		if err == nil && health == "healthy" {
			return nil
		}

		time.Sleep(time.Second)
	}

	return fmt.Errorf(
		"%s did not become healthy within %s",
		service,
		timeout,
	)
}

func (c Client) WaitForHealthStatus(
	service string,
	expected string,
	timeout time.Duration,
) error {

	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		health, err := c.HealthStatus(service)

		if err == nil && health == expected {
			return nil
		}

		time.Sleep(time.Second)
	}

	health, _ := c.HealthStatus(service)

	return fmt.Errorf(
		"%s did not reach health status %q within %s; current status: %q",
		service,
		expected,
		timeout,
		health,
	)
}

func (c Client) CreateLabVolume(name string) error {
	if !strings.HasPrefix(name, "platform-lab-lab-") {
		return fmt.Errorf("refusing to create non-Lab volume %q", name)
	}

	_, err := runner.Run(
		c.Root,
		"docker",
		"volume",
		"create",
		name,
	)
	if err != nil {
		return fmt.Errorf("create Lab volume %s: %w", name, err)
	}

	return nil
}

func (c Client) RemoveLabVolume(name string) error {
	if !strings.HasPrefix(name, "platform-lab-lab-") {
		return fmt.Errorf("refusing to remove non-Lab volume %q", name)
	}

	_, err := runner.Run(
		c.Root,
		"docker",
		"volume",
		"rm",
		name,
	)
	if err != nil {
		return fmt.Errorf("remove Lab volume %s: %w", name, err)
	}

	return nil
}

func (c Client) Exec(
	service string,
	args ...string,
) error {
	composeArgs := c.args(
		append(
			[]string{"exec", "-T", service},
			args...,
		)...,
	)

	_, err := runner.Run(
		c.Root,
		"docker",
		composeArgs...,
	)

	return err
}
