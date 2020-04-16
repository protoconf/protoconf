module github.com/protoconf/protoconf

go 1.13

require (
	cloud.google.com/go/firestore v1.2.0 // indirect
	cloud.google.com/go/spanner v1.5.0 // indirect
	github.com/anaskhan96/soup v1.1.1 // indirect
	github.com/apex/log v1.1.2 // indirect
	github.com/bazelbuild/bazel-gazelle v0.20.0
	github.com/coreos/etcd v3.3.18+incompatible // indirect
	github.com/docker/libkv v0.2.1 // indirect
	github.com/ernesto-jimenez/gogen v0.0.0-20180125220232-d7d4131e6607 // indirect
	github.com/etcd-io/etcd v3.3.18+incompatible // indirect
	github.com/fatih/structtag v1.2.0
	github.com/fsnotify/fsnotify v1.4.7
	github.com/ghodss/yaml v1.0.0
	github.com/go-sql-driver/mysql v1.5.0 // indirect
	github.com/gogo/protobuf v1.2.1
	github.com/golang/protobuf v1.3.5
	github.com/golangci/golangci-lint v1.18.0 // indirect
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/hashicorp/consul v1.6.2 // indirect
	github.com/hashicorp/consul/api v1.3.0
	github.com/hashicorp/go-getter v1.4.0
	github.com/hashicorp/go-plugin v1.0.1
	github.com/hashicorp/terraform v0.12.18
	github.com/jhump/protoreflect v1.4.5-0.20190719031800-6c4c7792338e
	github.com/mitchellh/cli v1.0.0
	github.com/mitchellh/go-homedir v1.1.0
	github.com/neverlee/keymutex v0.0.0-20171121013845-f593aa834bf9
	github.com/olekukonko/tablewriter v0.0.4 // indirect
	github.com/pkg/errors v0.8.1
	github.com/pkg/sftp v1.11.0 // indirect
	github.com/prometheus/client_golang v0.9.3
	github.com/qri-io/starlib v0.4.1
	github.com/rs/zerolog v1.18.0 // indirect
	github.com/samuel/go-zookeeper v0.0.0-20190923202752-2cc03de413da
	github.com/stretchr/testify v1.4.0
	github.com/zclconf/go-cty v1.2.0
	go.etcd.io/bbolt v1.3.3
	go.etcd.io/etcd v3.3.18+incompatible
	go.pedge.io/lion v0.0.0-20190619200210-304b2f426641 // indirect
	go.starlark.net v0.0.0-20191218235703-9fcb808a6221
	go.uber.org/zap v1.9.1
	golang.org/x/net v0.0.0-20200324143707-d3edc9973b7e
	golang.org/x/sync v0.0.0-20200317015054-43a5402ce75a
	golang.org/x/tools v0.0.0-20200413015812-1f08ef6002a8
	google.golang.org/genproto v0.0.0-20200413115906-b5235f65be36
	google.golang.org/grpc v1.28.1
	gopkg.in/inconshreveable/log15.v2 v2.0.0-20200109203555-b30bc20e4fd1 // indirect
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
