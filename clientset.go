package k8sutils

import (
	"fmt"
	"os"
	"sync"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

type Clientset struct {
	clientset *kubernetes.Clientset
}

var (
	cli              *Clientset
	initClientset    sync.Once
	serverVersion    *version.Info
	currentNamespace string
)

func NewClientSet() (*kubernetes.Clientset, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		var configPath string
		if p := os.Getenv(clientcmd.RecommendedConfigPathEnvVar); len(p) > 0 {
			configPath = p
		} else {
			configPath = clientcmd.RecommendedHomeFile
		}
		config, err = clientcmd.BuildConfigFromFlags("", configPath)
	}

	if err != nil {
		err = fmt.Errorf("error building kubeconfig: %w", err)
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}

func GetClientset() *Clientset {
	initClientset.Do(func() {
		var (
			err       error
			clientset *kubernetes.Clientset
		)
		cli = &Clientset{}

		defer func() {
			if err != nil {
				klog.Error(err.Error())
				panic(err.Error())
			}
		}()

		clientset, err = NewClientSet()
		if err != nil {
			err = fmt.Errorf("error creating Kubernetes client: %w", err)
			return
		}

		cli.clientset = clientset

		serverVersion, err = cli.clientset.Discovery().ServerVersion()
		if err != nil {
			err = fmt.Errorf("error getting server version: %w", err)
			return
		}

		// try to read current namespace
		if b, err2 := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err2 != nil {
			currentNamespace = corev1.NamespaceDefault // if error , set currentNamespace -> default
		} else {
			currentNamespace = string(b)
		}
	})

	return cli
}

func (c *Clientset) GetServerVersion() (string, error) {
	if serverVersion != nil {
		v := serverVersion
		_ = v
		return serverVersion.String(), nil
	}

	v, err := c.clientset.Discovery().ServerVersion()
	if err != nil {
		return "", err
	}
	serverVersion = v
	return serverVersion.String(), nil
}

func (c *Clientset) GetNamespace() string {
	return currentNamespace
}

func (c *Clientset) GetClientSet() *kubernetes.Clientset {
	return c.clientset
}
