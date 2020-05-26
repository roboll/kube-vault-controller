TAG     := $(shell git describe --tags --abbrev=0 HEAD)
PKGS    := $(shell go list ./... | grep -v /vendor/)
PREFIX  := quay.io/roboll

GO_VERSION := 1.7
GO_BUILD_PATH := /go/src/github.com/roboll/kube-vault-controller

generate:
	go generate ${PKGS}
.PHONY: generate

build:
	docker run \
		--rm \
		--volume ${PWD}:${GO_BUILD_PATH} \
		--workdir ${GO_BUILD_PATH} \
		--env GOOS=linux \
		--env GOARCH=amd64 \
		--env CGO_ENABLED=0 \
		golang:${GO_VERSION} \
		go build -a .
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
