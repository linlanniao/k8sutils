package k8sutils

import (
	"context"
	"fmt"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
)

// CreateJob creates a Job
func (c *Clientset) CreateJob(ctx context.Context, namespace string, job *batchv1.Job) (*batchv1.Job, error) {
	return c.clientset.BatchV1().Jobs(namespace).Create(ctx, job, metav1.CreateOptions{})
}

// GetJobs gets Jobs
func (c *Clientset) GetJobs(ctx context.Context, namespace string, selectedLabels map[string]string) (*batchv1.JobList, error) {
	return c.clientset.BatchV1().Jobs(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labels.Set(selectedLabels).AsSelector().String(),
	})
}

// GetJob gets a Job
func (c *Clientset) GetJob(ctx context.Context, namespace, jobName string) (*batchv1.Job, error) {
	return c.clientset.BatchV1().Jobs(namespace).Get(ctx, jobName, metav1.GetOptions{})
}

// GetPodsFromJob gets Pods from a Job
func (c *Clientset) GetPodsFromJob(ctx context.Context, namespace, jobName string) (*corev1.PodList, error) {
	pods, err := c.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labels.Set{"job-name": jobName}.AsSelector().String(),
	})
	if err != nil {
		return nil, err
	}
	if len(pods.Items) == 0 {
		return nil, fmt.Errorf("no pods found for namespace: %s , job %s", namespace, jobName)
	}
	return pods, nil
}

// WaitForJobActiveOrDone waits for a Job to be active or done
func (c *Clientset) WaitForJobActiveOrDone(ctx context.Context, namespace, jobName string) (status batchv1.JobStatus, err error) {
	err = wait.PollUntilContextTimeout(ctx, time.Second*3, time.Minute*5, false, func(ctx2 context.Context) (bool, error) {
		job, err := c.clientset.BatchV1().Jobs(namespace).Get(ctx2, jobName, metav1.GetOptions{})
		status = job.Status
		if err != nil {
			return false, err
		}
		if job.Status.Active > 0 {
			return true, nil
		}
		if job.Status.Succeeded > 0 {
			return true, nil
		}
		if job.Status.Failed > 0 {
			return true, nil
		}
		return false, nil
	})
	return status, err
}

// ListJob lists Jobs
func (c *Clientset) ListJob(ctx context.Context, namespace string) (*batchv1.JobList, error) {
	return c.clientset.BatchV1().Jobs(namespace).List(ctx, metav1.ListOptions{})
}

// DeleteJob deletes a Job
func (c *Clientset) DeleteJob(ctx context.Context, namespace, jobName string) error {
	policy := metav1.DeletePropagationForeground
	err := c.clientset.BatchV1().Jobs(namespace).Delete(ctx, jobName, metav1.DeleteOptions{
		PropagationPolicy: &policy,
	})
	if err != nil {
		return err
	}

	// Wait for the Job deletion to complete
	return wait.PollUntilContextTimeout(ctx, time.Second*2, time.Minute*2, true, func(ctx2 context.Context) (bool, error) {
		_, err := c.clientset.BatchV1().Jobs(namespace).Get(ctx2, jobName, metav1.GetOptions{})
		if err != nil && k8serrors.IsNotFound(err) {
			// Job is deleted
			return true, nil
		}
		if err != nil {
			return false, err
		}
		// Job still exists, continue waiting
		return false, nil
	})
}

// DeleteJobWithLabels deletes Jobs with labels
func (c *Clientset) DeleteJobWithLabels(ctx context.Context, namespace string, selectedLabels map[string]string) error {
	policy := metav1.DeletePropagationForeground
	delOpt := metav1.DeleteOptions{PropagationPolicy: &policy}
	lstOpt := metav1.ListOptions{LabelSelector: labels.Set(selectedLabels).AsSelector().String()}

	return c.clientset.BatchV1().Jobs(namespace).DeleteCollection(ctx, delOpt, lstOpt)
}
