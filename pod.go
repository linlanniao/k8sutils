package k8sutils

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
)

func (c *Clientset) CreatePod(ctx context.Context, namespace string, pod *corev1.Pod) (*corev1.Pod, error) {
	return c.clientset.CoreV1().Pods(namespace).Create(ctx, pod, metav1.CreateOptions{})
}

func (c *Clientset) DeletePod(ctx context.Context, namespace, podName string) error {
	policy := metav1.DeletePropagationForeground
	return c.clientset.CoreV1().Pods(namespace).Delete(ctx, podName, metav1.DeleteOptions{
		PropagationPolicy: &policy,
	})
}

func (c *Clientset) GetPod(ctx context.Context, namespace, podName string) (*corev1.Pod, error) {
	return c.clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
}

func (c *Clientset) GetPodStatus(ctx context.Context, namespace, podName string) (corev1.PodStatus, error) {
	pod, err := c.clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return corev1.PodStatus{}, err
	}
	return pod.Status, nil
}

// ListPod lists all pods in the specified namespace that match the specified labels.
// If no labels are specified, all pods in the namespace are returned.
func (c *Clientset) ListPod(ctx context.Context, namespace string, selectedLabels map[string]string) (*corev1.PodList, error) {
	selector := labels.NewSelector()

	if len(selectedLabels) > 0 {
		for key, value := range selectedLabels {
			req, err := labels.NewRequirement(key, selection.Equals, []string{value})
			if err != nil {
				return nil, fmt.Errorf("create label selector: %w", err)
			}
			selector = selector.Add(*req)
		}
	} else {
		selector = labels.Everything()
	}

	return c.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: selector.String(),
	})
}

// TailLogs tails the logs of the specified pod in the specified namespace and sends them to the specified channel.
// The function closes the channel when it's done.
//
// The function returns an error if it fails to tail the logs.
func (c *Clientset) TailLogs(ctx context.Context, namespace, podName string, logsCh chan<- string) error {
	defer close(logsCh)
	opts := &corev1.PodLogOptions{
		Timestamps: true,
		Follow:     true, // log tail
	}
	req := c.clientset.CoreV1().Pods(namespace).GetLogs(podName, opts)
	logStream, err := req.Stream(ctx)
	if err != nil {
		return err
	}
	defer logStream.Close()
	reader := bufio.NewReader(logStream)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		lst := strings.SplitN(line, " ", 2)
		timestamp, _ := time.Parse(time.RFC3339Nano, lst[0])
		logsCh <- timestamp.Format(`2006-01-02 15:04:05 MST`) + " | " + lst[1]
		//logsCh <- timestamp.Format(time.RFC3339) + " " + lst[1]
	}
}

// GetLogs returns the logs of the specified pod in the specified namespace.
//
// The function sends the log lines to the specified channel and closes the channel when it's done.
// If an error occurs, the function returns the error.
func (c *Clientset) GetLogs(ctx context.Context, namespace, podName string, logsCh chan<- string) error {
	// The function starts by closing the logs channel to prevent any further log lines from being sent.
	defer close(logsCh)

	// Create a new PodLogOptions struct and set the Timestamps field to true to include timestamps in the log lines.
	opts := &corev1.PodLogOptions{
		Timestamps: true,
	}

	// Create a new request to retrieve the logs of the specified pod.
	req := c.clientset.CoreV1().Pods(namespace).GetLogs(podName, opts)

	// Start streaming the logs from the Kubernetes API server.
	logStream, err := req.Stream(ctx)
	if err != nil {
		return err
	}
	defer logStream.Close()

	// Create a new buffered reader to read the logs from the streaming response.
	reader := bufio.NewReader(logStream)

	// Loop until the streaming response is closed, and send each log line to the logs channel.
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		lst := strings.SplitN(line, " ", 2)
		timestamp, _ := time.Parse(time.RFC3339Nano, lst[0])
		logsCh <- timestamp.Format(`2006-01-02 15:04:05 MST`) + " | " + lst[1]
		//logsCh <- timestamp.Format(time.RFC3339) + " " + lst[1]
	}
}

type LogLine struct {
	Timestamp time.Time
	Line      string
}

type LogLines []LogLine

// GetOrTailLogs returns the logs of the specified pod in the specified namespace.
//
// The function sends the log lines to the specified channel and closes the channel when it's done.
// If an error occurs, the function returns the error.
func (c *Clientset) GetOrTailLogs(ctx context.Context, namespace, podName string, logsCh chan<- LogLine, tail bool) error {
	// The function starts by closing the logs channel to prevent any further log lines from being sent.
	defer close(logsCh)

	// Create a new PodLogOptions struct and set the Timestamps field to true to include timestamps in the log lines.
	logOptions := &corev1.PodLogOptions{
		Timestamps: true,
		Follow:     tail, // log tail
	}

	// Create a new request to retrieve the logs of the specified pod.
	req := c.clientset.CoreV1().Pods(namespace).GetLogs(podName, logOptions)

	// Start streaming the logs from the Kubernetes API server.
	logStream, err := req.Stream(ctx)
	if err != nil {
		return err
	}
	defer logStream.Close()

	// Create a new buffered reader to read the logs from the streaming response.
	reader := bufio.NewReader(logStream)

	// Loop until the streaming response is closed, and send each log line to the logs channel.
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		lst := strings.SplitN(line, " ", 2)
		timestamp, _ := time.Parse(time.RFC3339Nano, lst[0])
		logsCh <- LogLine{
			Timestamp: timestamp,
			Line:      lst[1],
		}
	}
}
