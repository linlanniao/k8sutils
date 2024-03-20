package v2

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/linlanniao/k8sutils"
	"github.com/linlanniao/k8sutils/controller"
	"github.com/linlanniao/k8sutils/validate"
	batchv1 "k8s.io/api/batch/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

type manager struct {
	clientset      *k8sutils.Clientset
	mainController *controller.MasterController
	taskTracker    *sync.Map
	jobTracker     *sync.Map
	podTracker     *sync.Map

	iTaskService    ITaskService
	iTaskRunService ITaskRunService
}

var (
	singleMgr     *manager
	singleMgrOnce sync.Once
)

// Manager returns the single instance of manager
func Manager() *manager {
	singleMgrOnce.Do(func() {
		singleMgr = &manager{
			clientset:   k8sutils.GetClientset(),
			taskTracker: &sync.Map{},
			jobTracker:  &sync.Map{},
			podTracker:  &sync.Map{},
		}
	})

	return singleMgr
}

// InitController initializes the controller
func (m *manager) InitController(taskSvc ITaskService, taskRunSvc ITaskRunService) error {
	// skip init if already inited
	if m.mainController != nil {
		return errors.New("already inited")
	}

	m.iTaskService = taskSvc
	m.iTaskRunService = taskRunSvc

	// init job handler
	jobHandler := controller.NewJobHandler(
		m.iTaskService.Name(),
		m.iTaskService.Namespace(),
		m.iTaskService.Workers(),
		m.onJobAddedUpdated,
		m.onJobDeleted,
		m.clientset,
		managerAddLabelKey,
		managerAddLabelVal,
	)

	// init pod handler
	podHandler := controller.NewPodHandler(
		m.iTaskRunService.Name(),
		m.iTaskRunService.Namespace(),
		m.iTaskRunService.Workers(),
		nil,
		nil,
		//iTaskSvc.OnAddedUpdatedFunc(),
		//iTaskSvc.OnDeletedFunc(),
		m.clientset,
		managerAddLabelKey,
		managerAddLabelVal,
	)

	_ = podHandler // TODO finish this handler

	// init mainController
	m.mainController = controller.NewMasterController(controller.WithHandlers(
		jobHandler,
		//podHandler,
	))

	return nil
}

const (
	managerAddLabelKey = "kbatch.k8sutils.ppops.cn/manager"
	managerAddLabelVal = "alpha-v2"

	clusterRootLabelKey = "kbatch.k8sutils.ppops.cn/privilege"
	clusterRootLabelVal = "cluster-root"
)

var (
	// rules for k8s-manager.
	_k8sManagerRules = rbacv1.PolicyRule{
		Verbs:           []string{"*"}, // * Represent all permissions
		APIGroups:       []string{"*"}, // * Represent all API groups
		Resources:       []string{"*"}, // * Represent all resources
		NonResourceURLs: []string{"*"}, // * Represents all non-resource URLs
	}
)

func ManagerLabelsDefault() map[string]string {
	return map[string]string{
		managerAddLabelKey: managerAddLabelVal,
	}
}

func K8sManagerRules() rbacv1.PolicyRule {
	r := _k8sManagerRules.DeepCopy()
	return *r
}

// ApplyK8sManagerClusterRBAC applies the necessary RBAC resources for the k8s-manager.
func (m *manager) ApplyK8sManagerClusterRBAC(ctx context.Context) error {
	// labels is a map of labels to be applied to all resources created by the manager.
	labels := ManagerLabelsDefault()
	labels[clusterRootLabelKey] = clusterRootLabelVal

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

// storeTrackingTask stores the task in the tracking map.
func (m *manager) storeTrackingTask(task *Task) error {
	key, err := cache.MetaNamespaceKeyFunc(task)
	if err != nil {
		return err
	}

	m.taskTracker.Store(key, task)
	return nil
}

// deleteTrackingTask deletes the task from the tracking map.
func (m *manager) deleteTrackingTask(task *Task) error {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(task)
	if err != nil {
		return err
	}

	m.taskTracker.Delete(key)
	return nil
}

func (m *manager) storeTrackingJob(job *batchv1.Job) error {
	key, err := cache.MetaNamespaceKeyFunc(job)
	if err != nil {
		return err
	}

	m.jobTracker.Store(key, job)
	return nil
}

func (m *manager) loadTrackingJob(key string) (job *batchv1.Job, ok bool) {
	if obj, ok := m.jobTracker.Load(key); ok {
		if job, ok := obj.(*batchv1.Job); ok {
			return job, ok
		}
	}
	return nil, false
}

func (m *manager) deleteTrackingJob(job *batchv1.Job) error {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(job)
	if err != nil {
		return err
	}

	m.jobTracker.Delete(key)
	return nil
}

func (m *manager) RunTask(ctx context.Context, task *Task) (err error) {
	// Try to validate the task.
	if err = validate.Validate(task); err != nil {
		return err
	}

	// Add manager label to task
	task.SetLabels(ManagerLabelsDefault())

	// Add the task to the tracking map.
	err = m.storeTrackingTask(task)
	if err != nil {
		return err
	}

	// Defer a function that removes the task from the tracking map if the creation of the resources fails.
	defer func() {
		if err != nil {
			_ = m.deleteTrackingTask(task)
		}
	}()

	// generate script / configMap
	script, err := task.GenerateScript()
	if err != nil {
		return err
	}
	configMap, err := script.GenerateConfigMap()
	if err != nil {
		return err
	}

	// create configmap
	configMap, err = m.clientset.CreateConfigMap(ctx, script.GetNamespace(), configMap)
	if err != nil {
		return err
	}

	// update script status with the new configmap
	script.Status.IsConfigmapApplied = true
	script.Status.Configmap = configMap

	// generate job
	job, err := task.GenerateJob()
	if err != nil {
		return err
	}

	// create job
	job, err = m.clientset.CreateJob(ctx, job.GetNamespace(), job)
	if err != nil {
		return err
	}

	// update task status with the new job
	task.Status.IsJobApplied = true
	task.Status.Job = job

	// tracking job
	err = m.storeTrackingJob(job)
	if err != nil {
		return err
	}

	// Defer a function that removes the job from the tracking map if the creation of the resources fails.
	defer func() {
		if err != nil {
			_ = m.deleteTrackingJob(job)
		}
	}()

	return nil
}

// CleanupTask deletes all the resources created for the task.
func (m *manager) CleanupTask(ctx context.Context, task *Task) (err error) {
	labels := ManagerLabelsDefault()
	labels[TaskNameLabelKey] = task.GetName()

	// Query job with labels
	jobLst, err := m.clientset.ListJob(ctx, task.GetNamespace(), labels)
	if err != nil {
		return err
	}

	// Delete jobs
	for _, job := range jobLst.Items {
		if err := m.clientset.DeleteJob(ctx, job.GetNamespace(), job.GetName()); err != nil {
			return err
		}
	}

	// Query configmaps.
	cmLst, err := m.clientset.ListConfigMap(ctx, task.GetNamespace(), labels)
	if err != nil {
		return err
	}

	// Delete configmaps.
	for _, cm := range cmLst.Items {
		if err := m.clientset.DeleteConfigMap(ctx, cm.GetNamespace(), cm.GetName()); err != nil {
			return err
		}
	}

	return nil
}

func (m *manager) Start(ctx context.Context) error {
	if m.mainController == nil {
		return errors.New("mainController not inited")
	}

	// apply rbac configuration
	if err := m.ApplyK8sManagerClusterRBAC(ctx); err != nil {
		return err
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

const (
	callBackMaxRuntime = time.Second * 300
)

func (m *manager) onJobAddedUpdated(key string, job *batchv1.Job) error {
	//klog.Infof("onJobAddedUpdated, key=%s", key)

	if oldJob, ok := m.loadTrackingJob(key); ok {
		older := oldJob.Status
		newer := job.Status

		if older.Active == newer.Active &&
			older.Succeeded == newer.Succeeded &&
			older.Failed == newer.Succeeded &&
			len(older.Conditions) == len(newer.Conditions) {
			// The status of the new and old jobs is consistent, skip the follow-up processing.
			return nil
		}
	}

	do := func(job *batchv1.Job) error {
		// context
		ctx, cancel := context.WithTimeout(context.Background(), callBackMaxRuntime)
		defer cancel()

		// update tracking job
		err := m.storeTrackingJob(job)
		if err != nil {
			return err
		}

		task, err := ParseTaskFromJob(job)
		if err != nil {
			return err
		}
		status := job.Status

		task.Status.Job = job
		task.Status.IsJobApplied = true
		task.Status.Active = status.Active
		task.Status.Succeeded = status.Succeeded
		task.Status.Failed = status.Failed

		if len(status.Conditions) == 0 {
			// conditions is empty means the job is not done
			task.Status.Condition = nil
			m.iTaskService.OnTaskStatusUpdateFunc(ctx, task)

			return nil

		} else {
			// job is already done, update status and run callback function
			c0 := status.Conditions[0]
			task.Status.Condition = &TaskCondition{
				Type:               TaskConditionType(c0.Type),
				Status:             ConditionStatus(c0.Status),
				LastProbeTime:      c0.LastProbeTime,
				LastTransitionTime: c0.LastTransitionTime,
				Reason:             c0.Reason,
				Message:            c0.Message,
			}

			// callback
			m.iTaskService.OnTaskStatusUpdateFunc(ctx, task)
			m.iTaskService.OnTaskDoneFunc(ctx, task)

			// delete tracking job
			return m.deleteTrackingJob(job)
		}
	}

	err := do(job)
	if err != nil {
		klog.Errorf("onJobAddedUpdated, key=%s, err=%v", key, err)
		return err
	}
	klog.Infof("onJobAddedUpdated, key=%s", key)
	return nil

}

func (m *manager) onJobDeleted(key string) error {
	// Delete the value of jobTracker to avoid memory leakage
	m.jobTracker.Delete(key)
	klog.Infof("onJobDeleted, key=%s", key)
	return nil
}
