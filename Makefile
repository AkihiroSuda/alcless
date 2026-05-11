# Files are installed under $(DESTDIR)/$(PREFIX)
PREFIX ?= /usr/local
DEST := $(shell echo "$(DESTDIR)/$(PREFIX)" | sed 's:///*:/:g; s://*$$::')

VERSION ?=$(shell git describe --match 'v[0-9]*' --dirty='.m' --always --tags)
VERSION_TRIMMED := $(VERSION:v%=%)
VERSION_SYMBOL := github.com/AkihiroSuda/alcless/cmd/alclessctl/version.Version

export SOURCE_DATE_EPOCH ?= $(shell git log -1 --pretty=%ct)
# BSD date (macOS) uses `-r EPOCH`; GNU date (Linux) uses `-d @EPOCH`.
SOURCE_DATE_EPOCH_TOUCH := $(shell date -r $(SOURCE_DATE_EPOCH) +%Y%m%d%H%M.%S 2>/dev/null || date -d @$(SOURCE_DATE_EPOCH) +%Y%m%d%H%M.%S)

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
# BSD tar (macOS) and GNU tar (Linux) use different flags for reproducible
# archives. `--option !timestamp` and `gzip -n` both omit the gzip header mtime.
ifeq ($(shell $(TAR) --version 2>/dev/null | head -1 | grep -c bsdtar),1)
	TAR_REPRODUCIBLE_FLAGS := --uid 0 --gid 0 --option !timestamp -czvf
else
	TAR_REPRODUCIBLE_FLAGS := --owner=0 --group=0 --use-compress-program='gzip -n' -cvf
endif

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
	GOOS=$(1) GOARCH=$(2) make
	$(call touch_recursive,_output)
	$(TAR) -C _output/ --no-xattrs --numeric-owner $(TAR_REPRODUCIBLE_FLAGS) _artifacts/alcless-$(VERSION_TRIMMED)-$(3)-$(4).tar.gz ./
endef

.PHONY: artifacts
artifacts:
	rm -rf _artifacts
	mkdir -p _artifacts
	$(call make_artifact,darwin,amd64,Darwin,amd64)
	$(call make_artifact,darwin,arm64,Darwin,arm64)
	$(call make_artifact,linux,amd64,Linux,x86_64)
	$(call make_artifact,linux,arm64,Linux,aarch64)
	make clean
	(cd _artifacts ; sha256sum *) > SHA256SUMS
	mv SHA256SUMS _artifacts/SHA256SUMS
	$(call touch_recursive,_artifacts)
