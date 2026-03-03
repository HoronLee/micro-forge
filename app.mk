# Makefile for building servora micro service application
# This is a common Makefile template for all services in app/ directory

MKFILE_PATH := $(abspath $(lastword $(MAKEFILE_LIST)))
MKFILE_DIR  := $(dir $(MKFILE_PATH))
ENV_FILE    := $(MKFILE_DIR).env

# load environment variables from .env file if it exists
ifneq (,$(wildcard $(ENV_FILE)))
    include $(ENV_FILE)
    export
endif

GOPATH ?= $(shell go env GOPATH)
# GOVERSION is the current go version, e.g. go1.23.4
GOVERSION ?= $(shell go version | awk '{print $$3;}')

# Ensure GOPATH is set before running build process.
ifeq "$(GOPATH)" ""
  $(error Please set the environment variable GOPATH before running `make`)
endif

FAIL_ON_STDOUT := awk '{ print } END { if (NR > 0) { exit 1 } }'

GO_CMD     := GO111MODULE=on go
GIT_CMD    := git
DOCKER_CMD := docker

ARCH   := "`uname -s`"
LINUX  := "Linux"
MAC    := "Darwin"

DEFAULT_VERSION ?= $(SERVICE_APP_VERSION)

ifeq ($(OS),Windows_NT)
    IS_WINDOWS := TRUE
endif

ifneq (git,)
	GIT_EXIST := TRUE
endif

ifneq ("$(wildcard .git)", "")
	HAS_DOTGIT := TRUE
endif

ifeq ($(GIT_EXIST),TRUE)
ifeq ($(HAS_DOTGIT),TRUE)
	# CUR_TAG is the last git tag plus the delta from the current commit to the tag
	# e.g. v1.5.5-<nr of commits since>-g<current git sha>
	CUR_TAG ?= $(shell git describe --tags --first-parent 2>/dev/null || echo "dev")

	# LAST_TAG is the last git tag
    # e.g. v1.5.5
    LAST_TAG ?= $(shell git describe --match "v*" --abbrev=0 --tags --first-parent 2>/dev/null || echo "v0.0.1")

    # VERSION is the last git tag without the 'v'
    # e.g. 1.5.5
    VERSION ?= $(shell git describe --match "v*" --abbrev=0 --tags --first-parent 2>/dev/null | cut -c 2- || echo "0.0.1")
endif
endif

CUR_TAG  ?= $(DEFAULT_VERSION)
LAST_TAG ?= v$(DEFAULT_VERSION)
VERSION  ?= $(DEFAULT_VERSION)

# GOFLAGS is the flags for the go compiler.
LDFLAGS ?= -X main.version=$(VERSION)
GOFLAGS ?=

APP_RELATIVE_PATH := $(shell a=`basename $$PWD` && cd .. && b=`basename $$PWD` && echo $$b/$$a)
SERVICE_NAME      := $(shell a=`basename $$PWD` && cd .. && b=`basename $$PWD` && echo $$b)
APP_NAME          := $(shell echo $(APP_RELATIVE_PATH) | sed -En "s/\//-/p")

# Detect service-specific OpenAPI config file
# Format: buf.{service_name}.openapi.gen.yaml
OPENAPI_CONFIG := buf.$(SERVICE_NAME).openapi.gen.yaml
CONF ?= ./configs/config.yaml
RUN_DEPS ?= api openapi

.PHONY: build clean gen wire api openapi run app help env gen.gorm gen.ent

# show environment variables
env:
	@echo "GOPATH: $(GOPATH)"
	@echo "GOVERSION: $(GOVERSION)"
	@echo "GOFLAGS: $(GOFLAGS)"
	@echo "LDFLAGS: $(LDFLAGS)"
	@echo "PROJECT_NAME: $(PROJECT_NAME)"
	@echo "SERVICE_APP_VERSION: $(SERVICE_APP_VERSION)"
	@echo "APP_RELATIVE_PATH: $(APP_RELATIVE_PATH)"
	@echo "SERVICE_NAME: $(SERVICE_NAME)"
	@echo "APP_NAME: $(APP_NAME)"
	@echo "CUR_TAG: $(CUR_TAG)"
	@echo "LAST_TAG: $(LAST_TAG)"
	@echo "VERSION: $(VERSION)"
	@echo "OPENAPI_CONFIG: $(OPENAPI_CONFIG)"

# build golang application
build: api openapi
ifneq ("$(wildcard ./cmd)","")
	@go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o ./bin/ ./...
else
	@echo "No cmd directory found, skipping build for $(SERVICE_NAME)"
endif

# build golang application only
build_only:
ifneq ("$(wildcard ./cmd)","")
	@go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o ./bin/ ./...
else
	@echo "No cmd directory found, skipping build for $(SERVICE_NAME)"
endif

# run application
run: $(RUN_DEPS)
	-@go run $(GOFLAGS) -ldflags "$(LDFLAGS)" ./cmd/server -conf $(CONF)

# build service app
app: api openapi wire build

# clean build files
clean:
	@go clean
	$(if $(IS_WINDOWS), del "coverage.out", rm -f "coverage.out")
	@rm -f openapi.yaml

# generate code
gen: wire api openapi gen.ent

# generate GORM GEN PO and DAO code via centralized svr CLI
gen.gorm:
	@echo "Generating GORM DAO/PO..."
	@cd $(MKFILE_DIR) && go run ./cmd/svr gen gorm $(SERVICE_NAME)

# generate Ent code, if internal/data/generate.go exists
gen.ent:
ifneq ("$(wildcard ./internal/data/generate.go)","")
	@go generate ./internal/data
endif

# generate wire code
wire:
ifneq ("$(wildcard ./cmd/server)","")
	@go run -mod=mod github.com/google/wire/cmd/wire ./cmd/server
else
	@echo "No cmd/server directory found, skipping wire for $(SERVICE_NAME)"
endif

# generate protobuf api code
api:
	@cd ../../../api && \
	buf generate

# generate protobuf api OpenAPI v3 docs
openapi:
	@cd ../../../api && \
	buf generate --template $(OPENAPI_CONFIG)

# show help
help:
	@echo ""
	@echo "Usage:"
	@echo " make [target]"
	@echo ""
	@echo "Targets:"
	@awk '/^[a-zA-Z\-_0-9]+:/ { \
	helpMessage = match(lastLine, /^# (.*)/); \
		if (helpMessage) { \
			helpCommand = substr($$1, 0, index($$1, ":")-1); \
			helpMessage = substr(lastLine, RSTART + 2, RLENGTH); \
			printf "\033[36m%-22s\033[0m %s\n", helpCommand,helpMessage; \
		} \
	} \
	{ lastLine = $$0 }' $(MAKEFILE_LIST)

.DEFAULT_GOAL := help
