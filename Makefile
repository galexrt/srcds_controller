PROJECTNAME ?= srcds_controller
DESCRIPTION ?= srcds_controller - SRCDS Controller currently only for checking a SRCDS server using RCON.
MAINTAINER  ?= Alexander Trost <galexrt@googlemail.com>
HOMEPAGE    ?= https://github.com/galexrt/srcds_controller

GO111MODULE  ?= on
GO           := go
FPM          ?= fpm
PROMU        := $(GOPATH)/bin/promu
PREFIX       ?= $(shell pwd)
BIN_DIR      ?= $(PREFIX)/.build
TARBALL_DIR  ?= $(PREFIX)/.tarball
PACKAGE_DIR  ?= $(PREFIX)/.package
ARCH         ?= amd64
PACKAGE_ARCH ?= linux-amd64
VERSION      ?= $(shell cat VERSION)

# The GOHOSTARM and PROMU parts have been taken from the prometheus/promu repository
# which is licensed under Apache License 2.0 Copyright 2018 The Prometheus Authors
FIRST_GOPATH := $(firstword $(subst :, ,$(shell $(GO) env GOPATH)))

GOHOSTOS     ?= $(shell $(GO) env GOHOSTOS)
GOHOSTARCH   ?= $(shell $(GO) env GOHOSTARCH)

ifeq (arm, $(GOHOSTARCH))
	GOHOSTARM ?= $(shell GOARM= $(GO) env GOARM)
	GO_BUILD_PLATFORM ?= $(GOHOSTOS)-$(GOHOSTARCH)v$(GOHOSTARM)
else
	GO_BUILD_PLATFORM ?= $(GOHOSTOS)-$(GOHOSTARCH)
endif

PROMU_VERSION ?= 0.5.0
PROMU_URL     := https://github.com/prometheus/promu/releases/download/v$(PROMU_VERSION)/promu-$(PROMU_VERSION).$(GO_BUILD_PLATFORM).tar.gz
# END copied code

pkgs = $(shell go list ./... | grep -v /vendor/ | grep -v /test/)

DOCKER_IMAGE_NAME ?= galexrt/srcds_controller
DOCKER_IMAGE_TAG  ?= runner-$(subst /,-,$(shell git rev-parse --abbrev-ref HEAD))

all: format style vet test build

build: promu
	@echo ">> building binaries"
	GO111MODULE=$(GO111MODULE) $(PROMU) build --prefix $(PREFIX) $(PROMU_BINARIES)

crossbuild: promu
	@$(PROMU) crossbuild

docker:
	@echo ">> building docker image"
	docker build -t "$(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_TAG)" .

format:
	go fmt $(pkgs)

package-%: build
	mkdir -p -m0755 $(PACKAGE_DIR)/lib/systemd/system $(PACKAGE_DIR)/usr/bin
	mkdir -p $(PACKAGE_DIR)/etc/sysconfig
	cp .build/srcds_controller $(PACKAGE_DIR)/usr/bin
	cp systemd/srcds_controller.service $(PACKAGE_DIR)/lib/systemd/system
	cp systemd/sysconfig.srcds_controller $(PACKAGE_DIR)/etc/sysconfig/srcds_controller
	cd $(PACKAGE_DIR) && $(FPM) -s dir -t $(patsubst package-%, %, $@) \
	--deb-user root --deb-group root \
	--name $(PROJECTNAME) \
	--version $(VERSION) \
	--architecture $(PACKAGE_ARCH) \
	--description "$(DESCRIPTION)" \
	--maintainer "$(MAINTAINER)" \
	--url $(HOMEPAGE) \
	usr/ etc/

promu:
	$(eval PROMU_TMP := $(shell mktemp -d))
	curl -s -L $(PROMU_URL) | tar -xvzf - -C $(PROMU_TMP)
	mkdir -p $(FIRST_GOPATH)/bin
	cp $(PROMU_TMP)/promu-$(PROMU_VERSION).$(GO_BUILD_PLATFORM)/promu $(FIRST_GOPATH)/bin/promu
	rm -r $(PROMU_TMP)

srcds_run:
	$(GO) build -o srcds_run ./srcds_run.go
	chmod 755 srcds_run

style:
	@echo ">> checking code style"
	@! gofmt -d $(shell find . -path ./vendor -prune -o -name '*.go' -print) | grep '^'

tarball: promu
	@echo ">> building release tarball"
	@$(PROMU) tarball --prefix $(PREFIX) $(BIN_DIR)

test:
	@$(GO) test $(pkgs)

test-short:
	@echo ">> running short tests"
	@$(GO) test -short $(pkgs)

vet:
	@echo ">> vetting code"
	@$(GO) vet $(pkgs)

.PHONY: all build crossbuild docker format package promu style tarball test vet
