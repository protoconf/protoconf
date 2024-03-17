package inserter

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/kvtools/valkeyrie/store"
	"github.com/mitchellh/cli"
	"github.com/protoconf/protoconf/agent/dummykv"
	"github.com/protoconf/protoconf/utils/testdata"
	"github.com/stretchr/testify/assert"
)

func TestProtoconfInserter_InsertConfig(t *testing.T) {
	kvStore, _ := dummykv.New(context.Background(), []string{}, &dummykv.Config{})
	// kvStore, err := configmaps.New(context.Background(), []string{}, &configmaps.Config{Namespace: "default"})
	// require.NoError(t, err)
	testDir := testdata.SmallTestDir()

	i := NewProtoconfInserter(testDir, kvStore)
	type args struct {
		configFile string
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
		want    map[string]string
	}{
		{
			name:    "test",
			args:    args{configFile: "test.materialized_JSON"},
			wantErr: nil,
			want: map[string]string{
				"test/STABLE/config.data": `Cgp0ZXN0LnByb3RvEiwKH3R5cGUuZ29vZ2xlYXBpcy5jb20vVGVzdE1lc3NhZ2USCQoHSW0gaGVyZQ==`,
			},
		},
		{
			name:    "with_rollout_config",
			args:    args{configFile: "with_config_rollout.materialized_JSON"},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := i.InsertConfig(tt.args.configFile); !errors.Is(tt.wantErr, err) {
				t.Errorf("ProtoconfInserter.InsertConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
			for key, value := range tt.want {
				v, err := kvStore.Get(context.Background(), key, &store.ReadOptions{})
				assert.NoError(t, err)
				if string(v.Value) != value {
					t.Errorf("expected key %s to be: %v, got %s", key, value, v.Value)
				}
			}
		})
	}
}

func Test_cliCommand_Run(t *testing.T) {
	testDir := testdata.SmallTestDir()
	type args struct {
		args []string
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "test",
			args: args{args: []string{testDir, "test.materialized_JSON"}},
			want: 0,
		},
		{
			name: "no args",
			args: args{args: []string{}},
			want: 1,
		},
		{
			name: "etcd",
			args: args{args: []string{"-store", "etcd", testDir, "test.materialized_JSON"}},
			want: 0,
		},
		{
			name: "zookeeper",
			args: args{args: []string{"-store", "zookeeper", testDir, "test.materialized_JSON"}},
			want: 0,
		},
		{
			name: "delete",
			args: args{args: []string{"-d", testDir, "test.materialized_JSON"}},
			want: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &cliCommand{}
			if got := c.Run(tt.args.args); got != tt.want {
				t.Errorf("cliCommand.Run() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_cliCommand_Help(t *testing.T) {
	tests := []struct {
		name string
		want []string
	}{
		{
			name: "test",
			want: []string{
				`Insert a materialized config to the key-value store`,
				`Usage: [OPTION]... [protoconf_root] config...`,
				`-d`, `Delete a config from the key-value store`,
				`-prefix`,
				`Key-value store key prefix`,
				`-store`,
				`Key-value store type (consul/zookeeper/etcd) (default "consul")`,
				`-store-address`,
				`Key-value store address`,
			},
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &cliCommand{}
			got := c.Help()
			for _, w := range tt.want {
				if !strings.Contains(got, w) {
					t.Errorf("cliCommand.Help() = %v, want %v", got, w)
				}
			}

		})
	}
}

func TestCommand(t *testing.T) {
	tests := []struct {
		name    string
		want    cli.Command
		wantErr bool
	}{
		{name: "test", want: &cliCommand{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Command()
			if (err != nil) != tt.wantErr {
				t.Errorf("Command() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Command() = %v, want %v", got, tt.want)
			}
		})
	}
}
