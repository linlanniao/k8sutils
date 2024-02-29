package k8sutils

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	JobSharedVolumeName      = "shared-volume"
	JobSharedVolumeMountPath = "/workdir"
	JobWorkDir               = "/workdir"
	JobRequestCPU            = "100m"
	JobRequestMemory         = "100Mi"
	JobLimitCPU              = "2000m"
	JobLimitMemory           = "2000Mi"
	JobRunnerContainerImage  = "busybox:1.28.4"
)

var (
	JobDeletionGracePeriodSeconds = int64(120)
	JobParallelism                = int32(1)
	JobCompletions                = int32(1)
	JobBackoffLimit               = int32(0)
	JobPodTTLSecondsAfterFinished = int32(1800)
	JobRunnerContainerCommand     = []string{"sh", "-c", `echo runner`}
)

var TestJobSchema = batchv1.Job{
	ObjectMeta: metav1.ObjectMeta{
		Name:                       "",
		Namespace:                  "",
		DeletionGracePeriodSeconds: &JobDeletionGracePeriodSeconds,
		Labels:                     nil,
		Annotations:                nil,
	},
	Spec: batchv1.JobSpec{
		Parallelism:           &JobParallelism,
		Completions:           &JobCompletions,
		ActiveDeadlineSeconds: nil,
		BackoffLimit:          &JobBackoffLimit,
		Template: corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{
				Volumes: []corev1.Volume{
					{
						Name: JobSharedVolumeName,
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
					},
					{
						Name: "kubectl1",
						VolumeSource: corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{Path: "/usr/bin/kubectl"},
						},
					}, {
						Name: "kubectl2",
						VolumeSource: corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{Path: "/usr/local/bin/kubectl"},
						},
					}, {
						Name: "kubectl-neat",
						VolumeSource: corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{Path: "/usr/local/bin/kubectl-neat"},
						},
					}, {
						Name: "helm",
						VolumeSource: corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{Path: "/usr/local/bin/helm"},
						},
					}, {
						Name: "kustomize",
						VolumeSource: corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{Path: "/usr/local/bin/kustomize"},
						},
					}, {
						Name: "mc",
						VolumeSource: corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{Path: "/usr/local/bin/mc"},
						},
					}, {
						Name: "otk",
						VolumeSource: corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{Path: "/usr/local/bin/otk"},
						},
					}, {
						Name: "vela",
						VolumeSource: corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{Path: "/usr/local/bin/vela"},
						},
					}, {
						Name: "yq",
						VolumeSource: corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{Path: "/usr/local/bin/yq"},
						},
					},
				},
				Containers: []corev1.Container{
					{
						Name:       "runner",
						Image:      JobRunnerContainerImage,
						Command:    JobRunnerContainerCommand,
						Args:       nil,
						WorkingDir: JobWorkDir,
						Env: []corev1.EnvVar{
							{
								Name:  "PATH",
								Value: "/hostroot/usr/local/bin:/hostroot/usr/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
							},
						},
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse(JobLimitCPU),
								corev1.ResourceMemory: resource.MustParse(JobLimitMemory),
							},
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse(JobRequestCPU),
								corev1.ResourceMemory: resource.MustParse(JobRequestMemory),
							},
						},

						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      JobSharedVolumeName,
								MountPath: JobSharedVolumeMountPath,
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
							{
								Name:      "kubectl-neat",
								ReadOnly:  true,
								MountPath: "/hostroot/usr/local/bin/kubectl-neat",
							},
							{
								Name:      "helm",
								ReadOnly:  true,
								MountPath: "/hostroot/usr/local/bin/helm",
							},
							{
								Name:      "kustomize",
								ReadOnly:  true,
								MountPath: "/hostroot/usr/local/bin/kustomize",
							},
							//{
							//	Name:      "mc",
							//	ReadOnly:  true,
							//	MountPath: "/hostroot/usr/local/bin/mc",
							//},
							{
								Name:      "vela",
								ReadOnly:  true,
								MountPath: "/hostroot/usr/local/bin/vela",
							},
							{
								Name:      "yq",
								ReadOnly:  true,
								MountPath: "/hostroot/usr/local/bin/yq",
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
		},
		TTLSecondsAfterFinished: &JobPodTTLSecondsAfterFinished,
	},
}

var (
	JobRunnerNsenterContainerCommand = []string{
		"nsenter", "--target", "1", "--mount", "--uts", "--ipc", "--net", "--pid",
		"bash", "-c", `echo runner`,
	}
	JobRunnerNsenterSecurityContextPrivileged = true
	JobRunnerNsenterSecurityContext           = &corev1.SecurityContext{
		Privileged: &JobRunnerNsenterSecurityContextPrivileged,
	}
)

var TestJobSchemaNsenter = batchv1.Job{
	ObjectMeta: metav1.ObjectMeta{
		Name:                       "",
		Namespace:                  "",
		DeletionGracePeriodSeconds: &JobDeletionGracePeriodSeconds,
		Labels:                     nil,
		Annotations:                nil,
	},
	Spec: batchv1.JobSpec{
		Parallelism:           &JobParallelism,
		Completions:           &JobCompletions,
		ActiveDeadlineSeconds: nil,
		BackoffLimit:          &JobBackoffLimit,
		Template: corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:    "runner-nsenter",
						Image:   JobRunnerContainerImage,
						Command: JobRunnerNsenterContainerCommand,
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse(JobLimitCPU),
								corev1.ResourceMemory: resource.MustParse(JobLimitMemory),
							},
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse(JobRequestCPU),
								corev1.ResourceMemory: resource.MustParse(JobRequestMemory),
							},
						},
						ImagePullPolicy: corev1.PullIfNotPresent,
						SecurityContext: JobRunnerNsenterSecurityContext,
						Stdin:           true,
						StdinOnce:       true,
						TTY:             true,
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
		},
		TTLSecondsAfterFinished: &JobPodTTLSecondsAfterFinished,
	},
}

func createJob() (namespace, podName, jobName string, err error) {
	if err != nil {
		return "", "", "", fmt.Errorf("GetClientset() error = %v", err)
	}

	ActiveDeadlineSeconds := int64(900)
	jobTmpl := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "t-" + RandLowerStr(5),
			Namespace: "default",
		},
		Spec: batchv1.JobSpec{
			ActiveDeadlineSeconds: &ActiveDeadlineSeconds,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyOnFailure,
					Containers: []corev1.Container{
						{
							Name:  "example-container",
							Image: "nginx",
							Command: []string{
								"/bin/sh",
								"-c",
								`for i in $(seq 1 2000); do echo "$i longlonglonglongstring"; sleep 0.01; done`,
								//`for i in $(seq 1 30); do echo "$i longlonglonglongstring"; sleep 1; done`,
							},
							ImagePullPolicy: corev1.PullIfNotPresent,
						},
					},
				},
			},
		},
	}

	j, err := GetClientset().CreateJob(testCtx, jobTmpl.ObjectMeta.Namespace, jobTmpl)
	if err != nil {
		return "", "", "", fmt.Errorf("CreateJob() error = %v", err)
	}

	ctx := context.Background()
	pods, err := GetClientset().GetPodsFromJob(ctx, jobTmpl.GetNamespace(), jobTmpl.GetName())
	if err != nil {
		return "", "", "", fmt.Errorf("GetPodsFromJob() error = %v", err)
	}
	pod := pods.Items[0]
	return pod.Namespace, pod.GetName(), j.GetName(), nil
}

func TestClient_GetPods(t *testing.T) {
	ActiveDeadlineSeconds := int64(900)
	backoffLimit := int32(3)
	JobPodTTLSecondsAfterFinished = int32(1800)
	jobTmpl := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "t-" + RandLowerStr(5),
			Namespace: "default",
		},
		Spec: batchv1.JobSpec{
			ActiveDeadlineSeconds:   &ActiveDeadlineSeconds,
			BackoffLimit:            &backoffLimit,
			TTLSecondsAfterFinished: &JobPodTTLSecondsAfterFinished,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:  "example-container",
							Image: "nginx:latest",
							Command: []string{
								"/bin/sh",
								"-c",
								`echo hello_world && exit 1`,
								//`for i in $(seq 1 30); do echo "$i longlonglonglongstring"; sleep 1; done`,
							},
							ImagePullPolicy: corev1.PullIfNotPresent,
						},
					},
				},
			},
		},
	}
	_, err := GetClientset().CreateJob(testCtx, jobTmpl.ObjectMeta.Namespace, jobTmpl)
	assert.NoError(t, err)
	time.Sleep(30 * time.Second)
	ctx := context.Background()
	pods, err := GetClientset().GetPodsFromJob(ctx, jobTmpl.GetNamespace(), jobTmpl.GetName())
	assert.NoError(t, err)
	for _, pod := range pods.Items {
		t.Logf("pod: %s", pod.Name)
		t.Logf(pod.Status.StartTime.String())
	}
}

func TestClient_CreateDeleteJob(t *testing.T) {
	kc := GetClientset()
	currentJobs, err := kc.ListJob(testCtx, "default")
	assert.NoError(t, err)
	preJobNum := len(currentJobs.Items)

	jobTmpl := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			//Name:      "j1" + randomStr(5),
			Namespace: "default",
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyOnFailure,
					Containers: []corev1.Container{
						{
							Name:  "example-container",
							Image: "nginx",
							Command: []string{
								"/bin/sh",
								"-c",
								"echo Hello Kubernetes! && sleep 10",
							},
						},
					},
				},
			},
		},
	}
	tests := []struct {
		name string
	}{
		{
			name: "j1" + RandLowerStr(5),
		}, {
			name: "j2" + RandLowerStr(5),
		}, {
			name: "j3" + RandLowerStr(5),
		}, {
			name: "j4" + RandLowerStr(5),
		},
	}

	assert.NoError(t, err)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jobTmpl.ObjectMeta.Name = tt.name
			job, err := kc.CreateJob(testCtx, jobTmpl.ObjectMeta.Namespace, jobTmpl)
			assert.NoError(t, err)
			assert.NotNil(t, job)
		})
	}

	//time.Sleep(1 * time.Second)

	currentJobs, err = kc.ListJob(testCtx, "default")
	assert.NoError(t, err)
	assert.Equal(t, len(tests), len(currentJobs.Items)-preJobNum)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := kc.DeleteJob(testCtx, "default", tt.name)
			assert.NoError(t, err)
		})
	}
}

func TestClient_ListJob(t *testing.T) {
	jobs, err := GetClientset().clientset.BatchV1().Jobs("default").List(context.Background(), metav1.ListOptions{})
	if err != nil {
		t.Errorf("ListJob() error = %v", err)
		return
	}
	for _, job := range jobs.Items {
		t.Logf("job: %s", job.Name)
	}
}

func TestRFC3339(t *testing.T) {
	now := time.Now()
	t.Logf("now: %s", now.Format(time.RFC3339))
}

func TestClient_TailLogs(t *testing.T) {
	podNamespace, podName, _, err := createJob()
	assert.NoError(t, err)
	maxSize := 100
	logCh := make(chan string, maxSize) // 100 line
	ctx := context.Background()
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return GetClientset().TailLogs(ctx, podNamespace, podName, logCh)
	})
	g.Go(func() error {
		Duration := 5 * time.Second
		ticker := time.NewTicker(Duration)
		lines := make([]string, 0, maxSize)
		doFlush := func() error {
			if len(lines) == 0 {
				return nil
			}
			//t.Log(time.Now().Format(time.RFC3339))
			//t.Logf("logs: %s", strings.Join(lines, ""))
			fmt.Printf("%s", strings.Join(lines, ""))
			time.Sleep(1 * time.Second)
			lines = lines[:0]
			return nil
		}
		for {
			select {
			case line, ok := <-logCh:
				if !ok {
					ticker.Stop()
					if err := doFlush(); err != nil {
						return err
					}
					return nil
				}
				if len(lines) >= maxSize {
					if err := doFlush(); err != nil {
						return err
					}
					ticker.Reset(Duration)
				}
				lines = append(lines, line)

			case <-ticker.C:
				if err := doFlush(); err != nil {
					return err
				}
			}
		}
	})
	if err := g.Wait(); err != nil {
		t.Errorf("Wait() error = %v", err)
		return
	}
}

func TestCreateJobWithSchema(t *testing.T) {
	jobTmpl := &TestJobSchema
	jobTmpl.ObjectMeta.Name = "j1" + RandLowerStr(5)
	jobTmpl.ObjectMeta.Namespace = "default"
	job, err := GetClientset().CreateJob(testCtx, jobTmpl.ObjectMeta.Namespace, jobTmpl)
	assert.NoError(t, err)
	assert.NotNil(t, job)
}

func TestClient_CreateJob2(t *testing.T) {
	job := TestJobSchema
	job.Name = "rcadm-preplan-123-" + RandLowerStr(4)
	job.Namespace = GetClientset().GetNamespace()

	var executor string
	//executor = "python"
	executor = "bash"

	//var scriptPath string = fmt.Sprintf("%s/%s.sh", JobWorkDir, "hello_world")

	var uid string
	//uid = "2648a881-8ab8-43bb-98a9-3d07739f26a5" // hello world
	//uid = "7dc0a3ee-16a5-4055-b988-35f88f6d7151" // sleep 180
	//uid = "8fd6abce-f5f1-4c9b-bcfb-5a8bbcdaafcf" // regend-python-test
	//uid = "ebffc4fc-b500-4a6b-aa56-d626b1069e1e" // python-args-test
	uid = "6b18d4c4-ad12-444c-9bc6-eacce3848e22" // bash-args-test

	downloadUrl := fmt.Sprintf("http://sre-dev.rootcloud.info/api/preplan/v1/script/%s/content", uid)

	//// set init container
	//initCmd := []string{"sh", "-c", fmt.Sprintf("wget -O %s %s", scriptPath, downloadUrl)}
	//job.Spec.Template.Spec.InitContainers[0].Command = initCmd
	//job.Spec.Template.Spec.InitContainers[0].Image = "registry.rootcloud.com/devops/preplan-runner-bash:v0.1.0"

	// set runner container
	var args string
	args = "-f fffff -c ccccc -v true"

	runnerCmd := []string{"bash", "-c", fmt.Sprintf("curl -s %s | %s -s %s", downloadUrl, executor, args)} // bash
	//runnerCmd := []string{"bash", "-c", fmt.Sprintf("curl -s %s | %s - %s", downloadUrl, executor, args)}  // python
	job.Spec.Template.Spec.Containers[0].Command = runnerCmd
	job.Spec.Template.Spec.Containers[0].Image = "registry.rootcloud.com/devops/preplan-runner-python:v0.1.1"

	//set job labels
	labels := map[string]string{
		"preplan.sre.rootcloud.com/task-result-id": "123",
		"preplan.sre.rootcloud.com/task-id":        "not-set",
		"preplan.sre.rootcloud.com/script-id":      "2648a881-8ab8-43bb-98a9-3d07739f26a5",
	}
	job.ObjectMeta.Labels = labels

	// create k8s job
	_, err := GetClientset().CreateJob(testCtx, job.Namespace, &job)
	assert.NoError(t, err)
}

func TestClient_GetJobStatus(t *testing.T) {
	j1, _ := GetClientset().clientset.BatchV1().Jobs("default").Get(context.Background(), "j1", metav1.GetOptions{})
	j2, _ := GetClientset().clientset.BatchV1().Jobs("default").Get(context.Background(), "j2", metav1.GetOptions{})
	j3, _ := GetClientset().clientset.BatchV1().Jobs("default").Get(context.Background(), "j3", metav1.GetOptions{})
	t.Logf(j1.Status.String())
	t.Logf(j2.Status.String())
	t.Logf(j3.Status.String())
}

func TestClient_CreateNSenterJob(t *testing.T) {
	jobTmpl := &TestJobSchemaNsenter
	jobTmpl.ObjectMeta.Name = "j1" + RandLowerStr(5)
	jobTmpl.ObjectMeta.Namespace = "default"

	JobInitNsenterContainerCommand := []string{
		"nsenter", "--target", "1", "--mount", "--uts", "--ipc", "--net", "--pid",
		"bash", "-c", `echo "hostname && mkdir -p /tmp/testaaaa/ && touch /tmp/testaaaa/{1..20} && ls -ltrsh /tmp/testaaaa && sleep 10 && find /root" > /tmp/aaa.sh && chmod +x /tmp/aaa.sh`,
	}
	JobRunnerNsenterContainerCommand := []string{
		"nsenter", "--target", "1", "--mount", "--uts", "--ipc", "--net", "--pid",
		"bash", `/tmp/aaa.sh`,
	}

	jobTmpl.Spec.Template.Spec.InitContainers[0].Command = JobInitNsenterContainerCommand
	jobTmpl.Spec.Template.Spec.InitContainers[0].Image = "registry.rootcloud.com/devops/preplan-runner-bash:v0.1.0"
	jobTmpl.Spec.Template.Spec.Containers[0].Command = JobRunnerNsenterContainerCommand
	jobTmpl.Spec.Template.Spec.Containers[0].Image = "registry.rootcloud.com/devops/preplan-runner-bash:v0.1.0"

	job, err := GetClientset().CreateJob(testCtx, jobTmpl.ObjectMeta.Namespace, jobTmpl)
	assert.NoError(t, err)
	assert.NotNil(t, job)
}

func TestClient_CreateJobWithNodeAffinity(t *testing.T) {
	jobTmpl := &TestJobSchemaNsenter
	jobTmpl.ObjectMeta.Name = "j1" + RandLowerStr(5)
	jobTmpl.ObjectMeta.Namespace = "default"

	runCmd := []string{
		"nsenter", "--target", "1", "--mount", "--uts", "--ipc", "--net", "--pid",
		"bash", "-c", `hostname
kubectl version
kubectl get nodes`,
	}

	jobTmpl.Spec.Template.Spec.Containers[0].Command = runCmd
	jobTmpl.Spec.Template.Spec.Containers[0].Image = "registry.rootcloud.com/devops/preplan-runner-bash:v0.1.0"

	affinity := &corev1.Affinity{
		NodeAffinity: &corev1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
				NodeSelectorTerms: []corev1.NodeSelectorTerm{
					{
						MatchExpressions: []corev1.NodeSelectorRequirement{
							{
								Key:      "label1",
								Operator: corev1.NodeSelectorOpIn,
								Values:   []string{"value1"},
							},
						},
					},
				},
			},
		},
	}
	jobTmpl.Spec.Template.Spec.Affinity = affinity

	job, err := GetClientset().CreateJob(testCtx, jobTmpl.ObjectMeta.Namespace, jobTmpl)
	assert.NoError(t, err)
	assert.NotNil(t, job)
}

func TestClient_PreStopHook(t *testing.T) {
	ActiveDeadlineSeconds := int64(900)
	type dataElement struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}
	type T struct {
		Status string         `json:"status"`
		Msg    string         `json:"msg"`
		Data   []*dataElement `json:"data"`
	}

	testData := T{
		Status: "resolved",
		Msg:    time.Now().Format(time.RFC3339),
		Data: []*dataElement{
			{
				Name:  "--key1",
				Value: "value1",
			},
			{
				Name:  "--key2",
				Value: "value2",
			},
		},
	}
	b, _ := json.Marshal(testData)
	s := string(b)
	const (
		JobPreStopContentPrefix   = "PREPLAN_JOB_RESULT_JSON_CONTENT::"
		JobResultUpdateUrlPattern = "%s/api/preplan/v1/job/%d/result"
		JobResultFilePath         = "/tmp/output.json"
	)
	jobResultUploadUrl := fmt.Sprintf(JobResultUpdateUrlPattern, "http://preplan.sre-dev.rootcloud.info", 1103)
	entrypoint := fmt.Sprintf(`echo '%s' > %s && echo ok && sleep 15`, s, JobResultFilePath)
	preStopCmd := fmt.Sprintf(`
test -f %s && echo -n "%s" && jq -r -c . %s
test -f %s && curl -XPUT -H 'Content-Type: application/json' %s -d @%s >/dev/null 2>&1 || true
`, JobResultFilePath, JobPreStopContentPrefix, JobResultFilePath, JobResultFilePath, jobResultUploadUrl, JobResultFilePath)

	jobTmpl := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "t-" + RandLowerStr(5),
			Namespace: "default",
		},
		Spec: batchv1.JobSpec{
			ActiveDeadlineSeconds: &ActiveDeadlineSeconds,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyOnFailure,
					Containers: []corev1.Container{
						{
							Name:  "example-container",
							Image: "registry.rootcloud.com/devops/preplan-runner-python:v0.1.6",
							Command: []string{
								"/bin/sh",
								"-c",
								entrypoint + preStopCmd,
							},
							ImagePullPolicy: corev1.PullIfNotPresent,
						},
					},
				},
			},
		},
	}

	//	preStopCmd := fmt.Sprintf(`test -f %s && echo -n "%s" && jq -r -c . %s
	//test -f %s && curl -XPUT -H 'Content-Type: application/json' %s -d @%s || true
	//`,
	//		JobResultFilePath, JobPreStopContentPrefix, JobResultFilePath, JobResultFilePath,
	//		jobResultUploadUrl, JobResultFilePath)
	//jobTmpl.Spec.Template.Spec.Containers[0].Lifecycle = &corev1.Lifecycle{
	//	PreStop: &corev1.LifecycleHandler{
	//		Exec: &corev1.ExecAction{
	//			Command: []string{"/bin/sh", "-c", preStopCmd},
	//		},
	//	},
	//}
	//preStopCmd := fmt.Sprintf(` curl -XPUT -H 'Content-Type: application/json' %s -d @%s`,
	//	jobResultUploadUrl, JobResultFilePath)
	//jobTmpl.Spec.Template.Spec.Containers[0].Lifecycle = &corev1.Lifecycle{
	//	PreStop: &corev1.LifecycleHandler{
	//		Exec: &corev1.ExecAction{
	//			Command: []string{"/bin/sh", "-c", preStopCmd},
	//		},
	//	},
	//}

	j, err := GetClientset().CreateJob(testCtx, jobTmpl.ObjectMeta.Namespace, jobTmpl)
	assert.NoError(t, err)
	assert.NotNil(t, j)
}
