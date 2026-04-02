# aksctx

> AKS context switcher — discover and switch between AKS clusters across Azure subscriptions without memorizing resource group names.

```
$ aksctx switch

🔍 Discovering AKS clusters...

Select an AKS cluster
╭──────────────────────────────╮
│ type to filter...            │
╰──────────────────────────────╯

▶ aks-prod-eastus      prod-subscription / eastus / k8s 1.29.4 / 6 nodes
  aks-staging-westeu   dev-subscription / westeurope / k8s 1.28.9 / 3 nodes
  aks-dev-eastus        dev-subscription / eastus / k8s 1.28.9 / 2 nodes

  3/3  ↑↓ navigate  enter select  esc quit

✅ Switched to cluster: aks-prod-eastus
   Subscription : prod-subscription
   Resource Group: rg-platform-prod
   Location      : eastus
   K8s Version   : 1.29.4
```

## Why

Switching between AKS clusters normally requires:
```bash
az aks get-credentials --resource-group rg-platform-prod --name aks-prod-eastus --subscription <uuid>
kubectx aks-prod-eastus
```

You need to know the exact resource group and subscription ID by heart. `aksctx` discovers all clusters across all your subscriptions and lets you fuzzy-search and switch in one command.

## Commands

| Command          | Description                                              |
|------------------|----------------------------------------------------------|
| `aksctx switch`  | Interactive fuzzy picker — select and switch to a cluster |
| `aksctx list`    | List all AKS clusters across subscriptions in a table   |
| `aksctx current` | Show the current active context and API server           |
| `aksctx refresh` | Re-fetch credentials for the current cluster            |

## Install

**Homebrew (macOS/Linux):**
```bash
brew install amrinder/tap/aksctx
```

**Go install:**
```bash
go install github.com/amrinder/aksctx@latest
```

**Build from source:**
```bash
git clone https://github.com/amrinder/aksctx
cd aksctx
go build -o aksctx .
mv aksctx /usr/local/bin/
```

## Authentication

`aksctx` uses `DefaultAzureCredential` which tries the following in order:

1. **Environment variables** — `AZURE_CLIENT_ID`, `AZURE_CLIENT_SECRET`, `AZURE_TENANT_ID`
2. **Workload Identity** — for in-cluster or CI use
3. **Azure CLI** — if you've run `az login`
4. **Interactive browser** — fallback; opens a browser window

No extra configuration needed if you've already run `az login`.

## Architecture

```
aksctx switch
      │
      ▼
DefaultAzureCredential (azidentity)
      │
      ▼
armsubscription.NewSubscriptionsClient
  └─ list all accessible subscriptions
      │
      ▼
armcontainerservice.NewManagedClustersClient (per subscription)
  └─ list all AKS clusters
      │
      ▼
bubbletea TUI picker (fuzzy search)
      │
      ▼
ManagedClusters.ListClusterUserCredentials()
  └─ returns raw kubeconfig bytes
      │
      ▼
client-go clientcmd.Merge
  └─ merges into ~/.kube/config
  └─ sets current-context
```

## Requirements

- Go 1.22+
- Azure subscription(s) with AKS clusters
- `az login` or another supported credential method

## Contributing

PRs welcome. The core logic lives in:
- `internal/azure/clusters.go` — Azure SDK calls
- `internal/tui/picker.go` — bubbletea UI
- `internal/kubeconfig/merge.go` — kubeconfig manipulation
