package kubeconfig

import (
	"fmt"
	"sort"
	"strings"

	"k8s.io/client-go/tools/clientcmd"
)

// Context describes a kubeconfig context available for selection.
type Context struct {
	Name      string
	Cluster   string
	Namespace string
	Server    string
	IsCurrent bool
}

// ListContexts returns all contexts from the kubeconfig sorted by name,
// with the current context listed first when present.
func ListContexts() ([]Context, error) {
	kubeconfigPath, err := defaultKubeconfigPath()
	if err != nil {
		return nil, err
	}

	cfg, err := clientcmd.LoadFromFile(kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("loading kubeconfig: %w", err)
	}

	contexts := make([]Context, 0, len(cfg.Contexts))
	for name, ctx := range cfg.Contexts {
		context := Context{
			Name:      name,
			Namespace: strings.TrimSpace(ctx.Namespace),
			IsCurrent: name == cfg.CurrentContext,
		}
		if ctx != nil {
			context.Cluster = ctx.Cluster
		}
		if cluster, ok := cfg.Clusters[context.Cluster]; ok && cluster != nil {
			context.Server = cluster.Server
		}
		contexts = append(contexts, context)
	}

	if len(contexts) == 0 {
		return nil, nil
	}

	sort.Slice(contexts, func(i, j int) bool {
		if contexts[i].IsCurrent != contexts[j].IsCurrent {
			return contexts[i].IsCurrent
		}
		return strings.ToLower(contexts[i].Name) < strings.ToLower(contexts[j].Name)
	})

	return contexts, nil
}

// SetCurrentContext updates the kubeconfig current-context value.
func SetCurrentContext(contextName string) error {
	kubeconfigPath, err := defaultKubeconfigPath()
	if err != nil {
		return err
	}

	cfg, err := clientcmd.LoadFromFile(kubeconfigPath)
	if err != nil {
		return fmt.Errorf("loading kubeconfig: %w", err)
	}

	if _, ok := cfg.Contexts[contextName]; !ok {
		return fmt.Errorf("context %q not found in kubeconfig", contextName)
	}

	cfg.CurrentContext = contextName

	if err := clientcmd.WriteToFile(*cfg, kubeconfigPath); err != nil {
		return fmt.Errorf("writing kubeconfig: %w", err)
	}

	return nil
}
