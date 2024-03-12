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

// configMapTemplate creates a ConfigMap with a single key-value pair
type configMapTemplate struct {
	configMap     *corev1.ConfigMap
	name          string
	generateName  string
	namespace     string
	scriptName    string
	scriptContent string
}

// NewConfigMapTemplate creates a new configMapTemplate instance
func NewConfigMapTemplate(generateName, namespace, scriptName, scriptContent string) *configMapTemplate {

	if !strings.HasSuffix(generateName, "-") {
		generateName = generateName + "-"
	}

	c := &configMapTemplate{
		generateName:  generateName,
		namespace:     namespace,
		scriptName:    scriptName,
		scriptContent: scriptContent,
	}

	c.initConfigMap()

	return c
}

// Validate checks the validity of the configMapTemplate instance
func (c *configMapTemplate) Validate() error {
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

// ConfigMap returns the ConfigMap instance
func (c *configMapTemplate) ConfigMap() *corev1.ConfigMap {
	return c.configMap
}

// Name returns the name of the ConfigMap
func (c *configMapTemplate) Name() string {
	return c.name
}

// Namespace returns the namespace of the ConfigMap
func (c *configMapTemplate) Namespace() string {
	return c.namespace
}

// initConfigMap initializes the ConfigMap instance if it is not already initialized.
// It sets default values for the generateName and namespace fields, and generates a unique name for the ConfigMap.
// It also initializes the Data field with the script content.
func (c *configMapTemplate) initConfigMap() *configMapTemplate {
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

// SetLabels sets the labels of the configMapTemplate.
//
// This function initializes the ConfigMap instance if it is not already initialized.
// It then sets the labels on the ConfigMap instance and returns the configMapTemplate instance.
func (c *configMapTemplate) SetLabels(labels map[string]string) *configMapTemplate {
	c.initConfigMap()

	c.configMap.SetLabels(labels)
	return c
}

// SetLabel sets the value of a label on the configMapTemplate.
//
// This function initializes the ConfigMap instance if it is not already initialized.
// It then sets the label on the ConfigMap instance and returns the configMapTemplate instance.
func (c *configMapTemplate) SetLabel(key string, value string) *configMapTemplate {
	c.initConfigMap()

	if c.configMap.Labels == nil {
		c.configMap.Labels = map[string]string{}
	}
	c.configMap.Labels[key] = value
	return c
}
