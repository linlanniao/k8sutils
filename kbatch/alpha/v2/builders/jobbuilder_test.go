package builders_test

import (
	"context"
	"testing"
	"time"

	"github.com/linlanniao/k8sutils"
	"github.com/linlanniao/k8sutils/kbatch/alpha/v2/builders"
	"github.com/stretchr/testify/assert"
)

func TestJobBuilder(t *testing.T) {
	type config struct {
		name         string
		generateName string
		namespace    string
		isPrivileged bool
		image        string
	}

	tests := []config{
		{
			generateName: "a",
			namespace:    "default",
			isPrivileged: false,
			image:        "",
		},
		{
			generateName: "b",
			namespace:    "default",
			isPrivileged: true,
			image:        "",
		},
	}

	gcList := []config{}

	for _, tt := range tests {
		t.Run(tt.generateName, func(t *testing.T) {
			b := builders.JobBuilder(tt.generateName, tt.namespace, tt.isPrivileged, tt.image, nil)
			j := b.Job()
			assert.NotNil(t, j)
			t.Logf("job name: %s", tt.generateName)
			job, err := k8sutils.GetClientset().CreateJob(context.Background(), tt.namespace, j)
			assert.NoError(t, err)
			assert.NotNil(t, job)
			gcList = append(gcList, config{namespace: tt.namespace, name: job.GetName()})
		})
	}

	time.Sleep(10 * time.Second)

	// clean
	for _, tt := range gcList {
		t.Run(tt.name, func(t *testing.T) {
			err := k8sutils.GetClientset().DeleteJob(context.Background(), tt.namespace, tt.name)
			assert.NoError(t, err)
		})
	}
}
