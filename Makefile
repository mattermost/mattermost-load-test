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
	$(GO) install ./cmd/mcreate
	$(GO) install ./cmd/msetup
	$(GO) install ./cmd/mmanage
	$(GO) install ./cmd/loadtest

package: install
	rm -rf $(DIST_ROOT)
	mkdir -p $(DIST_PATH)/bin

	cp $(GOPATH)/bin/msetup $(DIST_PATH)/bin
	cp $(GOPATH)/bin/mmanage $(DIST_PATH)/bin
	cp $(GOPATH)/bin/loadtest $(DIST_PATH)/bin
	cp loadtestconfig.json $(DIST_PATH)
	cp README.md $(DIST_PATH)
	
	tar -C $(DIST_ROOT) -czf $(DIST_PATH).tar.gz $(DIST_FOLDER_NAME)

new-setup: install
	msetup

clean:
	rm -f errors.log cache.db stats.log status.log
	rm -f ./cmd/mmange/mmange
	rm -f ./cmd/mcreate/mcreate
	rm -f ./cmd/mcreate/msetup
	rm -f ./cmd/loadtest/loadtest
	rm -r .installdeps
	rm -rf vendor
	rm -rf $(DIST_ROOT)
