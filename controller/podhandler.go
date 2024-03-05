package controller

import (
	"github.com/linlanniao/k8sutils"
	corev1 "k8s.io/api/core/v1"
)

func NewPodHandler(
	name string,
	namespace string,
	workers int,
	onAddedUpdatedFuncs []OnAddedUpdatedFunc,
	onDeletedFuncs []OnDeletedFunc,
) *Handler {
	resource := "pods"
	kind := &corev1.Pod{}
	clientset := k8sutils.GetClientset()
	restClient := clientset.GetClientSet().CoreV1().RESTClient()
	h := NewHandler(name, resource, namespace, kind, restClient, workers, onAddedUpdatedFuncs, onDeletedFuncs)
	return h
}
