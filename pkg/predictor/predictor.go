package predictor

import (
	"github.com/lentil1016/descheduler/pkg/config"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type indexersType struct {
	nodeIndexer cache.Indexer
	rsIndexer   cache.Indexer
	podIndexer  cache.Indexer
}

var indexers indexersType
var client kubernetes.Interface
var conf config.ConfigSpec

func Init(nodeIndexer, rsIndexer, podIndexer cache.Indexer, client kubernetes.Interface) {
	indexers = indexersType{
		nodeIndexer: nodeIndexer,
		rsIndexer:   rsIndexer,
		podIndexer:  podIndexer,
	}
	client = client
	conf = config.GetConfig()
}
