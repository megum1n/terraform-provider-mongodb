GOLANG_CI_LINT_VERSION="v1.64.7"

BIN_NAME=terraform-provider-mongodb
CONTAINER_NAME=${BIN_NAME}

CURRENT_DIR=$(shell pwd)
DIST_DIR=${CURRENT_DIR}/bin


.PHONY: all
all: build


.PHONY: build
build:
	go get .
	CGO_ENABLED=0 go build -o ${DIST_DIR}/${BIN_NAME} ./


.PHONY: build.docker
build.docker:
	docker build -t ${CONTAINER_NAME} .


.PHONY: lint.docker
lint.docker:
	docker run -t --rm \
		-v $(shell pwd):/app \
		-w /app \
		golangci/golangci-lint:${GOLANG_CI_LINT_VERSION} \
		golangci-lint run -v


.PHONY: run
run: build
	${DIST_DIR}/${BIN_NAME}


.PHONY: clean
clean:
	go clean
	rm -rf ${DIST_DIR}


.PHONY: docs
docs:
	go get .
	go generate ./...
