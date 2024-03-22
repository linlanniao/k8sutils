package v2_test

import (
	"testing"

	v2 "github.com/linlanniao/k8sutils/kbatch/alpha/v2"
)

func TestRemoveSuffix(t *testing.T) {
	type args struct {
		input string
		sep   string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "case1",
			args: args{
				input: "hello/world",
				sep:   "/",
			},
			want:    "hello",
			wantErr: false,
		},
		{
			name: "case2",
			args: args{
				input: "hello",
				sep:   "/",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "case3",
			args: args{
				input: "default/test-pytask-o8nlsv-7i5848",
				sep:   "-",
			},
			want:    "default/test-pytask-o8nlsv",
			wantErr: false,
		},
		{
			name: "case4",
			args: args{
				input: "default/test7i5848",
				sep:   "-",
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := v2.RemoveSuffix(tt.args.input, tt.args.sep)
			if (err != nil) != tt.wantErr {
				t.Errorf("removeSuffix() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("removeSuffix() = %v, want %v", got, tt.want)
			}
		})
	}
}
