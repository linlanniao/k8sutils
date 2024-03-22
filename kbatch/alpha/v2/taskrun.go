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
	Pod *corev1.Pod `json:"pod,omitempty"` // if nil, the pod is not created

	// The phase of a Pod is a simple, high-level summary of where the Pod is in its lifecycle.
	// The conditions array, the reason and message fields, and the individual container status
	// arrays contain more detail about the pod's status.
	// There are five possible phase values:
	//
	// Pending: The pod has been accepted by the Kubernetes system, but one or more of the
	// container images has not been created. This includes time before being scheduled as
	// well as time spent downloading images over the network, which could take a while.
	// Running: The pod has been bound to a node, and all of the containers have been created.
	// At least one container is still running, or is in the process of starting or restarting.
	// Succeeded: All containers in the pod have terminated in success, and will not be restarted.
	// Failed: All containers in the pod have terminated, and at least one container has
	// terminated in failure. The container either exited with non-zero status or was terminated
	// by the system.
	// Unknown: For some reason the state of the pod could not be obtained, typically due to an
	// error in communicating with the host of the pod.
	//
	// More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#pod-phase
	// +optional
	Phase TaskRunPhase `json:"phase,omitempty"`

	// A human readable message indicating details about why the pod is in this condition.
	// +optional
	Message string `json:"message,omitempty" protobuf:"bytes,3,opt,name=message"`

	// A brief CamelCase message indicating details about why the pod is in this state.
	// e.g. 'Evicted'
	// +optional
	Reason string `json:"reason,omitempty" protobuf:"bytes,4,opt,name=reason"`

	// hostIP holds the IP address of the host to which the pod is assigned. Empty if the pod has not started yet.
	// A pod can be assigned to a node that has a problem in kubelet which in turns mean that HostIP will
	// not be updated even if there is a node is assigned to pod
	// +optional
	HostIP string `json:"hostIP,omitempty" protobuf:"bytes,5,opt,name=hostIP"`

	// podIP address allocated to the pod. Routable at least within the cluster.
	// Empty if not yet allocated.
	// +optional
	PodIP string `json:"podIP,omitempty" protobuf:"bytes,6,opt,name=podIP"`

	Result *TaskRunResult `json:"result,omitempty"`
}

// TaskRunPhase is a label for the condition of a pod at the current time.
// +enum
type TaskRunPhase string

// These are the valid statuses of pods.
const (
	// TaskRunPending means the pod has been accepted by the system, but one or more of the containers
	// has not been started. This includes time before being bound to a node, as well as time spent
	// pulling images onto the host.
	TaskRunPending TaskRunPhase = "Pending"
	// TaskRunRunning means the pod has been bound to a node and all of the containers have been started.
	// At least one container is still running or is in the process of being restarted.
	TaskRunRunning TaskRunPhase = "Running"
	// TaskRunSucceeded means that all containers in the pod have voluntarily terminated
	// with a container exit code of 0, and the system is not going to restart any of these containers.
	TaskRunSucceeded TaskRunPhase = "Succeeded"
	// TaskRunFailed means that all containers in the pod have terminated, and at least one container has
	// terminated in a failure (exited with a non-zero exit code or was stopped by the system).
	TaskRunFailed TaskRunPhase = "Failed"
	// TaskRunUnknown means that for some reason the state of the pod could not be obtained, typically due
	// to an error in communicating with the host of the pod.
	// Deprecated: It isn't being set since 2015 (74da3b14b0c0f658b3bb8d2def5094686d0e9095)
	TaskRunUnknown TaskRunPhase = "Unknown"
)

type TaskRunResult struct{}

type LogLine struct {
	Timestamp metav1.Time `json:"timestamp"`
	Message   string      `json:"message"`
}

type LogLines []*LogLine
