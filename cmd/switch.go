package cmd

import (
	"context"
	"fmt"

	"github.com/amrinder15/aksctx/internal/azure"
	"github.com/amrinder15/aksctx/internal/kubeconfig"
	"github.com/amrinder15/aksctx/internal/tui"
	"github.com/spf13/cobra"
)

var switchCmd = &cobra.Command{
	Use:     "switch",
	Aliases: []string{"s"},
	Short:   "Interactively select and switch to an AKS cluster",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		cred, err := azure.NewCredential()
		if err != nil {
			return err
		}

		if subscriptionName == "" {
			fmt.Println("🔍 Discovering AKS clusters...")
		} else {
			fmt.Printf("🔍 Discovering AKS clusters in subscription %q...\n", subscriptionName)
		}

		clusters, err := azure.ListAllClusters(ctx, cred, subscriptionName)
		if err != nil {
			return fmt.Errorf("listing clusters: %w", err)
		}

		if len(clusters) == 0 {
			if subscriptionName == "" {
				fmt.Println("No AKS clusters found across your subscriptions.")
			} else {
				fmt.Printf("No AKS clusters found in subscription %q.\n", subscriptionName)
			}
			return nil
		}

		// Build TUI items
		items := make([]tui.Item, len(clusters))
		for i, c := range clusters {
			items[i] = tui.Item{
				Label:       c.Name,
				Description: fmt.Sprintf("%s / %s / %s / k8s %s / %d nodes", c.SubscriptionName, c.Location, c.ResourceGroup, c.K8sVersion, c.NodeCount),
				Value:       c,
			}
		}

		result, err := tui.Run("Select an AKS cluster", items)
		if err != nil {
			return err
		}
		if result.Aborted || result.Selected == nil {
			fmt.Println("Cancelled.")
			return nil
		}

		selected := result.Selected.Value.(azure.Cluster)
		fmt.Printf("\n⬇  Fetching credentials for %s...\n", selected.Name)

		kubeconfigBytes, err := azure.GetClusterKubeconfig(ctx, cred, selected)
		if err != nil {
			return err
		}

		if err := kubeconfig.MergeAndSwitch(kubeconfigBytes, selected.Name); err != nil {
			return err
		}

		fmt.Printf("✅ Switched to cluster: %s\n", selected.Name)
		fmt.Printf("   Subscription : %s\n", selected.SubscriptionName)
		fmt.Printf("   Resource Group: %s\n", selected.ResourceGroup)
		fmt.Printf("   Location      : %s\n", selected.Location)
		fmt.Printf("   K8s Version   : %s\n", selected.K8sVersion)

		return nil
	},
}
