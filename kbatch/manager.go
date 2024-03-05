package kbatch

import (
	"context"
	"fmt"
	"sync"

	"github.com/linlanniao/k8sutils"
	"github.com/linlanniao/k8sutils/controller"
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

var (
	singleMgr     *manager
	singleMgrOnce sync.Once
)

type manager struct {
	clientset   *k8sutils.Clientset
	trackingMap *sync.Map
	controller  *controller.Controller
}

func Manager() *manager {
	singleMgrOnce.Do(func() {
		singleMgr = &manager{
			clientset:   k8sutils.GetClientset(),
			trackingMap: &sync.Map{},
		}
	})

	return singleMgr
}

func (m *manager) initController() error {
	// skip init if already inited
	if m.controller != nil {
		return nil
	}

	// init pod handler
	podHander := controller.NewPodHandler() // TODO

	// init controller
	m.controller = controller.NewController()

}

func (m *manager) Clientset() *k8sutils.Clientset {
	return m.clientset
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
	_, err = m.clientset.CreateConfigMap(ctx, cmTmpl.Namespace(), cmTmpl.ConfigMap())
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
	_, err = m.clientset.CreatePod(ctx, podTmpl.Namespace(), podTmpl.Pod())
	if err != nil {
		return err
	}

	return nil
}

func (m *manager) GetTrackingTask(key string) (*Task, error) {
	val, ok := m.trackingMap.Load(key)
	if !ok {
		return nil, fmt.Errorf("key %s not found in tracking map", key)
	}

	task, ok := val.(*Task)
	if !ok {
		return nil, fmt.Errorf("value of key %s is not a Task", key)
	}

	return task, nil
}

func (m *manager) CleanupTask(ctx context.Context, task *Task) (err error) {

	matchLabels := m.LabelDefault()
	matchLabels[TaskLabelAddKey] = task.GetName()

	// query pods
	podLst, err := m.clientset.ListPod(ctx, task.GetNamespace(), matchLabels)
	if err != nil {
		return err
	}
	// delete pod
	for _, pod := range podLst.Items {
		if err := m.clientset.DeletePod(ctx, pod.Namespace, pod.Name); err != nil {
			return err
		}
	}

	// query configmap
	cmLst, err := m.clientset.ListConfigMap(ctx, task.GetNamespace(), matchLabels)
	if err != nil {
		return err
	}

	// delete configmap
	for _, cm := range cmLst.Items {
		if err := m.clientset.DeleteConfigMap(ctx, cm.Namespace, cm.Name); err != nil {
			return err
		}
	}

	return nil

}
