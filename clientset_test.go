package k8sutils

import (
	"context"
	"testing"
)

var testCtx = context.Background()

func TestClient_GetServerVersion(t *testing.T) {
	version, err := GetClientset().GetServerVersion()
	if err != nil {
		t.Errorf("GetServerVersion() error = %v", err)
		return
	}
	t.Logf("version: %s", version)
}
