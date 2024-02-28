package controller

import (
	"github.com/linlanniao/k8sutils"
	corev1 "k8s.io/api/core/v1"
)

func NewPodHandler(
	name string,
	workers int,
	addedUpdatedFunctions []OnAddedUpdatedFunc,
	deletedFunctions []OnDeletedFunc,
) *Handler {

	resource := "pods"
	kind := &corev1.Pod{}
	clientset, err := k8sutils.GetClientset()
	if err != nil {
		panic(err.Error())
	}
	restClient := clientset.GetClientSet().CoreV1().RESTClient()

	h := NewHandler(
		name,
		resource,
		kind,
		restClient,
		workers,
		addedUpdatedFunctions,
		deletedFunctions,
	)
	return h
}
