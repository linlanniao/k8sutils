package v2_test

import (
	"testing"

	v2 "github.com/linlanniao/k8sutils/kbatch/alpha/v2"
	"github.com/stretchr/testify/assert"
)

func TestScript_GenerateConfigMap(t *testing.T) {
	s := v2.NewScript(
		"test-script",
		"default",
		`#!/bin/bash
echo "hello world"`,
		v2.ScriptExecutorBash,
	)
	cm, err := s.GenerateConfigMap()
	assert.NoError(t, err)
	assert.NotNil(t, cm)
	t.Log(s.Status.Configmap.String())
}
