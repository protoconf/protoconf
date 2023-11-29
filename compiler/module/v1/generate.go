package module

//go:generate protoc -I=. --go_out=.  --go_opt=paths=source_relative remote_repo.proto
