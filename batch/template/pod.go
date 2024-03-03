package template

import (
	"fmt"
	"path/filepath"

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
	//podRunnerContainerImage  = "busybox:1.28.4"

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

type PodTemplate struct {
	pod              *corev1.Pod
	namespace        string
	name             string
	namePrefix       string
	image            string
	isPrivileged     bool
	scriptExecutor   scriptExecutor
	scriptConfigMap  *corev1.ConfigMap // script content config map
	configMapDataKey string            // key for configmap.data field
	args             []string
}

func (p *PodTemplate) initPod() *PodTemplate {
	// skip if pod is already initialized
	if p.pod != nil {
		return p
	}

	p.pod = &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:         p.name,
			Namespace:    p.namespace,
			GenerateName: p.namePrefix,
		},
	}

	// init pod spec
	p.pod.Spec = corev1.PodSpec{
		Volumes:            nil,
		Containers:         nil,
		RestartPolicy:      corev1.RestartPolicyNever,
		NodeSelector:       nil,
		ServiceAccountName: "",
		NodeName:           "",
		HostNetwork:        false,
		Affinity:           nil,
	}
	if p.isPrivileged {
		p.pod.Spec.HostNetwork = true
		p.pod.Spec.HostPID = true
	}

	// init volumes
	if !p.isPrivileged {
		// The pod with ordinary permission needs to mount the command of the host kubectl into the container.
		p.pod.Spec.Volumes = []corev1.Volume{
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
		}
	}

	// init containers
	p.pod.Spec.Containers = make([]corev1.Container, 1)
	if !p.isPrivileged {
		p.pod.Spec.Containers[0] = corev1.Container{
			Name:       podContainerNormalName,
			Image:      p.image,
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
		p.pod.Spec.Containers[0] = corev1.Container{
			Name:    podContainerNsenterName,
			Image:   p.image,
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

	// init script executor
	if !p.isPrivileged {
		switch e := p.scriptExecutor; e {
		case scriptExecutorBash:
			p.pod.Spec.Containers[0].Command = []string{e.String(), "-l"}
		case scriptExecutorPython:
			p.pod.Spec.Containers[0].Command = []string{e.String()}
		}
		p.pod.Spec.Containers[0].Command = []string{p.scriptExecutor.String()}
	} else {
		p.pod.Spec.Containers[0].Command = []string{
			"nsenter", "--target", "1", "--mount", "--uts", "--ipc", "--net", "--pid",
			scriptExecutorBash.String(), "-l",
		}
	}

	return p

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

func (p *PodTemplate) SetScriptMount(scriptConfigMap *corev1.ConfigMap, dataKey string) *PodTemplate {
	p.initPod()

	if p.pod.Spec.Volumes == nil {
		p.pod.Spec.Volumes = []corev1.Volume{}
	}

	// upsert to PodTemplate
	p.scriptConfigMap = scriptConfigMap
	p.configMapDataKey = dataKey

	const volumeName = "script-volume"

	// If the volume already exists, skip it.
	for _, vol := range p.pod.Spec.Volumes {
		if vol.Name == volumeName {
			return p
		}
	}

	// Add the script volume.
	optional := false // configmap must exist

	// Nsenter mount files with pod names to avoid duplicate file names
	scriptName := fmt.Sprintf("%s-%s", "script", p.name)

	scriptVolume := corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: p.scriptConfigMap.GetName(),
				},
				Items: []corev1.KeyToPath{
					{
						Key:  p.configMapDataKey,
						Path: scriptName,
					},
				},
				Optional: &optional,
			},
		},
	}
	p.pod.Spec.Volumes = append(p.pod.Spec.Volumes, scriptVolume)

	// find runner container
	var runnerContainer *corev1.Container
	for _, c := range p.pod.Spec.Containers {
		if c.Name == podContainerNormalName || c.Name == podContainerNsenterName {
			runnerContainer = &c
			break
		}
	}
	if runnerContainer == nil {
		// if runner container not found, return
		return p
	}

	// Add the script volume mount.
	if runnerContainer.VolumeMounts == nil {
		runnerContainer.VolumeMounts = []corev1.VolumeMount{}
	}
	runnerContainer.VolumeMounts = append(runnerContainer.VolumeMounts, corev1.VolumeMount{
		Name:      volumeName,
		ReadOnly:  true,
		MountPath: scriptContentMountPath,
	})

	// setup args
	if runnerContainer.Args == nil {
		p.pod.Spec.Containers[0].Args = []string{}
	}
	// add script path
	scriptFullPath := filepath.Join(scriptContentMountPath, scriptName)
	runnerContainer.Args = append(runnerContainer.Args, scriptFullPath)

	// add extra args
	runnerContainer.Args = append(runnerContainer.Args, p.args...)

	return p
}

// TODO
func NewPodTemplate(
	namespace string,
	name string,
	isPrivileged bool,
	executor string,
	scriptContentConfigMap *corev1.ConfigMap,
	image string,

) *PodTemplate {
	tmpl := new(PodTemplate)
	tmpl.initPod()

	p.SetName(name)
	p.SetNamespace(namespace)

	// todo
	return nil
}
