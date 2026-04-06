package azure

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/subscription/armsubscription"
)

type Subscription struct {
	ID   string
	Name string
}

func ListSubscriptions(ctx context.Context, cred azcore.TokenCredential, subscriptionName string) ([]Subscription, error) {
	subs, err := listSubscriptions(ctx, cred, subscriptionName)
	if err != nil {
		return nil, err
	}

	sort.Slice(subs, func(i, j int) bool {
		return subs[i].Name < subs[j].Name
	})

	return subs, nil
}

func listSubscriptions(ctx context.Context, cred azcore.TokenCredential, subscriptionName string) ([]Subscription, error) {
	client, err := armsubscription.NewSubscriptionsClient(cred, nil)
	if err != nil {
		return nil, fmt.Errorf("creating subscriptions client: %w", err)
	}

	var subs []Subscription
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
				subs = append(subs, Subscription{ID: *s.SubscriptionID, Name: *s.DisplayName})
			}
		}
	}

	if filterName != "" && len(subs) == 0 {
		return nil, fmt.Errorf("no Azure subscription matched name %q", filterName)
	}

	return subs, nil
}
