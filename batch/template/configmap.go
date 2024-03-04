package template

import (
	"fmt"

	"github.com/linlanniao/k8sutils/common"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	cmNamePrefixDefault = "script"
	cmNamespaceDefault  = corev1.NamespaceDefault
)

type ConfigMapTemplate struct {
	cm            *corev1.ConfigMap
	name          string
	namePrefix    string
	namespace     string
	scriptName    string
	scriptContent string
}

func (c *ConfigMapTemplate) initConfigMap() *ConfigMapTemplate {
	// skip if configmap is already initialized
	if c.cm != nil {
		return c
	}

	// set default
	if len(c.namespace) == 0 {
		c.namespace = cmNamespaceDefault
	}
	if len(c.namePrefix) == 0 {
		c.namePrefix = cmNamePrefixDefault
	}
	if len(c.name) == 0 {
		c.name = fmt.Sprintf("%s-%s", c.namePrefix, common.RandLowerUpperNumStr(4))
	}

	// init configmap
	c.cm = &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.name,
			Namespace: c.namespace,
		},
		Data: map[string]string{
			c.scriptName: c.scriptContent,
		},
	}
	return c
}

func (c *ConfigMapTemplate) ConfigMap() *corev1.ConfigMap {
	return c.cm
}

func (c *ConfigMapTemplate) Name() string {
	return c.name
}
