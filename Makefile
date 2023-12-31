.PHONY: install-tools
install-tools:
	go install github.com/bufbuild/buf/cmd/buf@v1.28.1
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
	go install connectrpc.com/connect/cmd/protoc-gen-connect-go@latest

.PHONY: generate-proto
generate-proto:
	# buf lint
	buf generate

.PHONY: build
build:
	go build -o ./bin/koya ./cmd/koya/main.go

.PHONY: build-linux-amd64
build-linux-amd64:
	GOOS=linux GOARCH=amd64 go build -o ./bin/koya-linux-amd64 ./cmd/koya/main.go

.PHONY: deploy
deploy: build-linux-amd64
	./deploy/deploy.sh

.PHONY: docker-up
docker-up:
	docker compose up -d --force-recreate --build
