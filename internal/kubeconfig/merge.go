package kubeconfig

import (
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// MergeAndSwitch merges the provided kubeconfig bytes into ~/.kube/config
// and sets the current context to the cluster's context name.
func MergeAndSwitch(kubeconfigBytes []byte, clusterName string) error {
	// Parse the incoming kubeconfig
	incoming, err := clientcmd.Load(kubeconfigBytes)
	if err != nil {
		return fmt.Errorf("parsing kubeconfig: %w", err)
	}

	kubeconfigPath, err := defaultKubeconfigPath()
	if err != nil {
		return err
	}

	// Load existing kubeconfig (or start fresh if it doesn't exist)
	existing := clientcmdapi.NewConfig()
	if _, err := os.Stat(kubeconfigPath); err == nil {
		existing, err = clientcmd.LoadFromFile(kubeconfigPath)
		if err != nil {
			return fmt.Errorf("loading existing kubeconfig: %w", err)
		}
	}

	// Merge clusters, users, and contexts from incoming into existing
	for k, v := range incoming.Clusters {
		existing.Clusters[k] = v
	}
	for k, v := range incoming.AuthInfos {
		existing.AuthInfos[k] = v
	}
	for k, v := range incoming.Contexts {
		existing.Contexts[k] = v
	}

	// Set the current context to the first context in the incoming config
	// (AKS returns exactly one context per kubeconfig)
	if incoming.CurrentContext != "" {
		existing.CurrentContext = incoming.CurrentContext
	}

	// Write back
	if err := clientcmd.WriteToFile(*existing, kubeconfigPath); err != nil {
		return fmt.Errorf("writing kubeconfig: %w", err)
	}

	return nil
}

// CurrentContext returns the current kubeconfig context name and cluster server.
func CurrentContext() (contextName, server string, err error) {
	kubeconfigPath, err := defaultKubeconfigPath()
	if err != nil {
		return "", "", err
	}

	cfg, err := clientcmd.LoadFromFile(kubeconfigPath)
	if err != nil {
		return "", "", fmt.Errorf("loading kubeconfig: %w", err)
	}

	contextName = cfg.CurrentContext
	if contextName == "" {
		return "", "", fmt.Errorf("no current context set in kubeconfig")
	}

	ctx, ok := cfg.Contexts[contextName]
	if !ok {
		return contextName, "", nil
	}

	cluster, ok := cfg.Clusters[ctx.Cluster]
	if ok && cluster != nil {
		server = cluster.Server
	}

	return contextName, server, nil
}

func defaultKubeconfigPath() (string, error) {
	if env := os.Getenv("KUBECONFIG"); env != "" {
		return env, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("finding home directory: %w", err)
	}
	return filepath.Join(home, ".kube", "config"), nil
}
