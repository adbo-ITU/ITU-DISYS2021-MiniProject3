.PHONY: proto
proto:
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative ./service/service.proto

.PHONY: server
server:
	PORT=5000 go run server/*.go

.PHONY: frontend
frontend:
	go run frontend/*.go
