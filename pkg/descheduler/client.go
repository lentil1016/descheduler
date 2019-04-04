package descheduler

import (
	"fmt"
	"os"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func CreateClient(kubeConfigFile string) (kubernetes.Interface, error) {
	var cfg *rest.Config
	var err error
	if _, err = os.Stat(kubeConfigFile); os.IsNotExist(err) {
		cfg, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("Failed to build config from incluster config: %v", err)
		}
	} else {
		cfg, err = clientcmd.BuildConfigFromFlags("", kubeConfigFile)
		if err != nil {
			return nil, fmt.Errorf("Failed to build config from kubeconfig file: %v ", err)
		}
	}
	return kubernetes.NewForConfig(cfg)
}
