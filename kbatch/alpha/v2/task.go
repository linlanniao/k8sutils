package v2

import (
	"errors"
	"fmt"

	"github.com/linlanniao/k8sutils/common"
	"github.com/linlanniao/k8sutils/kbatch/alpha/v2/builders"
	"github.com/linlanniao/k8sutils/validate"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	if err := validate.Validate(t.Spec); err != nil {
		return err
	}

	return nil
}

type TaskSpec struct {
	ScriptSpec ScriptSpec `json:"script"`

	// This field is optional to allow higher level config management to default or override
	// container images in workload controllers like Deployments and StatefulSets.
	// +optional
	Image string `json:"image"`

	// This field is optional to allow higher level config management to default or override
	// +optional
	Privilege *TaskPrivilege `json:"privilege,optional,omitempty"`

	// This field is optional to parameters to the script.
	// +optional
	Parameters *Parameters `json:"parameters,optional,omitempty"`

	// Specifies the duration in seconds relative to the startTime that the task
	// may be continuously active before the system tries to terminate it; value
	// must be positive integer. If a Task is suspended (at creation or through an
	// update), this timer will effectively be stopped and reset when the Task is
	// resumed again.
	// +optional
	ActiveDeadlineSeconds *int64 `json:"activeDeadlineSeconds,omitempty"`

	// Specifies the number of retries before marking this task failed.
	// Defaults to 0, never retrying.
	// +optional
	BackoffLimit *int32 `json:"backoffLimit,omitempty"`

	// ttlSecondsAfterFinished limits the lifetime of a Task that has finished
	// execution (either Complete or Failed). If this field is set,
	// ttlSecondsAfterFinished after the Task finishes, it is eligible to be
	// automatically deleted. When the Task is being deleted, its lifecycle
	// guarantees (e.g. finalizers) will be honored. If this field is unset,
	// the Task won't be automatically deleted. If this field is set to zero,
	// the Task becomes eligible to be deleted immediately after it finishes.
	// +optional
	TTLSecondsAfterFinished *int32 `json:"ttlSecondsAfterFinished,omitempty"`

	// This field is a selector which must be true for the pod to fit on a node.
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// If specified, the pod's scheduling constraints'
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`
}

func (s *TaskSpec) Validate() error {
	if s.Image == "" {
		return errors.New("image cannot be empty")
	}

	if err := validate.Validate(s.ScriptSpec); err != nil {
		return err
	}

	if err := validate.Validate(s.Privilege); err != nil {
		return err
	}

	if err := validate.Validate(s.Parameters); err != nil {
		return err
	}

	if s.ActiveDeadlineSeconds != nil && *s.ActiveDeadlineSeconds <= 0 {
		return errors.New("activeDeadlineSeconds cannot be negative")
	}

	if s.BackoffLimit != nil && *s.BackoffLimit < 0 {
		return errors.New("backoffLimit cannot be negative")
	}

	if s.TTLSecondsAfterFinished != nil && *s.TTLSecondsAfterFinished <= 0 {
		return errors.New("ttlSecondsAfterFinished cannot be negative")
	}

	if err := validate.Validate(s.Affinity); err != nil {
		return err
	}

	return nil
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

type TaskStatus struct {
	Job             *batchv1.Job `json:"job,omitempty"`
	IsJobApplied    bool         `json:"isJobApplied,omitempty"`
	Script          *Script      `json:"script,omitempty"`
	IsScriptApplied bool         `json:"isScriptApplied,omitempty"`
}

func NewTask(
	generateName, namespace, image, scriptContent string,
	scriptExecutor ScriptExecutor,
	opts ...taskOption,
) *Task {
	t := new(Task)
	t.ObjectMeta = metav1.ObjectMeta{
		Namespace: namespace,
	}
	t.ObjectMeta.Name = common.GenerateName2Name(generateName)

	t.Spec = TaskSpec{
		ScriptSpec: ScriptSpec{Content: scriptContent, Executor: scriptExecutor},
		Image:      image,
	}

	for _, opt := range opts {
		opt(t)
	}

	return t
}

type taskOption func(t *Task)

func WithTaskAnnotations(annotations map[string]string) taskOption {
	return func(t *Task) {
		t.ObjectMeta.Annotations = annotations
	}
}

func WithTaskLabels(labels map[string]string) taskOption {
	return func(t *Task) {
		t.ObjectMeta.Labels = labels
	}
}

func WithTaskNodeSelector(nodeSelector map[string]string) taskOption {
	return func(t *Task) {
		t.Spec.NodeSelector = nodeSelector
	}
}

func WithTaskAffinity(affinity *corev1.Affinity) taskOption {
	return func(t *Task) {
		t.Spec.Affinity = affinity
	}
}

func WithTaskActiveDeadlineSeconds(activeDeadlineSeconds int64) taskOption {
	return func(t *Task) {
		t.Spec.ActiveDeadlineSeconds = &activeDeadlineSeconds
	}
}

func WithTaskBackoffLimit(backoffLimit int32) taskOption {
	return func(t *Task) {
		t.Spec.BackoffLimit = &backoffLimit
	}
}

const (
	TaskNameLabelKey = "v2.alpha.kbatch.k8sutils.ppops.cn/task"
)

func (t *Task) GenerateScript() (*Script, error) {
	if err := t.Validate(); err != nil {
		return nil, err
	}

	scriptOpts := make([]scriptOption, 0)

	// annotations
	if annotations := t.ObjectMeta.GetAnnotations(); len(annotations) > 0 {
		scriptOpts = append(scriptOpts, WithScriptAnnotations(annotations))
	}

	// labels
	labels := make(map[string]string)
	if x := t.GetLabels(); len(x) > 0 {
		labels = x
	}
	labels[TaskNameLabelKey] = t.GetName()
	scriptOpts = append(scriptOpts, WithScriptLabels(labels))

	t.Status.Script = NewScript(
		t.GetName(),
		t.GetNamespace(),
		t.Spec.ScriptSpec.Content,
		t.Spec.ScriptSpec.Executor,
		scriptOpts...,
	)

	// generate configmap
	_, err := t.Status.Script.GenerateConfigMap()
	if err != nil {
		return nil, err
	}

	return t.Status.Script, nil
}

func (t *Task) GenerateJob(script *Script) (*batchv1.Job, error) {
	if err := t.Validate(); err != nil {
		return nil, err
	}

	// privilege's setting
	isNsenter := false
	needsServiceAccount := false

	if s := t.Spec.Privilege; s != nil {
		switch *s {
		case TaskPrivilegeHostRoot:
			isNsenter = true
		case TaskPrivilegeClusterRoot:
			needsServiceAccount = true
		}
	}

	var args []string

	// arguments
	if params := t.Spec.Parameters; params != nil && !params.IsEmpty() {
		args = params.Args()
	}
	builder := builders.JobBuilder(t.GetName(), t.GetNamespace(), isNsenter, t.Spec.Image, args)

	// service account
	if needsServiceAccount {
		builder = builder.SetServiceAccount(K8sManagerSaName)
	}

	// script
	if t.Status.Script == nil {
		return nil, errors.New("script is not generated")
	}
	cm, err := t.Status.Script.ConfigMap()
	if err != nil {
		return nil, err
	}
	builder.SetScript(cm, scriptConfigMapDataKey, t.Spec.ScriptSpec.Executor.AsBuildersScriptExecutor())

	// annotations
	if annotations := t.GetAnnotations(); len(annotations) > 0 {
		builder.SetAnnotations(annotations)
	}

	// labels
	labels := make(map[string]string)
	if x := t.GetLabels(); len(x) > 0 {
		labels = x
	}
	labels[TaskNameLabelKey] = t.GetName()
	builder.SetLabels(labels)

	// affinity
	if x := t.Spec.Affinity; x != nil {
		builder.SetAffinity(x)
	}

	// node selector
	if x := t.Spec.NodeSelector; len(x) > 0 {
		builder.SetNodeSelector(x)
	}

	// active deadline seconds
	if t.Spec.ActiveDeadlineSeconds != nil {
		builder.SetActiveDeadlineSeconds(*t.Spec.ActiveDeadlineSeconds)
	}

	// backoff limit
	if t.Spec.BackoffLimit != nil {
		builder.SetBackoffLimit(*t.Spec.BackoffLimit)
	}

	// ttlSecondsAfterFinished
	if t.Spec.TTLSecondsAfterFinished != nil {
		builder.SetTTLSecondsAfterFinished(*t.Spec.TTLSecondsAfterFinished)
	}

	t.Status.Job = builder.Job()
	return t.Status.Job, nil
}
