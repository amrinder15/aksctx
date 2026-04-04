package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/amrinder15/aksctx/internal/azure"
	"github.com/amrinder15/aksctx/internal/kubeconfig"
	"github.com/amrinder15/aksctx/internal/tui"
	"github.com/spf13/cobra"
)

var switchDiscovery bool

var switchCmd = &cobra.Command{
	Use:     "switch",
	Aliases: []string{"s"},
	Short:   "Interactively switch to a local context or discover an AKS cluster",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !switchDiscovery {
			if strings.TrimSpace(subscriptionName) != "" {
				return fmt.Errorf("--subscription requires --discovery for the switch command")
			}
			return runLocalSwitch()
		}

		return runDiscoverySwitch()
	},
}

func init() {
	switchCmd.Flags().BoolVarP(&switchDiscovery, "discovery", "d", false, "Discover AKS clusters from Azure instead of selecting an existing kubeconfig context")
}

func runLocalSwitch() error {
	contexts, err := kubeconfig.ListContexts()
	if err != nil {
		return err
	}

	if len(contexts) == 0 {
		fmt.Println("No kubeconfig contexts found.")
		return nil
	}

	items := make([]tui.Item, len(contexts))
	for i, kubeContext := range contexts {
		items[i] = tui.Item{
			Label: kubeContext.Name,
			Value: kubeContext,
		}
	}

	result, err := tui.Run("Select a kubeconfig context", items)
	if err != nil {
		return err
	}
	if result.Aborted || result.Selected == nil {
		fmt.Println("Cancelled.")
		return nil
	}

	selected := result.Selected.Value.(kubeconfig.Context)
	if err := kubeconfig.SetCurrentContext(selected.Name); err != nil {
		return err
	}

	fmt.Println("\nSelect a default namespace")
	namespace, skipped, err := promptForNamespaceSelection(selected.Name)
	if err != nil {
		fmt.Printf("⚠  Unable to load namespaces for %s: %v\n", selected.Name, err)
	} else if skipped {
		fmt.Println("ℹ  Namespace selection skipped; keeping the context default.")
	} else {
		if err := kubeconfig.SetContextNamespace(selected.Name, namespace); err != nil {
			return err
		}
	}

	fmt.Printf("✅ Switched to context: %s\n", selected.Name)
	if selected.Cluster != "" {
		fmt.Printf("   Cluster       : %s\n", selected.Cluster)
	}
	if selected.Server != "" {
		fmt.Printf("   API server    : %s\n", selected.Server)
	}
	if namespace != "" {
		fmt.Printf("   Namespace     : %s\n", namespace)
	}

	return nil
}

func runDiscoverySwitch() error {
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

	contextName, _, err := kubeconfig.CurrentContext()
	if err != nil {
		return err
	}

	fmt.Println("\nSelect a default namespace")
	namespace, skipped, err := promptForNamespaceSelection(contextName)
	if err != nil {
		fmt.Printf("⚠  Unable to load namespaces for %s: %v\n", selected.Name, err)
	} else if skipped {
		fmt.Println("ℹ  Namespace selection skipped; keeping the context default.")
	} else {
		if err := kubeconfig.SetContextNamespace(contextName, namespace); err != nil {
			return err
		}
	}

	fmt.Printf("✅ Switched to cluster: %s\n", selected.Name)
	fmt.Printf("   Subscription : %s\n", selected.SubscriptionName)
	fmt.Printf("   Resource Group: %s\n", selected.ResourceGroup)
	fmt.Printf("   Location      : %s\n", selected.Location)
	fmt.Printf("   K8s Version   : %s\n", selected.K8sVersion)
	if namespace != "" {
		fmt.Printf("   Namespace     : %s\n", namespace)
	}

	return nil
}

func promptForNamespaceSelection(contextName string) (string, bool, error) {
	namespaces, err := kubeconfig.ListNamespaces(contextName)
	if err != nil {
		return "", false, err
	}
	if len(namespaces) == 0 {
		return "", true, nil
	}

	currentNamespace, err := kubeconfig.ContextNamespace(contextName)
	if err != nil {
		return "", false, err
	}
	if strings.TrimSpace(currentNamespace) == "" {
		currentNamespace = "default"
	}

	items := make([]tui.Item, len(namespaces))
	for i, namespace := range namespaces {
		description := ""
		if namespace == currentNamespace {
			description = "Current"
		}
		items[i] = tui.Item{
			Label:       namespace,
			Description: description,
			Value:       namespace,
		}
	}

	result, err := tui.Run("Select a default namespace", items)
	if err != nil {
		return "", false, err
	}
	if result.Aborted || result.Selected == nil {
		return "", true, nil
	}

	return result.Selected.Value.(string), false, nil
}
