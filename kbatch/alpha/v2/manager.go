package v2

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/linlanniao/k8sutils"
	"github.com/linlanniao/k8sutils/controller"
	"github.com/linlanniao/k8sutils/validate"
	"golang.org/x/sync/errgroup"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

type manager struct {
	clientset      *k8sutils.Clientset
	mainController *controller.MasterController
	taskTracker    *taskTracker
	jobTracker     *jobTracker
	podTracker     *podTracker

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
			taskTracker: newTaskTracker(),
			jobTracker:  newJobTracker(),
			podTracker:  newPodTracker(),
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

//// storeTrackingTask stores the task in the tracking map.
//func (m *manager) storeTrackingTask(task *Task) error {
//	key, err := cache.MetaNamespaceKeyFunc(task)
//	if err != nil {
//		return err
//	}
//
//	m.taskTracker.Store(key, task)
//	return nil
//}
//
//// deleteTrackingTask deletes the task from the tracking map.
//func (m *manager) deleteTrackingTask(task *Task) error {
//	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(task)
//	if err != nil {
//		return err
//	}
//
//	m.taskTracker.Delete(key)
//	return nil
//}
//
//func (m *manager) storeTrackingJob(job *batchv1.Job) error {
//	key, err := cache.MetaNamespaceKeyFunc(job)
//	if err != nil {
//		return err
//	}
//
//	m.jobTracker.Store(key, job)
//	return nil
//}
//
//func (m *manager) loadTrackingJob(key string) (*batchv1.Job, error) {
//	obj, ok := m.jobTracker.Load(key)
//	if !ok {
//		return nil, errors.New("not found")
//	}
//	job, ok := obj.(*batchv1.Job)
//	if !ok {
//		return nil, errors.New("object is not a job")
//	}
//
//	return job, nil
//}
//
//func (m *manager) deleteTrackingJob(job *batchv1.Job) error {
//	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(job)
//	if err != nil {
//		return err
//	}
//
//	m.jobTracker.Delete(key)
//	return nil
//}
//
//func (m *manager) loadTrackingPod(key string) (*corev1.Pod, error) {
//	obj, ok := m.podTracker.Load(key)
//	if !ok {
//		return nil, errors.New("not found")
//	}
//	pod, ok := obj.(*corev1.Pod)
//	if !ok {
//		return nil, errors.New("object is not a pod")
//	}
//
//	return pod, nil
//}

func (m *manager) RunTask(ctx context.Context, task *Task) (err error) {
	// Try to validate the task.
	if err = validate.Validate(task); err != nil {
		return err
	}

	// Add manager label to task
	task.SetLabels(ManagerLabelsDefault())

	// Add the task to the tracking map.
	err = m.taskTracker.store(task)
	if err != nil {
		return err
	}

	// Defer a function that removes the task from the tracking map if the creation of the resources fails.
	defer func() {
		if err != nil {
			_ = m.taskTracker.delete(task)
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
	err = m.jobTracker.store(job)
	if err != nil {
		return err
	}

	// Defer a function that removes the job from the tracking map if the creation of the resources fails.
	defer func() {
		if err != nil {
			_ = m.jobTracker.delete(job)
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
	// Try to get oldJob from JobTracker, compare the status of oldJob and newJob.
	// If the status is consistent, skip the subsequent operation.
	if oldJob, err := m.jobTracker.load(key); oldJob != nil && err == nil {
		older := oldJob.Status
		newer := job.Status

		if older.Active == newer.Active &&
			older.Succeeded == newer.Succeeded &&
			older.Failed == newer.Succeeded &&
			len(older.Conditions) == len(newer.Conditions) {
			return nil
		}
	}

	do := func(job *batchv1.Job) error {
		// context
		ctx, cancel := context.WithTimeout(context.Background(), callBackMaxRuntime)
		defer cancel()

		// update tracking job
		if err := m.jobTracker.store(job); err != nil {
			return err
		}

		// update task status with the new job
		task, err := Job2Task(job)
		if err != nil {
			return err
		}

		// status update callback
		m.iTaskService.OnStatusUpdate(ctx, task)

		if task.Status.Condition != nil {
			// Condition is not nil represents task has been completed.

			switch task.Status.Condition.Type {
			case TaskComplete:
				m.iTaskService.OnSucceed(ctx, task)
			default:
				m.iTaskService.OnFailed(ctx, task)
			}

			// delete tracking job
			return m.jobTracker.delete(job)
		}

		return nil
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

	// delete tracking job
	m.jobTracker.deleteByKey(key)

	// delete tracking task
	taskKey, err := RemoveSuffix(key, "-")
	if err != nil {
		return fmt.Errorf("onJobDeleted, failed to get taskKey, key=%s, err=%w", key, err)
	}
	m.taskTracker.deleteByKey(taskKey)

	klog.Infof("onJobDeleted, key=%s", key)

	return nil
}

const EndOfPodLine = "---------- END OF TASK-RUN ----------\n"

func (m *manager) onPodAddedUpdated(key string, pod *corev1.Pod) error {
	// Try to get oldPod from PodTracker, compare the status of oldPod and newPod.
	// If the status is consistent, skip the subsequent operation.
	oldPod, err := m.podTracker.load(key)
	if oldPod != nil && err == nil {
		older := oldPod.Status
		newer := pod.Status

		if older.Phase == newer.Phase {
			return nil
		}
	}

	do := func(older, newer *corev1.Pod) error {
		// context
		ctx, cancel := context.WithTimeout(context.Background(), callBackMaxRuntime)
		defer cancel()

		// update tracker
		if err := m.podTracker.store(newer); err != nil {
			return err
		}

		// parse taskRun from newer
		taskRun, err := Pod2TaskRun(newer)
		if err != nil {
			return err
		}

		// status update callback
		m.iTaskRunService.OnStatusUpdate(ctx, taskRun)

		// 处理日志, 只处理Running/Succeeded/Failed状态的pod, 其他状态的pod不处理日志
		initBufSize := 100 // 300 lines buffer
		tickerDuration := 10 * time.Second
		logCh := make(chan LogLine, initBufSize)
		g, ctx := errgroup.WithContext(ctx)

		switch taskRun.Status.Phase {
		case TaskRunRunning:
			g.Go(func() error {
				klog.Infof("Pod %s is running, start tracking logs (tail logs)", newer.GetName())
				return m.getOrTailLogs(ctx, newer.GetNamespace(), newer.GetName(), logCh, true)
			})

		case TaskRunFailed, TaskRunSucceeded:
			if older != nil && older.Status.Phase == corev1.PodRunning {
				// If it changes from Running state to Succeed/Failed state, the log will not be processed.
				close(logCh)
				return nil
			}

			g.Go(func() error {
				klog.Infof("Pod %s is over, start to refresh the logs in full (get logs)", newer.GetName())
				m.iTaskRunService.CleanAllLogs(ctx, taskRun) // Delete the old log first.
				return m.getOrTailLogs(ctx, newer.GetNamespace(), newer.GetName(), logCh, false)
			})

		default:
			// Pods in other states do not handle logs.
			close(logCh)
			return nil
		}

		// 处理日志的goroutine
		g.Go(func() error {
			ticker := time.NewTicker(tickerDuration)
			lines := make(LogLines, 0, initBufSize)
			doFlush := func(isEof bool) error { // flush日志
				if len(lines) == 0 && !isEof {
					return nil
				}
				logs := make(LogLines, len(lines))
				copy(logs, lines)
				if isEof {
					logs = append(logs, &LogLine{
						Timestamp: metav1.Now(),
						Message:   EndOfPodLine,
					})
				}
				lines = lines[:0]
				m.iTaskRunService.OnLogs(ctx, taskRun, logs)
				return nil
			}
			for {
				select {
				case line, ok := <-logCh: // Read the log from logCh

					if !ok {
						// logCh off, flush log
						ticker.Stop()
						if err := doFlush(true); err != nil {
							return err
						}
						return nil
					}

					if len(lines) >= initBufSize { // If the length of the line exceeds initBufSize, the flush log
						if err := doFlush(false); err != nil {
							return err
						}
						ticker.Reset(tickerDuration)
					}
					lines = append(lines, &line)

				case <-ticker.C: // Ticker timer trigger, flush log
					if err := doFlush(false); err != nil {
						return err
					}
				}
			}
		})
		return g.Wait()

	}

	err = do(oldPod, pod)
	if err != nil {
		klog.Errorf("onPodAddedUpdated, key=%s, err=%v", key, err)
		return err
	}
	klog.Infof("onPodAddedUpdated, key=%s", key)
	return nil
}

func (m *manager) onPodDeleted(key string) error {
	// Delete the value of podTracker to avoid memory leakage

	// delete tracking pod
	m.podTracker.deleteByKey(key)

	klog.Infof("onPodDeleted, key=%s", key)
	return nil
}

func (m *manager) getOrTailLogs(ctx context.Context, namespace, podName string, logsCh chan<- LogLine, tail bool) error {
	defer close(logsCh)
	logOptions := &corev1.PodLogOptions{
		Timestamps: true,
		Follow:     tail, // log tail
	}
	req := m.clientset.GetClientSet().CoreV1().Pods(namespace).GetLogs(podName, logOptions)
	logStream, err := req.Stream(ctx)
	if err != nil {
		return err
	}
	defer func(logStream io.ReadCloser) {
		_ = logStream.Close()
	}(logStream)
	reader := bufio.NewReader(logStream)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		lst := strings.SplitN(line, " ", 2)
		timestamp, _ := time.Parse(time.RFC3339Nano, lst[0])
		logsCh <- LogLine{
			Timestamp: metav1.NewTime(timestamp),
			Message:   lst[1],
		}
	}
}
