.PHONY: proto
proto:
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative ./service/service.proto

.PHONY: server
server:
	PORT=5000 go run server/*.go

.PHONY: frontend
frontend:
	go run frontend/*.go

.PHONY: build-server-container
build-server-container:
	docker build -t itudisysmp3server -f Dockerfile.server .

.PHONY: build-client-container
build-client-container:
	docker build -t itudisysmp3client -f Dockerfile.client .

.PHONY: build-all-containers
build-all-containers:
	make build-server-container
	make build-client-container
