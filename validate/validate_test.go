package validate_test

import (
	"testing"

	"github.com/linlanniao/k8sutils/validate"
	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {
	err := validate.Validate(nil)
	assert.NoError(t, err)
}
