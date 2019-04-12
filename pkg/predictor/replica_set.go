package predictor

import (
	"fmt"

	apps_v1 "k8s.io/api/apps/v1"
	api_v1 "k8s.io/api/core/v1"
)

func GetReplicaSetByKey(key string) *apps_v1.ReplicaSet {
	rs, _, err := indexers.rsIndexer.GetByKey(key)
	if err != nil {
		fmt.Println("Recover event abort because of error: ", err)
		return nil
	}
	return rs.(*apps_v1.ReplicaSet)
}

// check if there is peer pods on same node, then mark as evicted
func evictWithPeerOnOneNode(pods []*api_v1.Pod) (remainPods, evictPods []*api_v1.Pod) {
	rpm := make(map[string]*api_v1.Pod, 0)
	rsEvictedNames := make(map[string]bool, 0)
	var remains, evicts []*api_v1.Pod
	for _, pod := range pods {
		// if is a pod created by replica set
		ownerRefList := ownerRef(pod)
		if isReplicaSetPod(ownerRefList) {
			rsName := GetPodReplicaSetName(pod)
			// if there is another pod's ReplicaSet is the same with this one
			if peerPod, ok := rpm[rsName]; ok {
				fmt.Printf("Find peer %v on current node. %v marked as evicted\n", peerPod.Name, pod.Name)
				rsEvictedNames[rsName] = true
				evicts = append(evicts, pod)
			} else {
				rpm[rsName] = pod
			}
		} else {
			// not processed by this evict function
			remains = append(remains, pod)
		}
	}
	for rsName, pod := range rpm {
		if _, ok := rsEvictedNames[rsName]; ok {
			// this pod's peer on this node is marked as evicted,
			// should remain this one here.
			fmt.Printf("Pin pod %v on current node for this schedule term.\n", pod.Name)
		} else {
			remains = append(remains, pod)
		}
	}
	return remains, evicts
}

// Check if there is peer pods in cluster, then mark as evicted
func evictWithPeer(pods []*api_v1.Pod) (remainPods, evictPods []*api_v1.Pod) {
	var remains, evicts []*api_v1.Pod
	for _, pod := range pods {
		ownerRefList := ownerRef(pod)
		if isReplicaSetPod(ownerRefList) {
			rs := getPodReplicaSet(pod)
			if rs.Status.ReadyReplicas > 1 {
				// pod have living peer on other nodes.
				fmt.Printf("Find living peers. %v marked as evicted\n", pod.Name)
				evicts = append(evicts, pod)
			} else {
				remains = append(remains, pod)
			}
		} else {
			remains = append(remains, pod)
		}
	}
	return remains, evicts
}

func GetPodReplicaSetName(pod *api_v1.Pod) string {
	rs := getPodReplicaSet(pod)
	if rs != nil {
		return rs.ObjectMeta.Name
	}
	return ""
}

func IsReplicaSetReady(rs *apps_v1.ReplicaSet) bool {
	return rs.Status.Replicas == rs.Status.ReadyReplicas
}

func getPodReplicaSet(pod *api_v1.Pod) *apps_v1.ReplicaSet {
	ownerRefList := ownerRef(pod)
	if isReplicaSetPod(ownerRefList) {
		rss, err := rsLister.GetPodReplicaSets(pod)
		if err != nil || len(rss) != 1 {
			return nil
		}
		return rss[0]
	}
	return nil
}
