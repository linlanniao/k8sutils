package k8sutils

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateConfigMap creates a ConfigMap
func (c *Clientset) CreateConfigMap(
	ctx context.Context, namespace string, configMap *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	return c.clientset.CoreV1().ConfigMaps(namespace).Create(ctx, configMap, metav1.CreateOptions{})
}

// DeleteConfigMap deletes a ConfigMap.
func (c *Clientset) DeleteConfigMap(ctx context.Context, namespace, name string) error {
	policy := metav1.DeletePropagationForeground
	return c.clientset.CoreV1().ConfigMaps(namespace).Delete(ctx, name, metav1.DeleteOptions{
		PropagationPolicy: &policy,
	})
}

// UpdateConfigMap updates a ConfigMap.
func (c *Clientset) UpdateConfigMap(
	ctx context.Context, namespace string, configMap *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	return c.clientset.CoreV1().ConfigMaps(namespace).Update(ctx, configMap, metav1.UpdateOptions{})
}
