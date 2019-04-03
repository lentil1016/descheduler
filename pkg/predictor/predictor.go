package predictor

import (
	"k8s.io/client-go/tools/cache"
)

type indexersType struct {
	nodeIndexer cache.Indexer
	rsIndexer   cache.Indexer
	podIndexer  cache.Indexer
}

var indexers indexersType

func Init(nodeIndexer, rsIndexer, podIndexer cache.Indexer) {
	indexers = indexersType{
		nodeIndexer: nodeIndexer,
		rsIndexer:   rsIndexer,
		podIndexer:  podIndexer,
	}
}
