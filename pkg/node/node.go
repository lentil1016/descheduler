package node

import (
	api_v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

func GetReadyNodes(indexer cache.Indexer) ([]*api_v1.Node, error) {
	// Get all nodes
	var nodes []*api_v1.Node
	err := cache.ListAll(indexer, labels.Everything(), func(m interface{}) {
		nodes = append(nodes, m.(*api_v1.Node))
	})
	if err != nil {
		return []*api_v1.Node{}, err
	}

	// Select the nodes that is ready
	readyNodes := make([]*api_v1.Node, 0, len(nodes))
	for _, node := range nodes {
		if IsReady(node) {
			readyNodes = append(readyNodes, node)
		}
	}
	return nodes, nil
}

func IsReady(node *api_v1.Node) bool {
	ready := true
	for i := range node.Status.Conditions {
		cond := &node.Status.Conditions[i]
		if cond.Type == api_v1.NodeReady && cond.Status != api_v1.ConditionTrue {
			ready = false
		} else if cond.Type == api_v1.NodeOutOfDisk && cond.Status != api_v1.ConditionFalse {
			ready = false
		} else if cond.Type == api_v1.NodeNetworkUnavailable && cond.Status != api_v1.ConditionFalse {
			ready = false
		}
	}
	if node.Spec.Unschedulable {
		ready = false
	}
	return ready
}
