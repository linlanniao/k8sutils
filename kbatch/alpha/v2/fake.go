package v2

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
)

type FakeTaskService struct{}

func (f FakeTaskService) Get(ctx context.Context, name string) (*Task, error) {
	t := NewTask(
		"fake-task",
		"default",
		"python:3.12.2-alpine3.19",
		`
print("iam fake-task")
`,
		ScriptExecutorPython)
	return t, nil
}

func (f FakeTaskService) Create(ctx context.Context, task *Task) (*Task, error) {
	t := NewTask(
		"fake-task",
		"default",
		"python:3.12.2-alpine3.19",
		`
print("iam fake-task")
`,
		ScriptExecutorPython)
	return t, nil
}

func (f FakeTaskService) Update(ctx context.Context, task *Task) (*Task, error) {
	t := NewTask(
		"fake-task",
		"default",
		"python:3.12.2-alpine3.19",
		`
print("iam fake-task")
`,
		ScriptExecutorPython)
	return t, nil
}

func (f FakeTaskService) Delete(ctx context.Context, name string) error {
	return nil
}

func (f FakeTaskService) List(ctx context.Context, param ITaskListParams) ([]*Task, error) {
	lst := []*Task{
		NewTask(
			"fake-task1",
			"default",
			"python:3.12.2-alpine3.19",
			`
print("iam fake-task1")
`,
			ScriptExecutorPython),
		NewTask(
			"fake-task2",
			"default",
			"python:3.12.2-alpine3.19",
			`
print("iam fake-task2")
`,
			ScriptExecutorPython),
	}
	return lst, nil
}

func (f FakeTaskService) Name() string {
	return "fake-task"
}

func (f FakeTaskService) Namespace() string {
	return corev1.NamespaceDefault
}

func (f FakeTaskService) Workers() int {
	return 1
}

func (f FakeTaskService) OnStatusUpdate(ctx context.Context, task *Task) {
	klog.Infof(
		"OnTaskAddedUpdateFunc, task.status.Active: %v, task.status.Successed: %v, task.status.Failed: %v",
		task.Status.Active,
		task.Status.Succeeded,
		task.Status.Failed,
	)
}

func (f FakeTaskService) OnFailed(ctx context.Context, task *Task) {
	klog.Infof("OnFailed, task.status.condition: %v", task.Status.Condition)

}

func (f FakeTaskService) OnSucceed(ctx context.Context, task *Task) {
	klog.Infof("OnSucceed, task.status.condition: %v", task.Status.Condition)
}

var _ ITaskService = (*FakeTaskService)(nil)

type FakeTaskRunService struct{}

func (f FakeTaskRunService) CleanAllLogs(ctx context.Context, taskRun *TaskRun) {
	//TODO implement me
	panic("implement me")
}

func (f FakeTaskRunService) OnLogs(ctx context.Context, taskRun *TaskRun, logLines LogLines) {
	//TODO implement me
	panic("implement me")
}

func (f FakeTaskRunService) OnSucceed(ctx context.Context, taskRun *TaskRun) {
	//TODO implement me
	panic("implement me")
}

func (f FakeTaskRunService) Result(ctx context.Context, taskRun *TaskRun) (*TaskRunResult, error) {
	//TODO implement me
	panic("implement me")
}

func (f FakeTaskRunService) OnStatusUpdate(ctx context.Context, taskRun *TaskRun) {
	//TODO implement me
	panic("implement me")
}

func (f FakeTaskRunService) OnFailed(ctx context.Context, taskRun *TaskRun) {
	//TODO implement me
	panic("implement me")
}

func (f FakeTaskRunService) Get(ctx context.Context, name string) (*TaskRun, error) {
	//TODO implement me
	panic("implement me")
}

func (f FakeTaskRunService) Create(ctx context.Context, taskRun *TaskRun) (*TaskRun, error) {
	//TODO implement me
	panic("implement me")
}

func (f FakeTaskRunService) Update(ctx context.Context, taskRun *TaskRun) (*TaskRun, error) {
	//TODO implement me
	panic("implement me")
}

func (f FakeTaskRunService) Delete(ctx context.Context, name string) error {
	//TODO implement me
	panic("implement me")
}

func (f FakeTaskRunService) List(ctx context.Context, param ITaskRunListParams) ([]*TaskRun, error) {
	//TODO implement me
	panic("implement me")
}

func (f FakeTaskRunService) Name() string {
	return "fake-task-run"
}

func (f FakeTaskRunService) Namespace() string {
	return corev1.NamespaceDefault
}

func (f FakeTaskRunService) Workers() int {
	return 1
}

func (f FakeTaskRunService) OnPodCreatedFunc() (ctx context.Context, taskRun *TaskRun) {
	//TODO implement me
	panic("implement me")
}

var _ ITaskRunService = (*FakeTaskRunService)(nil)
