.PHONY: build run fmt vet test tidy deb clean

VERSION ?= 0.1.0
DEB_ARCH := $(shell dpkg --print-architecture 2>/dev/null || uname -m)

build:
	go build -o bin/startnow ./cmd/startnow

run:
	go run ./cmd/startnow

fmt:
	go fmt ./...

vet:
	go vet ./...

test:
	go test ./...

tidy:
	go mod tidy

deb: clean
	@test -n "$(VERSION)"
	@mkdir -p dist/deb/DEBIAN dist/deb/usr/bin dist/deb/usr/share/man/man1 \
		dist/deb/usr/share/applications dist/deb/usr/share/icons/hicolor/scalable/apps \
		dist/deb/etc/profile.d dist/deb/etc/fish/conf.d
	CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o dist/deb/usr/bin/startnow ./cmd/startnow
	install -m644 packaging/startnow.1 dist/deb/usr/share/man/man1/startnow.1
	install -m644 packaging/startnow.desktop dist/deb/usr/share/applications/startnow.desktop
	install -m644 packaging/startnow.svg dist/deb/usr/share/icons/hicolor/scalable/apps/startnow.svg
	install -m644 packaging/startnow.sh dist/deb/etc/profile.d/startnow.sh
	install -m644 packaging/startnow.fish dist/deb/etc/fish/conf.d/startnow.fish
	install -m755 packaging/postinst dist/deb/DEBIAN/postinst
	install -m755 packaging/postrm dist/deb/DEBIAN/postrm
	@printf 'Package: startnow\nVersion: %s\nSection: utils\nPriority: optional\nArchitecture: %s\nMaintainer: StartNow Developers <dev@startnow.local>\nDescription: Ninite-style developer tool installer for Linux\n A terminal UI installer for developer tools (Go, Node.js, Rust, Bun,\n LazyGit, ...) with checksum verification, version pinning, updates,\n uninstalls and a live system monitor. Linux only.\n' "$(VERSION)" "$(DEB_ARCH)" > dist/deb/DEBIAN/control
	dpkg-deb --build --root-owner-group dist/deb dist/startnow_$(VERSION)_$(DEB_ARCH).deb
	@rm -rf dist/deb
	@echo "built dist/startnow_$(VERSION)_$(DEB_ARCH).deb"

clean:
	rm -rf bin dist
