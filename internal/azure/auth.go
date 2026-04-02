package azure

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

// NewCredential returns a DefaultAzureCredential which automatically chains:
//  1. Environment variables (AZURE_CLIENT_ID, AZURE_CLIENT_SECRET, etc.)
//  2. Workload Identity (for in-cluster use)
//  3. Azure CLI credentials (az login)
//  4. Interactive browser login (fallback)
func NewCredential() (azcore.TokenCredential, error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to obtain Azure credential: %w\n\nTry running: az login", err)
	}
	return cred, nil
}
