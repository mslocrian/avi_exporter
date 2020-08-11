OUT = avi_exporter
GO           ?= go
GOFMT        ?= $(GO)fmt
FIRST_GOPATH := $(firstword $(subst :, ,$(shell $(GO) env GOPATH)))
GOOPTS       ?=
GOHOSTOS     ?= $(shell $(GO) env GOHOSTOS)
GOHOSTARCH   ?= $(shell $(GO) env GOHOSTARCH)
GO_VERSION        ?= $(shell $(GO) version)
GO_VERSION_NUMBER ?= $(word 3, $(GO_VERSION))
PRE_GO_111        ?= $(shell echo $(GO_VERSION_NUMBER) | grep -E 'go1\.(10|[0-9])\.')

GOVENDOR :=
GO111MODULE :=
pkgs = $(shell go list ./... | egrep -v "(vendor|gen)")
PROMU        := $(FIRST_GOPATH)/bin/promu

PREFIX                  ?= $(shell pwd)
BIN_DIR                 ?= $(shell pwd)

export DOCKERHUB_USER = $(or $(DEV_DOCKERHUB_REPO), mslocrian)
export DOCKERHUB_REPO = $(OUT)
export DOCKERHUB_VERSION = 0.2.11

GO_BUILD_PLATFORM ?= $(GOHOSTOS)-$(GOHOSTARCH)
PROMU_VERSION ?= 0.5.0
PROMU_URL     := https://github.com/prometheus/promu/releases/download/v$(PROMU_VERSION)/promu-$(PROMU_VERSION).$(GO_BUILD_PLATFORM).tar.gz

.PHONY: default
default:
	$(MAKE) clean
	$(MAKE) build-all

.PHONY: clean
clean:
	rm -rf avi_exporter

.PHONY: build-local
build-local: promu
	@echo ">> building binaries"
	GO111MODULE=$(GO111MODULE) $(PROMU) -v build --prefix $(PREFIX) $(PROMU_BINARIES)
#	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -tags netgo -ldflags '-w' -o target/linux/$(OUT)

.PHONY: build
build: format
	@docker build -f Dockerfile -t $(DOCKERHUB_REPO):$(DOCKERHUB_VERSION) .

.PHONY: push
push: DOCKER_IMAGE_ID = $(shell docker images -q $(DOCKERHUB_REPO):$(DOCKERHUB_VERSION))
push:
	docker tag $(DOCKER_IMAGE_ID) $(DOCKERHUB_USER)/$(DOCKERHUB_REPO):latest
	docker push $(DOCKERHUB_USER)/$(DOCKERHUB_REPO):latest
	docker tag $(DOCKER_IMAGE_ID) $(DOCKERHUB_USER)/$(DOCKERHUB_REPO):$(DOCKERHUB_VERSION)
	docker push $(DOCKERHUB_USER)/$(DOCKERHUB_REPO):$(DOCKERHUB_VERSION)

.PHONY: all
all:
	$(MAKE) clean
	$(MAKE) build-all

.PHONY: format
format:
	@echo ">> formatting code"
	@go fmt $(pkgs)

.PHONY: lint
lint:
	@echo ">> linting go files"
	@golint $(pkgs)

.PHONY: vet
vet:
	@echo ">> vetting go files"
	@go vet $(pkgs)

.PHONY: promu
promu: $(PROMU)

$(PROMU):
	$(eval PROMU_TMP := $(shell mktemp -d))
	curl -s -L $(PROMU_URL) | tar -xvzf - -C $(PROMU_TMP)
	mkdir -p $(FIRST_GOPATH)/bin
	cp $(PROMU_TMP)/promu-$(PROMU_VERSION).$(GO_BUILD_PLATFORM)/promu $(FIRST_GOPATH)/bin/promu
	rm -r $(PROMU_TMP)
