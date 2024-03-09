package template

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/linlanniao/k8sutils/common"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	jobSharedVolumeName      = "shared-volume"
	jobSharedVolumeMountPath = "/workdir"
	jobWorkDir               = "/workdir"
	jobRequestCPU            = "100m"
	jobRequestMemory         = "100Mi"
	jobLimitCPU              = "2000m"
	jobLimitMemory           = "2000Mi"

	JobContainerNormalName  = "runner"
	JobContainerNsenterName = "runner-nsenter"
	scriptContentMountPath  = "/tmp"

	jobEnvFromSecretOptional bool = true

	jobGenerateNameDefault = "runner-"
	jobNamespaceDefault    = corev1.NamespaceDefault
)

var (
	jobDeletionGracePeriodSeconds int64 = 120
	jobParallelism                int32 = 1
	jobCompletions                int32 = 1
	jobBackoffLimit               int32 = 0
	jobPodTTLSecondsAfterFinished int32 = 300
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

type jobTemplate struct {
	job              *batchv1.Job
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

// initJob initializes the job if it hasn't been initialized yet.
// It returns the jobTemplate instance.
func (j *jobTemplate) initJob() *jobTemplate {
	// skip if job is already initialized
	if j.job != nil {
		return j
	}

	j.job = &batchv1.Job{}

	// set default
	if len(j.namespace) == 0 {
		j.namespace = jobNamespaceDefault
	}
	if len(j.generateName) == 0 {
		j.generateName = jobGenerateNameDefault
	}
	j.name = common.GenerateName2Name(j.generateName)

	// init job metadata
	j.job.ObjectMeta = metav1.ObjectMeta{
		Name:      j.name,
		Namespace: j.namespace,
	}
	j.job.SetDeletionGracePeriodSeconds(&jobDeletionGracePeriodSeconds)

	// init job spec
	j.job.Spec = batchv1.JobSpec{
		Parallelism:           &jobParallelism,
		Completions:           &jobCompletions,
		ActiveDeadlineSeconds: nil,
		BackoffLimit:          &jobBackoffLimit,
		Template: corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{
				Volumes:            nil,
				Containers:         nil,
				RestartPolicy:      corev1.RestartPolicyNever,
				NodeSelector:       nil,
				ServiceAccountName: "",
				NodeName:           "",
				HostNetwork:        false,
				Affinity:           nil,
			},
		},
		TTLSecondsAfterFinished: &jobPodTTLSecondsAfterFinished,
	}
	if j.isPrivileged {
		j.job.Spec.Template.Spec.HostNetwork = true
		j.job.Spec.Template.Spec.HostPID = true
	}

	// init volumes
	if !j.isPrivileged {
		// The job with ordinary permission needs to mount the command of the host kubectl into the container.
		j.job.Spec.Template.Spec.Volumes = []corev1.Volume{
			{
				Name: jobSharedVolumeName,
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
	j.job.Spec.Template.Spec.Containers = make([]corev1.Container, 1)
	if !j.isPrivileged {
		j.job.Spec.Template.Spec.Containers[0] = corev1.Container{
			Name:       JobContainerNormalName,
			Image:      j.image,
			Command:    nil,
			Args:       nil,
			WorkingDir: jobWorkDir,
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
					corev1.ResourceCPU:    resource.MustParse(jobLimitCPU),
					corev1.ResourceMemory: resource.MustParse(jobLimitMemory),
				},
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse(jobRequestCPU),
					corev1.ResourceMemory: resource.MustParse(jobRequestMemory),
				},
			},

			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      jobSharedVolumeName,
					MountPath: jobSharedVolumeMountPath,
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
		j.job.Spec.Template.Spec.Containers[0] = corev1.Container{
			Name:    JobContainerNsenterName,
			Image:   j.image,
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
					corev1.ResourceCPU:    resource.MustParse(jobLimitCPU),
					corev1.ResourceMemory: resource.MustParse(jobLimitMemory),
				},
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse(jobRequestCPU),
					corev1.ResourceMemory: resource.MustParse(jobRequestMemory),
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

	return j

}

func (j *jobTemplate) Job() *batchv1.Job {
	return j.job
}

// Validate checks the validity of the job template.
func (j *jobTemplate) Validate() error {
	if j.job == nil {
		return errors.New("job is not initialized")
	}

	// Validate job name and generateName.
	if j.job.Name == "" ||
		j.generateName == "" ||
		strings.HasPrefix(j.generateName, j.name) ||
		j.name != j.job.Name {
		return errors.New("job name or generateName is not valid")
	}

	// Validate job namespace.
	if j.namespace == "" || j.namespace != j.job.Namespace {
		return errors.New("namespace is not valid")
	}

	// Validate image.
	if len(j.image) == 0 {
		return errors.New("image is empty")
	}

	// Validate script executor.
	if len(j.scriptExecutor.String()) == 0 {
		return errors.New("script executor is empty")
	}

	// Validate script configmap.
	if len(j.scriptConfigMap.Name) == 0 {
		return errors.New("script configmap is empty")
	}

	// Validate configmap data key.
	if len(j.configMapDataKey) == 0 {
		return errors.New("configmap data key is empty")
	}

	return nil
}

// setScriptExecutor sets the script executor for the jobTemplate and initializes the script executor.
// If the jobTemplate is not privileged, it sets the command for the container based on the script executor.
// If the jobTemplate is privileged, it sets the command for the container to use nsenter with the script executor.
func (j *jobTemplate) setScriptExecutor(executor ScriptExecutor) *jobTemplate {
	j.initJob()

	j.scriptExecutor = executor

	// init script executor
	var runnerContainer *corev1.Container
	for i, c := range j.job.Spec.Template.Spec.Containers {
		if c.Name == JobContainerNormalName || c.Name == JobContainerNsenterName {
			runnerContainer = &j.job.Spec.Template.Spec.Containers[i]
			break
		}
	}

	if !j.isPrivileged {
		switch e := j.scriptExecutor; e {
		case scriptExecutorBash:
			runnerContainer.Command = []string{e.String(), "-l"}
		case scriptExecutorPython:
			runnerContainer.Command = []string{e.String()}
		}
	} else {
		runnerContainer.Command = []string{
			"nsenter", "--target", "1", "--mount", "--uts", "--ipc", "--net", "--pid",
			scriptExecutorBash.String(), "-l",
		}
	}

	return j
}

// SetScript sets the script configmap and data key
func (j *jobTemplate) SetScript(configMapRef *corev1.ConfigMap, dataKey string, executor ScriptExecutor) *jobTemplate {
	j.initJob()

	j.setScriptExecutor(executor)

	if j.job.Spec.Template.Spec.Volumes == nil {
		j.job.Spec.Template.Spec.Volumes = []corev1.Volume{}
	}

	// upsert to jobTemplate
	j.scriptConfigMap = configMapRef
	j.configMapDataKey = dataKey

	const volumeName = "script-volume"

	// If the volume already exists, skip it.
	for _, vol := range j.job.Spec.Template.Spec.Volumes {
		if vol.Name == volumeName {
			return j
		}
	}

	// Add the script volume.
	optional := false // configmap must exist

	// Nsenter mount files with job names to avoid duplicate file names
	scriptName := fmt.Sprintf("%s-%s", "script", j.name)

	scriptVolume := corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: j.scriptConfigMap.GetName(),
				},
				Items: []corev1.KeyToPath{
					{
						Key:  j.configMapDataKey,
						Path: scriptName,
					},
				},
				Optional: &optional,
			},
		},
	}
	j.job.Spec.Template.Spec.Volumes = append(j.job.Spec.Template.Spec.Volumes, scriptVolume)

	// find runner container
	var runnerContainer *corev1.Container
	for i, c := range j.job.Spec.Template.Spec.Containers {
		if c.Name == JobContainerNormalName || c.Name == JobContainerNsenterName {
			runnerContainer = &j.job.Spec.Template.Spec.Containers[i]
			break
		}
	}
	if runnerContainer == nil {
		// if runner container not found, return
		return j
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
		runnerContainer.Args = []string{}
	}

	// add script path
	scriptFullPath := filepath.Join(scriptContentMountPath, scriptName)
	runnerContainer.Args = append(runnerContainer.Args, scriptFullPath)

	// add extra args
	runnerContainer.Args = append(runnerContainer.Args, j.args...)

	return j
}

// NewPodTemplate creates a new jobTemplate instance.
//
// Args:
// generateName: the generate name of the job.
// namespace: the namespace of the job.
// isPrivileged: whether the job is privileged.
// image: the image of the job.
//
// Returns:
// a new jobTemplate instance.
func NewPodTemplate(generateName string, namespace string, isPrivileged bool, image string) *jobTemplate {
	if !strings.HasSuffix(generateName, "-") {
		generateName = generateName + "-"
	}

	t := &jobTemplate{generateName: generateName, namespace: namespace, isPrivileged: isPrivileged, image: image}

	// initializes the job if it hasn't been initialized yet.
	t.initJob()

	return t
}

// SetGlobalConfigSecretName sets the name of the global configuration secret that will be mounted into the container as environment variables.
// The secret must contain key-value pairs of strings, where the keys are the names of the environment variables and the values are the values of the environment variables.
// If the secret does not exist or is empty, the environment variables will not be set.
// If the secret exists but some keys are missing or have empty values, the environment variables will be set with the default values.
// If the secret exists and all keys have non-empty values, the environment variables will be set with the values from the secret.
// If the secret exists and all keys have non-empty values, and some keys have empty values, the environment variables will be set with the values from the secret, except for the keys with empty values, which will be set with the default values.
func (j *jobTemplate) SetGlobalConfigSecretName(name string) *jobTemplate {
	j.initJob()

	secretOptional := jobEnvFromSecretOptional
	c0 := j.job.Spec.Template.Spec.Containers[0]
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
	return j
}

// SetAnnotations sets the annotations of the job.
func (j *jobTemplate) SetAnnotations(annotations map[string]string) *jobTemplate {
	j.initJob()

	j.job.SetAnnotations(annotations)
	j.job.Spec.Template.SetAnnotations(annotations)
	return j
}

// SetLabels sets the labels of the job.
func (j *jobTemplate) SetLabels(labels map[string]string) *jobTemplate {
	j.initJob()

	j.job.SetLabels(labels)
	j.job.Spec.Template.SetLabels(labels)
	return j
}

// SetLabel sets the label of the job.
func (j *jobTemplate) SetLabel(key string, value string) *jobTemplate {
	j.initJob()

	// job label
	if j.job.Labels == nil {
		j.job.Labels = map[string]string{}
	}
	j.job.Labels[key] = value

	// pod label
	if j.job.Spec.Template.Labels == nil {
		j.job.Spec.Template.Labels = map[string]string{}
	}
	j.job.Spec.Template.Labels[key] = value

	return j
}

// SetServiceAccountName sets the service account name of the job.
func (j *jobTemplate) SetServiceAccountName(saName string) *jobTemplate {
	j.initJob()

	j.job.Spec.Template.Spec.ServiceAccountName = saName
	return j
}

// SetName sets the name of the job.
func (j *jobTemplate) SetName(name string) *jobTemplate {
	j.initJob()

	j.name = name
	j.job.SetName(name)

	// clean namePrefix
	j.generateName = ""

	return j
}

// SetGenerateNameReGenerate sets the generateName and regenerates the name of the job.
func (j *jobTemplate) SetGenerateNameReGenerate(generateName string) *jobTemplate {
	j.initJob()

	if !strings.HasSuffix(generateName, "-") {
		generateName = generateName + "-"
	}

	j.generateName = generateName
	j.name = common.GenerateName2Name(j.generateName)
	j.job.SetName(j.name)
	return j
}

// SetAffinity sets the affinity of the job.
func (j *jobTemplate) SetAffinity(affinity *corev1.Affinity) *jobTemplate {
	j.initJob()

	j.job.Spec.Template.Spec.Affinity = affinity
	return j
}

// SetTTLSecondsAfterFinished sets the TTLSecondsAfterFinished field of the Job.
func (j *jobTemplate) SetTTLSecondsAfterFinished(ttl int32) *jobTemplate {
	j.job.Spec.TTLSecondsAfterFinished = &ttl
	return j
}

// AddEnv adds an environment variable to the container.
//
// Args:
// name: the name of the environment variable.
// value: the value of the environment variable.
//
// Returns:
// the jobTemplate instance.
func (j *jobTemplate) AddEnv(name, value string) *jobTemplate {
	j.initJob()

	name = strings.TrimSpace(name)
	value = strings.TrimSpace(value)

	// name and value can not be empty
	if name == "" || value == "" {
		return j
	}

	// add to container env
	for i, e := range j.job.Spec.Template.Spec.Containers {
		if e.Name == JobContainerNormalName || e.Name == JobContainerNsenterName {
			if j.job.Spec.Template.Spec.Containers[i].Env == nil {
				j.job.Spec.Template.Spec.Containers[i].Env = []corev1.EnvVar{}
			}

			j.job.Spec.Template.Spec.Containers[i].Env = append(
				j.job.Spec.Template.Spec.Containers[i].Env, corev1.EnvVar{
					Name:  name,
					Value: value,
				})
			return j
		}
	}

	return j
}

// Namespace returns the namespace of the jobTemplate.
func (j *jobTemplate) Namespace() string {
	return j.namespace
}

// Name returns the name of the jobTemplate.
func (j *jobTemplate) Name() string {
	return j.name
}
