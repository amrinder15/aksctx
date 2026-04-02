package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var subscriptionName string

var rootCmd = &cobra.Command{
	Use:   "aksctx",
	Short: "AKS context switcher — discover and switch between AKS clusters across Azure subscriptions",
	Long: `aksctx helps you discover and switch between AKS clusters across all your
Azure subscriptions without needing to remember resource group or cluster names.

It uses your existing Azure credentials (az login, environment variables,
or Workload Identity) and merges cluster credentials directly into your
~/.kube/config file.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&subscriptionName, "subscription", "", "Limit AKS discovery to a single Azure subscription name")

	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(switchCmd)
	rootCmd.AddCommand(currentCmd)
	rootCmd.AddCommand(refreshCmd)
}
