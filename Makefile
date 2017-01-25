.PHONY: install clean

GOFLAGS ?= $(GOFLAGS:)
GO=go

DIST_ROOT=dist
DIST_FOLDER_NAME=mattermost-load-test
DIST_PATH=$(DIST_ROOT)/$(DIST_FOLDER_NAME)


all: install

.installdeps:
	glide cache-clear
	glide update
	touch .installdeps

install: .installdeps
	$(GO) install ./cmd/loadtest

package: install
	rm -rf $(DIST_ROOT)
	mkdir -p $(DIST_PATH)/bin

	cp $(GOPATH)/bin/loadtest $(DIST_PATH)/bin
	cp loadtestconfig.json $(DIST_PATH)
	cp README.md $(DIST_PATH)
	cp -r testfiles $(DIST_PATH)
	
	tar -C $(DIST_ROOT) -czf $(DIST_PATH).tar.gz $(DIST_FOLDER_NAME)

clean:
	rm -f errors.log cache.db stats.log status.log
	rm -f ./cmd/loadtest/loadtest
	rm -f .installdeps
	rm -f loadtest.log
	rm -rf vendor
	rm -rf $(DIST_ROOT)
