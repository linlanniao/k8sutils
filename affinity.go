package k8sutils

import (
	"errors"
	"fmt"
	"regexp"

	corev1 "k8s.io/api/core/v1"
)

type NodeAffinity struct {
	Key      string               `json:"key"`
	Operator NodeAffinityOperator `json:"operator"`
	Values   []string             `json:"values"`
}

type NodeAffinityOperator string

func (n NodeAffinityOperator) String() string {
	return string(n)
}

const (
	NodeAffinityOpIn           NodeAffinityOperator = "In"
	NodeAffinityOpNotIn        NodeAffinityOperator = "NotIn"
	NodeAffinityOpExists       NodeAffinityOperator = "Exists"
	NodeAffinityOpDoesNotExist NodeAffinityOperator = "DoesNotExist"
	NodeAffinityOpGt           NodeAffinityOperator = "Gt"
	NodeAffinityOpLt           NodeAffinityOperator = "Lt"
)

func (n *NodeAffinity) Validate() error {
	// validate key
	if len(n.Key) == 0 || len(n.Key) >= 63 {
		return errors.New("invalid key, length must be between 1 and 63")
	}

	keyPattern := `([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]`
	if !regexp.MustCompile(keyPattern).MatchString(n.Key) {
		return fmt.Errorf("key is invalid, pattern: %s", keyPattern)
	}

	// validate operator
	switch n.Operator {
	case NodeAffinityOpIn, NodeAffinityOpNotIn, NodeAffinityOpExists, NodeAffinityOpDoesNotExist, NodeAffinityOpGt, NodeAffinityOpLt:
		// pass
	default:
		return fmt.Errorf("invalid operator: %s", n.Operator)
	}

	// validate values
	if len(n.Values) > 0 {
		for _, v := range n.Values {
			if v == "" {
				return fmt.Errorf("value is empty")
			}
			if len(v) > 63 {
				return fmt.Errorf("value is too long")
			}
			valuePattern := `(([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9])?`
			if !regexp.MustCompile(valuePattern).MatchString(v) {
				return fmt.Errorf("value is invalid, valuePattern: %s", valuePattern)
			}
		}
	}

	return nil
}

type NodeAffinities []*NodeAffinity

func (n NodeAffinities) Validate() error {
	if len(n) == 0 {
		return nil
	}
	for _, x := range n {
		x := x
		if err := x.Validate(); err != nil {
			return err
		}
	}
	return nil
}
func (n NodeAffinities) IsEmpty() bool {
	return len(n) == 0
}

// newAffinity
//
//	@Description: create affinity
//	@param bizNodeAffinities
//	@param matchAll, if true, all node affinities must be satisfied
//	@return *corev1.Affinity
//	@return error
func newAffinity(bizNodeAffinities NodeAffinities, matchAll bool) (*corev1.Affinity, error) {
	if bizNodeAffinities.IsEmpty() {
		return nil, fmt.Errorf("node affinities is empty")
	}

	if err := bizNodeAffinities.Validate(); err != nil {
		return nil, err
	}

	affinity := &corev1.Affinity{
		NodeAffinity: &corev1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
				NodeSelectorTerms: nil,
			},
		},
	}

	var terms []corev1.NodeSelectorTerm

	if matchAll {
		// match all
		terms = make([]corev1.NodeSelectorTerm, 1)
		requirements := make([]corev1.NodeSelectorRequirement, len(bizNodeAffinities))
		for i, entry := range bizNodeAffinities {
			entry := entry
			r := corev1.NodeSelectorRequirement{
				Key:      entry.Key,
				Operator: corev1.NodeSelectorOperator(entry.Operator),
				Values:   entry.Values,
			}
			requirements[i] = r
		}
		terms[0] = corev1.NodeSelectorTerm{
			MatchExpressions: requirements,
		}
	} else {
		// match any
		terms = make([]corev1.NodeSelectorTerm, len(bizNodeAffinities))
		for i, entry := range bizNodeAffinities {
			entry := entry
			r := corev1.NodeSelectorRequirement{
				Key:      entry.Key,
				Operator: corev1.NodeSelectorOperator(entry.Operator),
				Values:   entry.Values,
			}
			terms[i] = corev1.NodeSelectorTerm{
				MatchExpressions: []corev1.NodeSelectorRequirement{r},
			}
		}
	}

	affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms = terms
	return affinity, nil
}
