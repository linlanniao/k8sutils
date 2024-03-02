package template

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	podDefaultNamespace = corev1.NamespaceDefault

	podBaseLabelKey = "batch.k8sutils.ppops.cn/pod"
	podBaseLabelVal = "alpha1v1"

	podSharedVolumeName      = "shared-volume"
	podSharedVolumeMountPath = "/workdir"
	podWorkDir               = "/workdir"
	podRequestCPU            = "100m"
	podRequestMemory         = "100Mi"
	podLimitCPU              = "2000m"
	podLimitMemory           = "2000Mi"
	podRunnerContainerImage  = "busybox:1.28.4"

	podContainerNormalName  = "runner"
	podContainerNsenterName = "runner-nsenter"
	scriptContentMountPath  = "/tmp"
)

type scriptExecutor string

const (
	scriptExecutorBash   scriptExecutor = "bash"
	scriptExecutorPython scriptExecutor = "python"
)

func (s scriptExecutor) String() string {
	return string(s)
}

var (
	podNsenterSecurityContextPrivileged = true
)

func podBaseLabels(extraLabels ...map[string]string) map[string]string {
	l := map[string]string{
		podBaseLabelKey: podBaseLabelVal,
	}
	for _, el := range extraLabels {
		for k, v := range el {
			if k != podBaseLabelKey && v != podBaseLabelVal {
				l[k] = v
			}
		}
	}
	return l
}

func podNsenterContainerCommand() []string {
	return []string{
		"nsenter", "--target", "1", "--mount", "--uts", "--ipc", "--net", "--pid",
		//"sh", "-c", `echo runner`,
	}
}

func newPodContainers(isNsenter bool, executor scriptExecutor) (newerContainers []corev1.Container) {
	newerContainers = make([]corev1.Container, 1)

	// set containers
	if !isNsenter {
		newerContainers[0] = corev1.Container{
			Name:       "runner",
			Image:      podRunnerContainerImage,
			Command:    nil,
			Args:       nil,
			WorkingDir: podWorkDir,
			Env: []corev1.EnvVar{
				{
					Name:  "PATH",
					Value: "/hostroot/usr/local/bin:/hostroot/usr/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
				},
				{
					Name:  "TERM",
					Value: "dumb",
				},
			},
			Resources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse(podLimitCPU),
					corev1.ResourceMemory: resource.MustParse(podLimitMemory),
				},
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse(podRequestCPU),
					corev1.ResourceMemory: resource.MustParse(podRequestMemory),
				},
			},

			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      podSharedVolumeName,
					MountPath: podSharedVolumeMountPath,
				},
				{
					Name:      "kubectl1",
					ReadOnly:  true,
					MountPath: "/hostroot/usr/bin/kubectl",
					SubPath:   "",
				},
				{
					Name:      "kubectl2",
					ReadOnly:  true,
					MountPath: "/hostroot/usr/local/bin/kubectl",
				},
			},
			ImagePullPolicy: corev1.PullIfNotPresent,
		}
	} else {
		podNsenterSecurityContextPrivileged := true
		newerContainers[0] = corev1.Container{
			Name:    podContainerNsenterName,
			Image:   podRunnerContainerImage,
			Command: nil,
			Args:    nil,
			Env: []corev1.EnvVar{
				{
					Name:  "TERM",
					Value: "dumb",
				},
			},
			Resources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse(podLimitCPU),
					corev1.ResourceMemory: resource.MustParse(podLimitMemory),
				},
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse(podRequestCPU),
					corev1.ResourceMemory: resource.MustParse(podRequestMemory),
				},
			},
			ImagePullPolicy: corev1.PullIfNotPresent,
			SecurityContext: &corev1.SecurityContext{
				Privileged: &podNsenterSecurityContextPrivileged,
			},
			Stdin:     true,
			StdinOnce: true,
			TTY:       true,
		}
	}

	// set script executor
	if !isNsenter {
		switch executor {
		case scriptExecutorBash:
			newerContainers[0].Command = []string{executor.String(), "-c"}
		case scriptExecutorPython:
			newerContainers[0].Command = []string{executor.String()}
		}

	} else {
		newerContainers[0].Command = []string{
			"nsenter", "--target", "1", "--mount", "--uts", "--ipc", "--net", "--pid",
			"bash", "-c",
			//"sh", "-c", `echo runner`,
		}
	}

	return
}

// Template for normal's pod
var podTemplateNormal = corev1.Pod{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: podDefaultNamespace,
		Labels:    podBaseLabels(),
	},
	Spec: corev1.PodSpec{
		Volumes: []corev1.Volume{
			{
				Name: podSharedVolumeName,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
			{
				Name: "kubectl1",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{Path: "/usr/bin/kubectl"},
				},
			},
			{
				Name: "kubectl2",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{Path: "/usr/local/bin/kubectl"},
				},
			},
		},
		Containers: []corev1.Container{
			{
				Name:       "runner",
				Image:      podRunnerContainerImage,
				Args:       nil,
				WorkingDir: podWorkDir,
				Env: []corev1.EnvVar{
					{
						Name:  "PATH",
						Value: "/hostroot/usr/local/bin:/hostroot/usr/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
					},
					{
						Name:  "TERM",
						Value: "dumb",
					},
				},
				Resources: corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse(podLimitCPU),
						corev1.ResourceMemory: resource.MustParse(podLimitMemory),
					},
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse(podRequestCPU),
						corev1.ResourceMemory: resource.MustParse(podRequestMemory),
					},
				},

				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      podSharedVolumeName,
						MountPath: podSharedVolumeMountPath,
					},
					{
						Name:      "kubectl1",
						ReadOnly:  true,
						MountPath: "/hostroot/usr/bin/kubectl",
						SubPath:   "",
					},
					{
						Name:      "kubectl2",
						ReadOnly:  true,
						MountPath: "/hostroot/usr/local/bin/kubectl",
					},
				},
				ImagePullPolicy: corev1.PullIfNotPresent,
			},
		},
		RestartPolicy:      corev1.RestartPolicyNever,
		NodeSelector:       nil,
		ServiceAccountName: "",
		NodeName:           "",
		HostNetwork:        false,
		Affinity:           nil,
	},
}

// Template for nsenter's pod
var podTemplateNsenter = corev1.Pod{
	Spec: corev1.PodSpec{
		Containers: []corev1.Container{
			{
				Name:    "runner",
				Image:   podRunnerContainerImage,
				Command: podNsenterContainerCommand(),
				Env: []corev1.EnvVar{
					{
						Name:  "TERM",
						Value: "dumb",
					},
				},
				Resources: corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse(podLimitCPU),
						corev1.ResourceMemory: resource.MustParse(podLimitMemory),
					},
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse(podRequestCPU),
						corev1.ResourceMemory: resource.MustParse(podRequestMemory),
					},
				},
				ImagePullPolicy: corev1.PullIfNotPresent,
				SecurityContext: &corev1.SecurityContext{
					Privileged: &podNsenterSecurityContextPrivileged,
				},
				Stdin:     true,
				StdinOnce: true,
				TTY:       true,
			},
		},

		RestartPolicy:      corev1.RestartPolicyNever,
		NodeSelector:       nil,
		ServiceAccountName: "",
		NodeName:           "",
		HostNetwork:        true,
		HostPID:            true,
		Affinity:           nil,
	},
}

type PodTemplate struct {
	pod            *corev1.Pod
	namespace      string
	name           string
	isPrivileged   bool
	namePrefix     string
	scriptExecutor scriptExecutor
}

func (p *PodTemplate) Pod() *corev1.Pod {
	return p.pod
}

func (p *PodTemplate) SetDefaultNamespaceIfEmpty() *PodTemplate {
	if len(p.namespace) == 0 {
		p.namespace = corev1.NamespaceDefault
	}
	return p
}

func (p *PodTemplate) SetScriptVolume(scriptContentConfigMap *corev1.ConfigMap) *PodTemplate {
	if p.pod.Spec.Volumes == nil {
		p.pod.Spec.Volumes = []corev1.Volume{}
	}
	const volumeName = "script-volume"

	// If the volume already exists, skip it.
	for _, vol := range p.pod.Spec.Volumes {
		if vol.Name == volumeName {
			return p
		}
	}

	// Add the script volume.
	optional := false
	scriptName := fmt.Sprintf("%s-%s", "scriptcontent", p.name)
	scriptVolume := corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "",
				},
				Items: []corev1.KeyToPath{
					{
						Key:  "optional",
						Path: scriptName,
					},
				},
				Optional: &optional,
			},
		},
	}
	p.pod.Spec.Volumes = append(p.pod.Spec.Volumes, scriptVolume)

	// Add the script volume mount.
	if p.pod.Spec.Containers == nil {
		p.pod.Spec.Containers = newPodContainers(p.isPrivileged, p.scriptExecutor)
	}
	if p.pod.Spec.Containers[0].VolumeMounts == nil {
		p.pod.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{}
	}
	p.pod.Spec.Containers[0].VolumeMounts = append(p.pod.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
		Name:      volumeName,
		ReadOnly:  true,
		MountPath: scriptContentMountPath,
	})

	return p
}

func (p *PodTemplate) InitRunnerContainer() {

}

func NewPodTemplate(
	namespace string,
	name string,
	isPrivileged bool,
	executor string,
	scriptContentConfigMap *corev1.ConfigMap,
	image string,

) *PodTemplate {
	var p *corev1.Pod
	if !isPrivileged {
		p = podTemplateNormal.DeepCopy()
	} else {
		p = podTemplateNsenter.DeepCopy()
	}

	p.SetName(name)
	p.SetNamespace(namespace)

	// todo
	return nil
}
