# This Makefile is meant to be used by people that do not usually work
# with Go source code. If you know what GOPATH is then you probably
# don't need to bother with make.

.PHONY: gwat android ios gwat-cross swarm evm all test clean
.PHONY: gwat-linux gwat-linux-386 gwat-linux-amd64 gwat-linux-mips64 gwat-linux-mips64le
.PHONY: gwat-linux-arm gwat-linux-arm-5 gwat-linux-arm-6 gwat-linux-arm-7 gwat-linux-arm64
.PHONY: gwat-darwin gwat-darwin-386 gwat-darwin-amd64
.PHONY: gwat-windows gwat-windows-386 gwat-windows-amd64

GOBIN = $(shell pwd)/build/bin
GO ?= latest

gwat:
	build/env.sh go run build/ci.go install ./cmd/gwat
	@echo "Done building."
	@echo "Run \"$(GOBIN)/gwat\" to launch gwat."

swarm:
	build/env.sh go run build/ci.go install ./cmd/swarm
	@echo "Done building."
	@echo "Run \"$(GOBIN)/swarm\" to launch swarm."

all:
	build/env.sh go run build/ci.go install

android:
	build/env.sh go run build/ci.go aar --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/gwat.aar\" to use the library."

ios:
	build/env.sh go run build/ci.go xcode --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/Gwat.framework\" to use the library."

test: all
	build/env.sh go run build/ci.go test

clean:
	rm -fr build/_workspace/pkg/ $(GOBIN)/*

# The devtools target installs tools required for 'go generate'.
# You need to put $GOBIN (or $GOPATH/bin) in your PATH to use 'go generate'.

devtools:
	env GOBIN= go get -u golang.org/x/tools/cmd/stringer
	env GOBIN= go get -u github.com/kevinburke/go-bindata/go-bindata
	env GOBIN= go get -u github.com/fjl/gencodec
	env GOBIN= go get -u github.com/golang/protobuf/protoc-gen-go
	env GOBIN= go install ./cmd/abigen
	@type "npm" 2> /dev/null || echo 'Please install node.js and npm'
	@type "solc" 2> /dev/null || echo 'Please install solc'
	@type "protoc" 2> /dev/null || echo 'Please install protoc'

# Cross Compilation Targets (xgo)

gwat-cross: gwat-linux gwat-darwin gwat-windows gwat-android gwat-ios
	@echo "Full cross compilation done:"
	@ls -ld $(GOBIN)/gwat-*

gwat-linux: gwat-linux-386 gwat-linux-amd64 gwat-linux-arm gwat-linux-mips64 gwat-linux-mips64le
	@echo "Linux cross compilation done:"
	@ls -ld $(GOBIN)/gwat-linux-*

gwat-linux-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/386 -v ./cmd/gwat
	@echo "Linux 386 cross compilation done:"
	@ls -ld $(GOBIN)/gwat-linux-* | grep 386

gwat-linux-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/amd64 -v ./cmd/gwat
	@echo "Linux amd64 cross compilation done:"
	@ls -ld $(GOBIN)/gwat-linux-* | grep amd64

gwat-linux-arm: gwat-linux-arm-5 gwat-linux-arm-6 gwat-linux-arm-7 gwat-linux-arm64
	@echo "Linux ARM cross compilation done:"
	@ls -ld $(GOBIN)/gwat-linux-* | grep arm

gwat-linux-arm-5:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-5 -v ./cmd/gwat
	@echo "Linux ARMv5 cross compilation done:"
	@ls -ld $(GOBIN)/gwat-linux-* | grep arm-5

gwat-linux-arm-6:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-6 -v ./cmd/gwat
	@echo "Linux ARMv6 cross compilation done:"
	@ls -ld $(GOBIN)/gwat-linux-* | grep arm-6

gwat-linux-arm-7:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-7 -v ./cmd/gwat
	@echo "Linux ARMv7 cross compilation done:"
	@ls -ld $(GOBIN)/gwat-linux-* | grep arm-7

gwat-linux-arm64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm64 -v ./cmd/gwat
	@echo "Linux ARM64 cross compilation done:"
	@ls -ld $(GOBIN)/gwat-linux-* | grep arm64

gwat-linux-mips:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips --ldflags '-extldflags "-static"' -v ./cmd/gwat
	@echo "Linux MIPS cross compilation done:"
	@ls -ld $(GOBIN)/gwat-linux-* | grep mips

gwat-linux-mipsle:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mipsle --ldflags '-extldflags "-static"' -v ./cmd/gwat
	@echo "Linux MIPSle cross compilation done:"
	@ls -ld $(GOBIN)/gwat-linux-* | grep mipsle

gwat-linux-mips64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips64 --ldflags '-extldflags "-static"' -v ./cmd/gwat
	@echo "Linux MIPS64 cross compilation done:"
	@ls -ld $(GOBIN)/gwat-linux-* | grep mips64

gwat-linux-mips64le:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips64le --ldflags '-extldflags "-static"' -v ./cmd/gwat
	@echo "Linux MIPS64le cross compilation done:"
	@ls -ld $(GOBIN)/gwat-linux-* | grep mips64le

gwat-darwin: gwat-darwin-386 gwat-darwin-amd64
	@echo "Darwin cross compilation done:"
	@ls -ld $(GOBIN)/gwat-darwin-*

gwat-darwin-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=darwin/386 -v ./cmd/gwat
	@echo "Darwin 386 cross compilation done:"
	@ls -ld $(GOBIN)/gwat-darwin-* | grep 386

gwat-darwin-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=darwin/amd64 -v ./cmd/gwat
	@echo "Darwin amd64 cross compilation done:"
	@ls -ld $(GOBIN)/gwat-darwin-* | grep amd64

gwat-windows: gwat-windows-386 gwat-windows-amd64
	@echo "Windows cross compilation done:"
	@ls -ld $(GOBIN)/gwat-windows-*

gwat-windows-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=windows/386 -v ./cmd/gwat
	@echo "Windows 386 cross compilation done:"
	@ls -ld $(GOBIN)/gwat-windows-* | grep 386

gwat-windows-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=windows/amd64 -v ./cmd/gwat
	@echo "Windows amd64 cross compilation done:"
	@ls -ld $(GOBIN)/gwat-windows-* | grep amd64
