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

func (c *Clientset) GetLogs(ctx context.Context, namespace, podName string, logsCh chan<- string) error {
	defer close(logsCh)
	opts := &corev1.PodLogOptions{
		Timestamps: true,
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

type LogLine struct {
	Timestamp time.Time
	Line      string
}

type LogLines []LogLine

func (c *Clientset) GetOrTailLogs(ctx context.Context, namespace, podName string, logsCh chan<- LogLine, tail bool) error {
	defer close(logsCh)
	logOptions := &corev1.PodLogOptions{
		Timestamps: true,
		Follow:     tail, // log tail
	}
	req := c.clientset.CoreV1().Pods(namespace).GetLogs(podName, logOptions)
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
		logsCh <- LogLine{
			Timestamp: timestamp,
			Line:      lst[1],
		}
	}
}
