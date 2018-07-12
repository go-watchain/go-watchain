.PHONY: build cluster test

default: build

build:
	go build -o bin/swarm    github.com/watchain/go-watchain/cmd/swarm
	go build -o bin/gwat     github.com/watchain/go-watchain/cmd/gwat
	go build -o bin/bootnode github.com/watchain/go-watchain/cmd/bootnode

cluster: build
	scripts/boot-cluster.sh

test:
	go test -v github.com/watchain/go-watchain/swarm/...
