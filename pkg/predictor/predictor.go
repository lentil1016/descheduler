package predictor

import (
	"github.com/lentil1016/descheduler/pkg/config"
	"k8s.io/client-go/kubernetes"
	lister_appv1 "k8s.io/client-go/listers/apps/v1"
	lister_apiv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
)

type indexersType struct {
	nodeIndexer cache.Indexer
	rsIndexer   cache.Indexer
}

var indexers indexersType
var client kubernetes.Interface
var conf config.ConfigSpec
var nodeLister lister_apiv1.NodeLister
var rsLister lister_appv1.ReplicaSetLister

func Init(nodeIndexer, rsIndexer cache.Indexer, clientset kubernetes.Interface) {
	indexers = indexersType{
		nodeIndexer: nodeIndexer,
		rsIndexer:   rsIndexer,
	}
	client = clientset
	conf = config.GetConfig()
	nodeLister = lister_apiv1.NewNodeLister(nodeIndexer)
	rsLister = lister_appv1.NewReplicaSetLister(rsIndexer)
}

func scoreNode(cpuUsage, memUsage, podUsage float64) (float64, float64, float64) {
	var usageScore, sparedScore, normalScore float64
	usageScore, sparedScore, normalScore = scoreResource(cpuUsage,
		(100 - conf.Triggers.MinSparedPercentage.CPU),
		conf.Triggers.MaxSparedPercentage.CPU,
		usageScore, sparedScore, normalScore)
	usageScore, sparedScore, normalScore = scoreResource(memUsage,
		(100 - conf.Triggers.MinSparedPercentage.Memory),
		conf.Triggers.MaxSparedPercentage.Memory,
		usageScore, sparedScore, normalScore)
	usageScore, sparedScore, normalScore = scoreResource(podUsage,
		(100 - conf.Triggers.MinSparedPercentage.Pod),
		conf.Triggers.MaxSparedPercentage.Pod,
		usageScore, sparedScore, normalScore)
	return usageScore, sparedScore, normalScore
}

func scoreResource(usage, maxUsage, maxSpared, usageScore, sparedScore, normalScore float64) (float64, float64, float64) {
	spared := 100 - usage
	normalScore = normalScore + usage*usage/100
	if usage > maxUsage {
		usageScore = usageScore + usage*usage/100
	} else if spared > maxSpared {
		sparedScore = sparedScore + spared*spared/100
	}
	return usageScore, sparedScore, normalScore
}

// SupportEviction uses Discovery API to find out if the server support eviction subresource
// If support, it will return its groupVersion; Otherwise, it will return ""
func supportEviction() (string, error) {
	discoveryClient := client.Discovery()
	groupList, err := discoveryClient.ServerGroups()
	if err != nil {
		return "", err
	}
	foundPolicyGroup := false
	var policyGroupVersion string
	for _, group := range groupList.Groups {
		if group.Name == "policy" {
			foundPolicyGroup = true
			policyGroupVersion = group.PreferredVersion.GroupVersion
			break
		}
	}
	if !foundPolicyGroup {
		return "", nil
	}
	resourceList, err := discoveryClient.ServerResourcesForGroupVersion("v1")
	if err != nil {
		return "", err
	}
	for _, resource := range resourceList.APIResources {
		if resource.Name == "pods/eviction" && resource.Kind == "Eviction" {
			return policyGroupVersion, nil
		}
	}
	return "", nil
}
