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
	OnStatusUpdate(ctx context.Context, task *Task)
	OnFailed(ctx context.Context, task *Task)
	OnSucceed(ctx context.Context, task *Task)
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
	OnStatusUpdate(ctx context.Context, taskRun *TaskRun)
	OnFailed(ctx context.Context, taskRun *TaskRun)
	OnSucceed(ctx context.Context, taskRun *TaskRun)
	OnLog(ctx context.Context, taskRun *TaskRun, logLine *LogLine)
	Result(ctx context.Context, taskRun *TaskRun) (*TaskRunResult, error)
}

type ITaskRunService interface {
	ITaskRunCRUD
	ITaskRunCallback
}
