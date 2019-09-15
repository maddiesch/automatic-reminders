# Directories

# The root directory is the location of this Makefile
ROOT_DIR := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))

# The build directory is where binaries will be built to
BUILD_DIR := ${ROOT_DIR}/build

# The src directory is where your source code lives
SRC_DIR := ${ROOT_DIR}/src

# This is the environment for a lambda binary
GO_LAMBDA_ENV := GOOS=linux GOARCH=amd64

# This can be overriden in the .make-config to use whatever command you prefer
WATCH_COMAND := rerun --no-notify --pattern '*.go' -x

include $(ROOT_DIR)/.make-config

export AWS_PROFILE = $(AWS_SAM_PROFILE)

AWS_SAM_PACKAGE_FILE := $(ROOT_DIR)/package.yml
AWS_SAM_TEMPLATE_FILE := $(ROOT_DIR)/template.yml

.PHONY: all
all: clean test build

.PHONY: build
build:
	cd $(SRC_DIR)/function && $(GO_LAMBDA_ENV) go build -o $(BUILD_DIR)/function .

.PHONY: test
test:
	cd $(SRC_DIR)/function && go test -v ./...

.PHONY: clean
clean:
	rm -rf $(BUILD_DIR)
	-rm $(AWS_SAM_PACKAGE_FILE)

.PHONY: invoke
invoke: clean build
	sam local invoke FunctionHandler --event $(ROOT_DIR)/events/function.json

.PHONY: package
package: build
	sam package --template-file $(AWS_SAM_TEMPLATE_FILE) --output-template-file $(AWS_SAM_PACKAGE_FILE) --s3-bucket ${AWS_SAM_PACKAGE_BUCKET}

.PHONY: deploy
deploy: package
	aws cloudformation deploy --template-file $(AWS_SAM_PACKAGE_FILE) --stack-name $(AWS_CLOUDFORMATION_STACK_NAME) --capabilities CAPABILITY_IAM

.PHONY: watch
watch:
	$(WATCH_COMAND) $(MAKE) build

.PHONY: watch-test
watch-test:
	$(WATCH_COMAND) $(MAKE) test
