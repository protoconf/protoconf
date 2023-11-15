package agent

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

var jsonConfig string

func init() {
	dir := os.TempDir()
	jsonConfig = filepath.Join(dir, "config.json")
	os.WriteFile(jsonConfig, []byte(`{}`), 0644)
}
func Test_cliCommand_Run(t *testing.T) {
	type args struct {
		args []string
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "cannot listen port 10",
			args: args{
				args: []string{
					"-dev", "/hello",
					"-grpc-address", ":10",
				},
			},
			want: 1,
		},
		// {
		// 	name: "run dev server",
		// 	args: args{
		// 		args: []string{
		// 			"-dev", "/hello",
		// 			"-grpc-address", ":0",
		// 			"-http-address", ":0",
		// 			"-log-as-json",
		// 		},
		// 	},
		// 	want: 1,
		// },
		{
			name: "run consul server",
			args: args{
				args: []string{
					"-store", "consul",
				},
			},
			want: 1,
		},
		{
			name: "run etcd server",
			args: args{
				args: []string{
					"-store", "etcd",
				},
			},
			want: 1,
		},
		{
			name: "run zookeeper server",
			args: args{
				args: []string{
					"-store", "zookeeper",
				},
			},
			want: 1,
		},
		{
			name: "run unknown server",
			args: args{
				args: []string{
					"-store", "file",
				},
			},
			want: 1,
		},
		{
			name: "help",
			args: args{
				args: []string{
					"-h",
				},
			},
			want: 2,
		},
		{
			name: "config-file not exists",
			args: args{
				args: []string{
					"-config-file", "config",
				},
			},
			want: 2,
		},
		{
			name: "config-file empty",
			args: args{
				args: []string{
					"-config-file", os.DevNull,
				},
			},
			want: 2,
		},
		{
			name: "config-file non empty",
			args: args{
				args: []string{
					"-config-file", jsonConfig,
				},
			},
			want: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := Command()
			time.AfterFunc(time.Second*1, func() {
				p, _ := os.FindProcess(os.Getegid())
				p.Signal(os.Interrupt)
				t.Log("sent interrupt")
			})
			if got := c.Run(tt.args.args); got != tt.want {
				t.Errorf("cliCommand.Run() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_cliCommand_Help(t *testing.T) {
	c, _ := Command()
	c.Help()
}
