package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var subscriptionName string
var showVersion bool

// version is overridden at build time for release binaries.
var version = "dev"

var rootCmd = &cobra.Command{
	Use:   "aksctx",
	Short: "AKS context switcher — discover and switch between AKS clusters across Azure subscriptions",
	Long: `aksctx helps you discover and switch between AKS clusters across all your
Azure subscriptions without needing to remember resource group or cluster names.

It uses your existing Azure credentials (az login, environment variables,
or Workload Identity) and merges cluster credentials directly into your
	~/.kube/config file.`,
	Version: version,
	RunE: func(cmd *cobra.Command, args []string) error {
		if showVersion {
			fmt.Fprintln(cmd.OutOrStdout(), cmd.Version)
			return nil
		}
		return cmd.Help()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().BoolVarP(&showVersion, "version", "v", false, "Print version")
	rootCmd.PersistentFlags().StringVar(&subscriptionName, "subscription", "", "Limit AKS discovery to a single Azure subscription name")

	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(switchCmd)
	rootCmd.AddCommand(currentCmd)
	rootCmd.AddCommand(refreshCmd)
}
