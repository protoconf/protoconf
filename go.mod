module github.com/protoconf/protoconf

go 1.16

require (
	github.com/Masterminds/sprig/v3 v3.2.2
	github.com/fatih/structtag v1.2.0
	github.com/fsnotify/fsnotify v1.6.0
	github.com/ghodss/yaml v1.0.0
	github.com/golang/protobuf v1.5.2
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/hashicorp/go-getter v1.6.2
	github.com/hashicorp/go-plugin v1.4.6
	github.com/hashicorp/terraform v0.12.18
	github.com/jhump/protoreflect v1.12.0
	github.com/kvtools/valkeyrie v0.4.1
	github.com/mitchellh/cli v1.1.2
	github.com/mitchellh/go-homedir v1.1.0
	github.com/pelletier/go-toml v1.2.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.12.1
	github.com/qri-io/starlib v0.5.0
	github.com/sajari/fuzzy v1.0.0
	github.com/stretchr/testify v1.8.1
	github.com/zclconf/go-cty v1.10.0
	go.starlark.net v0.0.0-20220328144851-d1966c6b9fcd
	go.uber.org/zap v1.21.0
	golang.org/x/net v0.0.0-20220624214902-1bab6f366d9e
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	golang.org/x/tools v0.1.11
	google.golang.org/grpc v1.45.0
	google.golang.org/protobuf v1.28.0
)

replace github.com/hashicorp/consul v0.0.0-20171026175957-610f3c86a089 => github.com/hashicorp/consul v1.8.1
