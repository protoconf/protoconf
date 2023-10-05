package protoconf_agent_config

//go:generate protoc -I=. --go_out=.  --go_opt=paths=source_relative agent_config.proto
