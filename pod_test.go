package k8sutils

import (
	"strconv"
	"testing"
	"time"

	"github.com/linlanniao/k8sutils/common"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCreateDeletePod(t *testing.T) {
	namePrefix := "testpod-" + common.RandLowerStr(4)
	ns := "default"
	tmpl := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Labels: map[string]string{
				"handler.k8sutils.ppops.cn/pods": "banana",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "c",
					Image: "nginx:alpine",
				},
			},
		},
	}
	c := GetClientset()

	counting := func() int {
		currPods, err := c.ListPod(testCtx, ns, nil)
		assert.NoError(t, err)
		return len(currPods.Items)
	}
	prePodCount := counting()

	pods := make([]*corev1.Pod, 0)

	for i := 0; i < 10; i++ {
		pod := tmpl.DeepCopy()
		pod.SetName(namePrefix + "-" + strconv.Itoa(i))
		pods = append(pods, pod)
	}

	// create
	for _, pod := range pods {
		_, err := c.CreatePod(testCtx, ns, pod)
		if err != nil {
			t.Fatal(err)
		}
	}

	time.Sleep(time.Second * 3)
	// delete
	for _, pod := range pods {
		err := c.DeletePod(testCtx, ns, pod.GetName())
		assert.NoError(t, err)
	}

	time.Sleep(time.Second * 3)
	postPodCount := counting()
	assert.Equal(t, prePodCount, postPodCount)

}
