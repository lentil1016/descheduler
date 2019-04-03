package predictor

import (
	api_v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	api "k8s.io/kubernetes/pkg/apis/core"
)

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
