package v2

import (
	"errors"
	"fmt"

	"github.com/linlanniao/k8sutils/common"
	"github.com/linlanniao/k8sutils/kbatch/alpha/v2/builders"
	"github.com/linlanniao/k8sutils/validate"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Script struct {
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Spec              ScriptSpec   `json:"spec"`
	Status            ScriptStatus `json:"status,omitempty"`
}

func (s *Script) Validate() error {
	if s.ObjectMeta.Name == "" {
		return errors.New("name cannot be empty")
	}

	if s.ObjectMeta.Namespace == "" {
		return errors.New("namespace cannot be empty")
	}

	if err := validate.Validate(s.Spec); err != nil {
		return err
	}

	return nil
}

type ScriptSpec struct {
	Content  string         `json:"content"`
	Executor ScriptExecutor `json:"executor"`
}

func (s *ScriptSpec) Validate() error {
	if s.Content == "" {
		return errors.New("content cannot be empty")
	}
	if err := validate.Validate(s.Executor); err != nil {
		return err
	}
	return nil
}

type ScriptExecutor string

func (s ScriptExecutor) Validate() error {
	switch s {
	case ScriptExecutorSh, ScriptExecutorBash, ScriptExecutorPython:
		return nil
	default:
		return fmt.Errorf("invalid Script executor: %s", s)
	}
}

func (s ScriptExecutor) String() string {
	return string(s)
}

const (
	ScriptExecutorSh     ScriptExecutor = "sh"
	ScriptExecutorBash   ScriptExecutor = "bash"
	ScriptExecutorPython ScriptExecutor = "python"

	ScriptTypeDefault ScriptExecutor = ScriptExecutorSh
)

func (s ScriptExecutor) AsBuildersScriptExecutor() builders.ScriptExecutor {
	return builders.ScriptExecutor(s)
}

type ScriptStatus struct {
	Configmap          *corev1.ConfigMap `json:"configmap,omitempty"`
	IsConfigmapApplied bool              `json:"isConfigmapApplied,omitempty"`
}

func NewScript(
	generateName, namespace, content string,
	executor ScriptExecutor,
	opts ...scriptOption) *Script {
	s := new(Script)
	s.ObjectMeta = metav1.ObjectMeta{
		Namespace: namespace,
	}
	s.ObjectMeta.Name = common.GenerateName2Name(generateName)

	s.Spec = ScriptSpec{
		Content:  content,
		Executor: executor,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

type scriptOption func(s *Script)

func WithScriptAnnotations(annotations map[string]string) scriptOption {
	return func(s *Script) {
		s.ObjectMeta.Annotations = annotations
	}
}

func WithScriptLabels(labels map[string]string) scriptOption {
	return func(s *Script) {
		s.ObjectMeta.Labels = labels
	}
}

const (
	scriptNameLabelKey     = "v2.alpha.kbatch.k8sutils.ppops.cn/script"
	scriptExecutorLabelKey = "v2.alpha.kbatch.k8sutils.ppops.cn/executor"
	scriptConfigMapDataKey = "script"
)

func (s *Script) GenerateConfigMap() (*corev1.ConfigMap, error) {
	builder := builders.ConfigMapBuilder(
		s.GetName(),
		s.GetNamespace(),
		scriptConfigMapDataKey,
		s.Spec.Content,
	)

	if labels := s.ObjectMeta.GetLabels(); len(labels) > 0 {
		builder = builder.SetLabels(labels)
	}

	if annotations := s.ObjectMeta.GetAnnotations(); len(annotations) > 0 {
		builder = builder.SetAnnotations(annotations)
	}

	// set metadata with script info
	builder.SetLabel(scriptNameLabelKey, s.ObjectMeta.Name)
	builder.SetLabel(scriptExecutorLabelKey, s.Spec.Executor.String())

	if err := validate.Validate(builder); err != nil {
		return nil, err
	}

	cm := builder.ConfigMap()

	s.Status.Configmap = cm

	return cm, nil
}

func (s *Script) ConfigMap() (*corev1.ConfigMap, error) {
	if s.Status.Configmap == nil {
		return nil, errors.New("configmap is not generated")
	}

	return s.Status.Configmap, nil
}
