package v2

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type TaskRun struct {
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Spec              TaskRunSpec   `json:"spec"`
	Status            TaskRunStatus `json:"status,omitempty"`
}

type TaskRef struct {
	Name      string `json:"name,omitempty"`
	Namespace string `json:"namespace,omitempty"`
}

type TaskRunSpec struct {
	TaskRef *TaskRef `json:"taskRef,omitempty"`

	// This field is optional to allow higher level config management to default or override
	// container images in workload controllers like Deployments and StatefulSets.
	// +optional
	Image string `json:"image"`

	// This field is optional to allow higher level config management to default or override
	// +optional
	Privilege *TaskPrivilege `json:"privilege,omitempty"`

	// This field is optional to parameters to the script.
	// +optional
	Parameters *Parameters `json:"parameters,omitempty"`

	// NodeName is a request to schedule this pod onto a specific node. If it is non-empty,
	// the scheduler simply schedules this pod onto that node, assuming that it fits resource
	// requirements.
	// +optional
	NodeName string `json:"nodeName,omitempty"`

	// If specified, the pod's scheduling constraints'
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`
}

type TaskRunStatus struct {
	Pod       *corev1.Pod       `json:"pod,omitempty"`       // if nil, the pod is not created
	PodStatus *corev1.PodStatus `json:"podStatus,omitempty"` // if nil, the pod status is not created
}
