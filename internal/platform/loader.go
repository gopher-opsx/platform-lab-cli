package platform

import (
	"fmt"

	"gopkg.in/yaml.v3"

	platformfiles "github.com/gopher-opsx/platform-lab-cli/platforms"
)

func LoadPlatformLabV1() (Definition, error) {
	data, err := platformfiles.Files.ReadFile(
		"platform-lab-v1.yaml",
	)
	if err != nil {
		return Definition{}, fmt.Errorf(
			"read platform definition: %w",
			err,
		)
	}

	var definition Definition

	if err := yaml.Unmarshal(data, &definition); err != nil {
		return Definition{}, fmt.Errorf(
			"parse platform definition: %w",
			err,
		)
	}

	if definition.Name == "" {
		return Definition{}, fmt.Errorf(
			"platform name is required",
		)
	}

	if len(definition.ComposeFiles) == 0 {
		return Definition{}, fmt.Errorf(
			"at least one compose file is required",
		)
	}

	return definition, nil
}
