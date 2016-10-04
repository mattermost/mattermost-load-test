.PHONY: install

GOFLAGS ?= $(GOFLAGS:)
GO=go


all: install

install:
	$(GO) install ./cmd/mcreate
	$(GO) install ./cmd/mmanage
	$(GO) install ./cmd/loadtest

clean:
	rm -f errors.log cache.db stats.log
	rm -f ./cmd/mmange/mmange
	rm -f ./cmd/mcreate/mcreate
	rm -f ./cmd/loadtest/loadtest
