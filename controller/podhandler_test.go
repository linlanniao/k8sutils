package controller_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/linlanniao/k8sutils"
	"github.com/linlanniao/k8sutils/controller"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createPod() (string, error) {
	// 创建一个Pod对象
	// ...

	name := "testpod-" + k8sutils.RandLowerStr(5)
	ns := "default"
	var pod = &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
			Labels: map[string]string{
				"handler.k8sutils.ppops.cn/pods": "banana",
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
	clientset, _ := k8sutils.GetClientset()
	p, err := clientset.GetClientSet().CoreV1().Pods(ns).Create(context.TODO(), pod, metav1.CreateOptions{})
	if err != nil {
		return p.GetName(), err
	}
	return p.GetName(), nil
}

func deletePod(name string) error {
	c, _ := k8sutils.GetClientset()
	return c.GetClientSet().CoreV1().Pods("default").Delete(context.Background(), name, metav1.DeleteOptions{})
}

func TestCreatePod(t *testing.T) {
	name, err := createPod()
	assert.NotEmpty(t, name)
	assert.NoError(t, err)
}

func TestNewPodHandler(t *testing.T) {
	addedUpdateFunc := func(key string, obj any) error {
		fmt.Println(key)
		fmt.Println(obj)
		return nil
	}
	deletedFunc := func(key string) error {
		fmt.Println(key)
		return nil
	}
	h := controller.NewPodHandler(
		"banana",
		4,
		[]controller.OnAddedUpdatedFunc{addedUpdateFunc},
		[]controller.OnDeletedFunc{deletedFunc},
	)
	c := controller.NewController()
	c.AddHandler(h)
	go func() {
		err := c.Start(context.Background())
		assert.NoError(t, err)
	}()

	names := make([]string, 0)
	for i := 0; i < 10; i++ {
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
