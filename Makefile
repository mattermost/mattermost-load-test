.PHONY: install clean

GOFLAGS ?= $(GOFLAGS:)
GO=go


all: install

.installdeps:
	glide cache-clear
	glide update
	touch .installdeps

install: .installdeps
	$(GO) install ./cmd/mcreate
	$(GO) install ./cmd/mmanage
	$(GO) install ./cmd/loadtest

setup:
	./setup.sh

run:
	./run.sh

clean:
	rm -f errors.log cache.db stats.log status.log
	rm -f ./cmd/mmange/mmange
	rm -f ./cmd/mcreate/mcreate
	rm -f ./cmd/loadtest/loadtest
	rm -r .installdeps
