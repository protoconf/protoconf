#!/usr/bin/env bash
set -e

PROTOBUF_INCLUDE_DIR="/usr/include/google/protobuf/"
if [ ! -d "$PROTOBUF_INCLUDE_DIR" ]; then
    echo "Protobuf include files can't be found at $PROTOBUF_INCLUDE_DIR" >&2
    echo "Try installing libprotobuf-dev" >&2
    exit 1
fi

cd "$(dirname "$0")"

consul_path=${1:-"my_client_config"}
my_client_config_value=${2:-"$(date)"}

my_client_config_message=$(echo 'value: "'"$my_client_config_value"'"'| protoc -I../ ../protoconf/src/my_client/client-config.proto --encode=ClientConfig | sed -z 's/\n/\\n/g')
protoconf_value_message=$(echo 'value: {type_url: "type.googleapis.com/ClientConfig", value: "'"$my_client_config_message"'"}'| protoc -I../../ ../../types/proto/v1/protoconf-value.proto --encode=v1.ProtoconfValue)
consul kv put "$consul_path" "$protoconf_value_message"
