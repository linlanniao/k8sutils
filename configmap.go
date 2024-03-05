package k8sutils

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
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

// ListConfigMap lists ConfigMaps in a given namespace.
// If selectedLabels is not empty, the function returns only ConfigMaps with matching labels.
// The returned ConfigMapList is sorted by creation time, with the most recently created ConfigMap appearing first.
func (c *Clientset) ListConfigMap(
	ctx context.Context,
	namespace string,
	selectedLabels map[string]string) (*corev1.ConfigMapList, error) {

	// Create a label selector based on the selected labels.
	selector := labels.NewSelector()
	if len(selectedLabels) > 0 {
		requirements := make([]labels.Requirement, 0, len(selectedLabels))
		for key, value := range selectedLabels {
			requirement, err := labels.NewRequirement(key, selection.Equals, []string{value})
			if err != nil {
				return nil, fmt.Errorf("create label selector: %w", err)
			}
			requirements = append(requirements, *requirement)
		}
		selector = labels.NewSelector().Add(requirements...)
	}

	// List the ConfigMaps in the given namespace.
	list, err := c.clientset.CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("list ConfigMaps: %w", err)
	}

	return list, nil
}
