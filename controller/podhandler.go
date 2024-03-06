package controller

import (
	"fmt"

	"github.com/linlanniao/k8sutils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
)

type PodOnAddedUpdatedFunc func(key string, obj *corev1.Pod) error
type PodOnDeletedFunc func(key string) error

type PodHandler struct {
	name               string
	namespace          string
	workers            int
	onAddedUpdatedFunc PodOnAddedUpdatedFunc
	onDeletedFunc      PodOnDeletedFunc
	clientset          *k8sutils.Clientset
}

func NewPodHandler(
	name string,
	namespace string,
	workers int,
	onAddedUpdatedFunc PodOnAddedUpdatedFunc,
	onDeletedFunc PodOnDeletedFunc,
	clientset *k8sutils.Clientset,
) Handler {
	ph := &PodHandler{
		name:               name,
		clientset:          clientset,
		namespace:          namespace,
		workers:            workers,
		onAddedUpdatedFunc: onAddedUpdatedFunc,
		onDeletedFunc:      onDeletedFunc,
	}

	h := newBaseHandler(ph)
	return h
}

var _ handlerService = (*PodHandler)(nil) // check if PodHandler implements the handlerService interface

func (p PodHandler) Namespace() string {
	return p.namespace
}

func (p PodHandler) ClientSet() *kubernetes.Clientset {
	return p.clientset.GetClientSet()
}

func (p PodHandler) Name() string {
	return p.name
}

func (p PodHandler) OnAddedUpdated(key string, obj any) error {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return fmt.Errorf("unexpected type %T", obj)
	}

	return p.onAddedUpdatedFunc(key, pod)
}

func (p PodHandler) OnDeleted(key string) error {
	return p.onDeletedFunc(key)
}

func (p PodHandler) GetWorkers() int {
	return p.workers
}

func (p PodHandler) Kind() runtime.Object {
	return &corev1.Pod{}
}

func (p PodHandler) Resource() string {
	return "pods"
}
