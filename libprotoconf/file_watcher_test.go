package libprotoconf

import (
	"testing"
	"time"

	"github.com/protoconf/protoconf/utils/testdata"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

func Test_fileWatcher_Watch(t *testing.T) {
	type fields struct {
		protoconfRoot string
	}
	type args struct {
		path string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    Result
		wantErr bool
	}{
		// {
		// 	name: "empty",
		// 	fields: fields{
		// 		protoconfRoot: testdata.SmallTestDir(),
		// 	},
		// 	args:    args{path: "not_found"},
		// 	wantErr: true,
		// },
		{
			name: "test",
			fields: fields{
				protoconfRoot: testdata.LargeTestDir(),
			},
			args: args{
				path: "example/instance_0",
			},
			want: Result{
				Value: &anypb.Any{
					TypeUrl: "type.googleapis.com/terraform.v1.Terraform",
					Value:   []byte("\nM\xa2%J\n\ninstance_0\x12<\n\x0cami-830c94e3\x9a\x01\x08t2.micro\xb2\x02 \n\x04Name\x12\x18ExampleAppServerInstance"),
				},
			},
			wantErr: false,
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, _ := NewFileWatcher(tt.fields.protoconfRoot)
			stopCh := make(chan struct{})
			got, err := w.Watch(tt.args.path, stopCh)
			if (err != nil) != tt.wantErr {
				t.Errorf("fileWatcher.Watch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			go func() {
				// time.Sleep(500 * time.Millisecond)
				// err := os.WriteFile(
				// 	filepath.Join(
				// 		tt.fields.protoconfRoot,
				// 		consts.CompiledConfigPath,
				// 		tt.args.path+consts.CompiledConfigExtension),
				// 	[]byte(protojson.Format(&v1.ProtoconfValue{
				// 		Value: tt.want.Value,
				// 	})), 755)
				// if err != nil {
				// 	t.Log(err)
				// }
				time.Sleep(1 * time.Second)
				stopCh <- struct{}{}
			}()
			result := <-got
			if !proto.Equal(result.Value, tt.want.Value) {
				t.Errorf("fileWatcher.Watch() = %v, want %v", result, tt.want)
			}
		})
	}
}
