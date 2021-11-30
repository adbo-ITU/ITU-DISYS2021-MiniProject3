# Mini Project 3

## Requirements

In order to run this project, you will need to install the following dependencies:

- Docker
- docker-compose (included with Docker)
- GNU/Make

The GNU/Make tool is used to abstract away the build commands, such that they are shorter and easier to type in the CLI.

## Usage

To run the system, run the following command:

```bash
make run
```

## Simulate a replica manager crash

We leverage the docker and docker compose tool to simulate a replica manager crash and a subsequent reboot.

```bash
make simulate-crash
```

This kills the second replica manager and restarts it after 5 seconds.

## Development

To further develop this on your machine, you need to have Golang binaries installed natively, otherwise you can use the VS Code `devcontainer` that is provided within this project.

### Building the server docker image

```bash
make build-server-container
```

### Building the client docker image

```bash
make build-client-container
```

### Shorthand: build both images

```bash
make build
```

## Cleaning up

You can remove the containers with the built images from your local Docker installation

```bash
make clean
```
