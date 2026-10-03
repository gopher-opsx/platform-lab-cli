package compose

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gopher-opsx/platform-lab-cli/internal/runner"
)

const labOverrideFile = ".lab/compose.override.yaml"

func OverridePath(root string) string {
	return filepath.Join(
		root,
		".lab",
		"compose.override.yaml",
	)
}

func WriteOverride(
	root string,
	content string,
) error {

	dir := filepath.Join(root, ".lab")

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf(
			"create Lab runtime directory: %w",
			err,
		)
	}

	path := OverridePath(root)

	if err := os.WriteFile(
		path,
		[]byte(content),
		0644,
	); err != nil {
		return fmt.Errorf(
			"write Compose override: %w",
			err,
		)
	}

	return nil
}

func RemoveOverride(root string) error {
	path := OverridePath(root)

	err := os.Remove(path)

	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf(
			"remove Compose override: %w",
			err,
		)
	}

	return nil
}

func (c Client) RecreateWithOverride(
	service string,
) error {

	args := []string{"compose"}

	for _, file := range c.ComposeFiles {
		args = append(
			args,
			"-f",
			file,
		)
	}

	args = append(
		args,
		"-f",
		labOverrideFile,
		"up",
		"-d",
		"--no-deps",
		"--force-recreate",
		service,
	)

	_, err := runner.Run(
		c.Root,
		"docker",
		args...,
	)

	if err != nil {
		return fmt.Errorf(
			"recreate %s with Lab override: %w",
			service,
			err,
		)
	}

	return nil
}

func (c Client) RecreateBaseline(
	service string,
) error {

	args := []string{"compose"}

	for _, file := range c.ComposeFiles {
		args = append(
			args,
			"-f",
			file,
		)
	}

	args = append(
		args,
		"up",
		"-d",
		"--no-deps",
		"--force-recreate",
		service,
	)

	_, err := runner.Run(
		c.Root,
		"docker",
		args...,
	)

	if err != nil {
		return fmt.Errorf(
			"restore baseline service %s: %w",
			service,
			err,
		)
	}

	return nil
}
