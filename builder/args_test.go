package builder

import (
	"reflect"
	"testing"
)

func TestGetBuilderArgs(t *testing.T) {
	type args struct {
		args []string
	}
	tests := []struct {
		name    string
		args    args
		wantBa  *Args
		wantErr bool
	}{
		{"single", args{
			[]string{"testdata/1.yaml", "testdata", "testdata"},
		}, &Args{
			DataFiles:     []string{"testdata/1.yaml"},
			SourceFolders: []string{"testdata"},
			TargetFolder:  "testdata",
		}, false},
		{"multiple", args{
			[]string{"testdata/1.yaml", "testdata/2.yaml", "testdata", "testdata"},
		}, &Args{
			DataFiles:     []string{"testdata/1.yaml", "testdata/2.yaml"},
			SourceFolders: []string{"testdata"},
			TargetFolder:  "testdata",
		}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotBa, err := GetBuilderArgs(tt.args.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetBuilderArgs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotBa, tt.wantBa) {
				t.Errorf("GetBuilderArgs() gotBa = %v, want %v", gotBa, tt.wantBa)
			}
		})
	}
}
