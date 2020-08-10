OUT = avi_exporter
pkgs = $(shell go list ./... | egrep -v "(vendor|gen)")

export DOCKERHUB_USER = $(or $(DEV_DOCKERHUB_REPO), mslocrian)
export DOCKERHUB_REPO = avi_exporter
export DOCKERHUB_VERSION = 0.2.6

default:
	$(MAKE) clean
	$(MAKE) build-all

clean:
	rm -rf target

build-local:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -tags netgo -ldflags '-w' -o target/linux/$(OUT)

build: format
	@docker build -f Dockerfile -t $(DOCKERHUB_REPO):$(DOCKERHUB_VERSION) .

push: DOCKER_IMAGE_ID = $(shell docker images -q $(DOCKERHUB_REPO):$(DOCKERHUB_VERSION))
push:
	docker tag $(DOCKER_IMAGE_ID) $(DOCKERHUB_USER)/$(DOCKERHUB_REPO):latest
	docker push $(DOCKERHUB_USER)/$(DOCKERHUB_REPO):latest
	docker tag $(DOCKER_IMAGE_ID) $(DOCKERHUB_USER)/$(DOCKERHUB_REPO):$(DOCKERHUB_VERSION)
	docker push $(DOCKERHUB_USER)/$(DOCKERHUB_REPO):$(DOCKERHUB_VERSION)


all:
	$(MAKE) clean
	$(MAKE) build-all

format:
	@echo ">> formatting code"
	@go fmt $(pkgs)

lint:
	@echo ">> linting go files"
	@golint $(pkgs)

vet:
	@echo ">> vetting go files"
	@go vet $(pkgs)
