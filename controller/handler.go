package controller

import (
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

const (
	handlerKey         = "handler.k8sutils.ppops.cn"
	resyncPeriod       = 2 * time.Minute
	defaultWorkers int = 3
)

type OnAddedUpdatedFunc func(key string, obj any) error
type OnDeletedFunc func(key string) error

type Handler struct {
	name                string
	resource            string
	kind                runtime.Object
	indexer             cache.Indexer
	queue               workqueue.RateLimitingInterface
	informer            cache.Controller
	restClient          rest.Interface
	workers             int
	onAddedUpdatedFuncs []OnAddedUpdatedFunc
	onDeletedFuncs      []OnDeletedFunc
}

// NewHandler creates a new Handler.
func NewHandler(
	name string,
	resource string,
	namespace string,
	kind runtime.Object,
	restClient rest.Interface,
	workers int,
	onAddedUpdatedFuncs []OnAddedUpdatedFunc,
	onDeletedFuncs []OnDeletedFunc,
) *Handler {

	optionsModifier := func(options *metav1.ListOptions) {
		s := fmt.Sprintf("%s/%s=%s", handlerKey, resource, name)
		options.LabelSelector = s
	}

	listWatcher := cache.NewFilteredListWatchFromClient(restClient, resource, namespace, optionsModifier)

	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	indexer, informer := cache.NewIndexerInformer(listWatcher, kind, resyncPeriod, cache.ResourceEventHandlerFuncs{
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

	if workers >= 0 {
		workers = defaultWorkers
	}

	return &Handler{
		name:                name,
		resource:            resource,
		kind:                kind,
		indexer:             indexer,
		queue:               queue,
		informer:            informer,
		restClient:          restClient,
		workers:             workers,
		onAddedUpdatedFuncs: onAddedUpdatedFuncs,
		onDeletedFuncs:      onDeletedFuncs,
	}
}

func (h *Handler) processNextItem() bool {
	// Wait until there is a new item in the working queue
	key, quit := h.queue.Get()
	if quit {
		return false
	}
	// Tell the queue that we are done with processing this key. This unblocks the key for other workers
	// This allows safe parallel processing because two pods with the same key are never processed in
	// parallel.
	defer h.queue.Done(key)

	// Invoke the method containing the business logic
	err := h.processBusiness(key.(string))
	// Handle the error if something went wrong during the execution of the business logic
	h.handleErr(err, key)
	return true
}

// process is the business logic of the handler. In this handler it simply prints
// information about the pod to stdout. In case an error happened, it has to simply return the error.
// The retry logic should not be part of the business logic.
func (h *Handler) processBusiness(key string) error {
	obj, exists, err := h.indexer.GetByKey(key)
	if err != nil {
		klog.Errorf("fetching object with key %s from store failed with %v", key, err)
		return err
	}

	if !exists {
		// Below we will warm up our cache with a Obj, so that we will see a delete for one Obj
		klog.Infof("deleting object: %s", key)
		if len(h.onDeletedFuncs) == 0 {
			klog.Infof("onDeletedFuncs is empty, skip.")
			return nil
		}
		for _, fn := range h.onDeletedFuncs {
			if err := fn(key); err != nil {
				return err
			}
		}
		return nil
	} else {
		// Note that you also have to check the uid if you have a local controlled resource, which
		// is dependent on the actual instance, to detect that a Pod was recreated with the same name
		klog.Infof("sync/add/update for object: %s", key)
		if len(h.onAddedUpdatedFuncs) == 0 {
			klog.Info("onAddedUpdatedFuncs is empty, skip.")
		}
		for _, fn := range h.onAddedUpdatedFuncs {
			if err := fn(key, obj); err != nil {
				return err
			}
		}
		return nil
	}
}

// handleErr checks if an error happened and makes sure we will retry later.
func (h *Handler) handleErr(err error, key interface{}) {
	if err == nil {
		// Forget about the #AddRateLimited history of the key on every successful synchronization.
		// This ensures that future processing of updates for this key is not delayed because of
		// an outdated error history.
		h.queue.Forget(key)
		return
	}

	// This handler retries 5 times if something goes wrong. After that, it stops trying.
	if h.queue.NumRequeues(key) < 5 {
		klog.Infof("error syncing obj %v: %v", key, err)

		// Re-enqueue the key rate limited. Based on the rate limiter on the
		// queue and the re-enqueue history, the key will be processed later again.
		h.queue.AddRateLimited(key)
		return
	}

	h.queue.Forget(key)
	// Report to an external entity that, even after several retries, we could not successfully process this key
	utilruntime.HandleError(err)
	klog.Infof("dropping obj %q out of the queue: %v", key, err)
}

func (h *Handler) runWorker() {
	for h.processNextItem() {
	}
}

// Run begins watching and syncing.
func (h *Handler) Run(stopCh chan struct{}) {
	defer utilruntime.HandleCrash()

	// Let the workers stop when we are done
	defer h.queue.ShutDown()
	klog.Infof("starting handler, name=%s, resource=%s, workers=%d", h.name, h.resource, h.workers)

	go h.informer.Run(stopCh)

	// Wait for all involved caches to be synced, before processing items from the queue is started
	if !cache.WaitForCacheSync(stopCh, h.informer.HasSynced) {
		utilruntime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
		return
	}

	for i := 0; i < h.workers; i++ {
		go wait.Until(h.runWorker, time.Second, stopCh)
	}

	<-stopCh
	klog.Infof("stopping %s handler", h.name)
}
