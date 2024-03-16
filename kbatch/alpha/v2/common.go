package v2

import "fmt"

var (
	ErrNotSupported = fmt.Errorf("not supported")
)

const (
	K8sManagerSaName          = "k8sutils-k8s-manager-sa"
	K8sManagerRoleName        = "k8sutils-k8s-manager-cluster-role"
	K8sManagerRoleBindingName = "k8sutils-k8s-manager-cluster-role-binding"
)
