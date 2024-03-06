package kbatch

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/linlanniao/k8sutils"
	"github.com/linlanniao/k8sutils/controller"
	"github.com/linlanniao/k8sutils/kbatch/template"
	"github.com/linlanniao/k8sutils/validate"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	managerLabelKeyDefault   = "batch.k8sutils.ppops.cn/manager"
	managerLabelValueDefault = "v1"

	taskLabelAddKey = "batch.k8sutils.ppops.cn/task"

	managerConfigMapScriptName = "script"
)

var (
	singleMgr     *manager
	singleMgrOnce sync.Once
)

type manager struct {
	clientset      *k8sutils.Clientset
	taskTacker     *sync.Map
	podTacker      *sync.Map
	mainController *controller.MainController
	taskStorage    ITaskStorage
	taskCallback   ITaskCallback
}

func Manager() *manager {
	singleMgrOnce.Do(func() {
		singleMgr = &manager{
			clientset:  k8sutils.GetClientset(),
			taskTacker: &sync.Map{},
			podTacker:  &sync.Map{},
		}
	})

	return singleMgr
}

func (m *manager) InitController(iTaskSvc ITaskService) error {
	// skip init if already inited
	if m.mainController != nil {
		return errors.New("already inited")
	}

	// init pod handler
	podHandler := controller.NewPodHandler(
		iTaskSvc.Name(),
		iTaskSvc.Namespace(),
		iTaskSvc.Workers(),
		nil,
		nil,
		//iTaskSvc.OnAddedUpdatedFunc(),
		//iTaskSvc.OnDeletedFunc(),
		m.clientset,
	)

	// init mainController
	m.mainController = controller.NewController(controller.WithHandlers(podHandler))

	return nil
}

func (m *manager) SetTaskStorage(iTaskStorage ITaskStorage) {
	m.taskStorage = iTaskStorage
}
func (m *manager) SetTaskCallback(iTaskCallback ITaskCallback) {
	m.taskCallback = iTaskCallback
}

func (m *manager) onPodAdded() (ctx context.Context, pod *corev1.Pod) {
	// TODO
	panic("implement me")

	// add pod to podTacker

	// get taskName from label

	// get task from taskTacker

	// update task Status

	// update task to taskStorage

	// update task in taskTacker

}

func (m *manager) onPodUpdated() (ctx context.Context, pod *corev1.Pod) {
	// TODO
	panic("implement me")

	// get tackingPod from podTacker

	// if tackingPod.status.Phase ==  pod.status.Phase -- > do nothing

	// get taskName from label

	// get task from taskTacker

	// update task Status

	// update task to taskStorage

	// update task in taskTacker
}

func (m *manager) onPodDeleted() (ctx context.Context, pod *corev1.Pod) {
	// TODO
	panic("implement me")

	// get tackingPod from podTacker

	// get task from taskTacker

	// delete tackingPod from podTacker

	// delete tackingTask from taskTacker

}

func (m *manager) onPodAddUpdated() (ctx context.Context, pod *corev1.Pod) {
	// TODO
	panic("implement me")
	// if pod not in podTacker --> onPodAdded() else --> onPodUpdated()

}

func (m *manager) Clientset() *k8sutils.Clientset {
	return m.clientset
}

func (m *manager) LabelDefault() map[string]string {
	return map[string]string{managerLabelKeyDefault: managerLabelValueDefault}
}

// RunTask tries to create the necessary resources for the task.
// It updates the task status and adds the task to the tracking map.
// If the creation of the resources fails, the task is removed from the tracking map.
func (m *manager) RunTask(ctx context.Context, task *Task) (err error) {
	// Try to validate the task.
	if err = validate.Validate(task); err != nil {
		return err
	}

	// Update the task status.
	now := metav1.Now()
	task.Status.StartTime = &now

	// Add the task to the tracking map.
	trackingKey := task.Namespace + "/" + task.Name
	m.taskTacker.Store(trackingKey, task)

	// Defer a function that removes the task from the tracking map if the creation of the resources fails.
	defer func() {
		if err != nil {
			m.taskTacker.Delete(trackingKey)
		}
	}()

	// Create labels with default manager information and the task name.
	newLabels := m.LabelDefault()
	newLabels[taskLabelAddKey] = task.GetName()

	// Create a configmap template with the task information.
	cmTmpl := template.
		NewConfigMapTemplate(task.Name, task.Namespace, managerConfigMapScriptName, task.Spec.ScriptContent).
		SetLabels(newLabels)

	// Try to validate the configmap template.
	if err := validate.Validate(cmTmpl); err != nil {
		return err
	}

	// Create the configmap.
	_, err = m.clientset.CreateConfigMap(ctx, cmTmpl.Namespace(), cmTmpl.ConfigMap())
	if err != nil {
		return err
	}

	// Create a pod template with the task information.
	isPrivileged := false
	if task.Spec.Privilege != nil && *task.Spec.Privilege == TaskPrivilegeHostRoot {
		isPrivileged = true
	}

	podTmpl := template.
		NewPodTemplate(task.Name, task.Namespace, isPrivileged, task.Spec.Image).
		SetLabels(newLabels).
		SetScript(cmTmpl.ConfigMap(), managerConfigMapScriptName, task.Spec.ScriptType.AsExecutor())

	// Try to validate the pod template.
	if err := validate.Validate(podTmpl); err != nil {
		return err
	}

	// Create the pod.
	_, err = m.clientset.CreatePod(ctx, podTmpl.Namespace(), podTmpl.Pod())
	if err != nil {
		return err
	}

	// save task to taskStorage
	task, err = m.taskStorage.Create(ctx, task)

	// update task tacker
	m.taskTacker.Store(trackingKey, task)

	return nil
}

// GetTrackingTask returns the task with the given key from the tracking map.
// If the task does not exist, an error is returned.
func (m *manager) GetTrackingTask(key string) (*Task, error) {
	val, ok := m.taskTacker.Load(key)
	if !ok {
		return nil, fmt.Errorf("key %s not found in tracking map", key)
	}

	task, ok := val.(*Task)
	if !ok {
		return nil, fmt.Errorf("value of key %s is not a Task", key)
	}

	return task, nil
}

// GetTrackingPod returns the pod with the given key from the tracking map.
// If the pod does not exist, an error is returned.
func (m *manager) GetTrackingPod(key string) (*corev1.Pod, error) {
	val, ok := m.podTacker.Load(key)
	if !ok {
		return nil, fmt.Errorf("key %s not found in tracking map", key)
	}

	pod, ok := val.(*corev1.Pod)
	if !ok {
		return nil, fmt.Errorf("value of key %s is not a Pod", key)
	}

	return pod, nil
}

// CleanupTask deletes all the resources created for the task.
func (m *manager) CleanupTask(ctx context.Context, task *Task) (err error) {
	matchLabels := m.LabelDefault()
	matchLabels[taskLabelAddKey] = task.GetName()

	// Query pods.
	podLst, err := m.clientset.ListPod(ctx, task.GetNamespace(), matchLabels)
	if err != nil {
		return err
	}
	// Delete pod.
	for _, pod := range podLst.Items {
		if err := m.clientset.DeletePod(ctx, pod.Namespace, pod.Name); err != nil {
			return err
		}
	}

	// Query configmaps.
	cmLst, err := m.clientset.ListConfigMap(ctx, task.GetNamespace(), matchLabels)
	if err != nil {
		return err
	}

	// Delete configmap.
	for _, cm := range cmLst.Items {
		if err := m.clientset.DeleteConfigMap(ctx, cm.Namespace, cm.Name); err != nil {
			return err
		}
	}

	return nil
}

// Start starts the mainController.
func (m *manager) Start(ctx context.Context) error {
	if m.mainController == nil {
		return errors.New("mainController not inited")
	}

	return m.mainController.Run(ctx)
}
