package cmd

import (
	"github.com/massn/github-evaluator/pkg/stats"
	"github.com/spf13/cobra"
)

var (
	issuesCmd = &cobra.Command{
		Use: "issues",
		Args: func(cmd *cobra.Command, args []string) error {
			return checkArgs(args)
		},
		Run: func(cmd *cobra.Command, args []string) {
			err := stats.Stats(args[0], stats.Issues)
			if err != nil {
				panic(err)
			}
		},
	}
)

func init() {
	rootCmd.AddCommand(issuesCmd)
}
