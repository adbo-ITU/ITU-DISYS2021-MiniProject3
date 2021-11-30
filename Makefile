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

.PHONY: build
build:
	make build-server-container
	make build-client-container

.PHONY: run
run:
	docker-compose up

.PHONY: clean
clean:
	# Stop the containers
	docker-compose down

	# Removing client containers 
	docker ps --filter "ancestor=itudisysmp3client" -q -a | xargs docker rm

	# Removing client 
	docker ps --filter "ancestor=itudisysmp3server" -q -a | xargs docker rm

	docker rmi itudisysmp3server && docker rmi itudisysmp3client || true

.PHONY: simulate-crash
simulate-crash:
	# Killing the second server
	docker ps -a --filter "label=itudidsysmp3.app" | grep server2 | awk '{ print $1 }' | xargs docker kill || true

	# Waiting
	sleep 5

	docker compose restart auctionserver2

