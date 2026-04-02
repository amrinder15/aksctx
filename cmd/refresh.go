package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/amrinder15/aksctx/internal/azure"
	"github.com/amrinder15/aksctx/internal/kubeconfig"
	"github.com/spf13/cobra"
)

var refreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "Re-fetch credentials for the current cluster context",
	Long: `Re-fetches kubeconfig credentials for the currently active cluster.
Useful when your token has expired or credentials have rotated.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		contextName, _, err := kubeconfig.CurrentContext()
		if err != nil {
			return err
		}

		fmt.Printf("Current context: %s\n", contextName)

		cred, err := azure.NewCredential()
		if err != nil {
			return err
		}

		// Discover clusters and find the one matching current context
		if subscriptionName == "" {
			fmt.Println("🔍 Looking up cluster...")
		} else {
			fmt.Printf("🔍 Looking up cluster in subscription %q...\n", subscriptionName)
		}

		clusters, err := azure.ListAllClusters(ctx, cred, subscriptionName)
		if err != nil {
			return fmt.Errorf("listing clusters: %w", err)
		}

		var match *azure.Cluster
		for i, c := range clusters {
			// AKS context names are typically in format: <cluster-name> or <cluster-name>-admin
			if strings.EqualFold(c.Name, contextName) ||
				strings.HasPrefix(contextName, c.Name) {
				match = &clusters[i]
				break
			}
		}

		if match == nil {
			return fmt.Errorf("could not find an AKS cluster matching context %q\nTry running: aksctx switch", contextName)
		}

		fmt.Printf("⬇  Refreshing credentials for %s...\n", match.Name)
		kubeconfigBytes, err := azure.GetClusterKubeconfig(ctx, cred, *match)
		if err != nil {
			return err
		}

		if err := kubeconfig.MergeAndSwitch(kubeconfigBytes, match.Name); err != nil {
			return err
		}

		fmt.Printf("✅ Credentials refreshed for: %s\n", match.Name)
		return nil
	},
}
