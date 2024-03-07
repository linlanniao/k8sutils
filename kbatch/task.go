package kbatch

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/linlanniao/k8sutils/common"
	"github.com/linlanniao/k8sutils/kbatch/template"
	"github.com/linlanniao/k8sutils/validate"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	taskGenerateNameDefault = "task-"
	taskNamespaceDefault    = corev1.NamespaceDefault
)

type Task struct {
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Spec              TaskSpec   `json:"spec"`
	Status            TaskStatus `json:"status,omitempty"`
}

func (t *Task) Validate() error {
	if t.ObjectMeta.Name == "" {
		return errors.New("name cannot be empty")
	}

	if t.ObjectMeta.Namespace == "" {
		return errors.New("namespace cannot be empty")
	}

	if err := t.Spec.Validate(); err != nil {
		return err
	}

	return nil
}

type TaskSpec struct {
	Image         string         `json:"image"`
	ScriptContent string         `json:"scriptContent"`
	ScriptType    ScriptType     `json:"scriptType"`
	Privilege     *TaskPrivilege `json:"privilege,omitempty"`
	Parameters    *Parameters    `json:"parameters,omitempty"`

	// Specifies the number of retries before marking this task failed.
	// Defaults to 0
	// +optional
	MaxRetries *int32 `json:"retryTimes,omitempty"`

	// The interval between failed retries (seconds)
	// Defaults to 10
	CoolDown *int32 `json:"coolDown,omitempty"`

	ActiveDeadlineSeconds   *int64            `json:"activeDeadlineSeconds,omitempty"`
	TTLSecondsAfterFinished *int32            `json:"ttlSecondsAfterFinished,omitempty"`
	NodeSelector            map[string]string `json:"nodeSelector,omitempty"`
	Affinity                *corev1.Affinity  `json:"affinity,omitempty"`
}

func (ts *TaskSpec) Validate() error {
	if ts.Image == "" {
		return errors.New("image cannot be empty")
	}

	if ts.ScriptContent == "" {
		return errors.New("scriptContent cannot be empty")
	}

	if err := ts.ScriptType.Validate(); err != nil {
		return err
	}

	if ts.Privilege != nil {
		if err := ts.Privilege.Validate(); err != nil {
			return err
		}
	}

	if ts.Parameters != nil {
		if err := ts.Parameters.Validate(); err != nil {
			return err
		}
	}

	if ts.MaxRetries != nil && *ts.MaxRetries < 0 {
		return errors.New("retryTimes cannot be negative")
	}

	if ts.CoolDown != nil && *ts.CoolDown < 0 {
		return errors.New("coolDown cannot be negative")
	}

	// TODO ActiveDeadlineSeconds is not supported in this version
	if ts.ActiveDeadlineSeconds != nil {
		if *ts.ActiveDeadlineSeconds < 0 {
			return errors.New("activeDeadlineSeconds cannot be negative")
		}
		return fmt.Errorf("field: %s, %w", "activeDeadlineSeconds", ErrNotSupported)
	}

	// TODO NodeSelector is not supported in this version
	if len(ts.NodeSelector) != 0 {
		return fmt.Errorf("field: %s, %w", "nodeSelector", ErrNotSupported)
	}

	// TODO Affinity is not supported in this version
	if ts.Affinity != nil {
		return fmt.Errorf("field: %s, %w", "affinity", ErrNotSupported)
	}

	return nil
}

type TaskStatus struct {
	Conditions     []batchv1.JobCondition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
	StartTime      *metav1.Time           `json:"startTime,omitempty"`
	CompletionTime *metav1.Time           `json:"completionTime,omitempty"`
	Active         int32                  `json:"active,omitempty"`
	Succeeded      int32                  `json:"succeeded,omitempty"`
	Failed         int32                  `json:"failed,omitempty"`
	Terminating    *int32                 `json:"terminating,omitempty"`
}

type TaskPrivilege string

func (t TaskPrivilege) Validate() error {
	switch t {
	case TaskPrivilegeHostRoot, TaskPrivilegeClusterRoot:
		return nil
	default:
		return fmt.Errorf("invalid task privilege: %s", t)
	}
}

const (
	TaskPrivilegeHostRoot    TaskPrivilege = "HostRoot"
	TaskPrivilegeClusterRoot TaskPrivilege = "ClusterRoot"
)

func (t TaskPrivilege) String() string {
	return string(t)
}

type ScriptType string

func (s ScriptType) Validate() error {
	switch s {
	case ScriptTypePython, ScriptTypeBash:
		return nil
	default:
		return fmt.Errorf("invalid script type: %s", s)
	}
}

const (
	ScriptTypePython ScriptType = "python"
	ScriptTypeBash   ScriptType = "bash"
)

func (s ScriptType) AsExecutor() template.ScriptExecutor {
	return template.ScriptExecutor(s)
}

type TaskOption func(task *Task)

func WithPrivilege(privilege TaskPrivilege) TaskOption {
	return func(task *Task) {
		task.Spec.Privilege = &privilege
	}
}

func WithParameters(parameters ...Parameter) TaskOption {
	return func(task *Task) {
		l := len(parameters)
		if l == 0 {
			return
		}
		params := make(Parameters, l)
		for i, p := range parameters {
			p := p
			params[i] = &p
		}
		task.Spec.Parameters = &params
	}
}

func WithRetryTimes(retryTimes int32) TaskOption {
	return func(task *Task) {
		task.Spec.MaxRetries = &retryTimes
	}
}

func WithCoolDown(coolDown int32) TaskOption {
	return func(task *Task) {
		task.Spec.CoolDown = &coolDown
	}
}

func WithActiveDeadlineSeconds(activeDeadlineSeconds int64) TaskOption {
	return func(task *Task) {
		task.Spec.ActiveDeadlineSeconds = &activeDeadlineSeconds
	}
}

func WithTTLSecondsAfterFinished(ttl int32) TaskOption {
	return func(task *Task) {
		task.Spec.TTLSecondsAfterFinished = &ttl
	}
}

func WithNodeSelector(nodeSelector map[string]string) TaskOption {
	return func(task *Task) {
		task.Spec.NodeSelector = nodeSelector
	}
}

func WithAffinity(affinity *corev1.Affinity) TaskOption {
	return func(task *Task) {
		task.Spec.Affinity = affinity
	}
}

func NewTask(generateName, namespace, image, scriptContent string, scriptType ScriptType, opts ...TaskOption) (*Task, error) {
	t := new(Task)

	// set default
	if generateName == "" {
		generateName = taskGenerateNameDefault
	}
	if namespace == "" {
		namespace = taskNamespaceDefault
	}

	if !strings.HasSuffix(generateName, "-") {
		generateName = generateName + "-"
	}
	name := common.GenerateName2Name(generateName)

	t.ObjectMeta = metav1.ObjectMeta{
		Name:      name,
		Namespace: namespace,
	}

	t.Spec = TaskSpec{
		Image:         image,
		ScriptContent: scriptContent,
		ScriptType:    scriptType,
	}

	for _, opt := range opts {
		opt(t)
	}

	// try to validate the task
	if err := validate.Validate(t); err != nil {
		return nil, err
	}

	return t, nil
}

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
	//OnAddedUpdatedFunc() mainController.PodOnAddedUpdatedFunc
	//OnDeletedFunc() mainController.PodOnDeletedFunc
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
