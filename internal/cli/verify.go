package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/gopher-opsx/platform-lab-cli/internal/platform"
	"github.com/gopher-opsx/platform-lab-cli/internal/runner"
)

var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify the Platform Lab environment",
	Long: `Verify checks that Docker, Docker Compose, the expected
Platform Lab files, and required Compose services are available.

This command does not modify the environment.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		out := cmd.OutOrStdout()

		fmt.Fprintln(out, "Verifying Platform Lab...")
		fmt.Fprintln(out)

		definition, err := platform.LoadPlatformLabV1()
		if err != nil {
			return err
		}

		fmt.Fprintf(
			out,
			"✓ Platform definition: %s %s\n",
			definition.Name,
			definition.Version,
		)

		root, err := findPlatformRoot(definition.ComposeFiles)
		if err != nil {
			return err
		}

		fmt.Fprintf(
			out,
			"✓ Platform Lab repository: %s\n",
			root,
		)

		for _, composeFile := range definition.ComposeFiles {
			fullPath := filepath.Join(root, composeFile)

			if _, err := os.Stat(fullPath); err != nil {
				return fmt.Errorf(
					"required compose file not found: %s",
					fullPath,
				)
			}

			fmt.Fprintf(
				out,
				"✓ Compose file: %s\n",
				composeFile,
			)
		}

		// Run Docker from inside the Platform Lab repository.
		if _, err := runner.Run(
			root,
			"docker",
			"version",
		); err != nil {
			return fmt.Errorf(
				"Docker is not available: %w",
				err,
			)
		}

		fmt.Fprintln(out, "✓ Docker available")

		// Run Docker Compose from inside the Platform Lab repository.
		if _, err := runner.Run(
			root,
			"docker",
			"compose",
			"version",
		); err != nil {
			return fmt.Errorf(
				"Docker Compose is not available: %w",
				err,
			)
		}

		fmt.Fprintln(out, "✓ Docker Compose available")

		composeArgs := buildComposeArgs(
			root,
			definition.ComposeFiles,
		)

		composeArgs = append(
			composeArgs,
			"config",
			"--services",
		)

		// Read the actual Compose services from inside
		// the Platform Lab repository.
		result, err := runner.Run(
			root,
			"docker",
			composeArgs...,
		)
		if err != nil {
			return fmt.Errorf(
				"cannot read Platform Lab Compose configuration: %w",
				err,
			)
		}

		actualServices := parseServices(result.Stdout)

		if err := verifyRequiredServices(
			definition.RequiredServices,
			actualServices,
		); err != nil {
			return err
		}

		fmt.Fprintf(
			out,
			"✓ Required services found: %d\n",
			len(definition.RequiredServices),
		)

		fmt.Fprintln(out)
		fmt.Fprintln(
			out,
			"Platform Lab environment verified.",
		)

		fmt.Fprintln(
			out,
			"No changes were made.",
		)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(verifyCmd)
}

func findPlatformRoot(
	composeFiles []string,
) (string, error) {

	current, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf(
			"get current directory: %w",
			err,
		)
	}

	for {
		if hasRequiredFiles(current, composeFiles) {
			return current, nil
		}

		parent := filepath.Dir(current)

		if parent == current {
			break
		}

		current = parent
	}

	return "", fmt.Errorf(
		"Platform Lab repository not found; run lab from inside the Platform Lab repository",
	)
}

func hasRequiredFiles(
	root string,
	files []string,
) bool {

	for _, file := range files {
		path := filepath.Join(root, file)

		info, err := os.Stat(path)
		if err != nil || info.IsDir() {
			return false
		}
	}

	return true
}

func buildComposeArgs(
	root string,
	composeFiles []string,
) []string {

	args := []string{"compose"}

	for _, file := range composeFiles {
		args = append(
			args,
			"-f",
			filepath.Join(root, file),
		)
	}

	return args
}

func parseServices(output string) []string {
	lines := strings.Split(output, "\n")

	var services []string

	for _, line := range lines {
		service := strings.TrimSpace(line)

		if service != "" {
			services = append(
				services,
				service,
			)
		}
	}

	sort.Strings(services)

	return services
}

func verifyRequiredServices(
	required []string,
	actual []string,
) error {

	actualSet := make(map[string]bool)

	for _, service := range actual {
		actualSet[service] = true
	}

	var missing []string

	for _, service := range required {
		if !actualSet[service] {
			missing = append(
				missing,
				service,
			)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf(
			"Platform Lab is incompatible; missing services: %s",
			strings.Join(missing, ", "),
		)
	}

	return nil
}
