package template

import (
	"errors"
	"strings"

	"github.com/linlanniao/k8sutils/common"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	cmGenerateNameDefault = "script-"
	cmNamespaceDefault    = corev1.NamespaceDefault
)

// ConfigMapTemplate creates a ConfigMap with a single key-value pair
type ConfigMapTemplate struct {
	configMap     *corev1.ConfigMap
	name          string
	generateName  string
	namespace     string
	scriptName    string
	scriptContent string
}

// NewConfigMapTemplate creates a new ConfigMapTemplate instance
func NewConfigMapTemplate(generateName, namespace, scriptName, scriptContent string) *ConfigMapTemplate {

	if !strings.HasSuffix(generateName, "-") {
		generateName = generateName + "-"
	}

	c := &ConfigMapTemplate{
		generateName:  generateName,
		namespace:     namespace,
		scriptName:    scriptName,
		scriptContent: scriptContent,
	}

	c.initConfigMap()

	return c
}

func (c *ConfigMapTemplate) Validate() error {
	if c.configMap == nil {
		return errors.New("configMap is not initialized")
	}

	if c.configMap.Name == "" ||
		c.generateName == "" ||
		strings.HasPrefix(c.generateName, c.name) ||
		c.name != c.configMap.Name {
		return errors.New("configMap name or namePrefix is not valid")
	}

	if c.namespace == "" || c.namespace != c.configMap.Namespace {
		return errors.New("namespace is not valid")
	}

	if c.scriptName == "" {
		return errors.New("scriptName cannot be empty")
	}

	if _, ok := c.configMap.Data[c.scriptName]; !ok {
		return errors.New("scriptName is not valid")
	}

	if c.scriptContent == "" {
		return errors.New("scriptContent cannot be empty")
	}

	if _ok := c.configMap.Data[c.scriptName]; _ok != c.scriptContent {
		return errors.New("scriptContent is not valid")
	}

	return nil
}

// ConfigMap returns the ConfigMap object
func (c *ConfigMapTemplate) ConfigMap() *corev1.ConfigMap {
	return c.configMap
}

// Name returns the name of the ConfigMap
func (c *ConfigMapTemplate) Name() string {
	return c.name
}

// Namespace Get configMap namespace
func (c *ConfigMapTemplate) Namespace() string {
	return c.namespace
}

func (c *ConfigMapTemplate) initConfigMap() *ConfigMapTemplate {
	// skip if configmap is already initialized
	if c.configMap != nil {
		return c
	}

	// set default
	if len(c.namespace) == 0 {
		c.namespace = cmNamespaceDefault
	}
	if len(c.generateName) == 0 {
		c.generateName = cmGenerateNameDefault
	}
	c.name = common.GenerateName2Name(c.generateName)

	// init configmap
	c.configMap = &corev1.ConfigMap{
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

func (c *ConfigMapTemplate) SetLabels(labels map[string]string) *ConfigMapTemplate {
	c.initConfigMap()

	c.configMap.SetLabels(labels)
	return c
}

func (c *ConfigMapTemplate) SetLabel(key string, value string) *ConfigMapTemplate {
	c.initConfigMap()

	if c.configMap.Labels == nil {
		c.configMap.Labels = map[string]string{}
	}
	c.configMap.Labels[key] = value
	return c
}
