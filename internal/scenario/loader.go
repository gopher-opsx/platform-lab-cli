package scenario

import (
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	scenariofiles "github.com/gopher-opsx/platform-lab-cli/scenarios"
)

const scenarioDirectory = "course-01"

func LoadAll() ([]Scenario, error) {
	entries, err := fs.ReadDir(
		scenariofiles.Files,
		scenarioDirectory,
	)
	if err != nil {
		return nil, fmt.Errorf("read scenarios: %w", err)
	}

	var result []Scenario

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		path := scenarioDirectory + "/" + entry.Name()

		data, err := scenariofiles.Files.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf(
				"read scenario %s: %w",
				entry.Name(),
				err,
			)
		}

		var s Scenario

		if err := yaml.Unmarshal(data, &s); err != nil {
			return nil, fmt.Errorf(
				"parse scenario %s: %w",
				entry.Name(),
				err,
			)
		}

		if err := validate(s); err != nil {
			return nil, fmt.Errorf(
				"invalid scenario %s: %w",
				entry.Name(),
				err,
			)
		}

		result = append(result, s)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Lesson < result[j].Lesson
	})

	return result, nil
}

func Load(id string) (Scenario, error) {
	scenarios, err := LoadAll()
	if err != nil {
		return Scenario{}, err
	}

	for _, s := range scenarios {
		if s.ID == id {
			return s, nil
		}
	}

	return Scenario{}, fmt.Errorf(
		"scenario %q not found",
		id,
	)
}

func validate(s Scenario) error {
	if strings.TrimSpace(s.ID) == "" {
		return fmt.Errorf("id is required")
	}

	if strings.TrimSpace(s.Name) == "" {
		return fmt.Errorf("name is required")
	}

	if s.Course <= 0 {
		return fmt.Errorf("course must be greater than zero")
	}

	if s.Lesson <= 0 {
		return fmt.Errorf("lesson must be greater than zero")
	}

	if len(s.Inject) == 0 {
		return fmt.Errorf("at least one inject action is required")
	}

	return nil
}
