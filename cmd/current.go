package cmd

import (
	"fmt"

	"github.com/amrinder/aksctx/internal/kubeconfig"
	"github.com/spf13/cobra"
)

var currentCmd = &cobra.Command{
	Use:   "current",
	Short: "Show the current active AKS context",
	RunE: func(cmd *cobra.Command, args []string) error {
		contextName, server, err := kubeconfig.CurrentContext()
		if err != nil {
			return err
		}

		fmt.Printf("Current context : %s\n", contextName)
		if server != "" {
			fmt.Printf("API server      : %s\n", server)
		}
		return nil
	},
}
