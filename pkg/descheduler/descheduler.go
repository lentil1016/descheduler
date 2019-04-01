package descheduler

import (
	"fmt"
	"time"

	"github.com/lentil1016/descheduler/pkg/handler"
	api_v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

var serverStartTime time.Time

const maxRetries = 5

type Event struct {
	key          string
	eventType    string
	namespace    string
	resourceType string
}

type descheduler struct {
	clientset    kubernetes.Interface
	queue        workqueue.RateLimitingInterface
	podInformer  cache.SharedIndexInformer
	nodeInformer cache.SharedIndexInformer
	eventHandler handler.Handler
}

func CreateDescheduler(client kubernetes.Interface) (descheduler, error) {
	// create a work queue
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	var newEvent Event
	var err error

	// create a node informer
	nodeListWatcher := cache.NewListWatchFromClient(
		client.CoreV1().RESTClient(),
		"nodes",
		v1.NamespaceAll,
		fields.Everything(),
	)
	nodeInformer := cache.NewSharedIndexInformer(nodeListWatcher, &api_v1.Node{}, 0, cache.Indexers{})

	nodeInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			newEvent.key, err = cache.MetaNamespaceKeyFunc(obj)
			newEvent.eventType = "add"
			newEvent.resourceType = "node"
			if err == nil {
				queue.Add(newEvent)
			}
		},
	})

	// create a pod informer
	podListWatcher := cache.NewListWatchFromClient(
		client.CoreV1().RESTClient(),
		"pods",
		v1.NamespaceAll,
		fields.Everything(),
	)
	podInformer := cache.NewSharedIndexInformer(podListWatcher, &api_v1.Pod{}, 0, cache.Indexers{})

	podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			newEvent.key, err = cache.MetaNamespaceKeyFunc(obj)
			newEvent.eventType = "add"
			newEvent.resourceType = "pod"
			if err == nil {
				queue.Add(newEvent)
			}
		},
	})

	controller := &descheduler{
		clientset:    client,
		queue:        queue,
		podInformer:  podInformer,
		nodeInformer: nodeInformer,
	}
	return *controller, nil
}

func (c *descheduler) Run(stopCh <-chan struct{}) {
	defer runtime.HandleCrash()
	defer c.queue.ShutDown()

	fmt.Println("Starting descheduler")
	serverStartTime = time.Now().Local()

	go c.nodeInformer.Run(stopCh)
	if !cache.WaitForCacheSync(stopCh, c.nodeInformer.HasSynced) {
		runtime.HandleError(fmt.Errorf("Timed out waiting for caches to sync"))
		return
	}

	go c.podInformer.Run(stopCh)
	if !cache.WaitForCacheSync(stopCh, c.podInformer.HasSynced) {
		runtime.HandleError(fmt.Errorf("Timed out waiting for caches to sync"))
		return
	}

	fmt.Println("descheduler synced and ready")

	wait.Until(c.runWorker, time.Second, stopCh)
}

func (c *descheduler) runWorker() {
	for c.processNextItem() {
		// continue looping
	}
}

func (c *descheduler) processNextItem() bool {
	return true
}
