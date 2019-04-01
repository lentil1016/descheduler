package client

import (
	"fmt"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func CreateClient(kubeConfigFile string) (kubernetes.Interface, error) {
	var cfg *rest.Config
	var err error
	if kubeConfigFile != "" {
		cfg, err = clientcmd.BuildConfigFromFlags("", kubeConfigFile)
		if err != nil {
			return nil, fmt.Errorf("Failed to build config from kubeconfig file: %v ", err)
		}
	} else {
		cfg, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("Failed to build config from incluster config: %v", err)
		}
	}
	return kubernetes.NewForConfig(cfg)
}
