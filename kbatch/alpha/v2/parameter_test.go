package v2_test

import (
	"testing"

	v2 "github.com/linlanniao/k8sutils/kbatch/alpha/v2"
	"github.com/stretchr/testify/assert"
)

func TestArgs2Parameters(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    v2.Parameters
		wantErr bool
	}{
		{
			name:    "empty args",
			args:    []string{},
			wantErr: true,
		},
		{
			name: "one arg",
			args: []string{"/path/to/script.sh", "--key1", "value1"},
			want: v2.Parameters{
				{
					Key:   "--key1",
					Value: "value1",
				},
			},
		},
		{
			name: "even number of args",
			args: []string{"/path/to/script.sh", "--key1", "value1", "--key2", "value2"},
			want: v2.Parameters{
				{
					Key:   "--key1",
					Value: "value1",
				},
				{
					Key:   "--key2",
					Value: "value2",
				},
			},
		},
		{
			name:    "uneven number of args",
			args:    []string{"/path/to/script.sh", "key1", "value1", "key2"},
			wantErr: true,
		},
		{
			name:    "invalid arg",
			args:    []string{"/path/to/script.sh", "key1", "value1", "key2", "value2", "invalid"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := v2.Args2Parameters(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Args2Parameters() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !assert.Equal(t, tt.want, got) {
				t.Errorf("Args2Parameters() = %v, want %v", got, tt.want)
			}
		})
	}
}
