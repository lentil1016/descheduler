package predictor

import (
	"github.com/lentil1016/descheduler/pkg/config"
	"k8s.io/client-go/kubernetes"
	lister_appv1 "k8s.io/client-go/listers/apps/v1"
	"k8s.io/client-go/tools/cache"
)

type indexersType struct {
	nodeIndexer cache.Indexer
	rsIndexer   cache.Indexer
}

var indexers indexersType
var client kubernetes.Interface
var conf config.ConfigSpec
var rsLister lister_appv1.ReplicaSetLister

func Init(nodeIndexer, rsIndexer cache.Indexer, clientset kubernetes.Interface) {
	indexers = indexersType{
		nodeIndexer: nodeIndexer,
		rsIndexer:   rsIndexer,
	}
	client = clientset
	conf = config.GetConfig()
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
