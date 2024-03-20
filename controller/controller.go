package controller

import (
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

const (
	reSyncPeriod       = 5 * time.Minute
	defaultWorkers int = 3
)

type onAddedUpdatedFunc func(key string, obj any) error
type onDeletedFunc func(key string) error

type controller struct {
	name               string
	namespace          string
	resource           string
	kind               runtime.Object
	indexer            cache.Indexer
	queue              workqueue.RateLimitingInterface
	informer           cache.Controller
	workers            int
	onAddedUpdatedFunc onAddedUpdatedFunc
	onDeletedFunc      onDeletedFunc
}

type handler interface {
	Name() string
	Namespace() string
	OnAddedUpdated(key string, obj any) error
	OnDeleted(key string) error
	GetWorkers() int
	Kind() runtime.Object
	Resource() string
	RESTClient() rest.Interface
	WatchKeyValue() (key, value string)
}

// newController creates a new controller.
func newController(h handler) *controller {

	c := &controller{
		name:               h.Name(),
		resource:           h.Resource(),
		kind:               h.Kind(),
		indexer:            nil,
		queue:              nil,
		informer:           nil,
		workers:            h.GetWorkers(),
		onAddedUpdatedFunc: h.OnAddedUpdated,
		onDeletedFunc:      h.OnDeleted,
	}

	optionsModifier := func(options *metav1.ListOptions) {
		options.LabelSelector = newSelector(h.WatchKeyValue()).String()
	}

	listWatcher := cache.NewFilteredListWatchFromClient(
		h.RESTClient(),
		h.Resource(),
		h.Namespace(),
		optionsModifier,
	)

	// setup queue
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	c.queue = queue

	// setup informer and indexer
	indexer, informer := cache.NewIndexerInformer(listWatcher, h.Kind(), reSyncPeriod, cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(key)
			}
		},
		UpdateFunc: func(oldObj, newObj any) {
			key, err := cache.MetaNamespaceKeyFunc(newObj)
			if err == nil {
				queue.Add(key)
			}
		},
		DeleteFunc: func(obj any) {
			// IndexerInformer uses a delta queue, therefore for deletes we have to use this
			// key function.
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(key)
			}
		},
	}, cache.Indexers{})
	c.informer = informer
	c.indexer = indexer

	// setup workers
	if c.workers <= 0 {
		c.workers = defaultWorkers
	}

	return c
}

func (c *controller) processNextItem() bool {
	// Wait until there is a new item in the working queue
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	// Tell the queue that we are done with processing this key. This unblocks the key for other workers
	// This allows safe parallel processing because two pods with the same key are never processed in
	// parallel.
	defer c.queue.Done(key)

	// Invoke the method containing the business logic
	err := c.processBusiness(key.(string))
	// Handle the error if something went wrong during the execution of the business logic
	c.handleErr(err, key)
	return true
}

// process is the business logic of the controller. In this controller it simply prints
// information about the pod to stdout. In case an error happened, it has to simply return the error.
// The retry logic should not be part of the business logic.
func (c *controller) processBusiness(key string) error {
	obj, exists, err := c.indexer.GetByKey(key)
	if err != nil {
		klog.Errorf("fetching object with key %s from store failed with %v", key, err)
		return err
	}

	if !exists {
		// Below we will warm up our cache with a Obj, so that we will see a delete for one Obj
		klog.Infof("deleting object: %s", key)
		return c.onDeletedFunc(key)
	} else {
		// Note that you also have to check the uid if you have a local controlled resource, which
		// is dependent on the actual instance, to detect that a Pod was recreated with the same name
		klog.Infof("sync/add/update for object: %s", key)
		return c.onAddedUpdatedFunc(key, obj)
	}
}

// handleErr checks if an error happened and makes sure we will retry later.
func (c *controller) handleErr(err error, key interface{}) {
	if err == nil {
		// Forget about the #AddRateLimited history of the key on every successful synchronization.
		// This ensures that future processing of updates for this key is not delayed because of
		// an outdated error history.
		c.queue.Forget(key)
		return
	}

	// This controller retries 5 times if something goes wrong. After that, it stops trying.
	if c.queue.NumRequeues(key) < 5 {
		klog.Infof("error syncing obj %v: %v", key, err)

		// Re-enqueue the key rate limited. Based on the rate limiter on the
		// queue and the re-enqueue history, the key will be processed later again.
		c.queue.AddRateLimited(key)
		return
	}

	c.queue.Forget(key)
	// Report to an external entity that, even after several retries, we could not successfully process this key
	utilruntime.HandleError(err)
	klog.Infof("dropping obj %q out of the queue: %v", key, err)
}

func (c *controller) runWorker() {
	for c.processNextItem() {
	}
}

// Run begins watching and syncing.
func (c *controller) Run(stopCh chan struct{}) {
	defer utilruntime.HandleCrash()

	// Let the workers stop when we are done
	defer c.queue.ShutDown()
	klog.Infof("starting handler, name=%s, resource=%s, workers=%d", c.name, c.resource, c.workers)

	go c.informer.Run(stopCh)

	// Wait for all involved caches to be synced, before processing items from the queue is started
	if !cache.WaitForCacheSync(stopCh, c.informer.HasSynced) {
		utilruntime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
		return
	}

	for i := 0; i < c.workers; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	klog.Infof("stopping %s controller", c.name)
}

func newSelector(key, value string) labels.Selector {
	selector := labels.NewSelector()

	req, _ := labels.NewRequirement(key, selection.Equals, []string{value})

	selector = selector.Add(*req)

	return selector
}

func (c *controller) Namespace() string {
	return c.namespace
}
