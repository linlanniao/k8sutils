package v2

import (
	"errors"
	"fmt"

	"github.com/linlanniao/k8sutils/validate"
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

func Pod2TaskRun(pod *corev1.Pod) (*TaskRun, error) {
	if pod == nil {
		return nil, errors.New("pod is nil")
	}

	meta := pod.ObjectMeta.DeepCopy()
	taskRun := &TaskRun{
		ObjectMeta: *meta,
		Spec:       TaskRunSpec{},
		Status: TaskRunStatus{
			Pod:       pod,
			Phase:     TaskRunPhase(pod.Status.Phase),
			Message:   pod.Status.Message,
			Reason:    pod.Status.Reason,
			HostIP:    pod.Status.HostIP,
			PodIP:     pod.Status.PodIP,
			StartTime: pod.Status.StartTime.DeepCopy(),
		},
	}

	// spec.taskRef
	if taskName, ok := pod.Labels[TaskNameLabelKey]; ok {
		taskRun.Spec.TaskRef = &TaskRef{
			Name:      taskName,
			Namespace: pod.GetNamespace(),
		}
	}

	podSpec := pod.Spec
	if len(podSpec.Containers) == 0 {
		return nil, errors.New("pod has no containers")
	}
	c0 := podSpec.Containers[0]

	// spec.image
	taskRun.Spec.Image = c0.Image

	// spec.privilege
	if podSpec.HostNetwork && podSpec.HostPID {
		x := TaskPrivilegeHostRoot
		taskRun.Spec.Privilege = &x
	} else if podSpec.ServiceAccountName == K8sManagerSa {
		x := TaskPrivilegeClusterRoot
		taskRun.Spec.Privilege = &x
	}

	// spec.parameters
	if ps, err := Args2Parameters(c0.Args); err == nil {
		taskRun.Spec.Parameters = &ps
	}

	// spec.nodeName
	if podSpec.NodeName != "" {
		taskRun.Spec.NodeName = podSpec.NodeName
	}

	// spec.affinity
	if podSpec.Affinity != nil {
		taskRun.Spec.Affinity = podSpec.Affinity.DeepCopy()
	}

	// status
	podStatus := pod.Status
	taskRun.Status.Phase = TaskRunPhase(podStatus.Phase)
	if err := validate.Validate(taskRun.Status.Phase); err != nil {
		return nil, err
	}
	taskRun.Status.Message = podStatus.Message
	taskRun.Status.Reason = podStatus.Reason
	taskRun.Status.HostIP = podStatus.HostIP
	taskRun.Status.PodIP = podStatus.PodIP
	taskRun.Status.StartTime = podStatus.StartTime.DeepCopy()

	return taskRun, nil
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

	// RFC 3339 date and time at which the object was acknowledged by the Kubelet.
	// This is before the Kubelet pulled the container image(s) for the pod.
	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty" protobuf:"bytes,7,opt,name=startTime"`

	Result *TaskRunResult `json:"result,omitempty"` // TODO Need to design how to collect result data
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

func (p TaskRunPhase) Validate() error {
	switch p {
	case TaskRunPending, TaskRunRunning, TaskRunSucceeded, TaskRunFailed, TaskRunUnknown:
		return nil
	default:
		return fmt.Errorf("%s is not a valid TaskRunPhase", p)
	}
}

type TaskRunResult struct{}

type LogLine struct {
	Timestamp metav1.Time `json:"timestamp"`
	Message   string      `json:"message"`
}

type LogLines []*LogLine
