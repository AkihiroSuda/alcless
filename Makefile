# Files are installed under $(DESTDIR)/$(PREFIX)
PREFIX ?= /usr/local
DEST := $(shell echo "$(DESTDIR)/$(PREFIX)" | sed 's:///*:/:g; s://*$$::')

VERSION ?=$(shell git describe --match 'v[0-9]*' --dirty='.m' --always --tags)
VERSION_SYMBOL := github.com/AkihiroSuda/alcless/cmd/alclessctl/version.Version

export SOURCE_DATE_EPOCH ?= $(shell git log -1 --pretty=%ct)
SOURCE_DATE_EPOCH_TOUCH := $(shell date -r $(SOURCE_DATE_EPOCH) +%Y%m%d%H%M.%S)

GO ?= go
# Keep symbols by default, for supporting gomodjail
# https://github.com/AkihiroSuda/gomodjail
KEEP_SYMBOLS ?= 1
GO_BUILD_LDFLAGS_S := true
ifeq ($(KEEP_SYMBOLS),1)
	GO_BUILD_LDFLAGS_S = false
endif
GO_BUILD_LDFLAGS ?= -s=$(GO_BUILD_LDFLAGS_S) -w -X $(VERSION_SYMBOL)=$(VERSION)
GO_BUILD ?= $(GO) build -trimpath -ldflags="$(GO_BUILD_LDFLAGS)"
GOOS ?= $(shell $(GO) env GOOS)
GOARCH ?= $(shell $(GO) env GOARCH)

BINARIES := _output/bin/alclessctl _output/bin/alcless

TAR ?= tar

.PHONY: all
all: binaries

.PHONY: binaries
binaries: $(BINARIES)

.PHONY: _output/bin/alclessctl
_output/bin/alclessctl:
	$(GO_BUILD) -o "$@" ./cmd/alclessctl

.PHONY: _output/bin/alcless
_output/bin/alcless:
	cp -a ./cmd/alcless "$@"

.PHONY: install
install: uninstall
	mkdir -p "$(DEST)/bin"
	cp -a _output/bin/alclessctl "$(DEST)/bin/alclessctl"
	cp -a _output/bin/alcless "$(DEST)/bin/alcless"

.PHONY: uninstall
uninstall:
	rm -f "$(DEST)/bin/alclessctl" "$(DEST)/bin/alcless"


# clean does not remove _artifacts
.PHONY: clean
clean:
	rm -rf _output

define touch_recursive
	find "$(1)" -exec touch -t $(SOURCE_DATE_EPOCH_TOUCH) {} +
endef

define make_artifact
	make clean
	GOARCH=$(1) make
	$(call touch_recursive,_output)
	$(TAR) -C _output/ --no-xattrs --numeric-owner --uid 0 --gid 0 --option !timestamp -czvf _artifacts/alcless-$(VERSION).$(1).tar.gz ./
endef

# Needs to be executed on macOS
.PHONY: artifacts
artifacts:
	rm -rf _artifacts
	mkdir -p _artifacts
	$(call make_artifact,amd64)
	$(call make_artifact,arm64)
	make clean
	go version | tee _artifacts/build-env.txt
	echo --- >> _artifacts/build-env.txt
	sw_vers | tee -a _artifacts/build-env.txt
	echo --- >> _artifacts/build-env.txt
	pkgutil --pkg-info=com.apple.pkg.CLTools_Executables | tee -a _artifacts/build-env.txt
	echo --- >> _artifacts/build-env.txt
	$(CC) --version | tee -a _artifacts/build-env.txt
	(cd _artifacts ; sha256sum *) > SHA256SUMS
	mv SHA256SUMS _artifacts/SHA256SUMS
	$(call touch_recursive,_artifacts)
