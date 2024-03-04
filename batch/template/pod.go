package template

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/linlanniao/k8sutils/common"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	podSharedVolumeName      = "shared-volume"
	podSharedVolumeMountPath = "/workdir"
	podWorkDir               = "/workdir"
	podRequestCPU            = "100m"
	podRequestMemory         = "100Mi"
	podLimitCPU              = "2000m"
	podLimitMemory           = "2000Mi"

	podContainerNormalName  = "runner"
	podContainerNsenterName = "runner-nsenter"
	scriptContentMountPath  = "/tmp"

	podEnvFromSecretOptional bool = true

	//podNameDefault       = "runner"
	podNamePrefixDefault = "runner"
	podNamespaceDefault  = corev1.NamespaceDefault
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

	p.pod = &corev1.Pod{}

	// set default
	if len(p.namespace) == 0 {
		p.namespace = podNamespaceDefault
	}
	if len(p.namePrefix) == 0 {
		p.namePrefix = podNamePrefixDefault
	}
	if len(p.name) == 0 {
		p.name = fmt.Sprintf("%s-%s", p.namePrefix, common.RandLowerUpperNumStr(4))
	}

	// init pod metadata
	p.pod.ObjectMeta = metav1.ObjectMeta{
		Name:      p.name,
		Namespace: p.namespace,
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

	return p

}

func (p *PodTemplate) Pod() *corev1.Pod {
	return p.pod
}

func (p *PodTemplate) setScriptExecutor(executor scriptExecutor) *PodTemplate {
	p.initPod()

	p.scriptExecutor = executor

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

// SetScript sets the script configmap and data key
func (p *PodTemplate) SetScript(configMapRef *corev1.ConfigMap, dataKey string, executor scriptExecutor) *PodTemplate {
	p.initPod()

	p.setScriptExecutor(executor)

	if p.pod.Spec.Volumes == nil {
		p.pod.Spec.Volumes = []corev1.Volume{}
	}

	// upsert to PodTemplate
	p.scriptConfigMap = configMapRef
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

// NewPodTemplate Create PodTemplate
func NewPodTemplate(namespace string, namePrefix string, isPrivileged bool, image string) *PodTemplate {
	t := &PodTemplate{namespace: namespace, namePrefix: namePrefix, isPrivileged: isPrivileged, image: image}

	t.initPod()

	return t
}

// SetGlobalConfigSecretName Set the global configuration of SecretName
func (p *PodTemplate) SetGlobalConfigSecretName(name string) *PodTemplate {
	p.initPod()

	secretOptional := podEnvFromSecretOptional
	c0 := p.pod.Spec.Containers[0]
	if c0.EnvFrom == nil {
		c0.EnvFrom = []corev1.EnvFromSource{}
	}
	c0.EnvFrom = append(c0.EnvFrom, corev1.EnvFromSource{
		SecretRef: &corev1.SecretEnvSource{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: name,
			},
			Optional: &secretOptional,
		},
	})
	return p
}

// SetAnnotations Set pod Annotations
func (p *PodTemplate) SetAnnotations(annotations map[string]string) *PodTemplate {
	p.initPod()

	p.pod.SetAnnotations(annotations)
	return p
}

// SetLabels Set pod Labels
func (p *PodTemplate) SetLabels(labels map[string]string) *PodTemplate {
	p.initPod()

	p.pod.SetLabels(labels)
	return p
}

func (p *PodTemplate) SetLabel(key string, value string) *PodTemplate {
	p.initPod()

	if p.pod.Labels == nil {
		p.pod.Labels = map[string]string{}
	}
	p.pod.Labels[key] = value
	return p
}

// SetServiceAccountName Set pod serviceAccountName
func (p *PodTemplate) SetServiceAccountName(saName string) *PodTemplate {
	p.initPod()

	p.pod.Spec.ServiceAccountName = saName
	return p
}

// SetName Set pod name
func (p *PodTemplate) SetName(name string) *PodTemplate {
	p.initPod()

	p.name = name
	p.pod.SetName(name)

	// clean namePrefix
	p.namePrefix = ""

	return p
}

// SetNameAndPrefix Set pod name prefix, and update name
func (p *PodTemplate) SetNameAndPrefix(prefix string) *PodTemplate {
	p.initPod()

	p.namePrefix = prefix
	p.name = prefix + "-" + common.RandStr(5, true, false, true)
	p.pod.SetName(p.name)
	return p
}

// SetAffinity Set pod affinity
func (p *PodTemplate) SetAffinity(affinity *corev1.Affinity) *PodTemplate {
	p.initPod()

	p.pod.Spec.Affinity = affinity
	return p
}

// AddEnv Add env to pod container
func (p *PodTemplate) AddEnv(name, value string) *PodTemplate {
	p.initPod()

	name = strings.TrimSpace(name)
	value = strings.TrimSpace(value)

	// name and value can not be empty
	if name == "" || value == "" {
		return p
	}

	p.pod.Spec.Containers[0].Env = append(p.pod.Spec.Containers[0].Env, corev1.EnvVar{
		Name:  name,
		Value: value,
	})
	return p
}

// Namespace Get pod namespace
func (p *PodTemplate) Namespace() string {
	return p.namespace
}

func (p *PodTemplate) Name() string {
	return p.name
}
