package lib

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/protoconf/protoconf/compiler/module/v1"
	"github.com/protoconf/protoconf/utils/testdata"
)

func TestParseModulePath(t *testing.T) {
	type args struct {
		moduleName string
	}
	tests := []struct {
		name string
		args args
		want *ModulePath
	}{
		{
			name: "starlib",
			args: args{
				moduleName: "re.star",
			},
			want: &ModulePath{
				Filepath: "re.star",
				Ext:      ".star",
			},
		},
		{
			name: "@terraform util.pinc",
			args: args{
				moduleName: "@terraform_repo//terraform/v1/util.pinc",
			},
			want: &ModulePath{
				Repo:     "terraform_repo",
				Filepath: "//terraform/v1/util.pinc",
				Ext:      ".pinc",
			},
		},
		{
			name: "local util.pinc",
			args: args{
				moduleName: "//terraform/v1/util.pinc",
			},
			want: &ModulePath{
				Filepath: "//terraform/v1/util.pinc",
				Ext:      ".pinc",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseModulePath(tt.args.moduleName); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseModulePath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestModuleService_Sync(t *testing.T) {
	tests := []struct {
		name    string
		head    *module.RemoteRepo
		wantErr error
	}{
		{
			name: "empty",
			head: &module.RemoteRepo{},
		},
		{
			name: "no integrity",
			head: &module.RemoteRepo{
				Deps: map[string]*module.RemoteRepo{
					"terraform_repo": {
						Url:       "github.com/protoconf/protoconf-terraform",
						Pin:       &module.RemoteRepo_Tag{Tag: "v0.1.4"},
						GetterUrl: "git::https://github.com/protoconf/protoconf-terraform.git?ref=v0.1.4",
					},
				},
			},
		},
		{
			name:    "bad integrity",
			wantErr: ErrorRemoteRepoValidationFailed,
			head: &module.RemoteRepo{
				Deps: map[string]*module.RemoteRepo{
					"terraform_repo": {
						Url:       "github.com/protoconf/protoconf-terraform",
						Pin:       &module.RemoteRepo_Tag{Tag: "v0.1.4"},
						GetterUrl: "git::https://github.com/protoconf/protoconf-terraform.git?ref=v0.1.4",
						Integrity: "hello world",
					},
				},
			},
		},
		{
			name: "good integrity",
			head: &module.RemoteRepo{
				Deps: map[string]*module.RemoteRepo{
					"terraform_repo": {
						Url:       "github.com/protoconf/protoconf-terraform",
						Pin:       &module.RemoteRepo_Tag{Tag: "v0.1.4"},
						GetterUrl: "git::https://github.com/protoconf/protoconf-terraform.git?ref=v0.1.4",
						Integrity: "h1:P1NmV8U9zKkUSnj8Z8TTP95AN8jnHMk/1hnEeEjevg4=",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModuleService(testdata.SmallTestDir())
			m.head = tt.head
			if err := m.Sync(context.Background()); !errors.Is(err, tt.wantErr) {
				t.Errorf("ModuleService.Sync() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
