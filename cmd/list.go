package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/amrinder15/aksctx/internal/azure"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List AKS clusters across your Azure subscriptions",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		cred, err := azure.NewCredential()
		if err != nil {
			return err
		}

		if subscriptionName == "" {
			fmt.Println("🔍 Discovering AKS clusters across subscriptions...")
		} else {
			fmt.Printf("🔍 Discovering AKS clusters in subscription %q...\n", subscriptionName)
		}

		clusters, err := azure.ListAllClusters(ctx, cred, subscriptionName)
		if err != nil {
			return fmt.Errorf("listing clusters: %w", err)
		}

		if len(clusters) == 0 {
			fmt.Println("No AKS clusters found.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "CLUSTER\tSUBSCRIPTION\tRESOURCE GROUP\tLOCATION\tK8S VERSION\tNODES")
		fmt.Fprintln(w, "-------\t------------\t--------------\t--------\t-----------\t-----")
		for _, c := range clusters {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%d\n",
				c.Name,
				c.SubscriptionName,
				c.ResourceGroup,
				c.Location,
				c.K8sVersion,
				c.NodeCount,
			)
		}
		w.Flush()

		fmt.Printf("\n%d cluster(s) found.\n", len(clusters))
		return nil
	},
}
