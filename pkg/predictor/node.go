package predictor

import (
	"fmt"

	api_v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
	v1_resource "k8s.io/kubernetes/pkg/api/v1/resource"
)

// Splite node into high spared nodes list and low spared state nodes list
func InBadSparedState() (highSpared, lowSpared []*api_v1.Node, ok bool) {
	operatableNodes, _ := getOperatableNodes()
	if len(operatableNodes) < 2 {
		fmt.Println("Deschedule event droped because Operatable node is less than 2")
		return []*api_v1.Node{}, []*api_v1.Node{}, false
	}
	for _, node := range operatableNodes {
		cpuUsage, memUsage, podUsage, err := getNodeUsage(node)
		if err != nil {
			fmt.Println("Deschedule event droped, failed to get node usage, ", err)
			return []*api_v1.Node{}, []*api_v1.Node{}, false
		}
		usageScore, sparedScore := scoreNode(cpuUsage, memUsage, podUsage)
		fmt.Println(usageScore, sparedScore)
	}
	return []*api_v1.Node{}, []*api_v1.Node{}, true
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

func scoreNode(cpuUsage, memUsage, podUsage float64) (float64, float64) {
	var usageScore, sparedScore float64
	usageScore, sparedScore = scoreResource(cpuUsage,
		(100 - conf.Triggers.MinSparedPercentage.CPU),
		conf.Triggers.MaxSparedPercentage.CPU,
		usageScore, sparedScore)
	usageScore, sparedScore = scoreResource(memUsage,
		(100 - conf.Triggers.MinSparedPercentage.Memory),
		conf.Triggers.MaxSparedPercentage.Memory,
		usageScore, sparedScore)
	usageScore, sparedScore = scoreResource(podUsage,
		(100 - conf.Triggers.MinSparedPercentage.Pod),
		conf.Triggers.MaxSparedPercentage.Pod,
		usageScore, sparedScore)
	return usageScore, sparedScore
}

func scoreResource(usage, maxUsage, maxSpared, usageScore, sparedScore float64) (float64, float64) {
	spared := 100 - usage
	if usage > maxUsage {
		usageScore = usageScore + usage*usage/100
	} else if spared > maxSpared {
		sparedScore = sparedScore + spared*spared/100
	}
	return usageScore, sparedScore
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

func IsNodeReady(node *api_v1.Node) bool {
	return isNodeOperatable(node) && isNodeSchedulable(node)
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
