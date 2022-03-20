module github.com/protoconf/protoconf

go 1.16

require (
	github.com/fatih/structtag v1.2.0
	github.com/fsnotify/fsnotify v1.5.1
	github.com/ghodss/yaml v1.0.0
	github.com/golang/protobuf v1.5.2
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/hashicorp/go-getter v1.5.11
	github.com/hashicorp/go-plugin v1.4.3
	github.com/hashicorp/terraform v0.12.18
	github.com/jhump/protoreflect v1.10.1
	github.com/kvtools/valkeyrie v0.4.0
	github.com/mitchellh/cli v1.1.2
	github.com/mitchellh/go-homedir v1.1.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.12.1
	github.com/qri-io/starlib v0.5.0
	github.com/sajari/fuzzy v1.0.0
	github.com/stretchr/testify v1.7.0
	github.com/zclconf/go-cty v1.10.0
	go.starlark.net v0.0.0-20210602144842-1cdb82c9e17a
	go.uber.org/zap v1.20.0
	golang.org/x/net v0.0.0-20220225172249-27dd8689420f
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	golang.org/x/tools v0.1.8
	google.golang.org/grpc v1.45.0
	google.golang.org/protobuf v1.27.1
)

replace github.com/hashicorp/consul v0.0.0-20171026175957-610f3c86a089 => github.com/hashicorp/consul v1.8.1
