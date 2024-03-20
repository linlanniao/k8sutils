package controller

import (
	"fmt"

	"github.com/linlanniao/k8sutils"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type JobOnAddedUpdatedFunc func(key string, job *batchv1.Job) error
type JobOnDeletedFunc func(key string) error

type JobHandler struct {
	name               string
	namespace          string
	workers            int
	onAddedUpdatedFunc JobOnAddedUpdatedFunc
	onDeletedFunc      JobOnDeletedFunc
	clientset          *k8sutils.Clientset
	watchKey           string
	watchValue         string
}

func (j JobHandler) RESTClient() rest.Interface {
	return j.clientset.GetClientSet().BatchV1().RESTClient()
}

func (j JobHandler) WatchKeyValue() (key, value string) {
	return j.watchKey, j.watchValue
}

func NewJobHandler(
	name string,
	namespace string,
	workers int,
	onAddedUpdatedFunc JobOnAddedUpdatedFunc,
	onDeletedFunc JobOnDeletedFunc,
	clientset *k8sutils.Clientset,
	watchKey string,
	watchValue string,
) Controller {
	ph := &JobHandler{
		name:               name,
		clientset:          clientset,
		namespace:          namespace,
		workers:            workers,
		onAddedUpdatedFunc: onAddedUpdatedFunc,
		onDeletedFunc:      onDeletedFunc,
		watchKey:           watchKey,
		watchValue:         watchValue,
	}

	h := newController(ph)
	return h
}

var _ handler = (*JobHandler)(nil) // check if JobHandler implements the handler interface

func (j JobHandler) Name() string {
	return j.name
}

func (j JobHandler) Namespace() string {
	return j.namespace
}

func (j JobHandler) OnAddedUpdated(key string, obj any) error {
	job, ok := obj.(*batchv1.Job)
	if !ok {
		return fmt.Errorf("unexpected type %T", obj)
	}

	return j.onAddedUpdatedFunc(key, job)
}

func (j JobHandler) OnDeleted(key string) error {
	return j.onDeletedFunc(key)
}

func (j JobHandler) GetWorkers() int {
	return j.workers
}

func (j JobHandler) Kind() runtime.Object {
	return &batchv1.Job{}
}

func (j JobHandler) Resource() string {
	return "jobs"
}

func (j JobHandler) ClientSet() *kubernetes.Clientset {
	return j.clientset.GetClientSet()
}
