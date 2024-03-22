package v2

import (
	"errors"
	"sync"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

type taskTracker struct {
	m *sync.Map
}

func newTaskTracker() *taskTracker {
	return &taskTracker{m: &sync.Map{}}
}

func (t *taskTracker) store(task *Task) error {
	key, err := cache.MetaNamespaceKeyFunc(task)
	if err != nil {
		return err
	}

	t.m.Store(key, task)
	return nil
}
func (t *taskTracker) load(key string) (*Task, error) {
	obj, ok := t.m.Load(key)
	if !ok {
		return nil, errors.New("not found")
	}
	task, ok := obj.(*Task)
	if !ok {
		return nil, errors.New("object is not a task")
	}

	return task, nil
}

func (t *taskTracker) delete(task *Task) error {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(task)
	if err != nil {
		return err
	}

	t.m.Delete(key)
	return nil
}

func (t *taskTracker) deleteByKey(key string) {
	t.m.Delete(key)
}

type jobTracker struct {
	m *sync.Map
}

func newJobTracker() *jobTracker {
	return &jobTracker{m: &sync.Map{}}
}

func (j *jobTracker) store(job *batchv1.Job) error {
	key, err := cache.MetaNamespaceKeyFunc(job)
	if err != nil {
		return err
	}

	j.m.Store(key, job)
	return nil
}

func (j *jobTracker) load(key string) (*batchv1.Job, error) {
	obj, ok := j.m.Load(key)
	if !ok {
		return nil, errors.New("not found")
	}
	job, ok := obj.(*batchv1.Job)
	if !ok {
		return nil, errors.New("object is not a job")
	}

	return job, nil
}

func (j *jobTracker) delete(job *batchv1.Job) error {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(job)
	if err != nil {
		return err
	}

	j.m.Delete(key)
	return nil
}

func (j *jobTracker) deleteByKey(key string) {
	j.m.Delete(key)
}

type podTracker struct {
	m *sync.Map
}

func newPodTracker() *podTracker {
	return &podTracker{m: &sync.Map{}}
}

func (p *podTracker) store(pod *corev1.Pod) error {
	key, err := cache.MetaNamespaceKeyFunc(pod)
	if err != nil {
		return err
	}

	p.m.Store(key, pod)
	return nil
}

func (p *podTracker) load(key string) (*corev1.Pod, error) {
	obj, ok := p.m.Load(key)
	if !ok {
		return nil, errors.New("not found")
	}
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return nil, errors.New("object is not a pod")
	}

	return pod, nil
}

func (p *podTracker) delete(pod *corev1.Pod) error {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(pod)
	if err != nil {
		return err
	}

	p.m.Delete(key)
	return nil
}

func (p *podTracker) deleteByKey(key string) {
	p.m.Delete(key)
}
