package kubeconfig

import (
	"context"
	"fmt"
	"sort"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// ListNamespaces returns all namespaces visible from the provided context.
func ListNamespaces(contextName string) ([]string, error) {
	kubeconfigPath, err := defaultKubeconfigPath()
	if err != nil {
		return nil, err
	}

	loadingRules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath}
	overrides := &clientcmd.ConfigOverrides{CurrentContext: contextName}
	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides)

	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("building Kubernetes client config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("creating Kubernetes client: %w", err)
	}

	namespaceList, err := clientset.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing namespaces: %w", err)
	}

	names := make([]string, 0, len(namespaceList.Items))
	for _, namespace := range namespaceList.Items {
		names = append(names, namespace.Name)
	}

	sort.Strings(names)
	if defaultIdx := sort.SearchStrings(names, "default"); defaultIdx < len(names) && names[defaultIdx] == "default" {
		names[0], names[defaultIdx] = names[defaultIdx], names[0]
	}

	return names, nil
}

// ContextNamespace returns the configured default namespace for the context.
func ContextNamespace(contextName string) (string, error) {
	kubeconfigPath, err := defaultKubeconfigPath()
	if err != nil {
		return "", err
	}

	cfg, err := clientcmd.LoadFromFile(kubeconfigPath)
	if err != nil {
		return "", fmt.Errorf("loading kubeconfig: %w", err)
	}

	ctx, ok := cfg.Contexts[contextName]
	if !ok {
		return "", fmt.Errorf("context %q not found in kubeconfig", contextName)
	}

	return ctx.Namespace, nil
}

// SetContextNamespace updates the default namespace for the provided context.
func SetContextNamespace(contextName, namespace string) error {
	kubeconfigPath, err := defaultKubeconfigPath()
	if err != nil {
		return err
	}

	cfg, err := clientcmd.LoadFromFile(kubeconfigPath)
	if err != nil {
		return fmt.Errorf("loading kubeconfig: %w", err)
	}

	ctx, ok := cfg.Contexts[contextName]
	if !ok {
		return fmt.Errorf("context %q not found in kubeconfig", contextName)
	}

	ctx.Namespace = strings.TrimSpace(namespace)

	if err := clientcmd.WriteToFile(*cfg, kubeconfigPath); err != nil {
		return fmt.Errorf("writing kubeconfig: %w", err)
	}

	return nil
}
