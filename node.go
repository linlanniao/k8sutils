package k8sutils

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
)

func NewLabelSelector(nodeSelector corev1.NodeSelector) (labels.Selector, error) {
	labelSelector := labels.NewSelector()
	for _, term := range nodeSelector.NodeSelectorTerms {
		matchExpressions := term.MatchExpressions
		var op selection.Operator
		for _, nodeSelectorRequirement := range matchExpressions {
			switch nodeSelectorRequirement.Operator {
			case corev1.NodeSelectorOpIn:
				op = selection.In
			case corev1.NodeSelectorOpNotIn:
				op = selection.NotIn
			case corev1.NodeSelectorOpExists:
				op = selection.Exists
			case corev1.NodeSelectorOpDoesNotExist:
				op = selection.DoesNotExist
			case corev1.NodeSelectorOpGt:
				op = selection.GreaterThan
			case corev1.NodeSelectorOpLt:
				op = selection.LessThan
			default:
				return nil, fmt.Errorf("invalid operator: %s", nodeSelectorRequirement.Operator)
			}
			r, _ := labels.NewRequirement(
				nodeSelectorRequirement.Key,
				op,
				nodeSelectorRequirement.Values)
			labelSelector = labelSelector.Add(*r)
		}
	}
	return labelSelector, nil
}

// GetNodes
//
//	@Description: 获取节点, nodeSelector 与 selectedIps 是交集关系
//	@receiver c
//	@param ctx
//	@param nodeSelector 节点选择器
//	@param selectedIps 选择的ip
//	@return *corev1.NodeList
//	@return error
func (c *Clientset) GetNodes(ctx context.Context, nodeSelector *corev1.NodeSelector, selectedIps ...string) (*corev1.NodeList, error) {
	listOptions := metav1.ListOptions{}
	if nodeSelector != nil {
		labelSelector, err := NewLabelSelector(*nodeSelector)
		if err != nil {
			return nil, err
		}
		listOptions.LabelSelector = labelSelector.String()
	}

	lst, err := c.clientset.CoreV1().Nodes().List(ctx, listOptions)
	if err != nil {
		return nil, err
	}

	// 如果没有指定ip，则返回所有节点
	if len(selectedIps) == 0 {
		return lst, nil
	}
	ips := make(map[string]struct{})
	for _, ip := range selectedIps {
		ips[ip] = struct{}{}
	}
	item := make([]corev1.Node, 0)
	for _, node := range lst.Items {
		for _, addr := range node.Status.Addresses {
			if addr.Type == corev1.NodeInternalIP {
				if _, ok := ips[addr.Address]; ok {
					item = append(item, node)
					break
				}
			}
		}
	}
	lst.Items = item
	return lst, nil
}

// GetNodeIpToNameMapping
//
//	@Description: 获取节点ip与节点名称的映射, key: Ip, value: HostName
//	@receiver c
//	@param ctx
//	@param nodeSelector
//	@param selectedIps
//	@return map[string]string
func (c *Clientset) GetNodeIpToNameMapping(
	ctx context.Context, nodeSelector *corev1.NodeSelector, selectedIps ...string) map[string]string {
	lst, err := c.GetNodes(ctx, nodeSelector, selectedIps...)
	if err != nil {
		return nil
	}

	m := make(map[string]string)
	for _, node := range lst.Items {
		for _, addr := range node.Status.Addresses {
			if addr.Type == corev1.NodeInternalIP {
				m[addr.Address] = node.GetName()
				break
			}
		}
	}
	return m
}

func (c *Clientset) GetNodeName(ctx context.Context, ip string) (string, error) {
	lst, err := c.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to list nodes, err: %w", err)
	}
	if len(lst.Items) == 0 {
		return "", fmt.Errorf("no node found")
	}

	for _, node := range lst.Items {
		for _, addr := range node.Status.Addresses {
			if addr.Type == corev1.NodeInternalIP {
				if addr.Address == ip {
					return node.GetName(), nil
				}
			}
		}
	}
	return "", fmt.Errorf("no node found, ip: %s", ip)
}

func (c *Clientset) GetNodeIps(ctx context.Context, nodeSelector *corev1.NodeSelector) ([]string, error) {
	lst, err := c.GetNodes(ctx, nodeSelector)
	if err != nil {
		return nil, err
	}

	ips := make([]string, 0, len(lst.Items))
	for _, node := range lst.Items {
		for _, addr := range node.Status.Addresses {
			if addr.Type == corev1.NodeInternalIP {
				ips = append(ips, addr.Address)
				break
			}
		}
	}
	return ips, nil
}

func (c *Clientset) GetAllNodeIps(ctx context.Context) ([]string, error) {
	return c.GetNodeIps(ctx, nil)
}
