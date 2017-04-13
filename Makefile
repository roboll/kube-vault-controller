TAG     := $(shell git describe --tags --abbrev=0 HEAD)
PKGS    := $(shell go list ./... | grep -v /vendor/)
PREFIX  := quay.io/roboll

generate:
	go generate ${PKGS}
.PHONY: generate

deps:
	dep ensure
.PHONY: deps

build:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a .
.PHONY: build

check:
	go vet ${PKGS}
.PHONY: check

test:
	go test -v ${PKGS} -cover -race -p=1
.PHONY: test

image: build
	docker build -t ${PREFIX}/kube-vault-controller:${TAG} .
.PHONY: image

push: image
	docker push ${PREFIX}/kube-vault-controller:${TAG}
.PHONY: push
