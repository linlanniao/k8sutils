package controller

import (
	"fmt"

	"github.com/linlanniao/k8sutils"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
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
}

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
