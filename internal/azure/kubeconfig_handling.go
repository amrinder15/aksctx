package azure

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v6"
)

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
