package cmd

import (
	"errors"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "github-evaluator",
		Short: "An GitHub repositories evaluator",
		Args: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
)

// Execute the command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}

func checkArgs(args []string) error {
	if len(args) < 1 {
		return errors.New("requires repositories(yaml)")
	}
	return nil
}
