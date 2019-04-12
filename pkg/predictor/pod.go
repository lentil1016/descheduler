package predictor

import (
	"fmt"

	"github.com/lentil1016/descheduler/pkg/predicates"
	api_v1 "k8s.io/api/core/v1"
	policy "k8s.io/api/policy/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/kubelet/types"
)

// get evictable pods and rank them, then get the dedired number of pods to evict
func GetEvictPods(nodes []*api_v1.Node) ([]*api_v1.Pod, error) {
	evictSize := conf.Rules.MaxEvictSize
	var evictPods []*api_v1.Pod
	for _, node := range nodes {
		pods, err := getEvictablePods(node)
		if err != nil {
			fmt.Printf("Get evictable pods on %v failed, skipping this node. %v\n", node.ObjectMeta.Name, err)
		}
		rankedPods := rankEvictablePods(pods)
		evictPods = append(evictPods, rankedPods...)
		if len(evictPods) >= evictSize {
			fmt.Printf("maxEvictSize decide only top %v pods that marked as evict will be evicted.\n", evictSize)
			evictPods = evictPods[:evictSize]
			break
		}
	}
	return evictPods, nil
}

func rankEvictablePods(pods []*api_v1.Pod) []*api_v1.Pod {
	evicts := []*api_v1.Pod{}
	remains, newEvicts := evictUnfitPods(pods)
	evicts = append(evicts, newEvicts...)
	remains, newEvicts = evictWithPeerOnOneNode(remains)
	evicts = append(evicts, newEvicts...)
	remains, newEvicts = evictWithPeer(remains)
	evicts = append(evicts, newEvicts...)
	return evicts
}

// check if there is pod unfit its node, then mark as evicted
func evictUnfitPods(pods []*api_v1.Pod) (remainPods, evictPods []*api_v1.Pod) {
	var remains, evicts []*api_v1.Pod
	for _, pod := range pods {
		if pod.Spec.Affinity != nil &&
			pod.Spec.Affinity.NodeAffinity != nil &&
			pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution != nil &&
			!podFitsCurrentNode(pod) && podFitsAnySchedulableNode(pod) {
			// Pod have node affinity and can find a prefered node
			fmt.Printf("Find prefered node. %v marked as evicted\n", pod.Name)
			evicts = append(evicts, pod)
		} else {
			remains = append(remains, pod)
		}
	}
	return remains, evicts
}

func podFitsCurrentNode(pod *api_v1.Pod) bool {
	node, err := getPodNode(pod)
	if err != nil {
		fmt.Println("Get pod node failed, skipping process this pod", pod.Name, err)
		return true
	}
	ok, err := predicates.PodMatchNodeSelector(pod, node)

	if err != nil {
		fmt.Printf("Check if pod fit Current node failed, %v\n", err)
		return false
	}

	if !ok {
		fmt.Printf("Pod %v does not fit on node %v\n", pod.Name, node.Name)
		return false
	}

	fmt.Printf("Pod %v fits on node %v\n", pod.Name, node.Name)
	return true
}

func podFitsAnySchedulableNode(pod *api_v1.Pod) bool {

	nodes, err := getOperatableNodes()
	if err != nil {
		fmt.Println("Get operatable nodes failed, ", err)
	}

	for _, node := range nodes {
		ok, err := predicates.PodMatchNodeSelector(pod, node)
		if err != nil || !ok {
			continue
		}
		if ok {
			if isNodeSchedulable(node) {
				fmt.Printf("Pod %v can possibly be scheduled on %v\n", pod.Name, node.Name)
				return true
			}
			return false
		}
	}
	return false
}

func getEvictablePods(node *api_v1.Node) ([]*api_v1.Pod, error) {
	pods, err := getPodsOnNode(node)
	if err != nil {
		return []*api_v1.Pod{}, err
	}
	evictablePods := make([]*api_v1.Pod, 0)
	for _, pod := range pods {
		if !isEvictable(pod) {
			continue
		} else {
			evictablePods = append(evictablePods, pod)
			fmt.Println("Found pod that evictable:", pod.ObjectMeta.Name)
		}
	}
	return evictablePods, nil
}

// Checks if pod is evictable
func isEvictable(pod *api_v1.Pod) bool {
	ownerRefList := ownerRef(pod)
	if isMirrorPod(pod) ||
		isPodWithLocalStorage(pod) ||
		len(ownerRefList) == 0 ||
		isDaemonsetPod(ownerRefList) ||
		isCriticalPod(pod) {
		return false
	}
	return true
}

// ownerRef returns the ownerRefList for the pod.
func ownerRef(pod *api_v1.Pod) []v1.OwnerReference {
	return pod.ObjectMeta.GetOwnerReferences()
}

func isReplicaSetPod(ownerRefList []v1.OwnerReference) bool {
	for _, ownerRef := range ownerRefList {
		if ownerRef.Kind == "ReplicaSet" {
			return true
		}
	}
	return false
}

func isDaemonsetPod(ownerRefList []v1.OwnerReference) bool {
	for _, ownerRef := range ownerRefList {
		if ownerRef.Kind == "DaemonSet" {
			return true
		}
	}
	return false
}

// IsMirrorPod checks whether the pod is a mirror pod.
func isMirrorPod(pod *api_v1.Pod) bool {
	_, found := pod.ObjectMeta.Annotations[types.ConfigMirrorAnnotationKey]
	return found
}

func isPodWithLocalStorage(pod *api_v1.Pod) bool {
	for _, volume := range pod.Spec.Volumes {
		if volume.HostPath != nil || volume.EmptyDir != nil {
			return true
		}
	}
	return false
}

func isCriticalPod(pod *api_v1.Pod) bool {
	return types.IsCriticalPod(pod)
}

func MetaPodNodeIndexFunc(obj interface{}) ([]string, error) {
	meta, err := meta.Accessor(obj)
	if err != nil {
		return []string{""}, fmt.Errorf("object has no meta: %v", err)
	}
	return []string{meta.(*api_v1.Pod).Spec.NodeName}, nil
}

func getPodsOnNode(node *api_v1.Node) ([]*api_v1.Pod, error) {
	pods, err := indexers.podIndexer.ByIndex("byNode", node.ObjectMeta.Name)
	if err != nil {
		return []*api_v1.Pod{}, err
	}
	ret := []*api_v1.Pod{}
	for _, pod := range pods {
		ret = append(ret, pod.(*api_v1.Pod))
	}
	return ret, nil
}

func Evict(pods []*api_v1.Pod) {
	for _, pod := range pods {
		fmt.Println("Executing pod's eviction:", pod.ObjectMeta.Name)
		evictPod(pod)
	}
}

func evictPod(pod *api_v1.Pod) (bool, error) {
	if conf.DryRun {
		return true, nil
	}
	deleteOptions := &v1.DeleteOptions{}
	evictionVersion, _ := supportEviction()
	eviction := &policy.Eviction{
		TypeMeta: v1.TypeMeta{
			APIVersion: evictionVersion,
			Kind:       "Eviction",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      pod.Name,
			Namespace: pod.Namespace,
		},
		DeleteOptions: deleteOptions,
	}
	err := client.Policy().Evictions(eviction.Namespace).Evict(eviction)
	if err == nil {
		return true, nil
	} else if apierrors.IsTooManyRequests(err) {
		return false, fmt.Errorf("error when evicting pod (ignoring) %q: %v", pod.Name, err)
	} else if apierrors.IsNotFound(err) {
		return true, fmt.Errorf("pod not found when evicting %q: %v", pod.Name, err)
	} else {
		return false, err
	}
}
