package cli

import (
	"fmt"

	"github.com/gopher-opsx/platform-lab-cli/internal/compose"
	"github.com/gopher-opsx/platform-lab-cli/internal/platform"
	"github.com/gopher-opsx/platform-lab-cli/internal/state"
)

func platformClient() (string, compose.Client, error) {
	definition, err := platform.LoadPlatformLabV1()
	if err != nil {
		return "", compose.Client{}, err
	}

	root, err := findPlatformRoot(definition.ComposeFiles)
	if err != nil {
		return "", compose.Client{}, err
	}

	client := compose.Client{
		Root:         root,
		ComposeFiles: definition.ComposeFiles,
	}

	return root, client, nil
}

func ensureNoActiveScenario(root string) error {
	if state.Exists(root) {
		return fmt.Errorf(
			"another Lab scenario is already active; run 'lab reset' first",
		)
	}

	return nil
}
