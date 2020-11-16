package cmd

import (
	"errors"
	"github.com/massn/github-evaluator/pkg/stats"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "github-evaluator REPOS_YAML",
		Short: "An GitHub repositories evaluator",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return errors.New("requires repositories(yaml)")
			}
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			stats.Stats(args[0])
		},
	}
)

// Execute the command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}
