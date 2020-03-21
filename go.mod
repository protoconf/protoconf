module github.com/protoconf/protoconf

go 1.13

require (
	github.com/census-instrumentation/opencensus-proto v0.2.1 // indirect
	github.com/coreos/etcd v3.3.18+incompatible // indirect
	github.com/docker/libkv v0.2.1 // indirect
	github.com/envoyproxy/protoc-gen-validate v0.1.0 // indirect
	github.com/etcd-io/etcd v3.3.18+incompatible // indirect
	github.com/fsnotify/fsnotify v1.4.7
	github.com/golang/protobuf v1.3.2
	github.com/golangci/golangci-lint v1.18.0 // indirect
	github.com/hashicorp/consul v1.6.2 // indirect
	github.com/hashicorp/consul/api v1.3.0
	github.com/hashicorp/go-getter v1.4.0
	github.com/hashicorp/go-plugin v1.0.1
	github.com/hashicorp/terraform v0.12.18
	github.com/jhump/protoreflect v1.4.5-0.20190719031800-6c4c7792338e
	github.com/mitchellh/cli v1.0.0
	github.com/olekukonko/tablewriter v0.0.4 // indirect
	github.com/prometheus/client_model v0.0.0-20190812154241-14fe0d1b01d4 // indirect
	github.com/qri-io/starlib v0.4.1
	github.com/samuel/go-zookeeper v0.0.0-20190923202752-2cc03de413da
	github.com/stretchr/testify v1.4.0
	github.com/zclconf/go-cty v1.2.0
	go.etcd.io/bbolt v1.3.3
	go.etcd.io/etcd v3.3.18+incompatible
	go.starlark.net v0.0.0-20191218235703-9fcb808a6221
	go.uber.org/zap v1.9.1
	golang.org/x/net v0.0.0-20191209160850-c0dbc17a3553
	google.golang.org/genproto v0.0.0-20191223191004-3caeed10a8bf // indirect
	google.golang.org/grpc v1.23.0
	sigs.k8s.io/yaml v1.1.0 // indirect
)

replace (
	github.com/go-critic/go-critic v0.0.0-20181204210945-c3db6069acc5 => github.com/go-critic/go-critic v0.0.0-20190422201921-c3db6069acc5
	github.com/go-critic/go-critic v0.0.0-20181204210945-ee9bf5809ead => github.com/go-critic/go-critic v0.0.0-20190210220443-ee9bf5809ead
	github.com/golangci/errcheck v0.0.0-20181003203344-ef45e06d44b6 => github.com/golangci/errcheck v0.0.0-20181223084120-ef45e06d44b6
	github.com/golangci/go-tools v0.0.0-20180109140146-af6baa5dc196 => github.com/golangci/go-tools v0.0.0-20190318060251-af6baa5dc196
	github.com/golangci/gofmt v0.0.0-20181105071733-0b8337e80d98 => github.com/golangci/gofmt v0.0.0-20181222123516-0b8337e80d98
	github.com/golangci/gosec v0.0.0-20180901114220-66fb7fc33547 => github.com/golangci/gosec v0.0.0-20190211064107-66fb7fc33547
	github.com/golangci/lint-1 v0.0.0-20180610141402-ee948d087217 => github.com/golangci/lint-1 v0.0.0-20190420132249-ee948d087217
	google.golang.org/genproto v0.0.0-20170818100345-ee236bd376b0 => google.golang.org/genproto v0.0.0-20170818010345-ee236bd376b0
	mvdan.cc/unparam v0.0.0-20190124213536-fbb59629db34 => mvdan.cc/unparam v0.0.0-20190209190245-fbb59629db34
)
