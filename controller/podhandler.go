package controller

import (
	"github.com/linlanniao/k8sutils"
	corev1 "k8s.io/api/core/v1"
)

func NewPodHandler(
	name string,
	namespace string,
	workers int,
	addedUpdatedFunctions []OnAddedUpdatedFunc,
	deletedFunctions []OnDeletedFunc,
) *Handler {
	resource := "pods"
	kind := &corev1.Pod{}
	clientset := k8sutils.GetClientset()
	restClient := clientset.GetClientSet().CoreV1().RESTClient()
	h := NewHandler(name, resource, namespace, kind, restClient, workers, addedUpdatedFunctions, deletedFunctions)
	return h
}
