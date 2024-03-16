package v2

import (
	"context"
	"maps"
	"sync"

	"github.com/linlanniao/k8sutils"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/klog/v2"
)

type manager struct {
	clientset *k8sutils.Clientset
}

var (
	singleMgr     *manager
	singleMgrOnce sync.Once
)

// Manager returns the single instance of manager
func Manager() *manager {
	singleMgrOnce.Do(func() {
		singleMgr = &manager{
			clientset: k8sutils.GetClientset(),
		}
	})

	return singleMgr
}

var (
	// _k8sManagerLabels is a map of labels to be applied to all resources created by the manager.
	_k8sManagerLabels = map[string]string{
		"v2.alpha.kbatch.k8sutils.ppops.cn/app":       "k8s-manager",
		"v2.alpha.kbatch.k8sutils.ppops.cn/privilege": "cluster-root",
	}

	// k8sManagerRules k8s role rules for k8s-manager.
	_k8sManagerRules = rbacv1.PolicyRule{
		Verbs:           []string{"*"}, // * Represent all permissions
		APIGroups:       []string{"*"}, // * Represent all API groups
		Resources:       []string{"*"}, // * Represent all resources
		NonResourceURLs: []string{"*"}, // * Represents all non-resource URLs
	}
)

func K8sManagerLabels() map[string]string {
	return maps.Clone(_k8sManagerLabels)
}

func K8sManagerRules() rbacv1.PolicyRule {
	r := _k8sManagerRules.DeepCopy()
	return *r
}

// ApplyK8sManagerClusterRBAC applies the necessary RBAC resources for the k8s-manager.
func (m *manager) ApplyK8sManagerClusterRBAC(ctx context.Context) error {
	// labels is a map of labels to be applied to all resources created by the manager.
	labels := K8sManagerLabels()

	// applyServiceAccount applies the service account.
	if err := m.clientset.ApplyServiceAccount(ctx, K8sManagerSa, labels); err != nil {
		return err
	} else {
		klog.Infof("apply service account success, serviceAccount=%s", K8sManagerSa)
	}

	// applyClusterRole applies the cluster role.
	if err := m.clientset.ApplyClusterRole(ctx, K8sManagerClusterRole, K8sManagerRules(), labels); err != nil {
		return err
	} else {
		klog.Infof("apply cluster role success, clusterRole=%s", K8sManagerClusterRole)
	}

	// applyClusterRoleBinding applies the cluster role binding.
	if err := m.clientset.ApplyClusterRoleBinding(
		ctx,
		K8sManagerClusterRoleBinding,
		K8sManagerClusterRole,
		K8sManagerSa,
		labels,
	); err != nil {
		return err
	} else {
		klog.Infof("apply cluster role binding success, clusterRoleBinding=%s", K8sManagerClusterRoleBinding)
	}

	return nil
}
