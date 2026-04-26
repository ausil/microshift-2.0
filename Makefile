PROJECT_NAME := microshift
MODULE := github.com/ausil/microshift-2.0
BINARY := bin/$(PROJECT_NAME)
GO := go
GOFLAGS := -buildvcs=false
LDFLAGS := -X $(MODULE)/pkg/version.Version=$(VERSION) \
           -X $(MODULE)/pkg/version.GitCommit=$(GIT_COMMIT)

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

.PHONY: all build test lint clean install uninstall fmt vet srpm rpm bootc \
       disk-image disk-image-qcow2 disk-image-raw disk-image-iso disk-image-all

all: build

build:
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/microshift/

test:
	$(GO) test $(GOFLAGS) ./...

lint:
	golangci-lint run ./...

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

clean:
	rm -rf bin/
	$(GO) clean

install: build
	install -d $(DESTDIR)/usr/bin
	install -m 0755 $(BINARY) $(DESTDIR)/usr/bin/$(PROJECT_NAME)
	install -d $(DESTDIR)/etc/microshift
	install -m 0644 packaging/config/config.yaml $(DESTDIR)/etc/microshift/config.yaml
	install -d $(DESTDIR)/usr/lib/systemd/system
	install -m 0644 packaging/systemd/*.service $(DESTDIR)/usr/lib/systemd/system/
	install -d $(DESTDIR)/usr/share/microshift/assets
	cp -r assets/* $(DESTDIR)/usr/share/microshift/assets/

uninstall:
	rm -f $(DESTDIR)/usr/bin/$(PROJECT_NAME)
	rm -rf $(DESTDIR)/etc/microshift
	rm -f $(DESTDIR)/usr/lib/systemd/system/microshift*.service
	rm -rf $(DESTDIR)/usr/share/microshift

srpm:
	mkdir -p _output
	git archive --prefix=microshift-$(VERSION)/ HEAD -o _output/microshift-$(VERSION).tar.gz
	rpmbuild -bs packaging/rpm/microshift.spec \
		--define "_sourcedir $(CURDIR)/_output" \
		--define "_srcrpmdir $(CURDIR)/_output"

rpm: srpm
	mock -r fedora-rawhide-x86_64 --rebuild _output/microshift-$(VERSION)*.src.rpm \
		--resultdir=_output/rpms

bootc: build
	podman build -t microshift-bootc:$(VERSION) -f images/bootc/Containerfile .

DISK_IMAGE_FORMATS ?= qcow2

disk-image: bootc
	bash scripts/build-disk-images.sh --format $(DISK_IMAGE_FORMATS)

disk-image-qcow2: bootc
	bash scripts/build-disk-images.sh --format qcow2

disk-image-raw: bootc
	bash scripts/build-disk-images.sh --format raw

disk-image-iso: bootc
	bash scripts/build-disk-images.sh --format iso

disk-image-all: bootc
	bash scripts/build-disk-images.sh --format all

e2e:
	bash test/e2e/smoke_test.sh
