package v2_test

import (
	"context"
	"testing"
	"time"

	v2 "github.com/linlanniao/k8sutils/kbatch/alpha/v2"
	"github.com/stretchr/testify/assert"
)

func TestManager_ApplyK8sManagerClusterRBAC(t *testing.T) {
	mgr := v2.Manager()
	assert.NotNil(t, mgr)

	ctx := context.Background()
	err := mgr.ApplyK8sManagerClusterRBAC(ctx)
	assert.NoError(t, err)
}

func TestManager_RunTask(t *testing.T) {
	mgr := v2.Manager()
	assert.NotNil(t, mgr)

	ctx := context.Background()
	ns := "default"
	task := v2.NewTask(
		"test-pytask",
		ns,
		"nyurik/alpine-python3-requests",
		`
import requests
req = requests.get("http://www.baidu.com")
print(req.text)
`,
		v2.ScriptExecutorPython)

	// create task
	err := mgr.RunTask(ctx, task)
	assert.NoError(t, err)

	time.Sleep(time.Second * 10)

	// cleanup task
	err = mgr.CleanupTask(ctx, task)
	assert.NoError(t, err)

}
