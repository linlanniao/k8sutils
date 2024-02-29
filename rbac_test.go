package k8sutils

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	rbacv1 "k8s.io/api/rbac/v1"
)

func TestClient_ApplyRbac(t *testing.T) {
	saName := "preplan-k8s-manager-sa"
	clusterRoleName := "preplan-k8s-manager-cluster-role"
	clusterRoleBindingName := "preplan-k8s-manager-cluster-role-binding"
	rule := rbacv1.PolicyRule{
		Verbs:           []string{"*"},
		APIGroups:       []string{"*"},
		Resources:       []string{"*"},
		NonResourceURLs: []string{"*"},
	}
	labels := map[string]string{
		"preplan.sre.rootcloud.com/app":       "preplan",
		"preplan.sre.rootcloud.com/privilege": "host-root",
	}
	ctx := context.Background()
	var err error
	err = GetClientset().ApplyServiceAccount(ctx, saName, labels)
	assert.NoError(t, err)
	err = GetClientset().ApplyClusterRole(ctx, clusterRoleName, rule, labels)
	assert.NoError(t, err)
	err = GetClientset().ApplyClusterRoleBinding(ctx, clusterRoleBindingName, clusterRoleName, saName, labels)
	assert.NoError(t, err)
}

func TestClient_CreateJobWithRbac(t *testing.T) {
	// setup rbac
	saName := "preplan-k8s-manager-sa"
	clusterRoleName := "preplan-k8s-manager-cluster-role"
	clusterRoleBindingName := "preplan-k8s-manager-cluster-role-binding"
	rule := rbacv1.PolicyRule{
		Verbs:           []string{"*"},
		APIGroups:       []string{"*"},
		Resources:       []string{"*"},
		NonResourceURLs: []string{"*"},
	}
	labels := map[string]string{
		"preplan.sre.rootcloud.com/app":       "preplan",
		"preplan.sre.rootcloud.com/privilege": "host-root",
	}
	ctx := context.Background()
	var err error
	err = GetClientset().ApplyServiceAccount(ctx, saName, labels)
	assert.NoError(t, err)
	err = GetClientset().ApplyClusterRole(ctx, clusterRoleName, rule, labels)
	assert.NoError(t, err)
	err = GetClientset().ApplyClusterRoleBinding(ctx, clusterRoleBindingName, clusterRoleName, saName, labels)
	assert.NoError(t, err)

	// create job
	job := TestJobSchema
	job.Name = "rcadm-preplan-123-" + RandLowerStr(4)
	job.Namespace = GetClientset().GetNamespace()

	// set init container
	initCmd := []string{"bash", "-c", "echo initter"}
	job.Spec.Template.Spec.InitContainers[0].Command = initCmd
	job.Spec.Template.Spec.InitContainers[0].Image = "registry.rootcloud.com/devops/preplan-runner-bash:v0.1.1"

	// set runner container
	runnerCmd := []string{"bash", "-c", "kubectl get po -A && sleep 300"}
	job.Spec.Template.Spec.Containers[0].Command = runnerCmd
	job.Spec.Template.Spec.Containers[0].Image = "registry.rootcloud.com/devops/preplan-runner-bash:v0.1.1"

	//set job labels
	labels = map[string]string{
		"preplan.sre.rootcloud.com/task-result-id": "123",
		"preplan.sre.rootcloud.com/task-id":        "not-set",
		"preplan.sre.rootcloud.com/script-id":      "2648a881-8ab8-43bb-98a9-3d07739f26a5",
	}
	job.ObjectMeta.Labels = labels

	// set rbac
	job.Spec.Template.Spec.ServiceAccountName = saName

	// create k8s job
	_, err = GetClientset().CreateJob(ctx, job.Namespace, &job)
	assert.NoError(t, err)
}
