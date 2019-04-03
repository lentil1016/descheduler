package predictor

import (
	"fmt"

	"github.com/kubernetes/kubernetes/pkg/kubelet/types"
	api_v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	api "k8s.io/kubernetes/pkg/apis/core"
)

func GetEvictablePods(nodes []*api_v1.Node) ([]*api_v1.Pod, error) {
	pods, err := getPodsOnNode(nodes[0])
	if err != nil {
		return []*api_v1.Pod{}, err
	}
	evictablePods := make([]*api_v1.Pod, 0)
	for _, pod := range pods {
		if !isEvictable(pod) {
			fmt.Println("UnEvictable, ", pod.ObjectMeta.Name)
			continue
		} else {
			evictablePods = append(evictablePods, pod)
			fmt.Println(pod.ObjectMeta.Name)
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

func getUnfitPods(node *api_v1.Node) []*api_v1.Pod {
	return []*api_v1.Pod{}
}

func isWithReplica(node *api_v1.Node) []*api_v1.Pod {
	return []*api_v1.Pod{}

}

func getPodsHaveReplicas(node *api_v1.Node) []*api_v1.Pod {
	return []*api_v1.Pod{}
}

func getPodsOnNode(node *api_v1.Node) ([]*api_v1.Pod, error) {
	fieldSelector, err := fields.ParseSelector("spec.nodeName=" + node.Name + ",status.phase!=" + string(api.PodFailed))
	if err != nil {
		return []*api_v1.Pod{}, err
	}

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
