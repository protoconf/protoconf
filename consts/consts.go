package consts

var (
	Version = "0.0.1"
)

const (
	AgentDefaultAddress      = ":4300"
	CompiledConfigExtension  = ".materialized_JSON"
	CompiledConfigPath       = "materialized_config/"
	ConfigExtension          = ".pconf"
	EtcdDefaultAddress       = "127.0.0.1:2379"
	MultiConfigExtension     = ".mpconf"
	MutableConfigPath        = "mutable_config/"
	MutableConfigPrefix      = "mutable:"
	OutputsDir               = "outputs/"
	ProtoExtension           = ".proto"
	ServerDefaultAddress     = ":4301"
	SrcPath                  = "src/"
	TemplateExtension        = ".template"
	ValidatorExtensionSuffix = "-validator"
	ZookeeperDefaultAddress  = "127.0.0.1:2181"
)
