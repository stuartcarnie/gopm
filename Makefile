GO_TAGS =
GO_ARGS = -tags "$(GO_TAGS)"
GO_GENERATE=env go generate $(GO_ARGS)
GO_BUILD=env go build $(GO_ARGS)
GO_INSTALL=env go install $(GO_ARGS)

.DEFAULT_GOAL := all

.PHONY: all
all: bin/gopm bin/gopmctl

.PHONY: generate
generate: webgui/js/bundle.js
	go generate ./...

ifeq ($(RELEASE),1)
RELEASE_TAG=release
endif

webgui/js/bundle.js: rpc/javascript/gopm.ts rpc/javascript/service_grpc_web_pb.js rpc/javascript/service_pb.js
	cd rpc/javascript && npx webpack

bin/gopm: GO_TAGS += $(RELEASE_TAG)
bin/gopm: ./cmd/gopm
	$(GO_GENERATE)
	$(GO_BUILD) -o $@ ./$<

bin/gopmctl: ./cmd/gopmctl
	$(GO_BUILD) -o $@ ./$<

.PHONY: install
install: GO_TAGS += release
install:
	$(GO_GENERATE)
	$(GO_INSTALL) ./cmd/gopm
	$(GO_INSTALL) ./cmd/gopmctl

.PHONY: clean
clean:
	rm -rf bin/
