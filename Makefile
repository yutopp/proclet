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

.PHONY: build-linux-amd64
build-linux-amd64:
	GOOS=linux GOARCH=amd64 go build -o ./bin/koya-linux-amd64 ./cmd/koya/main.go

.PHONY: deploy
deploy: build-linux-amd64
	./deploy/deploy.sh

.PHONY: envoy
envoy:
	docker compose up envoy --force-recreate --build

.PHONY: prod
prod:
	docker compose up -d --force-recreate --build --env-file .env.prod
