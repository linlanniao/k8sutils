package v2_test

import (
	"context"
	"testing"

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
