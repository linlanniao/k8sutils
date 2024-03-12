package v2

import (
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Task struct {
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Spec              TaskSpec   `json:"spec"`
	Status            TaskStatus `json:"status,omitempty"`
}

type TaskSpec struct {
	Script ScriptSpec `json:"script"`

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
	Job       *batchv1.Job       `json:"job,omitempty"`       // if nil, the job is not created
	JobStatus *batchv1.JobStatus `json:"jobStatus,omitempty"` // if nil, the job status is not created
}
