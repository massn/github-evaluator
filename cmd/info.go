package cmd

import (
	"github.com/massn/github-evaluator/pkg/stats"
	"github.com/spf13/cobra"
)

var (
	infoCmd = &cobra.Command{
		Use: "info",
		Args: func(cmd *cobra.Command, args []string) error {
			return checkArgs(args)
		},
		Run: func(cmd *cobra.Command, args []string) {
			err := stats.Stats(args[0], stats.Info)
			if err != nil {
				panic(err)
			}
		},
	}
)

func init() {
	rootCmd.AddCommand(infoCmd)
}
