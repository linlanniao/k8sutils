package v2

import (
	"errors"
)

var (
	ErrNotSupported      = errors.New("not supported")
	ErrKeyNotFound       = errors.New("key not found")
	ErrValueTypeMismatch = errors.New("value type mismatch")
)

const (
	K8sManagerSa                 = "k8sutils-k8s-manager-sa"
	K8sManagerClusterRole        = "k8sutils-k8s-manager-cluster-role"
	K8sManagerClusterRoleBinding = "k8sutils-k8s-manager-cluster-role-binding"
)
