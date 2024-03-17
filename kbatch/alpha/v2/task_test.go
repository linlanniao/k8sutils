package v2_test

import (
	"context"
	"testing"
	"time"

	"github.com/linlanniao/k8sutils"
	v2 "github.com/linlanniao/k8sutils/kbatch/alpha/v2"
	"github.com/stretchr/testify/assert"
	rbacv1 "k8s.io/api/rbac/v1"
)

func TestTask_GenerateScript(t *testing.T) {
	task := v2.NewTask(
		"test-task",
		"default",
		"alpine:latest",
		`#!/bin/sh
echo "hello TestTask_GenerateScript"`,
		v2.ScriptExecutorSh)
	script, err := task.GenerateScript()
	assert.NoError(t, err)
	assert.NotNil(t, script)

	cm, err := script.ConfigMap()
	assert.NoError(t, err)
	assert.NotNil(t, cm)

	job, err := task.GenerateJob(script)
	assert.NoError(t, err)
	assert.NotNil(t, job)

	t.Log(cm.String())
	t.Log(job.String())

}

func TestTask_Apply(t *testing.T) {
	task := v2.NewTask(
		"test-task",
		"default",
		"alpine:latest",
		`#!/bin/sh
echo "hello TestTask_Apply"`,
		v2.ScriptExecutorSh)
	script, err := task.GenerateScript()
	assert.NoError(t, err)
	assert.NotNil(t, script)

	cm, err := script.ConfigMap()
	assert.NoError(t, err)
	assert.NotNil(t, cm)

	job, err := task.GenerateJob(script)
	assert.NoError(t, err)
	assert.NotNil(t, job)

	cli := k8sutils.GetClientset()
	ctx := context.Background()

	ns := "default"
	_, err = cli.CreateConfigMap(ctx, ns, cm)
	assert.NoError(t, err)
	_, err = cli.CreateJob(ctx, ns, job)
	assert.NoError(t, err)

	// cleanup
	time.Sleep(time.Second * 20)
	_ = cli.DeleteJob(ctx, ns, job.GetName())
	_ = cli.DeleteConfigMap(ctx, ns, cm.GetName())

}

func TestTask_Apply_py(t *testing.T) {
	task := v2.NewTask(
		"test-pytask",
		"default",
		"nyurik/alpine-python3-requests",
		`
import requests
req = requests.get("http://www.baidu.com")
print(req.text)
`,
		v2.ScriptExecutorPython)
	script, err := task.GenerateScript()
	assert.NoError(t, err)
	assert.NotNil(t, script)

	cm, err := script.ConfigMap()
	assert.NoError(t, err)
	assert.NotNil(t, cm)

	job, err := task.GenerateJob(script)
	assert.NoError(t, err)
	assert.NotNil(t, job)

	cli := k8sutils.GetClientset()
	ctx := context.Background()

	ns := "default"
	_, err = cli.CreateConfigMap(ctx, ns, cm)
	assert.NoError(t, err)
	_, err = cli.CreateJob(ctx, ns, job)
	assert.NoError(t, err)

	// cleanup
	time.Sleep(time.Second * 20)
	_ = cli.DeleteJob(ctx, ns, job.GetName())
	_ = cli.DeleteConfigMap(ctx, ns, cm.GetName())

}

func TestTask_Apply_kubectl(t *testing.T) {
	task := v2.NewTask(
		"test-pytask",
		"default",
		"alpine/k8s:1.27.11",
		`#!/bin/bash
kubectl get pod -A
`,
		v2.ScriptExecutorBash,
	)
	privilege := v2.TaskPrivilegeClusterRoot // k8s cluster root privilege
	task.Spec.Privilege = &privilege

	//  TODO apply rbac
	cli := k8sutils.GetClientset()
	ctx := context.Background()

	saName := v2.K8sManagerSa
	clusterRoleName := v2.K8sManagerClusterRole
	clusterRoleBindingName := v2.K8sManagerClusterRoleBinding
	rule := rbacv1.PolicyRule{
		Verbs:           []string{"*"},
		APIGroups:       []string{"*"},
		Resources:       []string{"*"},
		NonResourceURLs: []string{"*"},
	}
	labels := map[string]string{
		"kbatch.k8sutils.ppops.cn/sa":        "k8s-manager",
		"kbatch.k8sutils.ppops.cn/privilege": "host-root",
	}
	var err error
	err = cli.ApplyServiceAccount(ctx, saName, labels)
	assert.NoError(t, err)
	err = cli.ApplyClusterRole(ctx, clusterRoleName, rule, labels)
	assert.NoError(t, err)
	err = cli.ApplyClusterRoleBinding(ctx, clusterRoleBindingName, clusterRoleName, saName, labels)
	assert.NoError(t, err)

	script, err := task.GenerateScript()
	assert.NoError(t, err)
	assert.NotNil(t, script)

	cm, err := script.ConfigMap()
	assert.NoError(t, err)
	assert.NotNil(t, cm)

	job, err := task.GenerateJob(script)
	assert.NoError(t, err)
	assert.NotNil(t, job)

	ns := "default"
	_, err = cli.CreateConfigMap(ctx, ns, cm)
	assert.NoError(t, err)
	_, err = cli.CreateJob(ctx, ns, job)
	assert.NoError(t, err)

	// cleanup
	time.Sleep(time.Second * 20)
	_ = cli.DeleteJob(ctx, ns, job.GetName())
	_ = cli.DeleteConfigMap(ctx, ns, cm.GetName())

}
