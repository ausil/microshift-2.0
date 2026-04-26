PROJECT_NAME := microshift
MODULE := github.com/ausil/microshift-2.0
BINARY := bin/$(PROJECT_NAME)
GO := go
GOFLAGS :=
LDFLAGS := -X $(MODULE)/pkg/version.Version=$(VERSION) \
           -X $(MODULE)/pkg/version.GitCommit=$(GIT_COMMIT)

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

.PHONY: all build test lint clean install uninstall fmt vet

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

e2e:
	bash test/e2e/smoke_test.sh
