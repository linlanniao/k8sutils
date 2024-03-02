package batch

import (
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Task struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Spec              TaskSpec   `json:"spec"`
	Status            TaskStatus `json:"status,omitempty"`
}

type TaskSpec struct {
	Image                   string            `json:"image"`
	ScriptContent           string            `json:"scriptContent"`
	ScriptType              ScriptType        `json:"scriptType"`
	Privilege               *TaskPrivilege    `json:"privilege,omitempty"`
	Parameters              *Parameters       `json:"parameters,omitempty"`
	RetryTimes              *int32            `json:"retryTimes,omitempty"`
	CoolDown                *int32            `json:"coolDown,omitempty"`
	BackoffLimit            *int32            `json:"backoffLimit,omitempty"`
	ActiveDeadlineSeconds   *int64            `json:"activeDeadlineSeconds,omitempty"`
	TTLSecondsAfterFinished *int32            `json:"ttlSecondsAfterFinished,omitempty"`
	NodeSelector            map[string]string `json:"nodeSelector,omitempty"`
	Affinity                *corev1.Affinity  `json:"affinity,omitempty"`
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

func (t *Task) CreatePod() {

}