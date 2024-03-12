package v2

import (
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
	Status            scriptStatus `json:"status,omitempty"`
}

type ScriptSpec struct {
	Content  string         `json:"content"`
	Executor ScriptExecutor `json:"executor"`
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

type scriptStatus struct {
	Configmap *corev1.ConfigMap `json:"configmap,omitempty"` // if nil, the configmap is not created
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

func WithAnnotations(annotations map[string]string) scriptOption {
	return func(s *Script) {
		s.ObjectMeta.Annotations = annotations
	}
}

func WithLabels(labels map[string]string) scriptOption {
	return func(s *Script) {
		s.ObjectMeta.Labels = labels
	}
}

func (s *Script) Validate() error {
	if err := s.Spec.Executor.Validate(); err != nil {
		return err
	}
	return nil
}

const (
	scriptNameLabelKey     = "v2.alpha.kbatch.k8sutils.ppops.cn/script-name"
	scriptExecutorLabelKey = "v2.alpha.kbatch.k8sutils.ppops.cn/script-executor"
)

func (s *Script) AsConfigMap() (*corev1.ConfigMap, error) {
	builder := builders.ConfigMapBuilder(
		s.GetName(),
		s.GetNamespace(),
		"script",
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

	return builder.ConfigMap(), nil
}
