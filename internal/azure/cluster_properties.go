package azure

import (
	"sort"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v6"
)

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
