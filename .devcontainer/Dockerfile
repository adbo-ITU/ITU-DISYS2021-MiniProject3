# See here for image contents: https://github.com/microsoft/vscode-dev-containers/tree/v0.202.1/containers/go/.devcontainer/base.Dockerfile

ARG VARIANT="1.17-bullseye"
FROM mcr.microsoft.com/vscode/devcontainers/go:0-${VARIANT}

RUN apt-get update \
    && apt install -y protobuf-compiler

RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest \
    && go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
