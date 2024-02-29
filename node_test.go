package k8sutils

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClient_GetNodes(t *testing.T) {
	nas := NodeAffinities{
		{
			Key:      "kubernetes.io/hostname",
			Operator: NodeAffinityOpIn,
			Values: []string{
				"node01",
				//"node02",
			},
		},
	}
	affinity, _ := newAffinity(nas, true)
	t.Logf("affinity: %+v", affinity)
	selector := affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution
	nodes, err := GetClientset().GetNodes(context.Background(), selector)
	assert.NoError(t, err)
	assert.NotNil(t, nodes)
	assert.NotEmpty(t, nodes.Items)
	for _, node := range nodes.Items {
		t.Logf("nodeName: %+v", node.GetName())
	}
}

func TestClient_GetNodesByIps(t *testing.T) {
	ips := []string{
		"10.66.216.24", // node01
		"10.66.216.25", // node02   // labelSelect 会过滤掉这个节点
		"10.66.216.26", // node03
	}
	nas := NodeAffinities{
		{
			Key:      "kubernetes.io/hostname",
			Operator: NodeAffinityOpIn,
			Values: []string{
				"node01",
				//"node02",
				"node03",
			},
		},
	}
	affinity, _ := newAffinity(nas, true)
	t.Logf("affinity: %+v", affinity)
	selector := affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution
	nodes, err := GetClientset().GetNodes(context.Background(), selector, ips...)
	assert.NoError(t, err)
	assert.NotNil(t, nodes)
	assert.NotEmpty(t, nodes.Items)
	for _, node := range nodes.Items {
		t.Logf("nodeName: %+v", node.GetName())
	}
}
