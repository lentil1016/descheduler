package predictor

import (
	"fmt"

	apps_v1 "k8s.io/api/apps/v1"
	api_v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	api "k8s.io/kubernetes/pkg/apis/core"
)

func GetReplicaSetByKey(key string) *apps_v1.ReplicaSet {
	rs, _, err := indexers.rsIndexer.GetByKey(key)
	if err != nil {
		fmt.Println("Recover event abort because of error: ", err)
		return nil
	}
	return rs.(*apps_v1.ReplicaSet)
}

func CheckReplicas(pods []*api_v1.Pod) []*api_v1.Pod {
	for _, pod := range pods {
		rs := getPodReplicaSet(pod)
		if rs != nil {
			// TODO
			getPodsInReplicaSet(rs)
		}
	}
	return []*api_v1.Pod{}
}

func GetPodReplicaSetName(pod *api_v1.Pod) string {
	rs := getPodReplicaSet(pod)
	return rs.ObjectMeta.Name
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

func getPodsInReplicaSet(rs *apps_v1.ReplicaSet) ([]*api_v1.Pod, error) {
	fieldSelector, err := fields.ParseSelector("spec.nodeName=" + ",status.phase!=" + string(api.PodFailed))
	if err != nil {
		return []*api_v1.Pod{}, err
	}
	return getPods(fieldSelector)
}

func IsReplicaSetReady(rs *apps_v1.ReplicaSet) bool {
	return rs.Status.Replicas == rs.Status.ReadyReplicas
}
