package kbatch

import (
	"sync"

	"github.com/linlanniao/k8sutils"
	"github.com/linlanniao/k8sutils/kbatch/template"
	"github.com/linlanniao/k8sutils/validate"
)

const (
	ManagerLabelKey   = "batch.k8sutils.ppops.cn/manager"
	ManagerLabelValue = "v1"

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

func (m *manager) NewLabels() map[string]string {
	return map[string]string{ManagerLabelKey: ManagerLabelValue}
}

func (m *manager) RunTask(task *Task) error {
	// try to validate job
	if err := validate.Validate(task); err != nil {
		return err
	}

	// create label with task information
	// TODO  增加task信息
	newLabels := m.NewLabels()

	// create configmap
	cmTmpl := template.NewConfigMapTemplate(task.Name, task.Namespace, ManagerConfigMapScriptName, task.Spec.ScriptContent)

	cmTmpl.SetLabels(newLabels)

	// try to validate configmap
	if err := validate.Validate(cmTmpl); err != nil {
		return err
	}
	cmName := cmTmpl.Name()
	cm := cmTmpl.ConfigMap()
	_ = cmName // todo
	_ = cm     // todo

	// create pod
	isPrivileged := false
	if task.Spec.Privilege != nil && *task.Spec.Privilege == TaskPrivilegeHostRoot {
		isPrivileged = true
	}

	podTmpl := template.NewPodTemplate(task.Name, task.Namespace, isPrivileged, task.Spec.Image)
	podTmpl.SetLabels(newLabels)
	// todo set cm

	return nil
}
