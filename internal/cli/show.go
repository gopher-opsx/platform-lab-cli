package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/gopher-opsx/platform-lab-cli/internal/scenario"
)

var showCmd = &cobra.Command{
	Use:   "show <scenario>",
	Short: "Show an incident scenario",
	Args:  cobra.ExactArgs(1),

	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := scenario.Load(args[0])
		if err != nil {
			return err
		}

		out := cmd.OutOrStdout()

		fmt.Fprintf(out, "Scenario:    %s\n", s.ID)
		fmt.Fprintf(out, "Name:        %s\n", s.Name)
		fmt.Fprintf(out, "Course:      %d\n", s.Course)
		fmt.Fprintf(out, "Lesson:      %d\n", s.Lesson)
		fmt.Fprintf(out, "Platform:    %s %s\n",
			s.Platform.Name,
			s.Platform.Version,
		)

		fmt.Fprintln(out)
		fmt.Fprintln(out, s.Description)

		fmt.Fprintln(out)
		fmt.Fprintln(out, "Injection:")

		for _, action := range s.Inject {
			fmt.Fprintf(
				out,
				"  %s -> %s\n",
				action.Type,
				action.Target,
			)
		}

		fmt.Fprintln(out)
		fmt.Fprintln(out, "Rollback:")

		for _, action := range s.Rollback {
			fmt.Fprintf(
				out,
				"  %s -> %s\n",
				action.Type,
				action.Target,
			)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(showCmd)
}
