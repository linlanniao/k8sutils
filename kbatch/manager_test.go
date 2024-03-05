package kbatch_test

import (
	"context"
	"testing"
	"time"

	"github.com/linlanniao/k8sutils/kbatch"
	"github.com/stretchr/testify/assert"
)

func TestInitManager(t *testing.T) {
	mgr := kbatch.Manager()
	assert.NotNil(t, mgr)
	mgr2 := kbatch.Manager()

	// assert that mgr2 is the same as mgr
	assert.Equal(t, mgr, mgr2)
}

func TestManager_RunTask(t *testing.T) {
	mgr := kbatch.Manager()
	assert.NotNil(t, mgr)

	ns := "default"

	task, err := kbatch.NewTask(
		"test-task",
		ns,
		"nginx",
		"echo 'hello world'",
		kbatch.ScriptTypeBash)
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
