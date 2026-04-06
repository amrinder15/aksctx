package azure

import (
	"context"
	"fmt"
	"sort"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v6"
)

type AgentPool struct {
	Name              string
	Mode              string
	VMSize            string
	OSType            string
	Count             int32
	EnableAutoScaling bool
	MinCount          int32
	MaxCount          int32
}

// Cluster holds the relevant info for an AKS cluster.
type Cluster struct {
	Name                    string
	ID                      string
	ResourceGroup           string
	SubscriptionID          string
	SubscriptionName        string
	Location                string
	K8sVersion              string
	NodeCount               int32
	SKUTier                 string
	SupportPlan             string
	ProvisioningState       string
	PowerState              string
	EnableRBAC              bool
	EnableAzureRBAC         bool
	DisableLocalAccounts    bool
	PrivateCluster          bool
	AuthorizedIPRanges      int
	NetworkPlugin           string
	NetworkPolicy           string
	NetworkDataplane        string
	NetworkMode             string
	OutboundType            string
	DNSPrefix               string
	MonitoringAddonEnabled  bool
	AzurePolicyAddonEnabled bool
	NodePools               []AgentPool
}

// DisplayName returns a human-friendly label for the TUI picker.
func (c Cluster) DisplayName() string {
	return fmt.Sprintf("%s  [%s / %s]", c.Name, c.SubscriptionName, c.Location)
}

// ListAllClusters discovers AKS clusters across all accessible subscriptions,
// or within a single subscription when subscriptionName is provided.
func ListAllClusters(ctx context.Context, cred azcore.TokenCredential, subscriptionName string) ([]Cluster, error) {
	subs, err := listSubscriptions(ctx, cred, subscriptionName)
	if err != nil {
		return nil, err
	}

	var clusters []Cluster
	for _, sub := range subs {
		sc, err := listClustersInSubscription(ctx, cred, sub.ID, sub.Name)
		if err != nil {
			// Don't hard-fail on a single subscription (e.g. access denied)
			fmt.Printf("  ⚠ skipping subscription %s: %v\n", sub.Name, err)
			continue
		}
		clusters = append(clusters, sc...)
	}

	sort.Slice(clusters, func(i, j int) bool {
		if clusters[i].Name != clusters[j].Name {
			return clusters[i].Name < clusters[j].Name
		}
		if clusters[i].SubscriptionName != clusters[j].SubscriptionName {
			return clusters[i].SubscriptionName < clusters[j].SubscriptionName
		}
		return clusters[i].ResourceGroup < clusters[j].ResourceGroup
	})

	return clusters, nil
}

func ListClustersInSubscription(ctx context.Context, cred azcore.TokenCredential, sub Subscription) ([]Cluster, error) {
	clusters, err := listClustersInSubscription(ctx, cred, sub.ID, sub.Name)
	if err != nil {
		return nil, err
	}

	sort.Slice(clusters, func(i, j int) bool {
		if clusters[i].Name != clusters[j].Name {
			return clusters[i].Name < clusters[j].Name
		}
		return clusters[i].ResourceGroup < clusters[j].ResourceGroup
	})

	return clusters, nil
}

func listClustersInSubscription(ctx context.Context, cred azcore.TokenCredential, subID, subName string) ([]Cluster, error) {
	client, err := armcontainerservice.NewManagedClustersClient(subID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("creating AKS client: %w", err)
	}

	var clusters []Cluster
	pager := client.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing clusters: %w", err)
		}
		for _, mc := range page.Value {
			clusters = append(clusters, clusterFromManagedCluster(mc, subID, subName))
		}
	}
	return clusters, nil
}
