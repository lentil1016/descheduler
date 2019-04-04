package predictor

import (
	"fmt"
	"sort"

	api_v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
	v1_resource "k8s.io/kubernetes/pkg/api/v1/resource"
)

type nodeScore struct {
	node  *api_v1.Node
	score float64
}

// Splite node into high spared nodes list and low spared state nodes list
func GetBusyNodes() ([]*api_v1.Node, bool) {
	operatableNodes, _ := getOperatableNodes()
	// TODO: remove comment
	/*if len(operatableNodes) < 2 {
		fmt.Println("Deschedule event droped because Operatable node is less than 2")
		return []*api_v1.Node{}, []*api_v1.Node{}, false
	}
	*/
	// ranking nodes by most spared and most usage
	var sparedRank, usageRank, normalRank []nodeScore
	for _, node := range operatableNodes {
		nodeName := node.ObjectMeta.Name
		cpuUsage, memUsage, podUsage, err := getNodeUsage(node)
		if err != nil {
			fmt.Println("Deschedule event aborted, failed to get node usage, ", err)
			return []*api_v1.Node{}, false
		}
		usageScore, sparedScore, normalScore := scoreNode(cpuUsage, memUsage, podUsage)

		if usageScore != 0 {
			// High Usage node, marked if any resource is running low.
			fmt.Printf("Node %v is marked as a high usage node\n", nodeName)
			usageRank = append(usageRank, nodeScore{node, sparedScore})
		} else if sparedScore != 0 && isNodeSchedulable(node) {
			// High spared node, marked if some resource is highly spared
			// and node is schedulable, and no resource is running low.
			fmt.Printf("Node %v is marked as a high spared node\n", nodeName)
			sparedRank = append(sparedRank, nodeScore{node, sparedScore})
		} else {
			// Normal node, returned as usage node when there is no usage node.
			fmt.Printf("Node %v is marked as a normal node\n", nodeName)
			normalRank = append(normalRank, nodeScore{node, normalScore})
		}
	}
	// Do ranking
	sort.Slice(sparedRank, func(i, j int) bool { return sparedRank[i].score > sparedRank[j].score })
	sort.Slice(usageRank, func(i, j int) bool { return usageRank[i].score > usageRank[j].score })
	sort.Slice(normalRank, func(i, j int) bool { return normalRank[i].score > normalRank[j].score })

	// Put rank into slice
	var sparedNodes, usageNodes, normalNodes []*api_v1.Node
	for _, rankItem := range sparedRank {
		sparedNodes = append(sparedNodes, rankItem.node)
	}
	for _, rankItem := range usageRank {
		usageNodes = append(usageNodes, rankItem.node)
	}
	for _, rankItem := range normalRank {
		normalNodes = append(normalNodes, rankItem.node)
	}

	if len(normalNodes) == 0 {
		if len(usageNodes) == 0 {
			fmt.Println("Deschedule event aborted, all nodes are spared, nothing to deschedule")
			return []*api_v1.Node{}, false
		} else if len(sparedNodes) == 0 {
			fmt.Println("Deschedule event aborted, all nodes are busy, can't deschedule")
			return []*api_v1.Node{}, false
		}
	} else if len(usageNodes) == 0 {
		// TODO: remove comment
		/*if len(sparedNodes) == 0 {
			fmt.Println("Deschedule event aborted, all nodes are normal, nothing to deschedule")
			return []*api_v1.Node{}, false
		} else*/{
			fmt.Println("All nodes reserved sufficient resource. Try deschedule anyway.")
			return normalNodes, true
		}
	}
	return usageNodes, true
}

func IsNodeReady(node *api_v1.Node) bool {
	return isNodeOperatable(node) && isNodeSchedulable(node)
}

func getNodeUsage(node *api_v1.Node) (float64, float64, float64, error) {
	pods, err := getPodsOnNode(node)
	if err != nil {
		return 0, 0, 0, err
	}
	totalReqs := map[api_v1.ResourceName]resource.Quantity{}
	for _, pod := range pods {
		requests, _ := v1_resource.PodRequestsAndLimits(pod)
		for res, req := range requests {
			if res == api_v1.ResourceCPU || res == api_v1.ResourceMemory {
				if oldReq, ok := totalReqs[res]; !ok {
					totalReqs[res] = *req.Copy()
				} else {
					oldReq.Add(req)
					totalReqs[res] = oldReq
				}
			}
		}
	}
	nodeCapacity := node.Status.Capacity
	if len(node.Status.Allocatable) > 0 {
		nodeCapacity = node.Status.Allocatable
	}
	totalCPUReq := totalReqs[api_v1.ResourceCPU]
	totalMemReq := totalReqs[api_v1.ResourceMemory]
	totalPods := len(pods)
	cpuUsage := float64((float64(totalCPUReq.MilliValue()) * 100) / float64(nodeCapacity.Cpu().MilliValue()))
	memUsage := float64(float64(totalMemReq.Value()) / float64(nodeCapacity.Memory().Value()) * 100)
	podUsage := float64((float64(totalPods) * 100) / float64(nodeCapacity.Pods().Value()))
	return cpuUsage, memUsage, podUsage, nil
}

func getPodNode(pod *api_v1.Pod) (*api_v1.Node, error) {
	node, err := nodeLister.Get(pod.Spec.NodeName)
	return node, err
}

func getOperatableNodes() ([]*api_v1.Node, error) {
	// Get all nodes
	var nodes []*api_v1.Node
	err := cache.ListAll(indexers.nodeIndexer, labels.Everything(), func(m interface{}) {
		nodes = append(nodes, m.(*api_v1.Node))
	})
	if err != nil {
		return []*api_v1.Node{}, err
	}

	// Select the nodes that is ready
	readyNodes := make([]*api_v1.Node, 0, len(nodes))
	for _, node := range nodes {
		if isNodeOperatable(node) {
			readyNodes = append(readyNodes, node)
		}
	}
	return nodes, nil
}

func isNodeOperatable(node *api_v1.Node) bool {
	ready := true
	for i := range node.Status.Conditions {
		cond := &node.Status.Conditions[i]
		// Won't deschedule any pod in or out from the nodes that:
		// - is not in NodeReady condition, because default scheduler will do that.
		// - is in NodeNetworkUnavailable condition, because it will be a labor in vain
		// - is in NodeOutOfDisk condition, because it won't solve the real issue,
		// just delay it.
		if cond.Type == api_v1.NodeReady && cond.Status != api_v1.ConditionTrue {
			ready = false
		} else if cond.Type == api_v1.NodeOutOfDisk && cond.Status != api_v1.ConditionFalse {
			ready = false
		} else if cond.Type == api_v1.NodeNetworkUnavailable && cond.Status != api_v1.ConditionFalse {
			ready = false
		}
	}
	return ready
}

func isNodeSchedulable(node *api_v1.Node) bool {
	if node.Spec.Unschedulable {
		return false
	} else {
		return true
	}
}
