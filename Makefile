GO_TAGS =
GO = env go

.DEFAULT_GOAL := install

ifeq ($(GOPM_DEV),1)
	DEV_TAG=dev
endif

.PHONY: install
install:
	mkdir -p bin
	$(GO) build -tags "$(DEV_TAG)" -o bin ./...

# The install-dev target builds an executable that reads the webgui
# directory from disk rather than building it as an asset into the binary.
.PHONY: install-dev
install-dev:
	mkdir -p bin
	$(GO) build -tags dev -o bin ./...

.PHONY: test
test:
	$(GO) test ./...

.PHONY: all
all: generate install

.PHONY: generate
generate: go-generate webgui/js/bundle.js
	true

.PHONY: go-generate
go-generate:
	$(GO) generate ./...

.PHONY: clean
clean:
	rm -rf bin/

webgui/js/bundle.js: rpc/javascript/gopm.ts rpc/javascript/service_grpc_web_pb.js rpc/javascript/service_pb.js
	cd rpc/javascript && npx webpack
