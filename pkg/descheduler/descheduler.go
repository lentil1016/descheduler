package descheduler

import (
	"fmt"
	"time"

	"github.com/lentil1016/descheduler/pkg/config"
	"github.com/lentil1016/descheduler/pkg/handler"
	"github.com/lentil1016/descheduler/pkg/predictor"
	"github.com/lentil1016/descheduler/pkg/timer"
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

type descheduler struct {
	clientset    kubernetes.Interface
	queue        workqueue.RateLimitingInterface
	nodeInformer cache.SharedIndexInformer
	rsInformer   cache.SharedIndexInformer
	podInformer  cache.SharedIndexInformer
}

type Descheduler interface {
	Run(stopCh chan struct{})
}

func CreateDescheduler() (Descheduler, error) {
	conf := config.GetConfig()

	kubeconfig := conf.KubeConfigFile
	fmt.Println("Using kubeconfig file:", kubeconfig)
	client, err := CreateClient(kubeconfig)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	// create a work queue
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

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

	// create a replica set informer
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

	// create a pod informer, just used as cache, won't bind event handler.
	podInformer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (k8sruntime.Object, error) {
				return client.CoreV1().Pods("").List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				return client.CoreV1().Pods("").Watch(options)
			},
		},
		&api_v1.Pod{},
		0,
		cache.Indexers{"byNamespace": cache.MetaNamespaceIndexFunc})

	nodeInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		// Only handle the update event, because nodes get ready with an update event ultimately.
		UpdateFunc: func(old, new interface{}) {
			// Healthy nodes will push update event constantly.
			// Push event only when pod is getting ready.
			if !predictor.IsNodeReady(old.(*api_v1.Node)) && predictor.IsNodeReady(new.(*api_v1.Node)) {
				key, err := cache.MetaNamespaceKeyFunc(old)
				if err == nil {
					queue.Add(handler.NewEvent(key, "getReady", "node"))
				}
			}
		},
	})

	rsInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		// Only handle the update event, because replicaSet get ready with an update event.
		UpdateFunc: func(old, new interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(old)
			if err == nil {
				queue.Add(handler.NewEvent(key, "rsUpdate", "replicaSet"))
			}
		},
	})

	err = timer.InitTimer(func() {
		queue.Add(handler.NewEvent("", "onTime", "timer"))
	})
	if err != nil {
		return nil, err
	}

	predictor.Init(nodeInformer.GetIndexer(),
		rsInformer.GetIndexer(),
		podInformer.GetIndexer(),
		client)

	return &descheduler{
		clientset:    client,
		queue:        queue,
		nodeInformer: nodeInformer,
		rsInformer:   rsInformer,
		podInformer:  podInformer,
	}, nil
}

func (d *descheduler) Run(stopCh chan struct{}) {
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
		if !cache.WaitForCacheSync(ch, d.rsInformer.HasSynced) {
			runtime.HandleError(fmt.Errorf("Timed out waiting for raplica sets caches to sync"))
			return
		}
	}
	{
		ch := make(chan struct{})
		defer close(ch)
		go d.podInformer.Run(ch)
		if !cache.WaitForCacheSync(ch, d.podInformer.HasSynced) {
			runtime.HandleError(fmt.Errorf("Timed out waiting for raplica sets caches to sync"))
			return
		}
	}
	fmt.Println("descheduler synced and ready")

	// Timer will start if descheduler is configred as time triggered mode
	timer.RunTimer()

	wait.Until(d.runWorker, time.Second, stopCh)
}

func (d *descheduler) runWorker() {
	for d.processNextItem() {
		// continue looping
	}
}

func (d *descheduler) processNextItem() bool {
	newEvent, quit := d.queue.Get()
	if quit {
		return false
	}
	defer d.queue.Done(newEvent)

	event := newEvent.(handler.Event)
	handler.Type(event).Handle(event)
	return true
}
