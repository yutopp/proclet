.PHONY: install-tools
install-tools:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2

.PHONY: generate-proto
generate-proto:
	protoc \
		--go_out=./pkg --go_opt=paths=source_relative \
		--go-grpc_out=./pkg --go-grpc_opt=paths=source_relative \
		proto/api/v1/*.proto

.PHONY: build
build:
	go build -o ./bin/koya ./cmd/koya/main.go
