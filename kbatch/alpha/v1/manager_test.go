package v1_test

import (
	"context"
	"testing"
	"time"

	v12 "github.com/linlanniao/k8sutils/kbatch/alpha/v1"
	"github.com/stretchr/testify/assert"
)

func TestInitManager(t *testing.T) {
	mgr := v12.Manager()
	assert.NotNil(t, mgr)
	mgr2 := v12.Manager()

	// assert that mgr2 is the same as mgr
	assert.Equal(t, mgr, mgr2)
}

func TestManager_RunTask(t *testing.T) {
	mgr := v12.Manager()
	assert.NotNil(t, mgr)

	ns := "default"

	task, err := v12.NewTask(
		"test-task",
		ns,
		"nginx",
		"echo 'hello world'",
		v12.ScriptTypeBash)
	assert.NoError(t, err)
	assert.NotNil(t, task)

	ctx := context.Background()
	err = mgr.RunTask(ctx, task)
	assert.NoError(t, err)

	trackingTask, err := mgr.GetTrackingTask(ns + "/" + task.GetName())
	assert.NoError(t, err)
	assert.Equal(t, task.GetName(), trackingTask.GetName())
	assert.Equal(t, task.GetNamespace(), trackingTask.GetNamespace())

	time.Sleep(time.Second * 10)
	// cleanup task
	err = mgr.CleanupTask(ctx, task)
	assert.NoError(t, err)

}
