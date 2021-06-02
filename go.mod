module github.com/protoconf/protoconf

go 1.16

require (
	cloud.google.com/go v0.53.0 // indirect
	github.com/abronan/valkeyrie v0.1.0
	github.com/coreos/etcd v3.3.18+incompatible // indirect
	github.com/coreos/go-semver v0.3.0 // indirect
	github.com/fatih/structtag v1.2.0
	github.com/fsnotify/fsnotify v1.4.9
	github.com/ghodss/yaml v1.0.0
	github.com/gogo/protobuf v1.3.1 // indirect
	github.com/golang/mock v1.4.3 // indirect
	github.com/golang/protobuf v1.5.2
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/hashicorp/consul/api v1.8.1 // indirect
	github.com/hashicorp/go-getter v1.4.0
	github.com/hashicorp/go-plugin v1.4.1
	github.com/hashicorp/terraform v0.12.18
	github.com/jhump/protoreflect v1.8.2
	github.com/miekg/dns v1.1.29 // indirect
	github.com/mitchellh/cli v1.1.2
	github.com/mitchellh/go-homedir v1.1.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.6.0
	github.com/qri-io/starlib v0.5.0
	github.com/samuel/go-zookeeper v0.0.0-20190923202752-2cc03de413da // indirect
	github.com/stretchr/testify v1.7.0
	github.com/ulikunitz/xz v0.5.8 // indirect
	github.com/zclconf/go-cty v1.2.0
	go.etcd.io/etcd v3.3.18+incompatible // indirect
	go.starlark.net v0.0.0-20210511153848-cca21e7857d4
	go.uber.org/zap v1.15.0
	golang.org/x/exp v0.0.0-20200224162631-6cc2880d07d6 // indirect
	golang.org/x/lint v0.0.0-20200302205851-738671d3881b // indirect
	golang.org/x/net v0.0.0-20210525063256-abc453219eb5
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	golang.org/x/time v0.0.0-20191024005414-555d28b269f0 // indirect
	golang.org/x/tools v0.1.2
	google.golang.org/api v0.21.0 // indirect
	google.golang.org/grpc v1.38.0
	google.golang.org/protobuf v1.26.0
)

replace github.com/hashicorp/consul v0.0.0-20171026175957-610f3c86a089 => github.com/hashicorp/consul v1.8.1
