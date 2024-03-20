package v2

import (
	"context"
)

type ITaskListParams struct {
	Namespace *string `json:"namespace,optional"`
	IsRunning *bool   `json:"is_running,optional"`
	Limit     *int32  `json:"limit,optional"`
	Offset    *int32  `json:"offset,optional"`
}

type ITaskCRUD interface {
	Get(ctx context.Context, name string) (*Task, error)
	Create(ctx context.Context, task *Task) (*Task, error)
	Update(ctx context.Context, task *Task) (*Task, error)
	Delete(ctx context.Context, name string) error
	List(ctx context.Context, param ITaskListParams) ([]*Task, error)
}

type ITaskCallback interface {
	Name() string
	Namespace() string
	Workers() int

	//OnTaskRanFunc(ctx context.Context, task *Task)

	OnTaskStatusUpdateFunc(ctx context.Context, task *Task)
	OnTaskDoneFunc(ctx context.Context, task *Task)
}

type ITaskService interface {
	ITaskCRUD
	ITaskCallback
}

type ITaskRunListParams struct {
	Namespace *string `json:"namespace,optional"`
	IsRunning *bool   `json:"is_running,optional"`
	Limit     *int32  `json:"limit,optional"`
	Offset    *int32  `json:"offset,optional"`
}

type ITaskRunCRUD interface {
	Get(ctx context.Context, name string) (*TaskRun, error)
	Create(ctx context.Context, taskRun *TaskRun) (*TaskRun, error)
	Update(ctx context.Context, taskRun *TaskRun) (*TaskRun, error)
	Delete(ctx context.Context, name string) error
	List(ctx context.Context, param ITaskRunListParams) ([]*TaskRun, error)
}

type ITaskRunCallback interface {
	Name() string
	Namespace() string
	Workers() int
	OnPodCreatedFunc() (ctx context.Context, taskRun *TaskRun)
	//OnAddedUpdatedFunc() mainController.PodOnAddedUpdatedFunc
	//OnDeletedFunc() mainController.PodOnDeletedFunc
	//TODO OnPodAdd
	//TODO OnPodUpdate
	//TODO OnPodDelete
	//Log ?
	// status?
}

type ITaskRunService interface {
	ITaskRunCRUD
	ITaskRunCallback
}
