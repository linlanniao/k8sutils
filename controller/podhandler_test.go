package controller_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/linlanniao/k8sutils"
	"github.com/linlanniao/k8sutils/common"
	"github.com/linlanniao/k8sutils/controller"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createPod() (string, error) {
	// 创建一个Pod对象
	// ...

	k8sutils.GetClientset()
	name := "testpod-" + common.RandLowerStr(5)
	ns := "default"
	var pod = &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
			Labels: map[string]string{
				"controller.k8sutils.ppops.cn/pods": "banana",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "c1",
					Image: "nginx:alpine",
				},
			},
		},
	}
	clientset := k8sutils.GetClientset()
	p, err := clientset.GetClientSet().CoreV1().Pods(ns).Create(context.TODO(), pod, metav1.CreateOptions{})
	if err != nil {
		return p.GetName(), err
	}
	return p.GetName(), nil
}

func deletePod(name string) error {
	return k8sutils.GetClientset().GetClientSet().CoreV1().Pods("default").Delete(context.Background(), name, metav1.DeleteOptions{})
}

func TestCreatePod(t *testing.T) {
	name, err := createPod()
	assert.NotEmpty(t, name)
	assert.NoError(t, err)
}

func TestNewPodHandler(t *testing.T) {
	addedUpdateFunc := func(key string, pod *corev1.Pod) error {
		fmt.Println(key)
		fmt.Println(pod.GetName())
		return nil
	}
	deletedFunc := func(key string) error {
		fmt.Println(key)
		return nil
	}

	ph := controller.NewPodHandler(
		"banana",
		"default",
		3,
		addedUpdateFunc,
		deletedFunc,
		k8sutils.GetClientset(),
	)

	c := controller.NewMasterController()
	c.AddController(ph)
	go func() {
		err := c.Run(context.Background())
		assert.NoError(t, err)
	}()

	names := make([]string, 0)
	for i := 0; i < 3; i++ {
		podName, err := createPod()
		if err == nil {
			names = append(names, podName)
		}
	}

	time.Sleep(time.Second * 20)
	for _, name := range names {
		deletePod(name)
	}
}
