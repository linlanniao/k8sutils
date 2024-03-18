package v2

import (
	"context"
	"sync"

	"github.com/linlanniao/k8sutils"
	"github.com/linlanniao/k8sutils/validate"
	batchv1 "k8s.io/api/batch/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

type manager struct {
	clientset   *k8sutils.Clientset
	taskTracker *sync.Map
	jobTracker  *sync.Map
	podTracker  *sync.Map
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

const (
	managerAddLabelKey = "kbatch.k8sutils.ppops.cn/manager-version"
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

// startTrackingTask stores the task in the tracking map.
func (m *manager) startTrackingTask(task *Task) error {
	key, err := cache.MetaNamespaceKeyFunc(task)
	if err != nil {
		return err
	}

	m.taskTracker.Store(key, task)
	return nil
}

// stopTrackingTask deletes the task from the tracking map.
func (m *manager) stopTrackingTask(task *Task) error {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(task)
	if err != nil {
		return err
	}

	m.taskTracker.Delete(key)
	return nil
}

func (m *manager) startTrackingJob(job *batchv1.Job) error {
	key, err := cache.MetaNamespaceKeyFunc(job)
	if err != nil {
		return err
	}

	m.jobTracker.Store(key, job)
	return nil
}

func (m *manager) stopTrackingJob(job *batchv1.Job) error {
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
	err = m.startTrackingTask(task)
	if err != nil {
		return err
	}

	// Defer a function that removes the task from the tracking map if the creation of the resources fails.
	defer func() {
		if err != nil {
			_ = m.stopTrackingTask(task)
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
	job, err := task.GenerateJob(script)
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
	err = m.startTrackingJob(job)
	if err != nil {
		return err
	}

	// Defer a function that removes the job from the tracking map if the creation of the resources fails.
	defer func() {
		if err != nil {
			_ = m.stopTrackingJob(job)
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
