package compose

import (
	"fmt"
	"strings"

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
