package protoconf

//go:generate sh -xc "find . -name '*.proto' -not -path '*pb/*' -not -path '*utils/*' | xargs protoc -I=. --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative"
