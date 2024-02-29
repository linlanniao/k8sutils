package batch

import (
	"errors"
	"fmt"
	"regexp"

	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Task struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
}

type TaskSpec struct {
	Image                   string         `json:"image"`
	ScriptContent           string         `json:"scriptContent"`
	ScriptType              ScriptType     `json:"scriptType"`
	Privilege               *TaskPrivilege `json:"privilege,omitempty"`
	Parameters              *Parameters    `json:"parameters,omitempty"`
	RetryTimes              *int32         `json:"retryTimes,omitempty"`
	CoolDown                *int32         `json:"coolDown,omitempty"`
	BackoffLimit            *int32         `json:"backoffLimit,omitempty"`
	ActiveDeadlineSeconds   *int64         `json:"activeDeadlineSeconds,omitempty"`
	TTLSecondsAfterFinished *int32         `json:"ttlSecondsAfterFinished,omitempty"`
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

type Parameter struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (p Parameter) Validate() error {
	// key must not be empty, and length must be between 1 and 32
	if len(p.Key) == 0 || len(p.Key) >= 32 {
		return errors.New("invalid parameter name, length must be between 1 and 32")
	}
	pattern := `^[-]{1,2}[a-zA-Z0-9_-]+$`
	if res, _ := regexp.MatchString(pattern, p.Key); !res {
		return fmt.Errorf("invalid parameter name, must match %s", pattern)
	}

	// value must not be empty
	if p.Value == "" {
		return errors.New("invalid parameter value, must not be empty")
	}

	return nil
}

type Parameters []*Parameter

func (ps Parameters) Validate() error {
	if len(ps) == 0 {
		return nil
	}
	for _, x := range ps {
		x := x
		if err := x.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (ps Parameters) IsEmpty() bool {
	return len(ps) == 0
}
func (ps Parameters) Len() int {
	return len(ps)
}

func (ps Parameters) ToArgs() string {
	var s string
	for _, p := range ps {
		p := p
		s += p.Key + " " + p.Value + " "
	}
	return s
}

type TaskStatus struct {
	Conditions []batchv1.JobCondition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`

	StartTime *metav1.Time `json:"startTime,omitempty"`

	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	Active int32 `json:"active,omitempty"`

	Succeeded int32 `json:"succeeded,omitempty"`

	Failed int32 `json:"failed,omitempty"`

	Terminating *int32 `json:"terminating,omitempty"`
}
