package descheduler

import (
	"fmt"
	"time"

	"github.com/lentil1016/descheduler/pkg/client"
	"github.com/lentil1016/descheduler/pkg/config"
	"github.com/lentil1016/descheduler/pkg/node"
	apps_v1 "k8s.io/api/apps/v1"
	api_v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
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

type Descheduler struct {
	clientset    kubernetes.Interface
	queue        workqueue.RateLimitingInterface
	nodeInformer cache.SharedIndexInformer
	rsInformer   cache.SharedIndexInformer
}

var conf config.ConfigSpec

func CreateDescheduler() (Descheduler, error) {
	conf = config.GetConfig()

	kubeconfig := conf.KubeConfigFile
	fmt.Println("Using kubeconfig file:", kubeconfig)
	client, err := client.CreateClient(kubeconfig)
	if err != nil {
		fmt.Println(err)
		return Descheduler{}, err
	}
	// create a work queue
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	var newEvent Event

	// create a node informer with node selector
	nodeInformer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (k8sruntime.Object, error) {
				options.LabelSelector = conf.Rules.NodeSelector
				return client.CoreV1().Nodes().List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				options.LabelSelector = conf.Rules.NodeSelector
				return client.CoreV1().Nodes().Watch(options)
			},
		},
		&api_v1.Node{},
		0,
		cache.Indexers{})

	// create a replica set informer with node selector
	rsInformer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (k8sruntime.Object, error) {
				return client.AppsV1().ReplicaSets("").List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				return client.AppsV1().ReplicaSets("").Watch(options)
			},
		},
		&apps_v1.ReplicaSet{},
		0,
		cache.Indexers{"byNamespace": cache.MetaNamespaceIndexFunc})

	descheduler := &Descheduler{
		clientset:    client,
		queue:        queue,
		nodeInformer: nodeInformer,
		rsInformer:   rsInformer,
	}

	nodeInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		// Only handle the update event, because all nodes get ready with a update event ultimately.
		UpdateFunc: func(old, new interface{}) {
			// Healthy nodes will push update event constantly.
			// Push event only when pod is getting ready.
			if !node.IsReady(old.(*api_v1.Node)) && node.IsReady(new.(*api_v1.Node)) {
				newEvent.key, err = cache.MetaNamespaceKeyFunc(old)
				newEvent.eventType = "getReady"
				newEvent.resourceType = "node"
				if err == nil {
					queue.Add(newEvent)
				}
			}
		},
	})

	rsInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		// Only handle the update event, because all nodes get ready with a update event ultimately.
		UpdateFunc: func(old, new interface{}) {
			// Healthy nodes will push update event constantly.
			// Push event only when pod is getting ready.
			newEvent.key, err = cache.MetaNamespaceKeyFunc(old)
			newEvent.eventType = "rsUpdate"
			newEvent.resourceType = "replicaSet"
			if err == nil {
				queue.Add(newEvent)
			}
		},
	})

	return *descheduler, nil
}

func (d *Descheduler) Run(stopCh chan struct{}) {
	defer runtime.HandleCrash()
	defer d.queue.ShutDown()

	fmt.Println("Starting descheduler")
	serverStartTime = time.Now().Local()

	{
		ch := make(chan struct{})
		defer close(ch)
		go d.nodeInformer.Run(ch)
		if !cache.WaitForCacheSync(ch, d.nodeInformer.HasSynced) {
			runtime.HandleError(fmt.Errorf("Timed out waiting for nodes caches to sync"))
			return
		}
	}
	{
		ch := make(chan struct{})
		defer close(ch)
		go d.rsInformer.Run(ch)
		if !cache.WaitForCacheSync(ch, d.nodeInformer.HasSynced) {
			runtime.HandleError(fmt.Errorf("Timed out waiting for raplica sets caches to sync"))
			return
		}
	}

	fmt.Println("descheduler synced and ready")

	wait.Until(d.runWorker, time.Second, stopCh)
}

func (d *Descheduler) runWorker() {
	for d.processNextItem() {
		// continue looping
	}
}

func (d *Descheduler) processNextItem() bool {
	newEvent, quit := d.queue.Get()
	if quit {
		return false
	}

	defer d.queue.Done(newEvent)

	if newEvent.(Event).resourceType == "node" {
		err := d.processNodeItem(newEvent.(Event))
		if err != nil {
			return false
		}
	}
	return true
}

func (d *Descheduler) processNodeItem(newEvent Event) error {
	nodes, err := node.GetReadyNodes(d.nodeInformer.GetIndexer())
	if err != nil {
		fmt.Println("Failed to get ready nodes:", err)
		return err
	}
	fmt.Println(nodes)
	return nil
}
