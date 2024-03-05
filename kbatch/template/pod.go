package template

import (
	"errors"
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
	podGenerateNameDefault = "runner-"
	podNamespaceDefault    = corev1.NamespaceDefault
)

// ScriptExecutor executor for scripts like python / bash / groovy ...
type ScriptExecutor string

const (
	scriptExecutorBash   ScriptExecutor = "bash"
	scriptExecutorPython ScriptExecutor = "python"
)

func (s ScriptExecutor) String() string {
	return string(s)
}

type PodTemplate struct {
	pod              *corev1.Pod
	name             string
	generateName     string
	namespace        string
	image            string
	isPrivileged     bool
	scriptExecutor   ScriptExecutor
	scriptConfigMap  *corev1.ConfigMap // script content config map
	configMapDataKey string            // key for configmap.data field
	args             []string
}

// initPod initializes the pod if it hasn't been initialized yet.
// It returns the PodTemplate instance.
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
	if len(p.generateName) == 0 {
		p.generateName = podGenerateNameDefault
	}
	p.name = common.GenerateName2Name(p.generateName)

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

// Validate checks the validity of the pod template.
func (p *PodTemplate) Validate() error {
	if p.pod == nil {
		return errors.New("pod is not initialized")
	}

	// Validate pod name and generateName.
	if p.pod.Name == "" ||
		p.generateName == "" ||
		strings.HasPrefix(p.generateName, p.name) ||
		p.name != p.pod.Name {
		return errors.New("pod name or generateName is not valid")
	}

	// Validate pod namespace.
	if p.namespace == "" || p.namespace != p.pod.Namespace {
		return errors.New("namespace is not valid")
	}

	// Validate image.
	if len(p.image) == 0 {
		return errors.New("image is empty")
	}

	// Validate script executor.
	if len(p.scriptExecutor.String()) == 0 {
		return errors.New("script executor is empty")
	}

	// Validate script configmap.
	if len(p.scriptConfigMap.Name) == 0 {
		return errors.New("script configmap is empty")
	}

	// Validate configmap data key.
	if len(p.configMapDataKey) == 0 {
		return errors.New("configmap data key is empty")
	}

	return nil
}

// setScriptExecutor sets the script executor for the PodTemplate and initializes the script executor.
// If the PodTemplate is not privileged, it sets the command for the container based on the script executor.
// If the PodTemplate is privileged, it sets the command for the container to use nsenter with the script executor.
func (p *PodTemplate) setScriptExecutor(executor ScriptExecutor) *PodTemplate {
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
func (p *PodTemplate) SetScript(configMapRef *corev1.ConfigMap, dataKey string, executor ScriptExecutor) *PodTemplate {
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
	for idx := range p.pod.Spec.Containers {
		if p.pod.Spec.Containers[idx].Name == podContainerNormalName ||
			p.pod.Spec.Containers[idx].Name == podContainerNsenterName {
			runnerContainer = &p.pod.Spec.Containers[idx]
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

// NewPodTemplate creates a new PodTemplate instance.
//
// Args:
// generateName: the generate name of the pod.
// namespace: the namespace of the pod.
// isPrivileged: whether the pod is privileged.
// image: the image of the pod.
//
// Returns:
// a new PodTemplate instance.
func NewPodTemplate(generateName string, namespace string, isPrivileged bool, image string) *PodTemplate {
	if !strings.HasSuffix(generateName, "-") {
		generateName = generateName + "-"
	}

	t := &PodTemplate{generateName: generateName, namespace: namespace, isPrivileged: isPrivileged, image: image}

	// initializes the pod if it hasn't been initialized yet.
	t.initPod()

	return t
}

// SetGlobalConfigSecretName sets the name of the global configuration secret that will be mounted into the container as environment variables.
// The secret must contain key-value pairs of strings, where the keys are the names of the environment variables and the values are the values of the environment variables.
// If the secret does not exist or is empty, the environment variables will not be set.
// If the secret exists but some keys are missing or have empty values, the environment variables will be set with the default values.
// If the secret exists and all keys have non-empty values, the environment variables will be set with the values from the secret.
// If the secret exists and all keys have non-empty values, and some keys have empty values, the environment variables will be set with the values from the secret, except for the keys with empty values, which will be set with the default values.
// This function modifies the PodTemplate object.
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

// SetAnnotations sets the annotations of the pod.
// This function modifies the PodTemplate object.
func (p *PodTemplate) SetAnnotations(annotations map[string]string) *PodTemplate {
	p.initPod()

	p.pod.SetAnnotations(annotations)
	return p
}

// SetLabels sets the labels of the pod.
// This function modifies the PodTemplate object.
func (p *PodTemplate) SetLabels(labels map[string]string) *PodTemplate {
	p.initPod()

	p.pod.SetLabels(labels)
	return p
}

// SetLabel sets the label of the pod.
// This function modifies the PodTemplate object.
func (p *PodTemplate) SetLabel(key string, value string) *PodTemplate {
	p.initPod()

	if p.pod.Labels == nil {
		p.pod.Labels = map[string]string{}
	}
	p.pod.Labels[key] = value
	return p
}

// SetServiceAccountName sets the service account name of the pod.
// This function modifies the PodTemplate object.
func (p *PodTemplate) SetServiceAccountName(saName string) *PodTemplate {
	p.initPod()

	p.pod.Spec.ServiceAccountName = saName
	return p
}

// SetName sets the name of the pod.
// This function modifies the PodTemplate object.
func (p *PodTemplate) SetName(name string) *PodTemplate {
	p.initPod()

	p.name = name
	p.pod.SetName(name)

	// clean namePrefix
	p.generateName = ""

	return p
}

// SetGenerateNameReGenerate sets the generateName and regenerates the name of the pod.
// This function modifies the PodTemplate object.
func (p *PodTemplate) SetGenerateNameReGenerate(generateName string) *PodTemplate {
	p.initPod()

	if !strings.HasSuffix(generateName, "-") {
		generateName = generateName + "-"
	}

	p.generateName = generateName
	p.name = common.GenerateName2Name(p.generateName)
	p.pod.SetName(p.name)
	return p
}

// SetAffinity sets the affinity of the pod.
// This function modifies the PodTemplate object.
func (p *PodTemplate) SetAffinity(affinity *corev1.Affinity) *PodTemplate {
	p.initPod()

	p.pod.Spec.Affinity = affinity
	return p
}

// AddEnv adds an environment variable to the container.
//
// Args:
// name: the name of the environment variable.
// value: the value of the environment variable.
//
// Returns:
// the PodTemplate instance.
func (p *PodTemplate) AddEnv(name, value string) *PodTemplate {
	p.initPod()

	name = strings.TrimSpace(name)
	value = strings.TrimSpace(value)

	// name and value can not be empty
	if name == "" || value == "" {
		return p
	}

	// add to container env
	p.pod.Spec.Containers[0].Env = append(p.pod.Spec.Containers[0].Env, corev1.EnvVar{
		Name:  name,
		Value: value,
	})

	return p
}

// Namespace returns the namespace of the PodTemplate.
func (p *PodTemplate) Namespace() string {
	return p.namespace
}

// Name returns the name of the PodTemplate.
func (p *PodTemplate) Name() string {
	return p.name
}
