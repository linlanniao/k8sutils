package kbatch

import (
	"context"
	"sync"

	"github.com/linlanniao/k8sutils"
	"github.com/linlanniao/k8sutils/kbatch/template"
	"github.com/linlanniao/k8sutils/validate"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ManagerLabelKeyDefault   = "batch.k8sutils.ppops.cn/manager"
	ManagerLabelValueDefault = "v1"

	TaskLabelAddKey = "batch.k8sutils.ppops.cn/task"

	ManagerConfigMapScriptName = "script"
)

var singleMgr *manager

type manager struct {
	cli         *k8sutils.Clientset
	trackingMap *sync.Map
	once        sync.Once
}

func Manager() *manager {
	singleMgr.once.Do(func() {
		singleMgr = &manager{
			cli:         k8sutils.GetClientset(),
			trackingMap: &sync.Map{},
		}
	})

	return singleMgr
}

func (m *manager) LabelDefault() map[string]string {
	return map[string]string{ManagerLabelKeyDefault: ManagerLabelValueDefault}
}

func (m *manager) RunTask(ctx context.Context, task *Task) (err error) {
	// try to validate job
	if err = validate.Validate(task); err != nil {
		return err
	}

	// update task status
	now := metav1.Now()
	task.Status.StartTime = &now

	// add to tracking map
	trackingKey := task.Namespace + "/" + task.Name
	m.trackingMap.Store(trackingKey, task)

	// if the creation of pod or configmap fails, delete the task from the tracking map.
	defer func() {
		if err != nil {
			m.trackingMap.Delete(trackingKey)
		}
	}()

	// create label with task information
	newLabels := m.LabelDefault()
	newLabels[TaskLabelAddKey] = task.GetName()

	// create configmapTemplate
	cmTmpl := template.
		NewConfigMapTemplate(task.Name, task.Namespace, ManagerConfigMapScriptName, task.Spec.ScriptContent).
		SetLabels(newLabels)

	// try to validate configmap
	if err := validate.Validate(cmTmpl); err != nil {
		return err
	}

	// create configmap
	_, err = m.cli.CreateConfigMap(ctx, cmTmpl.Namespace(), cmTmpl.ConfigMap())
	if err != nil {
		return err
	}

	// create pod template
	isPrivileged := false
	if task.Spec.Privilege != nil && *task.Spec.Privilege == TaskPrivilegeHostRoot {
		isPrivileged = true
	}

	podTmpl := template.
		NewPodTemplate(task.Name, task.Namespace, isPrivileged, task.Spec.Image).
		SetLabels(newLabels).
		SetScript(cmTmpl.ConfigMap(), ManagerConfigMapScriptName, task.Spec.ScriptType.AsExecutor())

	// try to validate pod
	if err := validate.Validate(podTmpl); err != nil {
		return err
	}

	// create pod
	_, err = m.cli.CreatePod(ctx, podTmpl.Namespace(), podTmpl.Pod())
	if err != nil {
		return err
	}

	return nil
}
