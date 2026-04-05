package azure

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v6"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/subscription/armsubscription"
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

type Subscription struct {
	ID   string
	Name string
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
		sc, err := listClustersInSubscription(ctx, cred, sub.id, sub.name)
		if err != nil {
			// Don't hard-fail on a single subscription (e.g. access denied)
			fmt.Printf("  ⚠ skipping subscription %s: %v\n", sub.name, err)
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

type subscription struct {
	id   string
	name string
}

func ListSubscriptions(ctx context.Context, cred azcore.TokenCredential, subscriptionName string) ([]Subscription, error) {
	subs, err := listSubscriptions(ctx, cred, subscriptionName)
	if err != nil {
		return nil, err
	}

	result := make([]Subscription, 0, len(subs))
	for _, sub := range subs {
		result = append(result, Subscription{ID: sub.id, Name: sub.name})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result, nil
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

func listSubscriptions(ctx context.Context, cred azcore.TokenCredential, subscriptionName string) ([]subscription, error) {
	client, err := armsubscription.NewSubscriptionsClient(cred, nil)
	if err != nil {
		return nil, fmt.Errorf("creating subscriptions client: %w", err)
	}

	var subs []subscription
	filterName := strings.TrimSpace(subscriptionName)
	pager := client.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing subscriptions: %w", err)
		}
		for _, s := range page.Value {
			if s.SubscriptionID != nil && s.DisplayName != nil {
				if filterName != "" && !strings.EqualFold(*s.DisplayName, filterName) {
					continue
				}
				subs = append(subs, subscription{id: *s.SubscriptionID, name: *s.DisplayName})
			}
		}
	}

	if filterName != "" && len(subs) == 0 {
		return nil, fmt.Errorf("no Azure subscription matched name %q", filterName)
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
			clusters = append(clusters, clusterFromManagedCluster(mc, subID, subName))
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

	if strings.TrimSpace(cluster.ResourceGroup) == "" {
		return nil, fmt.Errorf("fetching kubeconfig for %s: resource group is empty", cluster.Name)
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
	if strings.TrimSpace(id) == "" {
		return ""
	}

	resourceID, err := arm.ParseResourceID(id)
	if err != nil || resourceID == nil {
		return ""
	}

	return resourceID.ResourceGroupName
}

func clusterFromManagedCluster(mc *armcontainerservice.ManagedCluster, subID, subName string) Cluster {
	c := Cluster{
		SubscriptionID:   subID,
		SubscriptionName: subName,
	}

	if mc == nil {
		return c
	}
	if mc.Name != nil {
		c.Name = *mc.Name
	}
	if mc.ID != nil {
		c.ID = *mc.ID
		c.ResourceGroup = resourceGroupFromID(*mc.ID)
	}
	if mc.Location != nil {
		c.Location = *mc.Location
	}
	if mc.SKU != nil && mc.SKU.Tier != nil {
		c.SKUTier = string(*mc.SKU.Tier)
	}

	if mc.Properties == nil {
		return c
	}

	props := mc.Properties
	if props.CurrentKubernetesVersion != nil && strings.TrimSpace(*props.CurrentKubernetesVersion) != "" {
		c.K8sVersion = *props.CurrentKubernetesVersion
	} else if props.KubernetesVersion != nil {
		c.K8sVersion = *props.KubernetesVersion
	}
	if props.SupportPlan != nil {
		c.SupportPlan = string(*props.SupportPlan)
	}
	if props.ProvisioningState != nil {
		c.ProvisioningState = *props.ProvisioningState
	}
	if props.PowerState != nil && props.PowerState.Code != nil {
		c.PowerState = string(*props.PowerState.Code)
	}
	if props.EnableRBAC != nil {
		c.EnableRBAC = *props.EnableRBAC
	}
	if props.AADProfile != nil && props.AADProfile.EnableAzureRBAC != nil {
		c.EnableAzureRBAC = *props.AADProfile.EnableAzureRBAC
	}
	if props.DisableLocalAccounts != nil {
		c.DisableLocalAccounts = *props.DisableLocalAccounts
	}
	if props.DNSPrefix != nil {
		c.DNSPrefix = *props.DNSPrefix
	}
	if props.APIServerAccessProfile != nil {
		if props.APIServerAccessProfile.EnablePrivateCluster != nil {
			c.PrivateCluster = *props.APIServerAccessProfile.EnablePrivateCluster
		}
		c.AuthorizedIPRanges = len(props.APIServerAccessProfile.AuthorizedIPRanges)
	}
	if props.NetworkProfile != nil {
		if props.NetworkProfile.NetworkPlugin != nil {
			c.NetworkPlugin = string(*props.NetworkProfile.NetworkPlugin)
		}
		if props.NetworkProfile.NetworkPolicy != nil {
			c.NetworkPolicy = string(*props.NetworkProfile.NetworkPolicy)
		}
		if props.NetworkProfile.NetworkDataplane != nil {
			c.NetworkDataplane = string(*props.NetworkProfile.NetworkDataplane)
		}
		if props.NetworkProfile.NetworkMode != nil {
			c.NetworkMode = string(*props.NetworkProfile.NetworkMode)
		}
		if props.NetworkProfile.OutboundType != nil {
			c.OutboundType = string(*props.NetworkProfile.OutboundType)
		}
	}
	if props.AddonProfiles != nil {
		c.MonitoringAddonEnabled = addonEnabled(props.AddonProfiles, "omsagent", "azuremonitor-containers")
		c.AzurePolicyAddonEnabled = addonEnabled(props.AddonProfiles, "azurepolicy")
	}
	if len(props.AgentPoolProfiles) > 0 {
		c.NodePools = make([]AgentPool, 0, len(props.AgentPoolProfiles))
		for _, pool := range props.AgentPoolProfiles {
			if pool == nil {
				continue
			}
			nodePool := AgentPool{}
			if pool.Name != nil {
				nodePool.Name = *pool.Name
			}
			if pool.Mode != nil {
				nodePool.Mode = string(*pool.Mode)
			}
			if pool.VMSize != nil {
				nodePool.VMSize = *pool.VMSize
			}
			if pool.OSType != nil {
				nodePool.OSType = string(*pool.OSType)
			}
			if pool.Count != nil {
				nodePool.Count = *pool.Count
				c.NodeCount += *pool.Count
			}
			if pool.EnableAutoScaling != nil {
				nodePool.EnableAutoScaling = *pool.EnableAutoScaling
			}
			if pool.MinCount != nil {
				nodePool.MinCount = *pool.MinCount
			}
			if pool.MaxCount != nil {
				nodePool.MaxCount = *pool.MaxCount
			}
			c.NodePools = append(c.NodePools, nodePool)
		}
		sort.Slice(c.NodePools, func(i, j int) bool {
			return c.NodePools[i].Name < c.NodePools[j].Name
		})
	}

	return c
}

func addonEnabled(addons map[string]*armcontainerservice.ManagedClusterAddonProfile, names ...string) bool {
	for _, name := range names {
		profile, ok := addons[name]
		if !ok || profile == nil || profile.Enabled == nil {
			continue
		}
		if *profile.Enabled {
			return true
		}
	}

	return false
}
