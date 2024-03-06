package kbatch

import (
	"context"

	corev1 "k8s.io/api/core/v1"
)

type ITaskStorage interface {
	Get(ctx context.Context, name string) (*Task, error)
	Create(ctx context.Context, task *Task) (*Task, error)
	Update(ctx context.Context, task *Task) (*Task, error)
	Delete(ctx context.Context, name string) error
	List(ctx context.Context) ([]*Task, error)
}

type ITaskCallback interface {
	Name() string
	Namespace() string
	Workers() int
	OnPodCreatedFunc() (ctx context.Context, pod *corev1.Pod)
	//OnAddedUpdatedFunc() controller.PodOnAddedUpdatedFunc
	//OnDeletedFunc() controller.PodOnDeletedFunc
	//TODO OnPodAdd
	//TODO OnPodUpdate
	//TODO OnPodDelete
	//Log ?
	// status?
}

type ITaskService interface {
	ITaskStorage
	ITaskCallback
}
