package k8sutils

import (
	"context"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	applycorev1 "k8s.io/client-go/applyconfigurations/core/v1"
	applyrbacv1 "k8s.io/client-go/applyconfigurations/rbac/v1"
)

func (c *Clientset) ApplyServiceAccount(ctx context.Context, name string, labels map[string]string) error {
	namespace := c.GetNamespace()

	sa := applycorev1.ServiceAccount(name, c.GetNamespace()).
		WithLabels(labels)

	_, err := c.clientset.CoreV1().ServiceAccounts(namespace).Apply(ctx, sa, metav1.ApplyOptions{FieldManager: name})
	if err != nil {
		return err
	}
	return nil
}

func (c *Clientset) ApplyClusterRole(ctx context.Context, name string, rule rbacv1.PolicyRule, labels map[string]string) error {
	r1 := applyrbacv1.PolicyRule().
		WithVerbs(rule.Verbs...).
		WithAPIGroups(rule.APIGroups...).
		WithResources(rule.Resources...)

	cr := applyrbacv1.ClusterRole(name).
		WithRules(r1).
		WithLabels(labels)

	if len(rule.NonResourceURLs) > 0 {
		r2 := applyrbacv1.PolicyRule().
			WithVerbs(rule.Verbs...).
			WithNonResourceURLs(rule.NonResourceURLs...)
		cr.WithRules(r2)
	}

	_, err := c.clientset.RbacV1().ClusterRoles().Apply(ctx, cr, metav1.ApplyOptions{FieldManager: name})
	if err != nil {
		return err
	}
	return nil
}

func (c *Clientset) ApplyClusterRoleBinding(
	ctx context.Context, name, clusterRoleName, serviceAccountName string, labels map[string]string) error {

	namespace := c.GetNamespace()

	cr := applyrbacv1.ClusterRoleBinding(name)

	cr.WithSubjects(applyrbacv1.Subject().
		WithKind("ServiceAccount").
		WithName(serviceAccountName).
		WithNamespace(namespace)).
		WithLabels(labels)

	cr.WithRoleRef(applyrbacv1.RoleRef().
		WithKind("ClusterRole").
		WithName(clusterRoleName).
		WithAPIGroup("rbac.authorization.k8s.io"))

	_, err := c.clientset.RbacV1().ClusterRoleBindings().Apply(ctx, cr, metav1.ApplyOptions{FieldManager: name})
	if err != nil {
		return err
	}
	return nil
}
