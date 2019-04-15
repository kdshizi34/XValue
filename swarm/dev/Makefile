.PHONY: build cluster test

default: build

build:
	go build -o bin/swarm    github.com/xvalue/go-xvalue/cmd/swarm
	go build -o bin/geth     github.com/xvalue/go-xvalue/cmd/geth
	go build -o bin/bootnode github.com/xvalue/go-xvalue/cmd/bootnode

cluster: build
	scripts/boot-cluster.sh

test:
	go test -v github.com/xvalue/go-xvalue/swarm/...
