package predictor

import (
	"fmt"

	api_v1 "k8s.io/api/core/v1"
	policy "k8s.io/api/policy/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	api "k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/kubernetes/pkg/kubelet/types"
)

func GetEvictPods(nodes []*api_v1.Node, evictSize int) ([]*api_v1.Pod, error) {
	var evictPods []*api_v1.Pod
	for _, node := range nodes {
		pods, err := getEvictablePods(node)
		if err != nil {
			fmt.Printf("Get evictable pods on %v failed, skipping this node. %v\n", node.ObjectMeta.Name, err)
		}
		rankedPods := rankEvictablePods(pods)
		evictPods = append(evictPods, rankedPods...)
		if len(evictPods) >= evictSize {
			evictPods = evictPods[:evictSize]
			break
		}
	}
	return evictPods, nil
}

func rankEvictablePods(pods []*api_v1.Pod) []*api_v1.Pod {
	return pods
}

func getUnfitPods(node *api_v1.Node) []*api_v1.Pod {
	return []*api_v1.Pod{}
}

func getPodsHaveReplicas(node *api_v1.Node) []*api_v1.Pod {
	return []*api_v1.Pod{}
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
			fmt.Println("Found pod that evictable: ", pod.ObjectMeta.Name)
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

func getPodsOnNode(node *api_v1.Node) ([]*api_v1.Pod, error) {
	fieldSelector, err := fields.ParseSelector("spec.nodeName=" + node.Name + ",status.phase!=" + string(api.PodFailed))
	if err != nil {
		return []*api_v1.Pod{}, err
	}
	return getPods(fieldSelector)
}

func getPods(fieldSelector fields.Selector) ([]*api_v1.Pod, error) {
	podList, err := client.CoreV1().Pods(api_v1.NamespaceAll).List(
		v1.ListOptions{FieldSelector: fieldSelector.String()})
	if err != nil {
		return []*api_v1.Pod{}, err
	}

	pods := make([]*api_v1.Pod, 0)
	for i := range podList.Items {
		pods = append(pods, &podList.Items[i])
	}
	return pods, nil
}

func Evict(pods []*api_v1.Pod) {
	for _, pod := range pods {
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
