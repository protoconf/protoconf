package lib

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/hashicorp/go-getter"
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
	testDir := testdata.SmallTestDir()
	tests := []struct {
		name    string
		head    *module.RemoteRepo
		wantErr error
	}{
		{
			name: "empty",
			head: &module.RemoteRepo{
				Url: ".",
			},
		},
		{
			name: "no integrity",
			head: &module.RemoteRepo{
				Url: ".",
				Deps: map[string]*module.RemoteRepo{
					"vizceral_repo": {
						Url: filepath.Join(testDir, "internal/vizceral.tgz"),
						Pin: &module.RemoteRepo_Checksum{Checksum: "896b13d56bd1787089ca5767656c7ef1"},
					},
				},
			},
		},
		{
			name:    "bad integrity",
			wantErr: ErrorRemoteRepoValidationFailed,
			head: &module.RemoteRepo{
				Url: ".",
				Deps: map[string]*module.RemoteRepo{
					"vizceral_repo": {
						Url:                  filepath.Join(testDir, "internal/vizceral.tgz"),
						Pin:                  &module.RemoteRepo_Checksum{Checksum: "896b13d56bd1787089ca5767656c7ef1"},
						FileDescriptorSetSum: "98173254590e7bfb78555e03033e756e",
						Integrity:            "hello world",
					},
				},
			},
		},
		{
			name: "good integrity",
			head: &module.RemoteRepo{
				Url: ".",
				Deps: map[string]*module.RemoteRepo{
					"vizceral_repo": {
						Url:       filepath.Join(testDir, "internal/vizceral.tgz"),
						Pin:       &module.RemoteRepo_Checksum{Checksum: "896b13d56bd1787089ca5767656c7ef1"},
						Integrity: "h1:mKU/VAicQpQB3uVzxxAlTZsKPewktnNXQvwmBXI5W9o=",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{})).With("test", t.Name())
			slog.SetDefault(logger)
			m := NewModuleService(testdata.SmallTestDir())
			m.head = tt.head
			m.Walk(func(r *module.RemoteRepo) error {
				if r.Url == "." {
					return nil
				}
				r.GetterUrl, _ = getter.Detect(r.Url, testDir, getter.Detectors)
				r.Label = repoLabel(r)
				return nil
			})
			if err := m.Sync(context.Background()); !errors.Is(err, tt.wantErr) {
				t.Errorf("ModuleService.Sync() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
