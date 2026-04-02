package azure

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v6"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/subscription/armsubscription"
)

// Cluster holds the relevant info for an AKS cluster.
type Cluster struct {
	Name           string
	ResourceGroup  string
	SubscriptionID string
	SubscriptionName string
	Location       string
	K8sVersion     string
	NodeCount      int32
}

// DisplayName returns a human-friendly label for the TUI picker.
func (c Cluster) DisplayName() string {
	return fmt.Sprintf("%s  [%s / %s]", c.Name, c.SubscriptionName, c.Location)
}

// ListAllClusters discovers AKS clusters across all accessible subscriptions.
func ListAllClusters(ctx context.Context, cred azcore.TokenCredential) ([]Cluster, error) {
	subs, err := listSubscriptions(ctx, cred)
	if err != nil {
		return nil, err
	}

	var clusters []Cluster
	for _, sub := range subs {
		sc, err := listClustersInSubscription(ctx, cred, sub.id, sub.name)
		if err != nil {
			// Don't hard-fail on a single subscription (e.g. access denied)
			fmt.Printf("  ⚠ skipping subscription %s: %v\n", sub.name, err)
			continue
		}
		clusters = append(clusters, sc...)
	}

	return clusters, nil
}

type subscription struct {
	id   string
	name string
}

func listSubscriptions(ctx context.Context, cred azcore.TokenCredential) ([]subscription, error) {
	client, err := armsubscription.NewSubscriptionsClient(cred, nil)
	if err != nil {
		return nil, fmt.Errorf("creating subscriptions client: %w", err)
	}

	var subs []subscription
	pager := client.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing subscriptions: %w", err)
		}
		for _, s := range page.Value {
			if s.SubscriptionID != nil && s.DisplayName != nil {
				subs = append(subs, subscription{id: *s.SubscriptionID, name: *s.DisplayName})
			}
		}
	}
	return subs, nil
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
			c := Cluster{
				SubscriptionID:   subID,
				SubscriptionName: subName,
			}
			if mc.Name != nil {
				c.Name = *mc.Name
			}
			if mc.Location != nil {
				c.Location = *mc.Location
			}
			if mc.Properties != nil && mc.Properties.KubernetesVersion != nil {
				c.K8sVersion = *mc.Properties.KubernetesVersion
			}
			// Extract resource group from the ARM resource ID
			if mc.ID != nil {
				c.ResourceGroup = resourceGroupFromID(*mc.ID)
			}
			// Sum node counts across all agent pools
			if mc.Properties != nil {
				for _, pool := range mc.Properties.AgentPoolProfiles {
					if pool.Count != nil {
						c.NodeCount += *pool.Count
					}
				}
			}
			clusters = append(clusters, c)
		}
	}
	return clusters, nil
}

// GetClusterKubeconfig fetches the admin kubeconfig for a given cluster.
func GetClusterKubeconfig(ctx context.Context, cred azcore.TokenCredential, cluster Cluster) ([]byte, error) {
	client, err := armcontainerservice.NewManagedClustersClient(cluster.SubscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("creating AKS client: %w", err)
	}

	result, err := client.ListClusterUserCredentials(ctx, cluster.ResourceGroup, cluster.Name, nil)
	if err != nil {
		return nil, fmt.Errorf("fetching kubeconfig for %s: %w", cluster.Name, err)
	}

	if len(result.Kubeconfigs) == 0 {
		return nil, fmt.Errorf("no kubeconfig returned for cluster %s", cluster.Name)
	}

	return result.Kubeconfigs[0].Value, nil
}

// resourceGroupFromID parses the resource group name from an ARM resource ID.
// e.g. /subscriptions/{sub}/resourceGroups/{rg}/providers/...
func resourceGroupFromID(id string) string {
	const marker = "/resourceGroups/"
	idx := indexOf(id, marker)
	if idx < 0 {
		return ""
	}
	rest := id[idx+len(marker):]
	end := indexOf(rest, "/")
	if end < 0 {
		return rest
	}
	return rest[:end]
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
