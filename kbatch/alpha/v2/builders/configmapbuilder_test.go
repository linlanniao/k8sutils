package builders

import (
	"reflect"
	"testing"
)

func TestConfigMapTemplate_SetLabels(t *testing.T) {
	tests := []struct {
		name   string
		labels map[string]string
	}{
		{name: "case1", labels: map[string]string{"foo": "bar"}},
		{name: "case2", labels: map[string]string{"a": "aa"}},
		{name: "case3", labels: map[string]string{"b": "bb"}},
		{name: "case4", labels: map[string]string{"c": "cc"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := ConfigMapBuilder("", "", "", "")
			if got := c.SetLabels(tt.labels).ConfigMap().Labels; !reflect.DeepEqual(got, tt.labels) {
				t.Errorf("configMapBuilder.SetLabels() = %v, want %v", got, tt.labels)
			}
			t.Log(c.ConfigMap().String())
		})
	}

}
