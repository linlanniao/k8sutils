package v1

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/linlanniao/k8sutils"
	"github.com/linlanniao/k8sutils/controller"
	"github.com/linlanniao/k8sutils/kbatch/alpha/v1/template"
	"github.com/linlanniao/k8sutils/validate"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

const (
	managerLabelKeyDefault   = "batch.k8sutils.ppops.cn/manager"
	managerLabelValueDefault = "alpha-v1"

	taskLabelAddKey = "batch.k8sutils.ppops.cn/task"

	managerConfigMapScriptName = "script"
)

var (
	singleMgr     *manager
	singleMgrOnce sync.Once

	ErrKeyNotFound       = errors.New("key not found")
	ErrValueTypeMismatch = errors.New("value type mismatch")
)

type manager struct {
	clientset      *k8sutils.Clientset
	taskTacker     *sync.Map
	podTacker      *sync.Map
	mainController *controller.MainController
	taskStorage    ITaskStorage
	taskCallback   ITaskCallback
}

// Manager returns the single instance of manager
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

// InitController initializes the controller
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

// SetTaskStorage sets the task storage
func (m *manager) SetTaskStorage(iTaskStorage ITaskStorage) {
	m.taskStorage = iTaskStorage
}

// SetTaskCallback sets the task callback
func (m *manager) SetTaskCallback(iTaskCallback ITaskCallback) {
	m.taskCallback = iTaskCallback
}

// UpdateTaskFromPod updates the task status based on the pod status.
// TODO task MaxRetry is not 0, how to retry ?
//
//	if retrying ?? how to deal with it?
func (m *manager) updateTaskFromPod(pod *corev1.Pod) error {
	// get taskName from label
	taskName, err := getTaskNameFromPod(pod)
	if err != nil {
		return err
	}

	// get task from taskTacker
	taskKey := pod.GetNamespace() + "/" + taskName
	task, err := m.GetTrackingTask(taskKey)
	if err != nil {
		return fmt.Errorf("task %s not found", taskKey)
	}

	// update task status
	switch pod.Status.Phase {
	case corev1.PodPending, corev1.PodRunning:
		task.Status.Active += 1

	case corev1.PodSucceeded:
		task.Status.Succeeded += 1
		if task.Status.Conditions == nil {
			task.Status.Conditions = make([]batchv1.JobCondition, 0)
		}
		task.Status.Conditions = append(task.Status.Conditions, batchv1.JobCondition{
			Type:               batchv1.JobComplete,
			Status:             corev1.ConditionTrue,
			LastProbeTime:      metav1.Now(),
			LastTransitionTime: metav1.Now(),
			Reason:             pod.Status.Reason,
			Message:            pod.Status.Message,
		})

		completionTime := metav1.Now()
		task.Status.CompletionTime = &completionTime

	case corev1.PodFailed:
		task.Status.Failed += 1
		if task.Status.Conditions == nil {
			task.Status.Conditions = make([]batchv1.JobCondition, 0)
		}

		var (
			reason  string
			message string
		)

		// try to get reason and message from container status
		if len(pod.Status.ContainerStatuses) > 0 {
			for _, c := range pod.Status.ContainerStatuses {
				if c.Name == template.PodContainerNormalName || c.Name == template.PodContainerNsenterName {
					if c.State.Terminated != nil {
						reason = c.State.Terminated.Reason
						message = c.State.Terminated.Message
						break
					}
				}
			}
		}
		if len(reason) == 0 {
			reason = pod.Status.Reason
		}
		if len(message) == 0 {
			message = pod.Status.Message
		}

		task.Status.Conditions = append(task.Status.Conditions, batchv1.JobCondition{
			Type:               batchv1.JobComplete,
			Status:             corev1.ConditionFalse,
			LastProbeTime:      metav1.Now(),
			LastTransitionTime: metav1.Now(),
			Reason:             reason,
			Message:            message,
		})
	}

	// update task to taskTacker
	if err = m.storeTackingTask(task); err != nil {
		return err
	}

	// storage task
	if m.taskStorage != nil {
		if _, err = m.taskStorage.Update(context.Background(), task); err != nil {
			return err
		}
	}
	return nil
}

// onPodAdded handles the pod added event
func (m *manager) onPodAdded(pod *corev1.Pod) (err error) {
	if pod == nil {
		return errors.New("pod is nil")
	}

	// add pod to podTacker
	if err := m.storeTrackingPod(pod); err != nil {
		return err
	}

	defer func() {
		if err != nil {
			_ = m.deleteTrackingPod(pod)
		}
	}()

	// update task from pod
	return m.updateTaskFromPod(pod)
}

// onPodUpdated handles the pod updated event
func (m *manager) onPodUpdated(oldPod, newPod *corev1.Pod) (err error) {
	if oldPod == nil || newPod == nil {
		return errors.New("pod is nil")
	}

	// skip if phase is the same
	if oldPod.Status.Phase == newPod.Status.Phase {
		return nil
	}

	// update pod to podTacker
	if err := m.storeTrackingPod(newPod); err != nil {
		return err
	}

	// update task from pod
	return m.updateTaskFromPod(newPod)
}

// onPodAddUpdated handles the pod added or updated event
func (m *manager) onPodAddUpdated(pod *corev1.Pod) {
	key, err := cache.MetaNamespaceKeyFunc(pod)
	if err != nil {
		klog.Errorf("failed to get key for pod %s/%s: %v", pod.GetNamespace(), pod.GetName(), err)
	}

	oldPod, err := m.getTrackingPod(key)
	if err != nil {
		if errors.Is(err, ErrKeyNotFound) {
			// pod not found, add it to podTacker, and run onPodAdded

			if err := m.onPodAdded(pod); err != nil {
				klog.Errorf(err.Error())
			}
		} else {
			// some other error ?
			klog.Errorf("failed to get pod %s/%s: %v", pod.GetNamespace(), pod.GetName(), err)
		}
	} else {
		// pod found, update it in podTacker, and run onPodUpdated

		if err := m.onPodUpdated(oldPod, pod); err != nil {
			klog.Errorf(err.Error())
		}
	}
}

// TODO finish this function
func (m *manager) onPodAddUpdated2(newPod *corev1.Pod) {
	// if pod is nil, skip it
	if newPod == nil {
		klog.Errorf("invalid pod")
		return
	}

	// try to get oldPod from podTacker, if not found, add it to podTacker
	newPodKey, _ := cache.MetaNamespaceKeyFunc(newPod)

	oldPod, err := m.getTrackingPod(newPodKey)
	if err != nil {
		if errors.Is(err, ErrKeyNotFound) {
			// pod not found, add it to podTacker
			if err := m.storeTrackingPod(newPod); err != nil {
				klog.Errorf("podTracker err: %s", err.Error())
				return
			}
		} else {
			// some other error?
			klog.Errorf("podTracker err: %s", err.Error())
			return
		}
	}

	// skip if phase is the same
	if oldPod != nil && oldPod.Status.Phase == newPod.Status.Phase {
		return
	}

	// update pod to podTacker
	if err := m.storeTrackingPod(newPod); err != nil {
		klog.Errorf("podTracker err: %s", err.Error())
		return
	}

	// get taskName / taskKey from pod
	taskName, err := getTaskNameFromPod(newPod)
	if err != nil {
		klog.Errorf(err.Error())
		// no task label found, skip it
		return
	}
	taskKey := newPod.GetNamespace() + "/" + taskName

	// try to get task from taskTacker
	//   if not found, try to query task from taskStorage
	task, err := m.GetTrackingTask(taskKey)
	if err != nil {
		if errors.Is(err, ErrKeyNotFound) {
			// task not found, try to query task from taskStorage
			task, err = m.taskStorage.Get(context.Background(), taskName)
			if err != nil {
				klog.Errorf(err.Error())
				return
			}

			// add task to taskTacker
			if err := m.storeTackingTask(task); err != nil {
				klog.Errorf(err.Error())
				return
			}

		} else {
			// some other error?
			klog.Errorf(err.Error())
			return
		}
	}

	// operate according to status.
	switch newPod.Status.Phase {
	case corev1.PodPending:
		task.Status.Active += 1

	case corev1.PodRunning:
		task.Status.Active += 1
		// TODO

	case corev1.PodSucceeded:
		task.Status.Succeeded += 1
		if task.Status.Conditions == nil {
			task.Status.Conditions = make([]batchv1.JobCondition, 0)
		}
		task.Status.Conditions = append(task.Status.Conditions, batchv1.JobCondition{
			Type:               batchv1.JobComplete,
			Status:             corev1.ConditionTrue,
			LastProbeTime:      metav1.Now(),
			LastTransitionTime: metav1.Now(),
			Reason:             newPod.Status.Reason,
			Message:            newPod.Status.Message,
		})

		completionTime := metav1.Now()
		task.Status.CompletionTime = &completionTime

	case corev1.PodFailed:
		task.Status.Failed += 1
		if task.Status.Conditions == nil {
			task.Status.Conditions = make([]batchv1.JobCondition, 0)
		}

		var (
			reason  string
			message string
		)

		// try to get reason and message from container status
		if len(newPod.Status.ContainerStatuses) > 0 {
			for _, c := range newPod.Status.ContainerStatuses {
				if c.Name == template.PodContainerNormalName || c.Name == template.PodContainerNsenterName {
					if c.State.Terminated != nil {
						reason = c.State.Terminated.Reason
						message = c.State.Terminated.Message
						break
					}
				}
			}
		}
		if len(reason) == 0 {
			reason = newPod.Status.Reason
		}
		if len(message) == 0 {
			message = newPod.Status.Message
		}

		task.Status.Conditions = append(task.Status.Conditions, batchv1.JobCondition{
			Type:               batchv1.JobComplete,
			Status:             corev1.ConditionFalse,
			LastProbeTime:      metav1.Now(),
			LastTransitionTime: metav1.Now(),
			Reason:             reason,
			Message:            message,
		})
	}

}

// onPodDeleted handles the pod deleted event
func (m *manager) onPodDeleted(pod *corev1.Pod) {
	// get task name from pod label
	taskName, err := getTaskNameFromPod(pod)
	if err != nil {
		err = fmt.Errorf("onPodDeleted: %w", err)
		klog.Errorf(err.Error())
		return
	}

	// get task from taskTacker
	taskKey := pod.GetNamespace() + "/" + taskName
	task, err := m.GetTrackingTask(taskKey)
	if err != nil {
		err = fmt.Errorf("onPodDeleted: %w", err)
		klog.Errorf(err.Error())
		return
	}

	// delete task from taskTacker, if task is done(succeeded or failed)
	if len(task.Status.Conditions) > 0 {
		if err := m.deleteTrackingPod(pod); err != nil {
			klog.Errorf(err.Error())
		}
	}

	// delete tackingPod from podTacker
	if err := m.deleteTrackingPod(pod); err != nil {
		klog.Errorf(err.Error())
	}
}

// Clientset returns the clientset
func (m *manager) Clientset() *k8sutils.Clientset {
	return m.clientset
}

// LabelDefault returns the default label
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
	err = m.storeTackingTask(task)
	if err != nil {
		return err
	}

	// Defer a function that removes the task from the tracking map if the creation of the resources fails.
	defer func() {
		if err != nil {
			_ = m.deleteTrackingTask(task)
		}
	}()

	// Create labels with default manager information and the task name.
	newLabels := m.LabelDefault()
	newLabels[taskLabelAddKey] = task.GetName()

	// Create a configmap template with the task information.
	cmTmpl := template.NewConfigMapTemplate(task.Name, task.Namespace, managerConfigMapScriptName, task.Spec.ScriptContent).
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

	podTmpl := template.NewPodTemplate(task.Name, task.Namespace, isPrivileged, task.Spec.Image).
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
	err = m.storeTackingTask(task)

	return nil
}

// GetTrackingTask returns the task with the given key from the tracking map.
// If the task does not exist, an error is returned.
func (m *manager) GetTrackingTask(key string) (*Task, error) {
	val, ok := m.taskTacker.Load(key)
	if !ok {
		return nil, fmt.Errorf("key: %s, err: %w", key, ErrKeyNotFound)
	}

	task, ok := val.(*Task)
	if !ok {
		return nil, fmt.Errorf("key: %s, err: %w", key, ErrValueTypeMismatch)
	}

	return task, nil
}

// storeTackingTask stores the task in the tracking map.
func (m *manager) storeTackingTask(task *Task) error {
	key, err := cache.MetaNamespaceKeyFunc(task)
	if err != nil {
		return err
	}

	m.taskTacker.Store(key, task)
	return nil
}

// deleteTrackingTask deletes the task from the tracking map.
func (m *manager) deleteTrackingTask(task *Task) error {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(task)
	if err != nil {
		return err
	}

	m.taskTacker.Delete(key)
	return nil
}

// getTrackingPod returns the pod with the given key from the tracking map.
// If the pod does not exist, an error is returned.
func (m *manager) getTrackingPod(key string) (*corev1.Pod, error) {
	val, ok := m.podTacker.Load(key)
	if !ok {
		return nil, fmt.Errorf("key: %s, err: %w", key, ErrKeyNotFound)
	}

	pod, ok := val.(*corev1.Pod)
	if !ok {
		return nil, fmt.Errorf("key: %s, err: %w", key, ErrValueTypeMismatch)
	}

	return pod, nil
}

// storeTrackingPod stores the pod in the tracking map.
func (m *manager) storeTrackingPod(pod *corev1.Pod) error {
	key, err := cache.MetaNamespaceKeyFunc(pod)
	if err != nil {
		return err
	}

	m.podTacker.Store(key, pod)
	return nil
}

// deleteTrackingPod deletes the pod from the tracking map.
func (m *manager) deleteTrackingPod(pod *corev1.Pod) error {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(pod)
	if err != nil {
		return err
	}

	m.podTacker.Delete(key)
	return nil
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
//
//	TODO finish this function.
func (m *manager) Start(ctx context.Context) error {
	if m.mainController == nil {
		return errors.New("mainController not inited")
	}

	// query tasks, filter tasks is Running.

	// add task to taskTacker

	// for loop forever, refactor with function?
	//   1. get task from taskTacker, filter tasks that are not active
	//   2. run task(create pod / create configmap)
	//   3. sleep 5s

	// run controller.
	return m.mainController.Run(ctx)

}

// getTaskNameFromPod returns the task name from the pod labels.
// It returns an error if the pod has no labels or the task label does not exist.
func getTaskNameFromPod(pod *corev1.Pod) (string, error) {
	labels := pod.GetLabels()
	if labels == nil {
		return "", errors.New("pod has no labels")
	}

	value, ok := labels[taskLabelAddKey]
	if !ok {
		return "", errors.New("pod has no task label")
	}

	return value, nil
}
