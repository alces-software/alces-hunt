# Copyright (C) 2026 Alces Software Ltd.
# SPDX-License-Identifier: EPL-2.0

PREFIX     ?= /opt/alces-hunt
BINDIR     ?= $(PREFIX)/bin
ETCDIR     ?= $(PREFIX)/etc
SYSTEMDDIR ?= /etc/systemd/system
VERSION    ?= 1.0.0
GO         ?= go
LDFLAGS    := -s -w -X github.com/sierra-tango-echo/alces-hunt/internal/version.Version=$(VERSION)

.PHONY: all build dist test install clean

all: build

build:
	mkdir -p bin
	$(GO) build -trimpath -ldflags "$(LDFLAGS)" -o bin/alces-hunt ./cmd/alces-hunt

# Cross-compiled release binaries (matches GitHub Actions).
dist:
	mkdir -p dist
	@for pair in linux/amd64 linux/arm64; do \
		goos=$${pair%/*}; goarch=$${pair#*/}; \
		name=alces-hunt-$${goos}-$${goarch}; \
		echo "building $${name}"; \
		CGO_ENABLED=0 GOOS=$${goos} GOARCH=$${goarch} $(GO) build -trimpath -ldflags "$(LDFLAGS)" -o dist/$${name} ./cmd/alces-hunt; \
	done

test:
	$(GO) test ./...

install: build
	install -d $(DESTDIR)$(BINDIR) $(DESTDIR)$(ETCDIR) $(DESTDIR)$(PREFIX)/var/buffer $(DESTDIR)$(PREFIX)/var/parsed $(DESTDIR)$(PREFIX)/var/log
	install -m 0755 bin/alces-hunt $(DESTDIR)$(BINDIR)/alces-hunt
	install -m 0755 bin/start $(DESTDIR)$(BINDIR)/start
	if [ ! -f $(DESTDIR)$(ETCDIR)/config.yml ]; then install -m 0644 etc/config.yml.ex $(DESTDIR)$(ETCDIR)/config.yml; fi
	if [ -d $(DESTDIR)$(SYSTEMDDIR) ] || [ -n "$(DESTDIR)" ]; then \
		install -d $(DESTDIR)$(SYSTEMDDIR); \
		install -m 0644 systemd/alces-hunt-server.service $(DESTDIR)$(SYSTEMDDIR)/alces-hunt-server.service; \
		install -m 0644 systemd/alces-hunt-send.service $(DESTDIR)$(SYSTEMDDIR)/alces-hunt-send.service; \
	fi
	install -d $(DESTDIR)/usr/local/bin
	ln -sfn $(BINDIR)/alces-hunt $(DESTDIR)/usr/local/bin/alces-hunt

clean:
	rm -f bin/alces-hunt
	rm -rf dist
