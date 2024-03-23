package v2

import (
	"encoding/json"
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
	Status            TaskStatus `json:"-"`
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

	// The number of pending and running pods.
	// +optional
	Active int32 `json:"active,omitempty"`

	// The number of pods which reached phase Succeeded.
	// +optional
	Succeeded int32 `json:"succeeded,omitempty"`

	// The number of pods which reached phase Failed.
	// +optional
	Failed int32 `json:"failed,omitempty"`

	// The latest available observations of an object's current state. when Condition is not nil, then the Task is done
	// +optional
	Condition *TaskCondition `json:"condition,omitempty"`
}

type TaskCondition struct {
	// Type of task condition, Complete or Failed.
	Type TaskConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status ConditionStatus `json:"status"`
	// Last time the condition was checked.
	// +optional
	LastProbeTime metav1.Time `json:"lastProbeTime,omitempty"`
	// Last time the condition transit from one status to another.
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// (brief) reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty"`
	// Human readable message indicating details about last transition.
	// +optional
	Message string `json:"message,omitempty"`
}

type TaskConditionType string

const (
	// TaskSuspended means the task has been suspended.
	TaskSuspended TaskConditionType = "Suspended"
	// TaskComplete means the task has completed its execution.
	TaskComplete TaskConditionType = "Complete"
	// TaskFailed means the task has failed its execution.
	TaskFailed TaskConditionType = "Failed"
	// TaskFailureTarget means the task is about to fail its execution.
	TaskFailureTarget TaskConditionType = "FailureTarget"
)

type ConditionStatus string

// These are valid condition statuses. "ConditionTrue" means a resource is in the condition.
// "ConditionFalse" means a resource is not in the condition. "ConditionUnknown" means kubernetes
// can't decide if a resource is in the condition or not. In the future, we could add other
// intermediate conditions, e.g. ConditionDegraded.
const (
	ConditionTrue    ConditionStatus = "True"
	ConditionFalse   ConditionStatus = "False"
	ConditionUnknown ConditionStatus = "Unknown"
)

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
	TaskNameLabelKey = "kbatch.k8sutils.ppops.cn/task"
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

func (t *Task) GenerateJob() (*batchv1.Job, error) {
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
		builder = builder.SetServiceAccount(K8sManagerSa)
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
	t.SetContentToAnnotation()
	if x := t.GetAnnotations(); len(x) > 0 {
		builder.SetAnnotations(x)
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
	//delete(t.Status.Job.Spec.Template.ObjectMeta.Annotations, TaskContentAnnotation) // do not recursion
	return t.Status.Job, nil
}

func (t *Task) SetLabel(key, value string) *Task {
	if t.ObjectMeta.Labels == nil {
		t.ObjectMeta.Labels = make(map[string]string)
	}
	t.ObjectMeta.Labels[key] = value
	return t
}

func (t *Task) SetLabels(labels map[string]string) *Task {
	if t.ObjectMeta.Labels == nil {
		t.ObjectMeta.Labels = make(map[string]string)
	}
	for k, v := range labels {
		t.ObjectMeta.Labels[k] = v
	}
	return t
}

func (t *Task) SetAnnotation(key, value string) *Task {
	if t.ObjectMeta.Annotations == nil {
		t.ObjectMeta.Annotations = make(map[string]string)
	}
	t.ObjectMeta.Annotations[key] = value
	return t
}

func (t *Task) SetAnnotations(annotations map[string]string) *Task {
	if t.ObjectMeta.Annotations == nil {
		t.ObjectMeta.Annotations = make(map[string]string)
	}
	for k, v := range annotations {
		t.ObjectMeta.Annotations[k] = v
	}
	return t
}

const (
	TaskContentAnnotation = "kbatch.k8sutils.ppops.cn/task-content"
)

func (t *Task) SetContentToAnnotation() *Task {
	if len(t.Annotations) > 0 {
		delete(t.Annotations, TaskContentAnnotation)
	}
	b, err := json.Marshal(t)
	if err != nil {
		return t
	}
	content := string(b)
	t.SetAnnotation(TaskContentAnnotation, content)
	return t
}

func Job2Task(job *batchv1.Job) (*Task, error) {
	if job == nil {
		return nil, errors.New("job is nil")
	}
	if job.ObjectMeta.Annotations == nil {
		return nil, errors.New("job annotations is nil")
	}
	if job.ObjectMeta.Annotations[TaskContentAnnotation] == "" {
		return nil, errors.New("task content is empty")
	}
	var task *Task

	// fill metadata && spec
	if err := json.Unmarshal([]byte(job.ObjectMeta.Annotations[TaskContentAnnotation]), &task); err != nil {
		return nil, err
	}

	// fill status
	status := job.Status

	task.Status.Job = job
	task.Status.IsJobApplied = true
	task.Status.Active = status.Active
	task.Status.Succeeded = status.Succeeded
	task.Status.Failed = status.Failed

	if len(status.Conditions) == 0 {
		// conditions is empty means the job is not done
		task.Status.Condition = nil
	} else {
		// job is already done, update status and run callback function
		c0 := status.Conditions[0]
		task.Status.Condition = &TaskCondition{
			Type:               TaskConditionType(c0.Type),
			Status:             ConditionStatus(c0.Status),
			LastProbeTime:      c0.LastProbeTime,
			LastTransitionTime: c0.LastTransitionTime,
			Reason:             c0.Reason,
			Message:            c0.Message,
		}
	}

	return task, nil
}
